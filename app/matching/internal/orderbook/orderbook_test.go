package orderbook

import (
	"sync"
	"testing"

	"github.com/tsfdsong/tradeengin/app/pkg/types"
)

func TestNewSkipTree(t *testing.T) {
	// 测试升序跳表
	ascTree := NewSkipTree(16, false)
	if ascTree.reverse {
		t.Error("Ascending tree should have reverse=false")
	}

	// 测试降序跳表
	descTree := NewSkipTree(16, true)
	if !descTree.reverse {
		t.Error("Descending tree should have reverse=true")
	}
}

func TestSkipTree_InsertAndGet(t *testing.T) {
	tree := NewSkipTree(16, false)

	// 插入价格层级
	level1 := &PriceLevel{Price: 100.0, TotalQty: 1000}
	level2 := &PriceLevel{Price: 200.0, TotalQty: 2000}
	level3 := &PriceLevel{Price: 150.0, TotalQty: 1500}

	tree.Insert(100.0, level1)
	tree.Insert(200.0, level2)
	tree.Insert(150.0, level3)

	if tree.Len() != 3 {
		t.Errorf("Expected length 3, got %d", tree.Len())
	}

	// 获取测试
	got := tree.Get(100.0)
	if got == nil || got.TotalQty != 1000 {
		t.Error("Get should return correct level")
	}

	got = tree.Get(999.0)
	if got != nil {
		t.Error("Get non-existent should return nil")
	}
}

func TestSkipTree_Remove(t *testing.T) {
	tree := NewSkipTree(16, false)

	tree.Insert(100.0, &PriceLevel{Price: 100.0, TotalQty: 1000})
	tree.Insert(200.0, &PriceLevel{Price: 200.0, TotalQty: 2000})

	if tree.Len() != 2 {
		t.Errorf("Expected length 2, got %d", tree.Len())
	}

	tree.Remove(100.0)
	if tree.Len() != 1 {
		t.Errorf("Expected length 1 after remove, got %d", tree.Len())
	}

	if tree.Get(100.0) != nil {
		t.Error("Removed level should not be found")
	}
}

func TestSkipTree_MinMaxPriceNode_Ascending(t *testing.T) {
	tree := NewSkipTree(16, false) // 升序

	tree.Insert(150.0, &PriceLevel{Price: 150.0, TotalQty: 1500})
	tree.Insert(100.0, &PriceLevel{Price: 100.0, TotalQty: 1000})
	tree.Insert(200.0, &PriceLevel{Price: 200.0, TotalQty: 2000})

	min := tree.MinPriceNode()
	if min == nil || min.Price != 100.0 {
		t.Errorf("Expected min price 100.0, got %v", min)
	}

	max := tree.MaxPriceNode()
	if max == nil || max.Price != 200.0 {
		t.Errorf("Expected max price 200.0, got %v", max)
	}
}

func TestSkipTree_MinMaxPriceNode_Descending(t *testing.T) {
	tree := NewSkipTree(16, true) // 降序

	tree.Insert(150.0, &PriceLevel{Price: 150.0, TotalQty: 1500})
	tree.Insert(100.0, &PriceLevel{Price: 100.0, TotalQty: 1000})
	tree.Insert(200.0, &PriceLevel{Price: 200.0, TotalQty: 2000})

	// 降序排列：200 -> 150 -> 100
	max := tree.MaxPriceNode()
	if max == nil || max.Price != 200.0 {
		t.Errorf("Expected max price 200.0, got %v", max)
	}

	min := tree.MinPriceNode()
	if min == nil || min.Price != 100.0 {
		t.Errorf("Expected min price 100.0, got %v", min)
	}
}

func TestSkipTree_GetTopLevels(t *testing.T) {
	tree := NewSkipTree(16, false)

	for i := 1; i <= 10; i++ {
		tree.Insert(float64(i*100), &PriceLevel{Price: float64(i * 100), TotalQty: int64(i * 1000)})
	}

	levels := tree.GetTopLevels(5)
	if len(levels) != 5 {
		t.Errorf("Expected 5 levels, got %d", len(levels))
	}

	// 升序应该从100开始
	if levels[0].Price != 100.0 {
		t.Errorf("Expected first level price 100.0, got %f", levels[0].Price)
	}
}

func TestSkipTree_Range(t *testing.T) {
	tree := NewSkipTree(16, false)

	for i := 1; i <= 5; i++ {
		tree.Insert(float64(i*100), &PriceLevel{Price: float64(i * 100)})
	}

	var prices []float64
	tree.Range(func(price float64, level *PriceLevel) bool {
		prices = append(prices, price)
		return true
	})

	if len(prices) != 5 {
		t.Errorf("Expected 5 prices, got %d", len(prices))
	}
}

func TestSkipTree_Validate(t *testing.T) {
	tree := NewSkipTree(16, false)

	for i := 1; i <= 100; i++ {
		tree.Insert(float64(i), &PriceLevel{Price: float64(i)})
	}

	if !tree.Validate() {
		t.Error("Tree should be valid")
	}
}

func TestSkipTree_Clear(t *testing.T) {
	tree := NewSkipTree(16, false)

	for i := 1; i <= 10; i++ {
		tree.Insert(float64(i), &PriceLevel{Price: float64(i)})
	}

	tree.Clear()

	if tree.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", tree.Len())
	}
}

