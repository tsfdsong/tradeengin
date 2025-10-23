package engine

import (
	"context"
	"time"
	"unsafe"

	"github.com/tsfdsong/tradeengin/app/matching/internal/config"
	"github.com/tsfdsong/tradeengin/app/matching/internal/monitor"
	"github.com/tsfdsong/tradeengin/app/matching/internal/orderbook"
	"github.com/tsfdsong/tradeengin/app/pkg/lockfree"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type MatchingWorker struct {
	id          int
	config      *config.Config
	orderBooks  map[string]*orderbook.HybridOrderBook
	inputQueues map[string]*lockfree.RingBuffer
	outputQueue *lockfree.RingBuffer
	batchSize   int
}

func NewMatchingWorker(
	id int,
	cfg *config.Config,
	orderBooks map[string]*orderbook.HybridOrderBook,
	inputQueues map[string]*lockfree.RingBuffer,
	outputQueue *lockfree.RingBuffer,
) *MatchingWorker {

	batchSize := cfg.Matching.BatchSize
	if batchSize <= 0 {
		batchSize = 128
	}

	return &MatchingWorker{
		id:          id,
		config:      cfg,
		orderBooks:  orderBooks,
		inputQueues: inputQueues,
		outputQueue: outputQueue,
		batchSize:   batchSize,
	}
}

func (w *MatchingWorker) Run(ctx context.Context) {
	logx.Infof("Matching worker %d started", w.id)

	for {
		select {
		case <-ctx.Done():
			logx.Infof("Matching worker %d stopped", w.id)
			return
		default:
			w.processBatch(ctx)
		}
	}
}

func (w *MatchingWorker) processBatch(ctx context.Context) {
	processed := 0
	startTime := time.Now()

	// 轮询所有交易对的队列
	for symbol, queue := range w.inputQueues {
		if processed >= w.batchSize {
			break
		}

		// 批量获取订单
		orders := queue.BatchPop(w.batchSize - processed)
		if len(orders) == 0 {
			continue
		}

		// 处理订单
		for _, orderPtr := range orders {
			select {
			case <-ctx.Done():
				return
			default:
				order := (*types.Order)(orderPtr)
				w.processOrder(symbol, order)
				processed++
			}
		}
	}

	// 记录处理指标 - 修正这里
	if processed > 0 {
		latency := time.Since(startTime)
		monitor.RecordBatchProcessed(processed, latency)
		monitor.RecordWorkerLoad(w.id, processed)
		monitor.RecordWorkerProcessed(w.id, processed)

		logx.Debugf("Worker %d processed %d orders in %v", w.id, processed, latency)
	} else {
		// 没有数据时短暂休眠
		time.Sleep(50 * time.Microsecond)
	}
}

func (w *MatchingWorker) processOrder(symbol string, order *types.Order) {
	startTime := time.Now()

	// 获取对应的订单簿
	orderBook, exists := w.orderBooks[symbol]
	if !exists {
		logx.Errorf("Order book not found for symbol: %s", symbol)
		monitor.RecordOrderRejected(symbol, "symbol_not_found")
		return
	}

	// 执行撮合
	result := orderBook.Match(order)

	// 记录撮合延迟
	matchingLatency := time.Since(startTime)
	monitor.RecordMatchingLatency(symbol, matchingLatency)

	// 记录交易
	for _, trade := range result.Trades {
		monitor.RecordTrade(symbol, trade.Quantity, trade.Price)
	}

	// 发送结果到输出队列
	if len(result.Trades) > 0 || result.Order.Quantity > 0 {
		resultPtr := unsafe.Pointer(result)
		if !w.outputQueue.Push(resultPtr) {
			// 输出队列满，记录警告
			monitor.RecordOrderRejected(symbol, "output_queue_full")
			logx.Infof("Output queue full, dropped match result for symbol: %s", symbol)

			// 归还结果对象到池
			types.PutMatchResultToPool(result)
		}
	} else {
		// 没有成交也没有剩余，归还结果对象
		types.PutMatchResultToPool(result)
	}

	// 归还订单对象到池
	types.PutOrderToPool(order)
}
