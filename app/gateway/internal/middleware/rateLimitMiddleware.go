package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// RateLimitMiddleware 限流中间件 - 使用go-zero的TokenLimiter
type RateLimitMiddleware struct {
	limiter *limit.TokenLimiter
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(limiter *limit.TokenLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// Handle 处理限流逻辑
func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.limiter == nil {
			next(w, r)
			return
		}

		// 尝试获取令牌
		if m.limiter.Allow() {
			next(w, r)
		} else {
			logx.WithContext(r.Context()).Slowf("rate limit exceeded for %s %s", r.Method, r.URL.Path)
			httpx.Error(w, NewRateLimitError())
		}
	}
}

// RateLimitError 限流错误
type RateLimitError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewRateLimitError() *RateLimitError {
	return &RateLimitError{
		Code:    429,
		Message: "Too many requests, please try again later",
	}
}

func (e *RateLimitError) Error() string {
	return e.Message
}
