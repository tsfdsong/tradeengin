package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	RedisConf RedisConfig
	Matching  zrpc.RpcClientConf
}

type RedisConfig struct {
	Host     string `json:",default=localhost:6379"`
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}
