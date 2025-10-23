package svc

import (
	"github.com/tsfdsong/tradeengin/app/gateway/internal/config"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/middleware"
	"github.com/tsfdsong/tradeengin/app/matching/matchservice"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	Metrics  rest.Middleware
	OrderRpc orderservice.OrderService
	MatchRpc matchservice.MatchService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		Metrics:  middleware.NewMetricsMiddleware().Handle,
		OrderRpc: orderservice.NewOrderService(zrpc.MustNewClient(c.OrderRpc)),
		MatchRpc: matchservice.NewMatchService(zrpc.MustNewClient(c.MatchRpc)),
	}
}
