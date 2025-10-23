package logic

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/types"
	"github.com/tsfdsong/tradeengin/app/order/order"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrInvalidSymbol   = errors.New("invalid symbol")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrInvalidPrice    = errors.New("invalid price")
	ErrInvalidSide     = errors.New("invalid side")
)

type CreateOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOrderLogic) validateOrder(req *types.OrderReq) error {
	if req.Symbol == "" {
		return ErrInvalidSymbol
	}
	if req.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if req.Price < 0 {
		return ErrInvalidPrice
	}
	if req.Side != 0 && req.Side != 1 {
		return ErrInvalidSide
	}
	return nil
}

func (l *CreateOrderLogic) CreateOrder(req *types.OrderReq) (*types.OrderResp, error) {
	// 参数校验
	if err := l.validateOrder(req); err != nil {
		return nil, err
	}

	// 调用 Order 服务创建订单
	orderResp, err := l.svcCtx.OrderRpc.CreateOrder(l.ctx, &orderservice.OrderRequest{
		Order: &order.Order{
			Symbol:    req.Symbol,
			Quantity:  req.Quantity,
			Price:     req.Price,
			Side:      int32(req.Side),
			Type:      int32(req.Type),
			ClientId:  req.ClientID,
			Timestamp: time.Now().UnixNano(),
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "CreateOrder: %+v", req)
	}

	// 返回结果
	return &types.OrderResp{
		OrderID:   orderResp.OrderId,
		Status:    int8(orderResp.Status),
		Timestamp: orderResp.Timestamp,
	}, nil
}
