package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	OrderRpc    zrpc.RpcClientConf
	MatchRpc    zrpc.RpcClientConf
	Gateway     GatewayConfig
	RedisConf   redis.RedisConf
	RateLimit   RateLimitConfig   // 新增: 限流配置
	Breaker     BreakerConfig     // 新增: 熔断配置
	HealthCheck HealthCheckConfig // 新增: 健康检查配置
}

type GatewayConfig struct {
	WorkerCount   int    `json:",default=16"`
	QueueSize     int    `json:",default=65536"`
	BatchSize     int    `json:",default=256"`
	FlushInterval string `json:",default=100ms"`
}

// RateLimitConfig 限流配置 - 使用go-zero的滑动窗口限流
type RateLimitConfig struct {
	Enabled    bool `json:",default=true"`
	Rate       int  `json:",default=10000"` // 每秒请求数
	Burst      int  `json:",default=20000"` // 突发容量
	OrderRate  int  `json:",default=5000"`  // 订单接口限流
	OrderBurst int  `json:",default=10000"` // 订单接口突发
}

// BreakerConfig 熔断配置 - 使用go-zero的Google SRE熔断器
type BreakerConfig struct {
	Enabled bool `json:",default=true"`
	// K值越大越容易熔断，默认1.5表示当成功率低于66.7%时熔断
	K float64 `json:",default=1.5"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool   `json:",default=true"`
	Path     string `json:",default=/health"`
	LivePath string `json:",default=/health/live"`
}
