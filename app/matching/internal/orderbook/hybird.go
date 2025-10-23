package orderbook

import (
	"sync"
	"time"
	"unsafe"

	"github.com/tsfdsong/tradeengin/app/matching/internal/monitor"
	"github.com/tsfdsong/tradeengin/app/pkg/lockfree"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// HybridOrderBook 高性能混合订单簿（使用跳表）
type HybridOrderBook struct {
	symbol   string
	buys     *SkipTree                        // 买盘 - 价格降序
	sells    *SkipTree                        // 卖盘 - 价格升序
	orderMap *sync.Map                        // orderID -> *Order
	seqMaps  map[float64]*lockfree.RingBuffer // 同价位订单队列
	mu       sync.RWMutex
	version  uint64
	depth    int
	stats    *OrderBookStats
}

// OrderBookStats 订单簿统计
type OrderBookStats struct {
	OrdersProcessed uint64
	TradesExecuted  uint64
	AvgLatency      time.Duration
	LastUpdate      time.Time
}

// NewHybridOrderBook 创建混合订单簿
func NewHybridOrderBook(symbol string) *HybridOrderBook {
	ob := &HybridOrderBook{
		symbol:   symbol,
		buys:     NewSkipTree(16, true),  // 买盘降序，最大16层
		sells:    NewSkipTree(16, false), // 卖盘升序，最大16层
		orderMap: &sync.Map{},
		seqMaps:  make(map[float64]*lockfree.RingBuffer),
		stats:    &OrderBookStats{},
		depth:    1000, // 默认深度
	}

	return ob
}

// Match 订单撮合
func (h *HybridOrderBook) Match(order *types.Order) *types.MatchResult {
	startTime := time.Now()

	h.mu.Lock()
	defer h.mu.Unlock()

	result := types.GetMatchResultFromPool()
	result.Order = order
	result.Timestamp = time.Now().UnixNano()

	var trades []*types.Trade
	remainingQty := order.Quantity

	if order.Side == types.SideBuy {
		// 买单匹配卖盘
		trades, remainingQty = h.matchBuyOrder(order, remainingQty)
	} else {
		// 卖单匹配卖盘
		trades, remainingQty = h.matchSellOrder(order, remainingQty)
	}

	result.Trades = trades

	// 剩余数量放入订单簿（限价单）
	if remainingQty > 0 && order.Type == types.TypeLimit {
		h.addOrderToBook(order, remainingQty)
	}

	// 更新统计
	h.updateStats(len(trades), time.Since(startTime))

	// 记录监控指标
	monitor.RecordOrderMatched(h.symbol, len(trades))
	if len(trades) > 0 {
		for _, trade := range trades {
			monitor.RecordTrade(h.symbol, trade.Quantity, trade.Price)
		}
	}

	h.version++
	return result
}

// matchBuyOrder 买单撮合逻辑
func (h *HybridOrderBook) matchBuyOrder(order *types.Order, remainingQty int64) ([]*types.Trade, int64) {
	var trades []*types.Trade

	for remainingQty > 0 && h.sells.Len() > 0 {
		bestAsk := h.sells.MinPriceNode()
		if bestAsk == nil {
			break
		}

		// 检查价格是否匹配
		if order.Type == types.TypeLimit && order.Price < bestAsk.Price {
			break // 限价单价格不匹配
		}

		// 计算匹配数量
		matchedQty := min(remainingQty, bestAsk.TotalQty)
		if matchedQty <= 0 {
			break
		}

		// 执行交易
		trade := h.executeTrade(order, bestAsk, matchedQty, bestAsk.Price)
		trades = append(trades, trade)

		// 更新数量
		remainingQty -= matchedQty
		bestAsk.TotalQty -= matchedQty

		// 移除已完全成交的价格层级
		if bestAsk.TotalQty == 0 {
			h.sells.Remove(bestAsk.Price)
			delete(h.seqMaps, bestAsk.Price)
		} else {
			// 部分成交，更新价格层级
			h.updatePriceLevel(bestAsk, -matchedQty)
		}

		// 记录交易
		h.recordTrade(trade)
	}

	return trades, remainingQty
}

// matchSellOrder 卖单撮合逻辑
func (h *HybridOrderBook) matchSellOrder(order *types.Order, remainingQty int64) ([]*types.Trade, int64) {
	var trades []*types.Trade

	for remainingQty > 0 && h.buys.Len() > 0 {
		bestBid := h.buys.MaxPriceNode()
		if bestBid == nil {
			break
		}

		// 检查价格是否匹配
		if order.Type == types.TypeLimit && order.Price > bestBid.Price {
			break // 限价单价格不匹配
		}

		// 计算匹配数量
		matchedQty := min(remainingQty, bestBid.TotalQty)
		if matchedQty <= 0 {
			break
		}

		// 执行交易
		trade := h.executeTrade(order, bestBid, matchedQty, bestBid.Price)
		trades = append(trades, trade)

		// 更新数量
		remainingQty -= matchedQty
		bestBid.TotalQty -= matchedQty

		// 移除已完全成交的价格层级
		if bestBid.TotalQty == 0 {
			h.buys.Remove(bestBid.Price)
			delete(h.seqMaps, bestBid.Price)
		} else {
			// 部分成交，更新价格层级
			h.updatePriceLevel(bestBid, -matchedQty)
		}

		// 记录交易
		h.recordTrade(trade)
	}

	return trades, remainingQty
}

// executeTrade 执行交易
func (h *HybridOrderBook) executeTrade(taker *types.Order, makerLevel *PriceLevel, qty int64, price float64) *types.Trade {
	// 从价格层级中获取最早的同价位订单
	makerOrder := h.getEarliestOrderFromLevel(makerLevel.Price)
	if makerOrder == nil {
		// 如果没有找到具体订单，创建一个虚拟的maker订单
		makerOrder = &types.Order{
			ID:       generateTradeID(),
			Symbol:   h.symbol,
			Price:    price,
			Quantity: qty,
			Side:     1 - taker.Side, // 相反方向
		}
	}

	trade := types.GetTradeFromPool()
	trade.TradeID = generateTradeID()
	trade.TakerOrderID = taker.ID
	trade.MakerOrderID = makerOrder.ID
	trade.Symbol = h.symbol
	trade.Price = price
	trade.Quantity = qty
	trade.Timestamp = time.Now().UnixNano()

	return trade
}

// addOrderToBook 添加订单到订单簿
func (h *HybridOrderBook) addOrderToBook(order *types.Order, qty int64) {
	var tree *SkipTree
	if order.Side == types.SideBuy {
		tree = h.buys
	} else {
		tree = h.sells
	}

	// 查找或创建价格层级
	level := tree.Get(order.Price)
	if level == nil {
		level = &PriceLevel{
			Price:    order.Price,
			TotalQty: 0,
			Orders:   make([]*types.Order, 0),
		}
		tree.Insert(order.Price, level)
	}

	// 添加订单到层级
	level.Orders = append(level.Orders, order)
	level.TotalQty += qty

	// 存储订单映射
	h.orderMap.Store(order.ID, order)

	// 初始化同价位订单队列
	if _, exists := h.seqMaps[order.Price]; !exists {
		h.seqMaps[order.Price] = lockfree.NewRingBuffer(1000)
	}

	// 将订单添加到顺序队列
	orderPtr := unsafe.Pointer(order)
	h.seqMaps[order.Price].Push(orderPtr)
}

// getEarliestOrderFromLevel 获取同价位最早的订单
func (h *HybridOrderBook) getEarliestOrderFromLevel(price float64) *types.Order {
	queue, exists := h.seqMaps[price]
	if !exists || queue == nil {
		return nil
	}

	item := queue.Pop()
	if item == nil {
		return nil
	}

	return (*types.Order)(item)
}

// updatePriceLevel 更新价格层级
func (h *HybridOrderBook) updatePriceLevel(level *PriceLevel, delta int64) {
	level.TotalQty += delta
	if level.TotalQty < 0 {
		level.TotalQty = 0
	}
}

// GetSnapshot 获取订单簿快照
func (h *HybridOrderBook) GetSnapshot(depth int) *types.OrderBook {
	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := &types.OrderBook{
		Symbol: h.symbol,
		Bids:   h.getTopLevels(h.buys, depth),
		Asks:   h.getTopLevels(h.sells, depth),
		Time:   time.Now().UnixMilli(),
	}

	return snapshot
}

// getTopLevels 获取顶部价格层级（适配跳表）
func (h *HybridOrderBook) getTopLevels(tree *SkipTree, depth int) []types.PriceLevel {
	levels := tree.GetTopLevels(depth)
	result := make([]types.PriceLevel, 0, len(levels))

	for _, level := range levels {
		priceLevel := types.PriceLevel{
			Price:    level.Price,
			Quantity: level.TotalQty,
			Count:    len(level.Orders),
		}
		result = append(result, priceLevel)
	}

	return result
}

// GetDepth 获取订单簿深度
func (h *HybridOrderBook) GetDepth(depth int) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.buys.Len() + h.sells.Len()
}

