package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Matching  MatchingConfig
	RedisConf redis.RedisConf // 修改为go-zero的Redis配置
}

type MatchingConfig struct {
	Symbols          []string `json:",default=[\"BTCUSD\", \"ETHUSD\"]"`
	OrderBookShards  int      `json:",default=16"`
	BatchSize        int      `json:",default=256"`
	WorkerCount      int      `json:",default=16"`
	SnapshotInterval string   `json:",default=30s"`
	PersistEnabled   bool     `json:",default=true"` // 新增: 是否启用持久化
	PersistInterval  string   `json:",default=5s"`   // 新增: 持久化间隔
}
