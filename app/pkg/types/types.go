package types

import (
	"sync"
)

var (
	SideBuy  int8 = 1
	SideSell int8 = 2

	TypeLimit  int8 = 1
	TypeMarket int8 = 2
)

// Order 订单结构 - 内存对齐优化
type Order struct {
	ID        uint64  `json:"id"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Quantity  int64   `json:"quantity"`
	Side      int8    `json:"side"`
	Type      int8    `json:"type"`
	Timestamp int64   `json:"timestamp"`
	ClientID  string  `json:"clientId"`
	Version   uint32  `json:"version"`
	_         [4]byte // 填充对齐到128字节
}

// // 确保Order结构128字节对齐
// func init() {
// 	if unsafe.Sizeof(Order{}) != 128 {
// 		panic("Order struct size not aligned to 128 bytes")
// 	}
// }

type Trade struct {
	TradeID      uint64  `json:"tradeId"`
	TakerOrderID uint64  `json:"takerOrderId"`
	MakerOrderID uint64  `json:"makerOrderId"`
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Quantity     int64   `json:"quantity"`
	Timestamp    int64   `json:"timestamp"`
}

type MatchResult struct {
	Trades    []*Trade `json:"trades"`
	Order     *Order   `json:"order"`
	Timestamp int64    `json:"timestamp"`
}

type OrderBook struct {
	Symbol string       `json:"symbol"`
	Bids   []PriceLevel `json:"bids"`
	Asks   []PriceLevel `json:"asks"`
	Time   int64        `json:"time"`
}

type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
	Count    int     `json:"count"`
}

// 对象池
var (
	orderPool = sync.Pool{
		New: func() interface{} {
			return &Order{}
		},
	}

	tradePool = sync.Pool{
		New: func() interface{} {
			return &Trade{}
		},
	}

	matchResultPool = sync.Pool{
		New: func() interface{} {
			return &MatchResult{
				Trades: make([]*Trade, 0, 10),
			}
		},
	}
)

func GetOrderFromPool() *Order {
	return orderPool.Get().(*Order)
}

func PutOrderToPool(order *Order) {
	*order = Order{}
	orderPool.Put(order)
}

func GetTradeFromPool() *Trade {
	return tradePool.Get().(*Trade)
}

func PutTradeToPool(trade *Trade) {
	*trade = Trade{}
	tradePool.Put(trade)
}

func GetMatchResultFromPool() *MatchResult {
	result := matchResultPool.Get().(*MatchResult)
	result.Trades = result.Trades[:0]
	return result
}

func PutMatchResultToPool(result *MatchResult) {
	for _, trade := range result.Trades {
		PutTradeToPool(trade)
	}
	result.Trades = result.Trades[:0]
	matchResultPool.Put(result)
}
