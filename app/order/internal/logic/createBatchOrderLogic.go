package logic

import (
	"context"

	"github.com/tsfdsong/tradeengin/app/order/internal/svc"
	"github.com/tsfdsong/tradeengin/app/order/order"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBatchOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateBatchOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBatchOrderLogic {
	return &CreateBatchOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateBatchOrderLogic) CreateBatchOrder(in *order.BatchOrderRequest) (*order.BatchOrderResponse, error) {

	return &orderservice.BatchOrderResponse{}, nil
}
