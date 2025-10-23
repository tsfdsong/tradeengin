package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Matching MatchingConfig
	// Redis    RedisConfig
}

type MatchingConfig struct {
	Symbols          []string `json:",default=[\"BTCUSD\", \"ETHUSD\"]"`
	OrderBookShards  int      `json:",default=16"`
	BatchSize        int      `json:",default=256"`
	WorkerCount      int      `json:",default=16"`
	SnapshotInterval string   `json:",default=30s"`
}

// type RedisConfig struct {
// 	Host     string `json:",default=localhost:6379"`
// 	Password string `json:",optional"`
// 	DB       int    `json:",default=0"`
// }
