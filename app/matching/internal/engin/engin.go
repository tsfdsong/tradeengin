package engine

import (
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"github.com/tsfdsong/tradeengin/app/matching/internal/config"
	"github.com/tsfdsong/tradeengin/app/matching/internal/monitor"
	"github.com/tsfdsong/tradeengin/app/matching/internal/orderbook"
	"github.com/tsfdsong/tradeengin/app/pkg/lockfree"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

var (
	ErrEngineAlreadyStarted = errors.New("matching engine already started")
	ErrSymbolNotFound       = errors.New("symbol not found")
	ErrQueueFull            = errors.New("input queue is full")
	ErrOrderNotFound        = errors.New("order not found")
	ErrDuplicateOrder       = errors.New("duplicate order")
)

// OrderStatus 订单状态
type OrderStatus int8

const (
	OrderStatusPending   OrderStatus = 0 // 挂单中
	OrderStatusPartial   OrderStatus = 1 // 部分成交
	OrderStatusFilled    OrderStatus = 2 // 完全成交
	OrderStatusCancelled OrderStatus = 3 // 已取消
)

// OrderState 订单状态信息
type OrderState struct {
	OrderID        uint64
	Symbol         string
	Status         OrderStatus
	FilledQuantity int64
	OriginalQty    int64
	CreateTime     int64
}

type MatchingEngine struct {
	config      *config.Config
	orderBooks  map[string]*orderbook.HybridOrderBook
	inputQueues map[string]*lockfree.RingBuffer
	outputQueue *lockfree.RingBuffer
	workers     []*MatchingWorker
	snapshotter *Snapshotter
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	mu          sync.RWMutex
	orderStates *sync.Map // 新增: 订单状态跟踪 (orderID -> *OrderState)
	processed   *sync.Map // 新增: 幂等性检查 (orderID -> bool)
}

func NewMatchingEngine(cfg *config.Config) *MatchingEngine {
	engine := &MatchingEngine{
		config:      cfg,
		orderBooks:  make(map[string]*orderbook.HybridOrderBook),
		inputQueues: make(map[string]*lockfree.RingBuffer),
		outputQueue: lockfree.NewRingBuffer(1024 * 1024),
		orderStates: &sync.Map{}, // 初始化订单状态跟踪
		processed:   &sync.Map{}, // 初始化幂等性检查
	}

	// 初始化订单簿
	symbols := cfg.Matching.Symbols
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	}

	for _, symbol := range symbols {
		engine.orderBooks[symbol] = orderbook.NewHybridOrderBook(symbol)
		engine.inputQueues[symbol] = lockfree.NewRingBuffer(uint64(65536))
	}

	return engine
}

func (e *MatchingEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started {
		return ErrEngineAlreadyStarted
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	// 启动工作协程
	e.startWorkers(ctx)

	// 启动快照服务
	e.startSnapshotter(ctx)

	// 启动结果处理器
	e.startResultProcessor(ctx)

	e.started = true
	logx.Info("Matching engine started successfully")

	return nil
}

func (e *MatchingEngine) startWorkers(ctx context.Context) {
	workerCount := e.config.Matching.WorkerCount
	if workerCount <= 0 {
		workerCount = 32
	}

	for i := 0; i < workerCount; i++ {
		worker := NewMatchingWorker(i, e.config, e.orderBooks, e.inputQueues, e.outputQueue)
		e.workers = append(e.workers, worker)

		e.wg.Add(1)
		threading.GoSafe(func() {
			defer e.wg.Done()
			worker.Run(ctx)
		})
	}
}

func (e *MatchingEngine) startSnapshotter(ctx context.Context) {
	e.snapshotter = NewSnapshotter(e.orderBooks, e.config.Matching.SnapshotInterval)

	e.wg.Add(1)
	threading.GoSafe(func() {
		defer e.wg.Done()
		e.snapshotter.Run(ctx)
	})
}

func (e *MatchingEngine) startResultProcessor(ctx context.Context) {
	e.wg.Add(1)
	threading.GoSafe(func() {
		defer e.wg.Done()
		e.processResults(ctx)
	})
}

func (e *MatchingEngine) processResults(ctx context.Context) {
	batchSize := e.config.Matching.BatchSize
	if batchSize <= 0 {
		batchSize = 128
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 批量获取结果
			results := e.outputQueue.BatchPop(batchSize)
			if len(results) == 0 {
				time.Sleep(100 * time.Microsecond)
				continue
			}

			// 处理撮合结果
			for _, resultPtr := range results {
				result := (*types.MatchResult)(resultPtr)
				e.handleMatchResult(result)
			}
		}
	}
}

