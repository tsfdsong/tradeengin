package sequencer

import "time"

func NextID() uint64 {
	// 这里使用一个简单的时间戳加随机数的方式生成唯一ID
	// 在生产环境中，建议使用更复杂的分布式ID生成算法，如Snowflake
	return uint64(time.Now().UnixNano())
}
