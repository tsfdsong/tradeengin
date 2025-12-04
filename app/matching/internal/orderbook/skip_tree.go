package orderbook

import (
	"math/rand"
	"sync"
	"time"

	"github.com/tsfdsong/tradeengin/app/pkg/types"
)

// SkipNode 跳表节点
type SkipNode struct {
	key     float64
	value   *PriceLevel
	forward []*SkipNode
	level   int
}

// SkipTree 基于跳表的价格树
type SkipTree struct {
	header   *SkipNode
	level    int
	maxLevel int
	reverse  bool // true: 降序, false: 升序
	size     int
	mu       sync.RWMutex
	rngMu    sync.Mutex // 新增: 随机数生成器锁
	rng      *rand.Rand
}

// PriceLevel 价格层级,按照价格点位聚合相同价格的订单集合,优化撮合性能
type PriceLevel struct {
	Price    float64
	TotalQty int64
	Orders   []*types.Order
}

// NewSkipTree 创建跳表价格树
func NewSkipTree(maxLevel int, reverse bool) *SkipTree {
	source := rand.NewSource(time.Now().UnixNano())

	st := &SkipTree{
		header: &SkipNode{
			key:     0,
			value:   nil,
			forward: make([]*SkipNode, maxLevel+1),
			level:   maxLevel,
		},
		level:    0,
		maxLevel: maxLevel,
		reverse:  reverse,
		size:     0,
		rng:      rand.New(source),
	}

	// 初始化头节点的forward指针
	for i := 0; i <= maxLevel; i++ {
		st.header.forward[i] = nil
	}

	return st
}

// randomLevel 随机生成层级 - 修复并发安全问题
func (st *SkipTree) randomLevel() int {
	st.rngMu.Lock()
	defer st.rngMu.Unlock()

	level := 0
	for st.rng.Float32() < 0.5 && level < st.maxLevel {
		level++
	}
	return level
}

// compare 比较函数
func (st *SkipTree) compare(a, b float64) int {
	const epsilon = 1e-10
	diff := a - b

	if diff > epsilon {
		if st.reverse {
			return -1 // 降序：大的在前
		}
		return 1 // 升序：大的在后
	} else if diff < -epsilon {
		if st.reverse {
			return 1 // 降序：小的在后
		}
		return -1 // 升序：小的在前
	}
	return 0
}

// Insert 插入价格层级
func (st *SkipTree) Insert(price float64, level *PriceLevel) {
	st.mu.Lock()
	defer st.mu.Unlock()

	update := make([]*SkipNode, st.maxLevel+1)
	current := st.header

	// 从最高层开始查找插入位置
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil && st.compare(current.forward[i].key, price) < 0 {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]

	// 如果key已存在，更新值
	if current != nil && st.compare(current.key, price) == 0 {
		current.value = level
		return
	}

	// 创建新节点
	newLevel := st.randomLevel()
	if newLevel > st.level {
		for i := st.level + 1; i <= newLevel; i++ {
			update[i] = st.header
		}
		st.level = newLevel
	}

	newNode := &SkipNode{
		key:     price,
		value:   level,
		forward: make([]*SkipNode, newLevel+1),
		level:   newLevel,
	}

	// 更新指针
	for i := 0; i <= newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	st.size++
}

// Remove 移除价格层级
func (st *SkipTree) Remove(price float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	update := make([]*SkipNode, st.maxLevel+1)
	current := st.header

	// 查找要删除的节点
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil && st.compare(current.forward[i].key, price) < 0 {
			current = current.forward[i]
		}
		update[i] = current
	}

	current = current.forward[0]
	if current == nil || st.compare(current.key, price) != 0 {
		return // 节点不存在
	}

	// 更新指针
	for i := 0; i <= st.level; i++ {
		if update[i].forward[i] != current {
			break
		}
		update[i].forward[i] = current.forward[i]
	}

	// 调整层级
	for st.level > 0 && st.header.forward[st.level] == nil {
		st.level--
	}

	st.size--
}

// Get 获取价格层级
func (st *SkipTree) Get(price float64) *PriceLevel {
	st.mu.RLock()
	defer st.mu.RUnlock()

	current := st.header
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil && st.compare(current.forward[i].key, price) < 0 {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	if current != nil && st.compare(current.key, price) == 0 {
		return current.value
	}

	return nil
}

// Len 获取价格层级数量
func (st *SkipTree) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.size
}

// MinPriceNode 获取最小价格节点
func (st *SkipTree) MinPriceNode() *PriceLevel {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if st.size == 0 {
		return nil
	}

	if st.reverse {
		// 降序排列，最小值在最后
		current := st.header
		for i := st.level; i >= 0; i-- {
			for current.forward[i] != nil {
				current = current.forward[i]
			}
		}
		return current.value
	} else {
		// 升序排列，最小值在最前
		current := st.header.forward[0]
		if current != nil {
			return current.value
		}
		return nil
	}
}

