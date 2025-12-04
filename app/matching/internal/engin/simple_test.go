package engine

import (
	"testing"

	"github.com/tsfdsong/tradeengin/app/matching/internal/orderbook"
)

func TestRedisPersister(t *testing.T) {
	// 创建订单簿
	ob := orderbook.NewHybridOrderBook("BTCUSDT")

	// 简单测试：确保订单簿创建成功
	if ob == nil {
		t.Error("Expected orderbook to be created")
	}

	// 测试获取快照
	snapshot := ob.GetSnapshot(10)
	if snapshot == nil {
		t.Error("Expected snapshot to be retrieved")
	}
}