// CancelOrder 取消订单
func (h *HybridOrderBook) CancelOrder(orderID uint64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从订单映射中查找订单
	orderInterface, exists := h.orderMap.Load(orderID)
	if !exists {
		return false
	}

	order := orderInterface.(*types.Order)

	// 从对应的价格树中移除
	var tree *SkipTree
	if order.Side == types.SideBuy {
		tree = h.buys
	} else {
		tree = h.sells
	}

	level := tree.Get(order.Price)
	if level == nil {
		return false
	}

	// 从层级中移除订单
	for i, o := range level.Orders {
		if o.ID == orderID {
			// 移除订单
			level.Orders = append(level.Orders[:i], level.Orders[i+1:]...)
			level.TotalQty -= order.Quantity

			// 如果层级为空，移除整个层级
			if level.TotalQty == 0 {
				tree.Remove(order.Price)
				delete(h.seqMaps, order.Price)
			}

			// 从订单映射中移除
			h.orderMap.Delete(orderID)

			// 归还订单对象到池
			types.PutOrderToPool(order)

			h.version++
			return true
		}
	}

	return false
}

// GetBestBid 获取最优买价
func (h *HybridOrderBook) GetBestBid() (float64, int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.buys.Len() == 0 {
		return 0, 0
	}

	bestBid := h.buys.MaxPriceNode()
	if bestBid == nil {
		return 0, 0
	}

	return bestBid.Price, bestBid.TotalQty
}

