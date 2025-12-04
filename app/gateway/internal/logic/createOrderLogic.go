package logic

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/types"
	"github.com/tsfdsong/tradeengin/app/order/order"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"
	pkgtypes "github.com/tsfdsong/tradeengin/app/pkg/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrInvalidSymbol   = errors.New("invalid symbol")
	ErrInvalidQuantity = errors.New("invalid quantity: must be positive")
	ErrInvalidPrice    = errors.New("invalid price: must be non-negative")
	ErrInvalidSide     = errors.New("invalid side: must be 1(buy) or 2(sell)")
	ErrInvalidType     = errors.New("invalid type: must be 1(limit) or 2(market)")
	ErrLimitPriceZero  = errors.New("limit order must have positive price")
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
	// 校验交易对
	if req.Symbol == "" {
		return ErrInvalidSymbol
	}
	// 校验数量
	if req.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	// 校验价格
	if req.Price < 0 {
		return ErrInvalidPrice
	}
	// 校验方向 - 修复: Side应为1(买)或2(卖)
	if req.Side != pkgtypes.SideBuyValue && req.Side != pkgtypes.SideSellValue {
		return ErrInvalidSide
	}
	// 校验订单类型
	if req.Type != pkgtypes.TypeLimitValue && req.Type != pkgtypes.TypeMarketValue {
		return ErrInvalidType
	}
	// 限价单必须有价格
	if req.Type == pkgtypes.TypeLimitValue && req.Price <= 0 {
		return ErrLimitPriceZero
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
