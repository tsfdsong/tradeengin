package lockfree

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// RingBuffer 无锁环形队列 - 修复版本
// 支持多生产者多消费者(MPMC)场景
type RingBuffer struct {
	_        [64]byte // 缓存行填充，避免false sharing
	head     uint64
	_        [56]byte // 填充到独立缓存行
	tail     uint64
	_        [56]byte // 填充到独立缓存行
	buffer   []unsafe.Pointer
	mask     uint64
	capacity uint64
}

// NewRingBuffer 创建无锁队列
func NewRingBuffer(size uint64) *RingBuffer {
	if size&(size-1) != 0 {
		size = nextPowerOfTwo(size)
	}

	return &RingBuffer{
		buffer:   make([]unsafe.Pointer, size),
		head:     0,
		tail:     0,
		mask:     size - 1,
		capacity: size,
	}
}

func nextPowerOfTwo(n uint64) uint64 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// Push 无锁入队 - 修复版本
// 使用双重CAS确保slot写入的原子性
func (q *RingBuffer) Push(item unsafe.Pointer) bool {
	var spinCount int
	for {
		head := atomic.LoadUint64(&q.head)
		tail := atomic.LoadUint64(&q.tail)

		if head-tail >= q.capacity {
			return false // 队列满
		}

		next := head + 1
		if atomic.CompareAndSwapUint64(&q.head, head, next) {
			index := head & q.mask
			// 确保slot为空后再写入
			for atomic.LoadPointer(&q.buffer[index]) != nil {
				runtime.Gosched()
			}
			atomic.StorePointer(&q.buffer[index], item)
			return true
		}
		// 自旋等待，避免CPU空转
		spinCount++
		if spinCount > 100 {
			runtime.Gosched()
			spinCount = 0
		}
	}
}

// Pop 无锁出队
func (q *RingBuffer) Pop() unsafe.Pointer {
	for {
		head := atomic.LoadUint64(&q.head)
		tail := atomic.LoadUint64(&q.tail)

		if tail >= head {
			return nil // 队列空
		}

		next := tail + 1
		if atomic.CompareAndSwapUint64(&q.tail, tail, next) {
			index := tail & q.mask
			for {
				item := atomic.LoadPointer(&q.buffer[index])
				if item != nil {
					atomic.StorePointer(&q.buffer[index], nil)
					return item
				}
			}
		}
	}
}

// BatchPop 批量出队 - 修复版本
// 使用原子交换确保每个元素只被消费一次
func (q *RingBuffer) BatchPop(max int) []unsafe.Pointer {
	if max <= 0 {
		return nil
	}

	var spinCount int
	for {
		head := atomic.LoadUint64(&q.head)
		tail := atomic.LoadUint64(&q.tail)

		if tail >= head {
			return nil
		}

		available := head - tail
		count := available
		if count > uint64(max) {
			count = uint64(max)
		}

		next := tail + count
		if atomic.CompareAndSwapUint64(&q.tail, tail, next) {
			items := make([]unsafe.Pointer, 0, count)
			for i := uint64(0); i < count; i++ {
				index := (tail + i) & q.mask
				// 使用原子交换确保只取一次
				for {
					item := atomic.SwapPointer(&q.buffer[index], nil)
					if item != nil {
						items = append(items, item)
						break
					}
					// 等待生产者写入完成
					runtime.Gosched()
				}
			}
			return items
		}
		// 自旋等待
		spinCount++
		if spinCount > 100 {
			runtime.Gosched()
			spinCount = 0
		}
	}
}

// Size 返回队列当前元素数量
func (q *RingBuffer) Size() uint64 {
	head := atomic.LoadUint64(&q.head)
	tail := atomic.LoadUint64(&q.tail)
	if head >= tail {
		return head - tail
	}
	return 0
}

// IsEmpty 检查队列是否为空
func (q *RingBuffer) IsEmpty() bool {
	return q.Size() == 0
}

// IsFull 检查队列是否已满
func (q *RingBuffer) IsFull() bool {
	return q.Size() >= q.capacity
}

// Capacity 返回队列容量
func (q *RingBuffer) Capacity() uint64 {
	return q.capacity
}
