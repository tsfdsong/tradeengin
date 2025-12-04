package monitor

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 订单相关指标
	ordersReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_orders_received_total",
		Help: "Total number of orders received",
	}, []string{"symbol", "side"})

	ordersMatched = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_orders_matched_total",
		Help: "Total number of orders matched",
	}, []string{"symbol"})

	ordersRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_orders_rejected_total",
		Help: "Total number of orders rejected",
	}, []string{"symbol", "reason"})

	// 延迟指标
	orderLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "trading_order_latency_seconds",
		Help:    "Order processing latency distribution",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 16), // 100微秒到3.2秒
	}, []string{"symbol"})

	matchingLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "trading_matching_latency_seconds",
		Help:    "Order matching latency distribution",
		Buckets: prometheus.ExponentialBuckets(0.000001, 2, 20), // 1微秒到1秒
	}, []string{"symbol"})

	batchProcessingLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "trading_batch_processing_latency_seconds",
		Help:    "Batch processing latency distribution",
		Buckets: prometheus.ExponentialBuckets(0.00001, 2, 16), // 10微秒到0.3秒
	})

	// 队列和深度指标
	queueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trading_queue_depth",
		Help: "Current depth of input queues",
	}, []string{"symbol"})

	orderBookDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trading_orderbook_depth",
		Help: "Current depth of order books",
	}, []string{"symbol"})

	orderBookSpread = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trading_orderbook_spread",
		Help: "Current spread of order books",
	}, []string{"symbol"})

	// 批量处理指标
	batchSizeHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "trading_batch_size",
		Help:    "Batch processing size distribution",
		Buckets: prometheus.LinearBuckets(1, 10, 20), // 1到200
	})

	batchProcessingCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trading_batch_processing_total",
		Help: "Total number of batch processing operations",
	})

	// 工作协程指标
	workerLoad = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trading_worker_load",
		Help: "Current load of matching workers",
	}, []string{"worker_id"})

	workerProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_worker_processed_total",
		Help: "Total orders processed by each worker",
	}, []string{"worker_id"})

	// 内存和GC指标
	memoryAlloc = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "trading_memory_alloc_bytes",
		Help: "Current memory allocation in bytes",
	})

	goroutineCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "trading_goroutine_count",
		Help: "Current number of goroutines",
	})

	// 交易指标
	tradesExecuted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_trades_executed_total",
		Help: "Total number of trades executed",
	}, []string{"symbol"})

	tradeVolume = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trading_trade_volume_total",
		Help: "Total trade volume",
	}, []string{"symbol"})
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	// 可以添加自定义收集逻辑
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordOrderReceived 记录订单接收
func (m *MetricsCollector) RecordOrderReceived(symbol string, side int8) {
	sideStr := "buy"
	if side == 1 {
		sideStr = "sell"
	}
	ordersReceived.WithLabelValues(symbol, sideStr).Inc()
}

// RecordOrderMatched 记录订单成交
func (m *MetricsCollector) RecordOrderMatched(symbol string, tradeCount int) {
	ordersMatched.WithLabelValues(symbol).Add(float64(tradeCount))
}

// RecordOrderRejected 记录订单拒绝
func (m *MetricsCollector) RecordOrderRejected(symbol string, reason string) {
	ordersRejected.WithLabelValues(symbol, reason).Inc()
}

// RecordOrderLatency 记录订单处理延迟
func (m *MetricsCollector) RecordOrderLatency(symbol string, latency time.Duration) {
	orderLatency.WithLabelValues(symbol).Observe(latency.Seconds())
}

// RecordMatchingLatency 记录撮合延迟
func (m *MetricsCollector) RecordMatchingLatency(symbol string, latency time.Duration) {
	matchingLatency.WithLabelValues(symbol).Observe(latency.Seconds())
}

// RecordBatchProcessed 记录批量处理
func (m *MetricsCollector) RecordBatchProcessed(batchSize int, latency time.Duration) {
	batchSizeHistogram.Observe(float64(batchSize))
	batchProcessingLatency.Observe(latency.Seconds())
	batchProcessingCount.Inc()
}

