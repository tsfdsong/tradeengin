package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// BreakerMiddleware 熔断中间件 - 使用go-zero的Google SRE熔断器
type BreakerMiddleware struct {
	brk breaker.Breaker
}

// NewBreakerMiddleware 创建熔断中间件
func NewBreakerMiddleware(brk breaker.Breaker) *BreakerMiddleware {
	return &BreakerMiddleware{
		brk: brk,
	}
}

// Handle 处理熔断逻辑
func (m *BreakerMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.brk == nil {
			next(w, r)
			return
		}

		// 使用熔断器包装请求
		promise, err := m.brk.Allow()
		if err != nil {
			// 熔断器打开，拒绝请求
			logx.WithContext(r.Context()).Slowf("circuit breaker open for %s %s", r.Method, r.URL.Path)
			httpx.Error(w, NewCircuitBreakerError())
			return
		}

		// 创建响应记录器来捕获状态码
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 执行请求
		next(rw, r)

		// 根据响应状态标记成功或失败
		if rw.statusCode >= 500 {
			promise.Reject("server error")
		} else {
			promise.Accept()
		}
	}
}

// responseWriter 包装ResponseWriter以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CircuitBreakerError 熔断错误
type CircuitBreakerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewCircuitBreakerError() *CircuitBreakerError {
	return &CircuitBreakerError{
		Code:    503,
		Message: "Service temporarily unavailable, please try again later",
	}
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}