func TestSkipTree_ConcurrentInsert(t *testing.T) {
	tree := NewSkipTree(16, false)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				price := float64(id*100 + j)
				tree.Insert(price, &PriceLevel{Price: price})
			}
		}(i)
	}

	wg.Wait()

	if tree.Len() != 1000 {
		t.Errorf("Expected 1000 levels, got %d", tree.Len())
	}
}

func TestHybridOrderBook_Match_BuyOrder(t *testing.T) {
	ob := NewHybridOrderBook("BTCUSDT")

	// 添加卖单到订单簿
	sellOrder := &types.Order{
		ID:       1,
		Symbol:   "BTCUSDT",
		Price:    50000.0,
		Quantity: 100,
		Side:     types.SideSell,
		Type:     types.TypeLimit,
	}
	ob.addOrderToBook(sellOrder, 100)

	// 买单撮合
	buyOrder := &types.Order{
		ID:       2,
		Symbol:   "BTCUSDT",
		Price:    50000.0,
		Quantity: 50,
		Side:     types.SideBuy,
		Type:     types.TypeLimit,
	}

	result := ob.Match(buyOrder)

	if len(result.Trades) != 1 {
		t.Errorf("Expected 1 trade, got %d", len(result.Trades))
	}

	if result.Trades[0].Quantity != 50 {
		t.Errorf("Expected trade quantity 50, got %d", result.Trades[0].Quantity)
	}
}

func TestHybridOrderBook_Match_NoMatch(t *testing.T) {
	ob := NewHybridOrderBook("BTCUSDT")

	// 添加卖单
	sellOrder := &types.Order{
		ID:       1,
		Symbol:   "BTCUSDT",
		Price:    51000.0, // 卖价高于买价
		Quantity: 100,
		Side:     types.SideSell,
		Type:     types.TypeLimit,
	}
	ob.addOrderToBook(sellOrder, 100)

	// 买单无法成交
	buyOrder := &types.Order{
		ID:       2,
		Symbol:   "BTCUSDT",
		Price:    50000.0, // 买价低于卖价
		Quantity: 50,
		Side:     types.SideBuy,
		Type:     types.TypeLimit,
	}

	result := ob.Match(buyOrder)

	if len(result.Trades) != 0 {
		t.Error("Should not have trades when prices don't match")
	}
}

func TestHybridOrderBook_CancelOrder(t *testing.T) {
	ob := NewHybridOrderBook("BTCUSDT")

	order := &types.Order{
		ID:       1,
		Symbol:   "BTCUSDT",
		Price:    50000.0,
		Quantity: 100,
		Side:     types.SideBuy,
		Type:     types.TypeLimit,
	}
	ob.addOrderToBook(order, 100)

	// 取消订单
	success := ob.CancelOrder(1)
	if !success {
		t.Error("CancelOrder should succeed")
	}

	// 再次取消应该失败
	success = ob.CancelOrder(1)
	if success {
		t.Error("CancelOrder same order twice should fail")
	}
}

func TestHybridOrderBook_GetBestBidAsk(t *testing.T) {
	ob := NewHybridOrderBook("BTCUSDT")

	// 添加买单
	buyOrder := &types.Order{
		ID:       1,
		Symbol:   "BTCUSDT",
		Price:    49900.0,
		Quantity: 100,
		Side:     types.SideBuy,
		Type:     types.TypeLimit,
	}
	ob.addOrderToBook(buyOrder, 100)

	// 添加卖单
	sellOrder := &types.Order{
		ID:       2,
		Symbol:   "BTCUSDT",
		Price:    50100.0,
		Quantity: 100,
		Side:     types.SideSell,
		Type:     types.TypeLimit,
	}
	ob.addOrderToBook(sellOrder, 100)

	bid, _ := ob.GetBestBid()
	ask, _ := ob.GetBestAsk()

	if bid != 49900.0 {
		t.Errorf("Expected best bid 49900.0, got %f", bid)
	}
	if ask != 50100.0 {
		t.Errorf("Expected best ask 50100.0, got %f", ask)
	}
}

func BenchmarkSkipTree_Insert(b *testing.B) {
	tree := NewSkipTree(16, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Insert(float64(i), &PriceLevel{Price: float64(i)})
	}
}

func BenchmarkSkipTree_Get(b *testing.B) {
	tree := NewSkipTree(16, false)
	for i := 0; i < 10000; i++ {
		tree.Insert(float64(i), &PriceLevel{Price: float64(i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Get(float64(i % 10000))
	}
}

func BenchmarkHybridOrderBook_Match(b *testing.B) {
	ob := NewHybridOrderBook("BTCUSDT")

	// 预填充订单
	for i := 0; i < 1000; i++ {
		order := &types.Order{
			ID:       uint64(i),
			Symbol:   "BTCUSDT",
			Price:    50000.0 + float64(i),
			Quantity: 100,
			Side:     types.SideSell,
			Type:     types.TypeLimit,
		}
		ob.addOrderToBook(order, 100)
	}

	buyOrder := &types.Order{
		ID:       99999,
		Symbol:   "BTCUSDT",
		Price:    51000.0,
		Quantity: 10,
		Side:     types.SideBuy,
		Type:     types.TypeLimit,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buyOrder.ID = uint64(100000 + i)
		ob.Match(buyOrder)
	}
}
