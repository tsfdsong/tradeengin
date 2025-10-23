package logic

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tsfdsong/tradeengin/app/matching/internal/svc"
	"github.com/tsfdsong/tradeengin/app/matching/match"
	"github.com/tsfdsong/tradeengin/app/pkg/types"
	"github.com/tsfdsong/tradeengin/app/pkg/xerr"
)

type ProcessOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProcessOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProcessOrderLogic {
	return &ProcessOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProcessOrderLogic) ProcessOrder(in *match.Order) (*match.MatchResult, error) {
	// 转换订单类型
	order := &types.Order{
		ID:        in.Id,
		Symbol:    in.Symbol,
		Price:     in.Price,
		Quantity:  in.Quantity,
		Side:      int8(in.Side),
		Type:      int8(in.Type),
		Timestamp: in.Timestamp,
		ClientID:  in.ClientId,
	}

	// 处理订单
	result, err := l.svcCtx.Engine.ProcessOrder(order)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("server internal error"), "engin process order failed: %+v, err: %v", in, err)
	}

	// 转换结果类型
	var trades []*match.Trade
	for _, trade := range result.Trades {
		trades = append(trades, &match.Trade{
			TradeId:      trade.TradeID,
			TakerOrderId: trade.TakerOrderID,
			MakerOrderId: trade.MakerOrderID,
			Symbol:       trade.Symbol,
			Price:        trade.Price,
			Quantity:     trade.Quantity,
			Timestamp:    trade.Timestamp,
		})
	}

	return &match.MatchResult{
		Trades:    trades,
		Order:     in,
		Timestamp: result.Timestamp,
	}, nil
}
