package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// RingBuffer 无锁环形队列
type RingBuffer struct {
	buffer   []unsafe.Pointer
	head     uint64
	tail     uint64
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

// Push 无锁入队
func (q *RingBuffer) Push(item unsafe.Pointer) bool {
	for {
		head := atomic.LoadUint64(&q.head)
		tail := atomic.LoadUint64(&q.tail)

		if head-tail >= q.capacity {
			return false // 队列满
		}

		next := head + 1
		if atomic.CompareAndSwapUint64(&q.head, head, next) {
			index := head & q.mask
			atomic.StorePointer(&q.buffer[index], item)
			return true
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

// BatchPop 批量出队
func (q *RingBuffer) BatchPop(max int) []unsafe.Pointer {
	if max <= 0 {
		return nil
	}

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
			items := make([]unsafe.Pointer, count)
			for i := uint64(0); i < count; i++ {
				index := (tail + i) & q.mask
				for {
					item := atomic.LoadPointer(&q.buffer[index])
					if item != nil {
						items[i] = item
						atomic.StorePointer(&q.buffer[index], nil)
						break
					}
				}
			}
			return items
		}
	}
}
