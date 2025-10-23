package svc

import (
	"github.com/tsfdsong/tradeengin/app/matching/matchservice"
	"github.com/tsfdsong/tradeengin/app/order/internal/config"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	MatchRpc matchservice.MatchService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		MatchRpc: matchservice.NewMatchService(zrpc.MustNewClient(c.Matching)),
	}
}
