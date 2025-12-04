package types

import (
	"testing"
)

func TestOrder_Reset(t *testing.T) {
	order := &Order{
		ID:        12345,
		Symbol:    "BTCUSDT",
		Price:     50000.0,
		Quantity:  100,
		Side:      SideBuy,
		Type:      TypeLimit,
		Timestamp: 1234567890,
		ClientID:  "client123",
		Version:   1,
	}

	order.Reset()

	if order.ID != 0 || order.Symbol != "" || order.Price != 0 ||
		order.Quantity != 0 || order.Side != 0 || order.Type != 0 ||
		order.Timestamp != 0 || order.ClientID != "" || order.Version != 0 {
		t.Error("Reset should clear all fields")
	}
}

func TestOrder_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		order    Order
		expected bool
	}{
		{
			name: "Valid limit buy order",
			order: Order{
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 100,
				Side:     SideBuy,
				Type:     TypeLimit,
			},
			expected: true,
		},
		{
			name: "Valid market sell order",
			order: Order{
				Symbol:   "ETHUSDT",
				Price:    0, // 市价单可以没有价格
				Quantity: 50,
				Side:     SideSell,
				Type:     TypeMarket,
			},
			expected: true,
		},
		{
			name: "Invalid - empty symbol",
			order: Order{
				Symbol:   "",
				Price:    50000.0,
				Quantity: 100,
				Side:     SideBuy,
				Type:     TypeLimit,
			},
			expected: false,
		},
		{
			name: "Invalid - zero quantity",
			order: Order{
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 0,
				Side:     SideBuy,
				Type:     TypeLimit,
			},
			expected: false,
		},
		{
			name: "Invalid - wrong side",
			order: Order{
				Symbol:   "BTCUSDT",
				Price:    50000.0,
				Quantity: 100,
				Side:     3, // 无效的Side
				Type:     TypeLimit,
			},
			expected: false,
		},
		{
			name: "Invalid - limit order without price",
			order: Order{
				Symbol:   "BTCUSDT",
				Price:    0,
				Quantity: 100,
				Side:     SideBuy,
				Type:     TypeLimit,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.order.IsValid() != tt.expected {
				t.Errorf("Expected IsValid() = %v", tt.expected)
			}
		})
	}
}

func TestOrder_IsBuySell(t *testing.T) {
	buyOrder := Order{Side: SideBuy}
	sellOrder := Order{Side: SideSell}

	if !buyOrder.IsBuy() {
		t.Error("Buy order should return true for IsBuy()")
	}
	if buyOrder.IsSell() {
		t.Error("Buy order should return false for IsSell()")
	}

	if sellOrder.IsBuy() {
		t.Error("Sell order should return false for IsBuy()")
	}
	if !sellOrder.IsSell() {
		t.Error("Sell order should return true for IsSell()")
	}
}

func TestTrade_Reset(t *testing.T) {
	trade := &Trade{
		TradeID:      12345,
		TakerOrderID: 111,
		MakerOrderID: 222,
		Symbol:       "BTCUSDT",
		Price:        50000.0,
		Quantity:     100,
		Timestamp:    1234567890,
		TakerSide:    SideBuy,
	}

	trade.Reset()

	if trade.TradeID != 0 || trade.Symbol != "" || trade.Price != 0 ||
		trade.Quantity != 0 || trade.TakerSide != 0 {
		t.Error("Reset should clear all fields")
	}
}

func TestMatchResult_HasTrades(t *testing.T) {
	result := &MatchResult{}
	if result.HasTrades() {
		t.Error("Empty result should not have trades")
	}

	result.Trades = append(result.Trades, &Trade{})
	if !result.HasTrades() {
		t.Error("Result with trades should return true for HasTrades()")
	}
}

func TestMatchResult_TotalFilledQty(t *testing.T) {
	result := &MatchResult{
		Trades: []*Trade{
			{Quantity: 100},
			{Quantity: 50},
			{Quantity: 25},
		},
	}

	if result.TotalFilledQty() != 175 {
		t.Errorf("Expected TotalFilledQty = 175, got %d", result.TotalFilledQty())
	}
}

func TestOrderPool(t *testing.T) {
	// 获取对象
	order1 := GetOrderFromPool()
	if order1 == nil {
		t.Fatal("GetOrderFromPool should not return nil")
	}

	// 设置一些值
	order1.ID = 12345
	order1.Symbol = "BTCUSDT"

	// 归还对象
	PutOrderToPool(order1)

	// 再次获取，应该被重置
	order2 := GetOrderFromPool()
	if order2.ID != 0 || order2.Symbol != "" {
		t.Error("Pool object should be reset after put")
	}
}

func TestTradePool(t *testing.T) {
	trade := GetTradeFromPool()
	if trade == nil {
		t.Fatal("GetTradeFromPool should not return nil")
	}

	trade.TradeID = 12345
	trade.Symbol = "BTCUSDT"

	PutTradeToPool(trade)

	trade2 := GetTradeFromPool()
	if trade2.TradeID != 0 || trade2.Symbol != "" {
		t.Error("Pool object should be reset after put")
	}
}

func TestMatchResultPool(t *testing.T) {
	result := GetMatchResultFromPool()
	if result == nil {
		t.Fatal("GetMatchResultFromPool should not return nil")
	}

	// 添加一些trades
	result.Trades = append(result.Trades, GetTradeFromPool())
	result.Trades = append(result.Trades, GetTradeFromPool())

	PutMatchResultToPool(result)

	result2 := GetMatchResultFromPool()
	if len(result2.Trades) != 0 {
		t.Error("Pool result should have empty trades after put")
	}
}

func BenchmarkOrderPool_GetPut(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			order := GetOrderFromPool()
			order.ID = 12345
			order.Symbol = "BTCUSDT"
			PutOrderToPool(order)
		}
	})
}

func BenchmarkOrder_IsValid(b *testing.B) {
	order := Order{
		Symbol:   "BTCUSDT",
		Price:    50000.0,
		Quantity: 100,
		Side:     SideBuy,
		Type:     TypeLimit,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		order.IsValid()
	}
}
