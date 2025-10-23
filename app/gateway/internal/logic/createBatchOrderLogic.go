package logic

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/svc"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/types"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBatchOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateBatchOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBatchOrderLogic {
	return &CreateBatchOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateBatchOrderLogic) CreateBatchOrder(req *types.BatchOrderReq) (*types.BatchOrderResp, error) {
	var orders []*orderservice.Order
	for _, orderReq := range req.Orders {
		orders = append(orders, &orderservice.Order{
			Symbol:    orderReq.Symbol,
			Price:     orderReq.Price,
			Quantity:  orderReq.Quantity,
			Side:      int32(orderReq.Side),
			Type:      int32(orderReq.Type),
			ClientId:  orderReq.ClientID,
			Timestamp: time.Now().UnixNano(),
		})
	}

	resp, err := l.svcCtx.OrderRpc.CreateBatchOrder(l.ctx, &orderservice.BatchOrderRequest{
		Orders: orders,
	})

	if err != nil {
		return nil, errors.Wrapf(err, "CreateBatchOrder: %+v", req)
	}

	var results []types.OrderResp
	for _, result := range resp.Results {
		results = append(results, types.OrderResp{
			OrderID:   result.OrderId,
			Status:    int8(result.Status),
			Timestamp: result.Timestamp,
		})
	}

	return &types.BatchOrderResp{
		Results: results,
	}, nil
}