// MaxPriceNode 获取最大价格节点
func (st *SkipTree) MaxPriceNode() *PriceLevel {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if st.size == 0 {
		return nil
	}

	if st.reverse {
		// 降序排列，最大值在最前
		current := st.header.forward[0]
		if current != nil {
			return current.value
		}
		return nil
	} else {
		// 升序排列，最大值在最后
		current := st.header
		for i := st.level; i >= 0; i-- {
			for current.forward[i] != nil {
				current = current.forward[i]
			}
		}
		return current.value
	}
}

// GetTopLevels 获取顶部价格层级
func (st *SkipTree) GetTopLevels(limit int) []*PriceLevel {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if st.size == 0 {
		return nil
	}

	var levels []*PriceLevel
	current := st.header

	if st.reverse {
		// 降序：从头开始取前limit个
		current = st.header.forward[0]
		count := 0
		for current != nil && count < limit {
			levels = append(levels, current.value)
			current = current.forward[0]
			count++
		}
	} else {
		// 升序：从头开始取前limit个
		current = st.header.forward[0]
		count := 0
		for current != nil && count < limit {
			levels = append(levels, current.value)
			current = current.forward[0]
			count++
		}
	}

	return levels
}

// Range 范围遍历
func (st *SkipTree) Range(fn func(price float64, level *PriceLevel) bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	current := st.header.forward[0]
	for current != nil {
		if !fn(current.key, current.value) {
			break
		}
		current = current.forward[0]
	}
}

// RangeFrom 从指定价格开始遍历
func (st *SkipTree) RangeFrom(startPrice float64, fn func(price float64, level *PriceLevel) bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	current := st.header
	// 找到起始位置
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil && st.compare(current.forward[i].key, startPrice) < 0 {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	for current != nil {
		if !fn(current.key, current.value) {
			break
		}
		current = current.forward[0]
	}
}

// RangeBetween 在价格范围内遍历
func (st *SkipTree) RangeBetween(minPrice, maxPrice float64, fn func(price float64, level *PriceLevel) bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	current := st.header
	// 找到起始位置
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil && st.compare(current.forward[i].key, minPrice) < 0 {
			current = current.forward[i]
		}
	}

	current = current.forward[0]
	for current != nil && st.compare(current.key, maxPrice) <= 0 {
		if !fn(current.key, current.value) {
			break
		}
		current = current.forward[0]
	}
}

// Clear 清空跳表
func (st *SkipTree) Clear() {
	st.mu.Lock()
	defer st.mu.Unlock()

	// 重置头节点
	for i := 0; i <= st.maxLevel; i++ {
		st.header.forward[i] = nil
	}
	st.level = 0
	st.size = 0
}

// GetAllLevels 获取所有价格层级
func (st *SkipTree) GetAllLevels() []*PriceLevel {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var levels []*PriceLevel
	current := st.header.forward[0]

	for current != nil {
		levels = append(levels, current.value)
		current = current.forward[0]
	}

	return levels
}

// GetPriceRange 获取价格范围
func (st *SkipTree) GetPriceRange() (min, max float64) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if st.size == 0 {
		return 0, 0
	}

	// 最小值
	minNode := st.header.forward[0]
	if minNode != nil {
		min = minNode.key
	}

	// 最大值
	current := st.header
	for i := st.level; i >= 0; i-- {
		for current.forward[i] != nil {
			current = current.forward[i]
		}
	}
	if current != st.header {
		max = current.key
	}

	return min, max
}

// GetTotalQuantity 获取总数量
func (st *SkipTree) GetTotalQuantity() int64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var total int64
	current := st.header.forward[0]

	for current != nil {
		total += current.value.TotalQty
		current = current.forward[0]
	}

	return total
}

// GetOrderCount 获取订单数量
func (st *SkipTree) GetOrderCount() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var count int
	current := st.header.forward[0]

	for current != nil {
		count += len(current.value.Orders)
		current = current.forward[0]
	}

	return count
}

// PrintStructure 打印跳表结构（用于调试）
func (st *SkipTree) PrintStructure() {
	st.mu.RLock()
	defer st.mu.RUnlock()

	for i := st.level; i >= 0; i-- {
		current := st.header.forward[i]
		print("Level ", i, ": ")
		for current != nil {
			print(current.key, " -> ")
			current = current.forward[i]
		}
		println("nil")
	}
}

// Validate 验证跳表结构完整性
func (st *SkipTree) Validate() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// 检查每一层是否有序
	for i := 0; i <= st.level; i++ {
		current := st.header.forward[i]
		prev := st.header

		for current != nil {
			// 检查顺序
			if st.compare(prev.key, current.key) > 0 {
				return false
			}

			// 检查层级是否有效
			if current.level < i {
				return false
			}

			prev = current
			current = current.forward[i]
		}
	}

	return true
}
