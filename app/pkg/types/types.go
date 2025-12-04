package types

import (
	"sync"
)

// 订单方向常量
const (
	SideBuyValue  int8 = 1
	SideSellValue int8 = 2
)

// 订单类型常量
const (
	TypeLimitValue  int8 = 1
	TypeMarketValue int8 = 2
)

var (
	SideBuy  int8 = SideBuyValue
	SideSell int8 = SideSellValue

	TypeLimit  int8 = TypeLimitValue
	TypeMarket int8 = TypeMarketValue
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

// Reset 重置Order对象，用于对象池回收
func (o *Order) Reset() {
	o.ID = 0
	o.Symbol = ""
	o.Price = 0
	o.Quantity = 0
	o.Side = 0
	o.Type = 0
	o.Timestamp = 0
	o.ClientID = ""
	o.Version = 0
}

// IsValid 检查订单是否有效
func (o *Order) IsValid() bool {
	if o.Symbol == "" {
		return false
	}
	if o.Quantity <= 0 {
		return false
	}
	if o.Side != SideBuy && o.Side != SideSell {
		return false
	}
	if o.Type != TypeLimit && o.Type != TypeMarket {
		return false
	}
	if o.Type == TypeLimit && o.Price <= 0 {
		return false
	}
	return true
}

// IsBuy 是否是买单
func (o *Order) IsBuy() bool {
	return o.Side == SideBuy
}

// IsSell 是否是卖单
func (o *Order) IsSell() bool {
	return o.Side == SideSell
}

// IsLimit 是否是限价单
func (o *Order) IsLimit() bool {
	return o.Type == TypeLimit
}

// IsMarket 是否是市价单
func (o *Order) IsMarket() bool {
	return o.Type == TypeMarket
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
	TakerSide    int8    `json:"takerSide"` // 新增: Taker方向
}

// Reset 重置Trade对象
func (t *Trade) Reset() {
	t.TradeID = 0
	t.TakerOrderID = 0
	t.MakerOrderID = 0
	t.Symbol = ""
	t.Price = 0
	t.Quantity = 0
	t.Timestamp = 0
	t.TakerSide = 0
}

type MatchResult struct {
	Trades    []*Trade `json:"trades"`
	Order     *Order   `json:"order"`
	Timestamp int64    `json:"timestamp"`
}

// Reset 重置MatchResult对象
func (m *MatchResult) Reset() {
	m.Trades = m.Trades[:0]
	m.Order = nil
	m.Timestamp = 0
}

// HasTrades 是否有成交
func (m *MatchResult) HasTrades() bool {
	return len(m.Trades) > 0
}

// TotalFilledQty 总成交数量
func (m *MatchResult) TotalFilledQty() int64 {
	var total int64
	for _, t := range m.Trades {
		total += t.Quantity
	}
	return total
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
	order.Reset() // 使用Reset方法完全清理
	orderPool.Put(order)
}

func GetTradeFromPool() *Trade {
	return tradePool.Get().(*Trade)
}

func PutTradeToPool(trade *Trade) {
	trade.Reset() // 使用Reset方法完全清理
	tradePool.Put(trade)
}

func GetMatchResultFromPool() *MatchResult {
	result := matchResultPool.Get().(*MatchResult)
	result.Trades = result.Trades[:0]
	return result
}

func PutMatchResultToPool(result *MatchResult) {
	// 先回收trades
	for _, trade := range result.Trades {
		PutTradeToPool(trade)
	}
	result.Reset() // 使用Reset方法完全清理
	matchResultPool.Put(result)
}
