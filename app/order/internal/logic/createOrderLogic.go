package logic

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/tsfdsong/tradeengin/app/matching/matchservice"
	"github.com/tsfdsong/tradeengin/app/order/internal/svc"
	"github.com/tsfdsong/tradeengin/app/order/order"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"
	"github.com/tsfdsong/tradeengin/app/pkg/sequencer"
	"github.com/tsfdsong/tradeengin/app/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	// 订单状态常量
	StatusPending  int32 = 0
	StatusPartial  int32 = 1
	StatusFilled   int32 = 2
	StatusRejected int32 = 3
)

type CreateOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateOrderLogic) CreateOrder(in *order.OrderRequest) (*order.OrderResponse, error) {
	// 生成订单ID
	orderID := sequencer.NextID()

	// 设置订单ID和时间戳
	in.Order.Id = orderID
	if in.Order.Timestamp == 0 {
		in.Order.Timestamp = time.Now().UnixNano()
	}

	// 调用 Matching 服务进行撮合
	matchResp, err := l.svcCtx.MatchRpc.ProcessOrder(l.ctx, &matchservice.Order{
		Id:        in.Order.Id,
		Symbol:    in.Order.Symbol,
		Price:     in.Order.Price,
		Quantity:  in.Order.Quantity,
		Side:      in.Order.Side,
		Type:      in.Order.Type,
		ClientId:  in.Order.ClientId,
		Timestamp: in.Order.Timestamp,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrMsg("server internal error"), "match server process order failed: %+v, err: %v", in, err)
	}

	// 根据撮合结果确定订单状态
	status := determineOrderStatus(matchResp)

	return &orderservice.OrderResponse{
		OrderId:   orderID,
		Status:    int32(status),
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func determineOrderStatus(matchResult *matchservice.MatchResult) int32 {
	if len(matchResult.Trades) == 0 {
		return StatusPending
	}

	// 检查是否完全成交
	remainingQty := matchResult.Order.Quantity
	for _, trade := range matchResult.Trades {
		remainingQty -= trade.Quantity
	}

	if remainingQty == 0 {
		return StatusFilled
	} else {
		return StatusPartial
	}
}
