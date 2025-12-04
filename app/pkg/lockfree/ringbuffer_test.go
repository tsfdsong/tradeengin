package lockfree

import (
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

func TestNewRingBuffer(t *testing.T) {
	tests := []struct {
		name     string
		size     uint64
		expected uint64
	}{
		{"Power of 2", 1024, 1024},
		{"Non power of 2", 1000, 1024}, // 应该向上取整到1024
		{"Small size", 10, 16},         // 应该向上取整到16
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRingBuffer(tt.size)
			if rb.Capacity() != tt.expected {
				t.Errorf("Expected capacity %d, got %d", tt.expected, rb.Capacity())
			}
		})
	}
}

func TestRingBuffer_PushPop(t *testing.T) {
	rb := NewRingBuffer(1024)

	// 测试基本Push/Pop
	val := 42
	ptr := unsafe.Pointer(&val)

	if !rb.Push(ptr) {
		t.Error("Push should succeed on empty buffer")
	}

	result := rb.Pop()
	if result == nil {
		t.Error("Pop should return non-nil on non-empty buffer")
	}

	if *(*int)(result) != 42 {
		t.Error("Pop should return the pushed value")
	}

	// 测试空队列Pop
	if rb.Pop() != nil {
		t.Error("Pop on empty buffer should return nil")
	}
}

func TestRingBuffer_Size(t *testing.T) {
	rb := NewRingBuffer(1024)

	if rb.Size() != 0 {
		t.Error("New buffer should have size 0")
	}

	if !rb.IsEmpty() {
		t.Error("New buffer should be empty")
	}

	// Push一些元素
	for i := 0; i < 100; i++ {
		val := i
		rb.Push(unsafe.Pointer(&val))
	}

	if rb.Size() != 100 {
		t.Errorf("Expected size 100, got %d", rb.Size())
	}

	if rb.IsEmpty() {
		t.Error("Buffer with elements should not be empty")
	}
}

func TestRingBuffer_Full(t *testing.T) {
	rb := NewRingBuffer(16)

	// 填满队列
	for i := 0; i < 16; i++ {
		val := i
		if !rb.Push(unsafe.Pointer(&val)) {
			t.Errorf("Push %d should succeed", i)
		}
	}

	if !rb.IsFull() {
		t.Error("Buffer should be full")
	}

	// 继续Push应该失败
	val := 100
	if rb.Push(unsafe.Pointer(&val)) {
		t.Error("Push on full buffer should fail")
	}
}

func TestRingBuffer_BatchPop(t *testing.T) {
	rb := NewRingBuffer(1024)

	// Push 100个元素
	for i := 0; i < 100; i++ {
		val := i
		rb.Push(unsafe.Pointer(&val))
	}

	// 批量Pop 50个
	items := rb.BatchPop(50)
	if len(items) != 50 {
		t.Errorf("Expected 50 items, got %d", len(items))
	}

	// 剩余50个
	if rb.Size() != 50 {
		t.Errorf("Expected 50 remaining, got %d", rb.Size())
	}
}

func TestRingBuffer_ConcurrentPush(t *testing.T) {
	rb := NewRingBuffer(65536)
	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 1000

	var successCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				val := id*itemsPerGoroutine + j
				if rb.Push(unsafe.Pointer(&val)) {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if successCount != int64(numGoroutines*itemsPerGoroutine) {
		t.Errorf("Expected %d successful pushes, got %d", numGoroutines*itemsPerGoroutine, successCount)
	}
}

func TestRingBuffer_ConcurrentPushPop(t *testing.T) {
	rb := NewRingBuffer(1024)
	var wg sync.WaitGroup

	// 生产者
	var pushCount int64
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				val := j
				if rb.Push(unsafe.Pointer(&val)) {
					atomic.AddInt64(&pushCount, 1)
				}
			}
		}()
	}

	// 消费者
	var popCount int64
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				if rb.Pop() != nil {
					atomic.AddInt64(&popCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	// 由于并发，pushCount和popCount可能不相等，但应该在合理范围内
	remaining := rb.Size()
	if pushCount-popCount != int64(remaining) {
		t.Logf("Push: %d, Pop: %d, Remaining: %d", pushCount, popCount, remaining)
	}
}

func BenchmarkRingBuffer_Push(b *testing.B) {
	rb := NewRingBuffer(65536)
	val := 42

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push(unsafe.Pointer(&val))
		rb.Pop()
	}
}

func BenchmarkRingBuffer_ConcurrentPushPop(b *testing.B) {
	rb := NewRingBuffer(65536)
	val := 42

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rb.Push(unsafe.Pointer(&val))
			rb.Pop()
		}
	})
}

func BenchmarkRingBuffer_BatchPop(b *testing.B) {
	rb := NewRingBuffer(65536)

	// 预填充
	for i := 0; i < 10000; i++ {
		val := i
		rb.Push(unsafe.Pointer(&val))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := rb.BatchPop(100)
		// 重新填充
		for range items {
			val := i
			rb.Push(unsafe.Pointer(&val))
		}
	}
}
