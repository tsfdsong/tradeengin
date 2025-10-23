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
)

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
}

func NewMatchingEngine(cfg *config.Config) *MatchingEngine {
	engine := &MatchingEngine{
		config:      cfg,
		orderBooks:  make(map[string]*orderbook.HybridOrderBook),
		inputQueues: make(map[string]*lockfree.RingBuffer),
		outputQueue: lockfree.NewRingBuffer(1024 * 1024),
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
	queue, exists := e.inputQueues[order.Symbol]
	if !exists {
		return nil, ErrSymbolNotFound
	}

	// 使用对象池获取订单指针
	orderPtr := types.GetOrderFromPool()
	*orderPtr = *order

	if !queue.Push(unsafe.Pointer(orderPtr)) {
		types.PutOrderToPool(orderPtr)
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
