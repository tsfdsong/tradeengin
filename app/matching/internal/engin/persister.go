package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tsfdsong/tradeengin/app/matching/internal/orderbook"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RedisPersister Redis持久化服务 - 使用go-zero的Redis客户端
type RedisPersister struct {
	client     *redis.Redis
	orderBooks map[string]*orderbook.HybridOrderBook
	interval   time.Duration
	keyPrefix  string
	enabled    bool
}

// NewRedisPersister 创建Redis持久化服务
func NewRedisPersister(
	client *redis.Redis,
	orderBooks map[string]*orderbook.HybridOrderBook,
	interval string,
	enabled bool,
) *RedisPersister {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		duration = 5 * time.Second
	}

	return &RedisPersister{
		client:     client,
		orderBooks: orderBooks,
		interval:   duration,
		keyPrefix:  "matching:orderbook:",
		enabled:    enabled,
	}
}

// Run 运行持久化服务
func (p *RedisPersister) Run(ctx context.Context) {
	if !p.enabled || p.client == nil {
		logx.Info("Redis persister disabled or client not available")
		return
	}

	logx.Info("Redis persister started")
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 最后一次持久化
			p.persistAll()
			logx.Info("Redis persister stopped")
			return
		case <-ticker.C:
			p.persistAll()
		}
	}
}

// persistAll 持久化所有订单簿
func (p *RedisPersister) persistAll() {
	for symbol, orderBook := range p.orderBooks {
		if err := p.persistOrderBook(symbol, orderBook); err != nil {
			logx.Errorf("Failed to persist orderbook for %s: %v", symbol, err)
		}
	}
}

// persistOrderBook 持久化单个订单簿
func (p *RedisPersister) persistOrderBook(symbol string, orderBook *orderbook.HybridOrderBook) error {
	snapshot := orderBook.GetSnapshot(1000) // 保存前1000档

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	key := p.keyPrefix + symbol
	if err := p.client.Set(key, string(data)); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	// 同时保存统计信息
	stats := orderBook.GetStats()
	statsData, _ := json.Marshal(stats)
	statsKey := p.keyPrefix + symbol + ":stats"
	_ = p.client.Set(statsKey, string(statsData))

	logx.Debugf("Persisted orderbook for %s, size=%d bytes", symbol, len(data))
	return nil
}

// RestoreOrderBook 从Redis恢复订单簿
func (p *RedisPersister) RestoreOrderBook(symbol string) (*types.OrderBook, error) {
	if p.client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := p.keyPrefix + symbol
	data, err := p.client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	if data == "" {
		return nil, nil // 没有数据
	}

	var snapshot types.OrderBook
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	logx.Infof("Restored orderbook for %s from Redis", symbol)
	return &snapshot, nil
}

// SaveOrder 保存订单到Redis（用于订单恢复）
func (p *RedisPersister) SaveOrder(order *types.Order) error {
	if p.client == nil {
		return nil
	}

	key := fmt.Sprintf("matching:order:%d", order.ID)
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// 设置24小时过期
	return p.client.Setex(key, string(data), 86400)
}

// GetOrder 从Redis获取订单
func (p *RedisPersister) GetOrder(orderID uint64) (*types.Order, error) {
	if p.client == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("matching:order:%d", orderID)
	data, err := p.client.Get(key)
	if err != nil {
		return nil, err
	}

	if data == "" {
		return nil, nil
	}

	var order types.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		return nil, err
	}

	return &order, nil
}

// DeleteOrder 从Redis删除订单
func (p *RedisPersister) DeleteOrder(orderID uint64) error {
	if p.client == nil {
		return nil
	}

	key := fmt.Sprintf("matching:order:%d", orderID)
	_, err := p.client.Del(key)
	return err
}

// SaveTrade 保存成交记录
func (p *RedisPersister) SaveTrade(trade *types.Trade) error {
	if p.client == nil {
		return nil
	}

	key := fmt.Sprintf("matching:trade:%d", trade.TradeID)
	data, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	// 添加到有序集合（按时间戳排序）
	listKey := fmt.Sprintf("matching:trades:%s", trade.Symbol)
	_, _ = p.client.Zadd(listKey, trade.Timestamp, string(data))

	// 保存单条记录，24小时过期
	return p.client.Setex(key, string(data), 86400)
}

// GetRecentTrades 获取最近成交记录
func (p *RedisPersister) GetRecentTrades(symbol string, limit int) ([]*types.Trade, error) {
	if p.client == nil {
		return nil, nil
	}

	listKey := fmt.Sprintf("matching:trades:%s", symbol)
	// 获取最新的limit条记录
	results, err := p.client.ZrevrangeCtx(context.Background(), listKey, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}

	trades := make([]*types.Trade, 0, len(results))
	for _, data := range results {
		var trade types.Trade
		if err := json.Unmarshal([]byte(data), &trade); err != nil {
			continue
		}
		trades = append(trades, &trade)
	}

	return trades, nil
}
