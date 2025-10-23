package model

import (
	"sync"

	"github.com/tsfdsong/tradeengin/app/pkg/types"
)

var (
	orderPool = sync.Pool{
		New: func() interface{} {
			return &types.Order{}
		},
	}

	tradePool = sync.Pool{
		New: func() interface{} {
			return &types.Trade{}
		},
	}

	matchResultPool = sync.Pool{
		New: func() interface{} {
			return &types.MatchResult{
				Trades: make([]*types.Trade, 0, 10),
			}
		},
	}
)

func GetOrderFromPool() *types.Order {
	return orderPool.Get().(*types.Order)
}

func PutOrderToPool(order *types.Order) {
	// 重置订单字段
	*order = types.Order{}
	orderPool.Put(order)
}

func GetTradeFromPool() *types.Trade {
	return tradePool.Get().(*types.Trade)
}

func PutTradeToPool(trade *types.Trade) {
	*trade = types.Trade{}
	tradePool.Put(trade)
}

func GetMatchResultFromPool() *types.MatchResult {
	result := matchResultPool.Get().(*types.MatchResult)
	// 清空交易列表但保持容量
	result.Trades = result.Trades[:0]
	return result
}

func PutMatchResultToPool(result *types.MatchResult) {
	// 归还交易对象到池
	for _, trade := range result.Trades {
		PutTradeToPool(trade)
	}
	result.Trades = result.Trades[:0]
	matchResultPool.Put(result)
}
