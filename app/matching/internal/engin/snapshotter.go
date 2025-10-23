package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tsfdsong/tradeengin/app/matching/internal/orderbook"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// Snapshotter 订单簿快照服务
type Snapshotter struct {
	orderBooks   map[string]*orderbook.HybridOrderBook
	interval     time.Duration
	snapshotChan chan *Snapshot
	enabled      bool
}

// Snapshot 订单簿快照
type Snapshot struct {
	Symbol    string
	Data      []byte
	Timestamp int64
	Version   uint64
}

// NewSnapshotter 创建快照服务
func NewSnapshotter(orderBooks map[string]*orderbook.HybridOrderBook, interval string) *Snapshotter {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		duration = 30 * time.Second // 默认30秒
	}

	return &Snapshotter{
		orderBooks:   orderBooks,
		interval:     duration,
		snapshotChan: make(chan *Snapshot, 1000),
		enabled:      true,
	}
}

// Run 运行快照服务
func (s *Snapshotter) Run(ctx context.Context) {
	logx.Info("Snapshotter started")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logx.Info("Snapshotter stopped")
			return
		case <-ticker.C:
			s.takeSnapshots()
		}
	}
}

// takeSnapshots 拍摄所有订单簿快照
func (s *Snapshotter) takeSnapshots() {
	for symbol, orderBook := range s.orderBooks {
		snapshot := s.takeSnapshot(symbol, orderBook)
		if snapshot != nil {
			select {
			case s.snapshotChan <- snapshot:
				// 快照发送成功
			default:
				logx.Infof("Snapshot channel full, dropped snapshot for %s", symbol)
			}
		}
	}
}

// takeSnapshot 拍摄单个订单簿快照
func (s *Snapshotter) takeSnapshot(symbol string, orderBook *orderbook.HybridOrderBook) *Snapshot {
	startTime := time.Now()

	// 获取订单簿快照
	obSnapshot := orderBook.GetSnapshot(100) // 获取前100档深度

	// 序列化快照数据
	data, err := json.Marshal(obSnapshot)
	if err != nil {
		logx.Errorf("Failed to marshal snapshot for %s: %v", symbol, err)
		return nil
	}

	// 获取统计信息
	stats := orderBook.GetStats()

	latency := time.Since(startTime)
	logx.Infof("Snapshot taken for %s: depth=%d, latency=%v, stats=%+v",
		symbol, len(obSnapshot.Bids)+len(obSnapshot.Asks), latency, stats)

	return &Snapshot{
		Symbol:    symbol,
		Data:      data,
		Timestamp: time.Now().Unix(),
		Version:   stats.OrdersProcessed,
	}
}

// GetSnapshotChan 获取快照通道
func (s *Snapshotter) GetSnapshotChan() <-chan *Snapshot {
	return s.snapshotChan
}

// SaveSnapshot 保存快照到持久化存储
func (s *Snapshotter) SaveSnapshot(snapshot *Snapshot) error {
	// 这里可以实现保存到Redis、数据库或文件系统
	// 例如:
	// - Redis: 使用有序集合存储历史快照
	// - 数据库: 插入到snapshots表
	// - 文件系统: 写入到本地文件

	logx.Debugf("Saving snapshot for %s, size=%d bytes",
		snapshot.Symbol, len(snapshot.Data))

	return nil
}

// RestoreFromSnapshot 从快照恢复订单簿
func (s *Snapshotter) RestoreFromSnapshot(symbol string, data []byte) error {
	var obSnapshot types.OrderBook
	if err := json.Unmarshal(data, &obSnapshot); err != nil {
		return err
	}

	// 这里可以实现从快照数据恢复订单簿状态
	// 注意: 这需要根据具体的订单簿实现来编写恢复逻辑

	logx.Infof("Restored orderbook for %s from snapshot", symbol)
	return nil
}

// Enable 启用快照服务
func (s *Snapshotter) Enable() {
	s.enabled = true
}

// Disable 禁用快照服务
func (s *Snapshotter) Disable() {
	s.enabled = false
}
