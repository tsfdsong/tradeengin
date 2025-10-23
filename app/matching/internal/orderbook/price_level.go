package orderbook

import (
	"sync"
	"sync/atomic"

	"github.com/tsfdsong/tradeengin/app/pkg/types"
)

// PriceLevelManager 价格层级管理器
type PriceLevelManager struct {
	levels     map[float64]*PriceLevel
	orderMap   map[uint64]*types.Order // orderID -> Order
	seqMap     map[float64]*OrderQueue // 同价位订单队列
	mu         sync.RWMutex
	totalQty   int64
	orderCount int32
}

// OrderQueue 订单队列
type OrderQueue struct {
	orders []*types.Order
	head   int
	tail   int
	size   int
	mu     sync.Mutex
}

// NewPriceLevelManager 创建价格层级管理器
func NewPriceLevelManager() *PriceLevelManager {
	return &PriceLevelManager{
		levels:   make(map[float64]*PriceLevel),
		orderMap: make(map[uint64]*types.Order),
		seqMap:   make(map[float64]*OrderQueue),
	}
}

// AddOrder 添加订单到价格层级
func (plm *PriceLevelManager) AddOrder(order *types.Order) {
	plm.mu.Lock()
	defer plm.mu.Unlock()

	price := order.Price

	// 查找或创建价格层级
	level, exists := plm.levels[price]
	if !exists {
		level = &PriceLevel{
			Price:    price,
			TotalQty: 0,
			Orders:   make([]*types.Order, 0),
		}
		plm.levels[price] = level
	}

	// 添加到层级
	level.Orders = append(level.Orders, order)
	level.TotalQty += order.Quantity

	// 添加到订单映射
	plm.orderMap[order.ID] = order

	// 初始化同价位订单队列
	if _, exists := plm.seqMap[price]; !exists {
		plm.seqMap[price] = NewOrderQueue(1000)
	}

	// 添加到顺序队列
	plm.seqMap[price].Push(order)

	// 更新统计
	atomic.AddInt64(&plm.totalQty, order.Quantity)
	atomic.AddInt32(&plm.orderCount, 1)
}

// RemoveOrder 从价格层级移除订单
func (plm *PriceLevelManager) RemoveOrder(orderID uint64) bool {
	plm.mu.Lock()
	defer plm.mu.Unlock()

	order, exists := plm.orderMap[orderID]
	if !exists {
		return false
	}

	price := order.Price
	level, exists := plm.levels[price]
	if !exists {
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
				delete(plm.levels, price)
				delete(plm.seqMap, price)
			}

			// 从订单映射中移除
			delete(plm.orderMap, orderID)

			// 从顺序队列中移除（简化处理，实际应该更精确）
			if queue, exists := plm.seqMap[price]; exists {
				queue.Remove(orderID)
			}

			// 更新统计
			atomic.AddInt64(&plm.totalQty, -order.Quantity)
			atomic.AddInt32(&plm.orderCount, -1)

			return true
		}
	}

	return false
}

// GetEarliestOrder 获取同价位最早的订单
func (plm *PriceLevelManager) GetEarliestOrder(price float64) *types.Order {
	plm.mu.RLock()
	defer plm.mu.RUnlock()

	queue, exists := plm.seqMap[price]
	if !exists {
		return nil
	}

	return queue.Peek()
}

// GetLevel 获取价格层级
func (plm *PriceLevelManager) GetLevel(price float64) *PriceLevel {
	plm.mu.RLock()
	defer plm.mu.RUnlock()

	return plm.levels[price]
}

// GetTotalQuantity 获取总数量
func (plm *PriceLevelManager) GetTotalQuantity() int64 {
	return atomic.LoadInt64(&plm.totalQty)
}

// GetOrderCount 获取订单数量
func (plm *PriceLevelManager) GetOrderCount() int32 {
	return atomic.LoadInt32(&plm.orderCount)
}

// NewOrderQueue 创建订单队列
func NewOrderQueue(capacity int) *OrderQueue {
	return &OrderQueue{
		orders: make([]*types.Order, capacity),
		head:   0,
		tail:   0,
		size:   0,
	}
}

// Push 添加订单到队列
func (oq *OrderQueue) Push(order *types.Order) {
	oq.mu.Lock()
	defer oq.mu.Unlock()

	if oq.size == len(oq.orders) {
		// 队列满，扩容
		newOrders := make([]*types.Order, len(oq.orders)*2)
		copy(newOrders, oq.orders[oq.head:])
		copy(newOrders[len(oq.orders)-oq.head:], oq.orders[:oq.head])
		oq.orders = newOrders
		oq.head = 0
		oq.tail = oq.size
	}

	oq.orders[oq.tail] = order
	oq.tail = (oq.tail + 1) % len(oq.orders)
	oq.size++
}

// Pop 从队列取出订单
func (oq *OrderQueue) Pop() *types.Order {
	oq.mu.Lock()
	defer oq.mu.Unlock()

	if oq.size == 0 {
		return nil
	}

	order := oq.orders[oq.head]
	oq.orders[oq.head] = nil
	oq.head = (oq.head + 1) % len(oq.orders)
	oq.size--

	return order
}

// Peek 查看队列头部订单
func (oq *OrderQueue) Peek() *types.Order {
	oq.mu.Lock()
	defer oq.mu.Unlock()

	if oq.size == 0 {
		return nil
	}

	return oq.orders[oq.head]
}

// Remove 从队列中移除指定订单
func (oq *OrderQueue) Remove(orderID uint64) bool {
	oq.mu.Lock()
	defer oq.mu.Unlock()

	for i := 0; i < oq.size; i++ {
		idx := (oq.head + i) % len(oq.orders)
		if oq.orders[idx].ID == orderID {
			// 移除订单
			oq.orders[idx] = nil

			// 移动后续元素
			for j := i; j < oq.size-1; j++ {
				srcIdx := (oq.head + j + 1) % len(oq.orders)
				dstIdx := (oq.head + j) % len(oq.orders)
				oq.orders[dstIdx] = oq.orders[srcIdx]
			}

			oq.tail = (oq.tail - 1 + len(oq.orders)) % len(oq.orders)
			oq.size--

			return true
		}
	}

	return false
}

// Size 获取队列大小
func (oq *OrderQueue) Size() int {
	oq.mu.Lock()
	defer oq.mu.Unlock()

	return oq.size
}
