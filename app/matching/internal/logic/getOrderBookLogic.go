package logic

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tsfdsong/tradeengin/app/matching/internal/svc"
	"github.com/tsfdsong/tradeengin/app/matching/match"
	"github.com/tsfdsong/tradeengin/app/pkg/xerr"
)

type GetOrderBookLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderBookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderBookLogic {
	return &GetOrderBookLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderBookLogic) GetOrderBook(in *match.OrderBookRequest) (*match.OrderBookSnapshot, error) {
	orderBook, err := l.svcCtx.Engine.GetOrderBook(in.Symbol, int(in.Depth))
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("get orderbook failed"), "get orderbook failed: %+v, err: %v", in, err)
	}

	var bids []*match.PriceLevel
	var asks []*match.PriceLevel

	for _, bid := range orderBook.Bids {
		bids = append(bids, &match.PriceLevel{
			Price:      bid.Price,
			Quantity:   bid.Quantity,
			OrderCount: int32(bid.Count),
		})
	}

	for _, ask := range orderBook.Asks {
		asks = append(asks, &match.PriceLevel{
			Price:      ask.Price,
			Quantity:   ask.Quantity,
			OrderCount: int32(ask.Count),
		})
	}

	return &match.OrderBookSnapshot{
		Symbol:    orderBook.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: orderBook.Time,
	}, nil
}