func (e *MatchingEngine) handleMatchResult(result *types.MatchResult) {
	// 记录交易指标
	monitor.RecordOrderMatched(result.Order.Symbol, len(result.Trades))

	// 更新订单簿深度
	if orderBook, exists := e.orderBooks[result.Order.Symbol]; exists {
		depth := orderBook.GetDepth(10)
		monitor.SetOrderBookDepth(result.Order.Symbol, depth)
	}
}

func (e *MatchingEngine) ProcessOrder(order *types.Order) (*types.MatchResult, error) {
	// 幂等性检查
	if _, exists := e.processed.LoadOrStore(order.ID, true); exists {
		return nil, ErrDuplicateOrder
	}

	queue, exists := e.inputQueues[order.Symbol]
	if !exists {
		e.processed.Delete(order.ID) // 回滚幂等性标记
		return nil, ErrSymbolNotFound
	}

	// 记录订单状态
	e.orderStates.Store(order.ID, &OrderState{
		OrderID:        order.ID,
		Symbol:         order.Symbol,
		Status:         OrderStatusPending,
		FilledQuantity: 0,
		OriginalQty:    order.Quantity,
		CreateTime:     order.Timestamp,
	})

	// 使用对象池获取订单指针
	orderPtr := types.GetOrderFromPool()
	*orderPtr = *order

	if !queue.Push(unsafe.Pointer(orderPtr)) {
		types.PutOrderToPool(orderPtr)
		e.processed.Delete(order.ID) // 回滚幂等性标记
		e.orderStates.Delete(order.ID)
		return nil, ErrQueueFull
	}

	// 在实际生产环境中，这里应该等待处理结果
	// 简化处理，直接返回空结果
	return &types.MatchResult{
		Order:     order,
		Timestamp: time.Now().UnixNano(),
	}, nil
}

func (e *MatchingEngine) GetOrderBook(symbol string, depth int) (*types.OrderBook, error) {
	orderBook, exists := e.orderBooks[symbol]
	if !exists {
		return nil, ErrSymbolNotFound
	}

	return orderBook.GetSnapshot(depth), nil
}

func (e *MatchingEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started {
		return
	}

	if e.cancel != nil {
		e.cancel()
	}

	e.wg.Wait()
	e.started = false

	logx.Info("Matching engine stopped successfully")
}

// CancelOrder 取消订单
func (e *MatchingEngine) CancelOrder(orderID uint64, symbol string) (bool, error) {
	orderBook, exists := e.orderBooks[symbol]
	if !exists {
		return false, ErrSymbolNotFound
	}

	// 尝试从订单簿中取消
	success := orderBook.CancelOrder(orderID)
	if !success {
		return false, ErrOrderNotFound
	}

	// 更新订单状态
	if state, ok := e.orderStates.Load(orderID); ok {
		os := state.(*OrderState)
		os.Status = OrderStatusCancelled
	}

	logx.Infof("Order %d cancelled successfully", orderID)
	return true, nil
}

// GetOrderState 查询订单状态
func (e *MatchingEngine) GetOrderState(orderID uint64) (*OrderState, error) {
	state, exists := e.orderStates.Load(orderID)
	if !exists {
		return nil, ErrOrderNotFound
	}
	return state.(*OrderState), nil
}

// UpdateOrderState 更新订单状态(内部使用)
func (e *MatchingEngine) UpdateOrderState(orderID uint64, filledQty int64) {
	state, exists := e.orderStates.Load(orderID)
	if !exists {
		return
	}

	os := state.(*OrderState)
	os.FilledQuantity += filledQty

	// 更新状态
	if os.FilledQuantity >= os.OriginalQty {
		os.Status = OrderStatusFilled
	} else if os.FilledQuantity > 0 {
		os.Status = OrderStatusPartial
	}
}

// GetSymbols 获取所有交易对
func (e *MatchingEngine) GetSymbols() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	symbols := make([]string, 0, len(e.orderBooks))
	for symbol := range e.orderBooks {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetQueueSize 获取队列大小
func (e *MatchingEngine) GetQueueSize(symbol string) (uint64, error) {
	queue, exists := e.inputQueues[symbol]
	if !exists {
		return 0, ErrSymbolNotFound
	}
	return queue.Size(), nil
}

// GetOrderBooks 获取所有订单簿的引用
func (e *MatchingEngine) GetOrderBooks() map[string]*orderbook.HybridOrderBook {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.orderBooks
}

// GetOrderBookBySymbol 获取指定交易对的订单簿
func (e *MatchingEngine) GetOrderBookBySymbol(symbol string) (*orderbook.HybridOrderBook, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ob, exists := e.orderBooks[symbol]
	if !exists {
		return nil, ErrSymbolNotFound
	}
	return ob, nil
}
