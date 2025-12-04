package interceptor

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// TraceIDKey 链路追踪ID的key
	TraceIDKey = "x-trace-id"
	// DefaultTimeout 默认超时时间
	DefaultTimeout = 30 * time.Second
	// SlowThreshold 慢请求阈值
	SlowThreshold = 500 * time.Millisecond
)

// UnaryServerInterceptor 服务端一元拦截器
// 提供链路追踪、超时控制、慢请求日志等功能
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := timex.Now()

		// 提取trace id
		traceID := extractTraceID(ctx)

		// 添加超时控制
		ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()

		// 添加traceID到日志上下文
		ctx = logx.ContextWithFields(ctx, logx.Field("traceId", traceID))

		// 执行处理器
		resp, err := handler(ctx, req)

		// 计算耗时
		duration := timex.Since(start)

		// 记录日志
		logRequest(ctx, info.FullMethod, duration, err)

		return resp, err
	}
}

// UnaryClientInterceptor 客户端一元拦截器
// 提供链路追踪传递、超时控制、重试等功能
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := timex.Now()

		// 传递trace id
		ctx = injectTraceID(ctx)

		// 添加超时控制（如果没有设置的话）
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
			defer cancel()
		}

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 计算耗时
		duration := timex.Since(start)

		// 记录日志
		logRequest(ctx, method, duration, err)

		return err
	}
}

// StreamServerInterceptor 服务端流式拦截器
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := timex.Now()
		ctx := ss.Context()

		// 提取trace id
		traceID := extractTraceID(ctx)
		ctx = logx.ContextWithFields(ctx, logx.Field("traceId", traceID))

		// 包装stream
		wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}

		// 执行处理器
		err := handler(srv, wrapped)

		// 计算耗时
		duration := timex.Since(start)

		// 记录日志
		logRequest(ctx, info.FullMethod, duration, err)

		return err
	}
}

// wrappedServerStream 包装的ServerStream
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// extractTraceID 从context中提取trace id
func extractTraceID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return generateTraceID()
	}

	values := md.Get(TraceIDKey)
	if len(values) == 0 {
		return generateTraceID()
	}

	return values[0]
}

// injectTraceID 向context中注入trace id
func injectTraceID(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	// 如果没有trace id，生成一个
	if len(md.Get(TraceIDKey)) == 0 {
		traceID := generateTraceID()
		md = metadata.Join(md, metadata.Pairs(TraceIDKey, traceID))
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// generateTraceID 生成trace id
func generateTraceID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// logRequest 记录请求日志
func logRequest(ctx context.Context, method string, duration time.Duration, err error) {
	if err != nil {
		code := status.Code(err)
		if code == codes.DeadlineExceeded {
			logx.WithContext(ctx).Slowf("[gRPC] timeout | method=%s | duration=%v | error=%v",
				method, duration, err)
		} else {
			logx.WithContext(ctx).Errorf("[gRPC] error | method=%s | duration=%v | error=%v",
				method, duration, err)
		}
		return
	}

	if duration > SlowThreshold {
		logx.WithContext(ctx).Slowf("[gRPC] slow | method=%s | duration=%v", method, duration)
	} else {
		logx.WithContext(ctx).Infof("[gRPC] success | method=%s | duration=%v", method, duration)
	}
}
