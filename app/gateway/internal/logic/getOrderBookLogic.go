package logic

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/types"
	"github.com/tsfdsong/tradeengin/app/matching/matchservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderBookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderBookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderBookLogic {
	return &GetOrderBookLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderBookLogic) GetOrderBook(req *types.OrderBookReq) (*types.OrderBookResp, error) {
	// 调用 Matching RPC 服务获取订单簿
	resp, err := l.svcCtx.MatchRpc.GetOrderBook(l.ctx, &matchservice.OrderBookRequest{
		Symbol: req.Symbol,
		Depth:  int32(req.Depth),
	})

	if err != nil {
		return nil, errors.Wrapf(err, "GetOrderBook: %+v", req)
	}

	var bids []types.PriceLevel
	var asks []types.PriceLevel

	for _, bid := range resp.Bids {
		bids = append(bids, types.PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
			Count:    int(bid.OrderCount),
		})
	}

	for _, ask := range resp.Asks {
		asks = append(asks, types.PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
			Count:    int(ask.OrderCount),
		})
	}

	return &types.OrderBookResp{
		Symbol: resp.Symbol,
		Bids:   bids,
		Asks:   asks,
		Time:   resp.Timestamp,
	}, nil
}
