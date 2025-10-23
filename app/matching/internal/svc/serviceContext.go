package svc

import (
	"github.com/tsfdsong/tradeengin/app/matching/internal/config"
	engine "github.com/tsfdsong/tradeengin/app/matching/internal/engin"
)

type ServiceContext struct {
	Config config.Config
	Engine *engine.MatchingEngine
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 创建撮合引擎
	matchingEngine := engine.NewMatchingEngine(&c)

	// 启动引擎
	if err := matchingEngine.Start(); err != nil {
		panic(err)
	}
	return &ServiceContext{
		Config: c,
		Engine: matchingEngine,
	}
}
