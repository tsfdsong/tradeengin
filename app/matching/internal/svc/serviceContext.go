package svc

import (
	"github.com/tsfdsong/tradeengin/app/matching/internal/config"
	engine "github.com/tsfdsong/tradeengin/app/matching/internal/engin"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config      config.Config
	Engine      *engine.MatchingEngine
	RedisClient *redis.Redis           // 新增: Redis客户端
	Persister   *engine.RedisPersister // 新增: Redis持久化服务
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config: c,
	}

	// 初始化Redis客户端
	if c.RedisConf.Host != "" {
		svcCtx.RedisClient = redis.MustNewRedis(c.RedisConf)
		logx.Info("Redis client initialized for matching service")
	}

	// 创建撮合引擎
	svcCtx.Engine = engine.NewMatchingEngine(&c)

	// 初始化持久化服务
	if c.Matching.PersistEnabled && svcCtx.RedisClient != nil {
		svcCtx.Persister = engine.NewRedisPersister(
			svcCtx.RedisClient,
			svcCtx.Engine.GetOrderBooks(),
			c.Matching.PersistInterval,
			true,
		)
		logx.Info("Redis persister initialized")
	}

	// 启动引擎
	if err := svcCtx.Engine.Start(); err != nil {
		panic(err)
	}

	return svcCtx
}

// Close 关闭服务上下文
func (s *ServiceContext) Close() {
	if s.Engine != nil {
		s.Engine.Stop()
		logx.Info("Matching engine stopped")
	}
}
