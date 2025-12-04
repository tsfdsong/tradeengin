package interceptor

import (
	"context"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerInterceptor(t *testing.T) {
	// 初始化日志
	logx.Disable()

	interceptor := UnaryServerInterceptor()

	// 创建测试请求上下文
	ctx := context.Background()

	// 创建测试处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond) // 模拟处理时间
		return "response", nil
	}

	// 创建UnaryServerInfo
	info := &grpc.UnaryServerInfo{
		Server:     nil,
		FullMethod: "/test.TestService/TestMethod",
	}

	// 执行拦截器
	resp, err := interceptor(ctx, "request", info, handler)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if resp != "response" {
		t.Errorf("Expected response 'response', got %v", resp)
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	// 初始化日志
	logx.Disable()

	interceptor := UnaryClientInterceptor()

	// 创建测试请求上下文
	ctx := context.Background()

	// 创建测试调用器
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		time.Sleep(10 * time.Millisecond) // 模拟调用时间
		return nil
	}

	// 执行拦截器
	err := interceptor(ctx, "/test.TestService/TestMethod", "request", "reply", nil, invoker)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestExtractTraceID(t *testing.T) {
	// 测试从metadata提取trace id
	md := metadata.Pairs("x-trace-id", "test-trace-id")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	traceID := extractTraceID(ctx)

	if traceID != "test-trace-id" {
		t.Errorf("Expected trace id 'test-trace-id', got '%s'", traceID)
	}
}

func TestInjectTraceID(t *testing.T) {
	// 测试向metadata注入trace id
	ctx := context.Background()

	newCtx := injectTraceID(ctx)

	md, ok := metadata.FromOutgoingContext(newCtx)
	if !ok {
		t.Error("Expected metadata in context")
		return
	}

	values := md.Get("x-trace-id")
	if len(values) == 0 {
		t.Error("Expected trace id in metadata")
		return
	}

	if len(values[0]) == 0 {
		t.Error("Expected non-empty trace id")
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()

	if len(id1) == 0 {
		t.Error("Expected non-empty trace id")
	}

	if id1 == id2 {
		t.Error("Expected different trace ids")
	}
}

func TestRandomString(t *testing.T) {
	str := randomString(10)

	if len(str) != 10 {
		t.Errorf("Expected length 10, got %d", len(str))
	}
}
