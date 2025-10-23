package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	OrderRpc  zrpc.RpcClientConf
	MatchRpc  zrpc.RpcClientConf
	Gateway   GatewayConfig
	RedisConf RedisConfig
}

type GatewayConfig struct {
	WorkerCount   int    `json:",default=16"`
	QueueSize     int    `json:",default=65536"`
	BatchSize     int    `json:",default=256"`
	FlushInterval string `json:",default=100ms"`
}

type RedisConfig struct {
	Host     string `json:",default=localhost:6379"`
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}
