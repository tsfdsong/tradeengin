package svc

import (
	"github.com/tsfdsong/tradeengin/app/gateway/internal/config"
	"github.com/tsfdsong/tradeengin/app/gateway/internal/middleware"
	"github.com/tsfdsong/tradeengin/app/matching/matchservice"
	"github.com/tsfdsong/tradeengin/app/order/orderservice"
	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	Metrics      rest.Middleware
	RateLimiter  rest.Middleware // 新增: 限流中间件
	OrderRpc     orderservice.OrderService
	MatchRpc     matchservice.MatchService
	RedisClient  *redis.Redis        // 新增: Redis客户端
	OrderLimiter *limit.TokenLimiter // 新增: 订单接口限流器
	Breaker      breaker.Breaker     // 新增: 熔断器
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config:   c,
		Metrics:  middleware.NewMetricsMiddleware().Handle,
		OrderRpc: orderservice.NewOrderService(zrpc.MustNewClient(c.OrderRpc)),
		MatchRpc: matchservice.NewMatchService(zrpc.MustNewClient(c.MatchRpc)),
	}

	// 初始化Redis客户端
	if c.RedisConf.Host != "" {
		svcCtx.RedisClient = redis.MustNewRedis(c.RedisConf)
		logx.Info("Redis client initialized")
	}

	// 初始化限流器 - 使用go-zero的令牌桶限流
	if c.RateLimit.Enabled && svcCtx.RedisClient != nil {
		svcCtx.OrderLimiter = limit.NewTokenLimiter(
			c.RateLimit.OrderRate,
			c.RateLimit.OrderBurst,
			svcCtx.RedisClient,
			"order:ratelimit",
		)
		svcCtx.RateLimiter = middleware.NewRateLimitMiddleware(svcCtx.OrderLimiter).Handle
		logx.Infof("Rate limiter initialized: rate=%d, burst=%d", c.RateLimit.OrderRate, c.RateLimit.OrderBurst)
	}

	// 初始化熔断器 - 使用go-zero的Google SRE熔断器
	if c.Breaker.Enabled {
		svcCtx.Breaker = breaker.NewBreaker(breaker.WithName("gateway"))
		logx.Info("Circuit breaker initialized")
	}

	return svcCtx
}