// RecordWorkerLoad 记录工作协程负载 - 修复 workerID 转换问题
func (m *MetricsCollector) RecordWorkerLoad(workerID int, load int) {
	workerLoad.WithLabelValues(fmt.Sprintf("%d", workerID)).Set(float64(load))
}

// RecordWorkerProcessed 记录工作协程处理数量 - 修复 workerID 转换问题
func (m *MetricsCollector) RecordWorkerProcessed(workerID int, count int) {
	workerProcessed.WithLabelValues(fmt.Sprintf("%d", workerID)).Add(float64(count))
}

// SetQueueDepth 设置队列深度
func (m *MetricsCollector) SetQueueDepth(symbol string, depth int) {
	queueDepth.WithLabelValues(symbol).Set(float64(depth))
}

// SetOrderBookDepth 设置订单簿深度
func (m *MetricsCollector) SetOrderBookDepth(symbol string, depth int) {
	orderBookDepth.WithLabelValues(symbol).Set(float64(depth))
}

// SetOrderBookSpread 设置订单簿价差
func (m *MetricsCollector) SetOrderBookSpread(symbol string, spread float64) {
	orderBookSpread.WithLabelValues(symbol).Set(spread)
}

// RecordTrade 记录交易
func (m *MetricsCollector) RecordTrade(symbol string, quantity int64, price float64) {
	tradesExecuted.WithLabelValues(symbol).Inc()
	tradeVolume.WithLabelValues(symbol).Add(float64(quantity) * price)
}

// UpdateMemoryMetrics 更新内存指标
func (m *MetricsCollector) UpdateMemoryMetrics() {
	// 这里可以使用 runtime.ReadMemStats 来获取详细内存信息
	// 简化实现，只记录基本指标
	// var memStats runtime.MemStats
	// runtime.ReadMemStats(&memStats)
	// memoryAlloc.Set(float64(memStats.Alloc))
}

// UpdateGoroutineMetrics 更新协程指标
func (m *MetricsCollector) UpdateGoroutineMetrics() {
	// 可以通过 runtime.NumGoroutine() 获取协程数量
	// goroutineCount.Set(float64(runtime.NumGoroutine()))
}

// 全局函数 - 为了向后兼容
func RecordOrderReceived(symbol string, side int8) {
	collector := NewMetricsCollector()
	collector.RecordOrderReceived(symbol, side)
}

func RecordOrderMatched(symbol string, tradeCount int) {
	collector := NewMetricsCollector()
	collector.RecordOrderMatched(symbol, tradeCount)
}

func RecordMatchingLatency(symbol string, latency time.Duration) {
	collector := NewMetricsCollector()
	collector.RecordMatchingLatency(symbol, latency)
}

func RecordBatchProcessed(batchSize int, latency time.Duration) {
	collector := NewMetricsCollector()
	collector.RecordBatchProcessed(batchSize, latency)
}

func RecordWorkerLoad(workerID int, load int) {
	collector := NewMetricsCollector()
	collector.RecordWorkerLoad(workerID, load)
}

func RecordWorkerProcessed(workerID int, load int) {
	collector := NewMetricsCollector()
	collector.RecordWorkerProcessed(workerID, load)
}

func SetOrderBookDepth(symbol string, depth int) {
	collector := NewMetricsCollector()
	collector.SetOrderBookDepth(symbol, depth)
}

func SetQueueDepth(symbol string, depth int) {
	collector := NewMetricsCollector()
	collector.SetQueueDepth(symbol, depth)
}

func RecordOrderRejected(symbol string, reason string) {
	collector := NewMetricsCollector()
	collector.RecordOrderRejected(symbol, reason)
}

func RecordTrade(symbol string, quantity int64, price float64) {
	collector := NewMetricsCollector()
	collector.RecordTrade(symbol, quantity, price)
}

func SetOrderBookSpread(symbol string, spread float64) {
	collector := NewMetricsCollector()
	collector.SetOrderBookSpread(symbol, spread)
}