// GetBestAsk 获取最优卖价
func (h *HybridOrderBook) GetBestAsk() (float64, int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.sells.Len() == 0 {
		return 0, 0
	}

	bestAsk := h.sells.MinPriceNode()
	if bestAsk == nil {
		return 0, 0
	}

	return bestAsk.Price, bestAsk.TotalQty
}

// GetSpread 获取买卖价差
func (h *HybridOrderBook) GetSpread() float64 {
	bid, _ := h.GetBestBid()
	ask, _ := h.GetBestAsk()

	if bid == 0 || ask == 0 {
		return 0
	}

	return ask - bid
}

// GetBestBidAndAsk 获取最优买卖价
func (h *HybridOrderBook) GetBestBidAndAsk() (float64, float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var bestBid, bestAsk float64

	if h.buys.Len() > 0 {
		if bestBidNode := h.buys.MaxPriceNode(); bestBidNode != nil {
			bestBid = bestBidNode.Price
		}
	}

	if h.sells.Len() > 0 {
		if bestAskNode := h.sells.MinPriceNode(); bestAskNode != nil {
			bestAsk = bestAskNode.Price
		}
	}

	return bestBid, bestAsk
}

// UpdateSpreadMetrics 更新价差指标
func (h *HybridOrderBook) UpdateSpreadMetrics() {
	bid, ask := h.GetBestBidAndAsk()
	if bid > 0 && ask > 0 {
		spread := ask - bid
		monitor.SetOrderBookSpread(h.symbol, spread)
	}
}

// ValidateOrderBooks 验证订单簿完整性（调试用）
func (h *HybridOrderBook) ValidateOrderBooks() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	validBuys := h.buys.Validate()
	validSells := h.sells.Validate()

	if !validBuys {
		logx.Error("Buy order book validation failed")
	}
	if !validSells {
		logx.Error("Sell order book validation failed")
	}

	return validBuys && validSells
}

// PrintOrderBookStructure 打印订单簿结构（调试用）
func (h *HybridOrderBook) PrintOrderBookStructure() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	println("=== Buy Order Book (Descending) ===")
	h.buys.PrintStructure()

	println("=== Sell Order Book (Ascending) ===")
	h.sells.PrintStructure()
}

// updateStats 更新统计信息
func (h *HybridOrderBook) updateStats(tradeCount int, latency time.Duration) {
	h.stats.OrdersProcessed++
	h.stats.TradesExecuted += uint64(tradeCount)
	h.stats.LastUpdate = time.Now()

	// 更新平均延迟
	if h.stats.AvgLatency == 0 {
		h.stats.AvgLatency = latency
	} else {
		h.stats.AvgLatency = (h.stats.AvgLatency*time.Duration(h.stats.OrdersProcessed-1) + latency) /
			time.Duration(h.stats.OrdersProcessed)
	}
}

// GetStats 获取统计信息
func (h *HybridOrderBook) GetStats() *OrderBookStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := *h.stats // 返回副本
	return &stats
}

// recordTrade 记录交易
func (h *HybridOrderBook) recordTrade(trade *types.Trade) {
	// 这里可以添加交易记录到持久化存储或消息队列
	// 例如: h.tradeLogger.Log(trade)
}

// generateTradeID 生成交易ID
func generateTradeID() uint64 {
	return uint64(time.Now().UnixNano())
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
