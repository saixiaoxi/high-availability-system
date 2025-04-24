package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/saixiaoxi/high-availability-system/pkg/retry"
)

// RetryConfig 定义重试中间件的配置
type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	RandomFactor    float64
	RetryableStatus []int
	ErrorHandler    func(*gin.Context, error)
}

// DefaultRetryConfig 返回默认的重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		Multiplier:      2.0,
		RandomFactor:    0.5,
		RetryableStatus: []int{
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}
}

// RetryMiddleware 创建一个Gin重试中间件
func RetryMiddleware(config *RetryConfig) gin.HandlerFunc {
	// 如果没有提供配置，使用默认配置
	if config == nil {
		config = DefaultRetryConfig()
	}

	// 转换为重试包的配置
	retryConfig := &retry.Config{
		MaxAttempts:         config.MaxAttempts,
		InitialInterval:     config.InitialInterval,
		MaxInterval:         config.MaxInterval,
		Multiplier:          config.Multiplier,
		RandomizationFactor: config.RandomFactor,
	}

	return func(c *gin.Context) {
		// 只为外部服务请求创建重试
		if c.Request.URL.Path == "/internal/healthcheck" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// 如果请求有body，保存它以便重试
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body.Close()
			// 重置Body以便后面的处理器可以读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应记录器来捕获响应
		w := &responseRecorder{ResponseWriter: c.Writer, statusCode: http.StatusOK}
		c.Writer = w

		// 创建重试上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// 执行带重试的处理
		err := retry.DoWithContext(ctx, func(ctx context.Context) error {
			// 重置响应记录器
			w.statusCode = http.StatusOK
			w.body.Reset()

			// 重置请求体 (如果有)
			if len(requestBody) > 0 {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// 执行请求
			c.Next()

			// 检查是否需要重试
			if shouldRetry(w.statusCode, config.RetryableStatus) {
				return &httpError{statusCode: w.statusCode}
			}

			return nil
		}, retryConfig)

		// 处理最终错误
		if err != nil && config.ErrorHandler != nil {
			config.ErrorHandler(c, err)
		}
	}
}

// 检查状态码是否应该重试
func shouldRetry(statusCode int, retryableStatus []int) bool {
	for _, code := range retryableStatus {
		if statusCode == code {
			return true
		}
	}
	return false
}

// HTTP错误包装
type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return http.StatusText(e.statusCode)
}

// 响应记录器用于捕获响应
type responseRecorder struct {
	gin.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

// 实现ResponseWriter接口的WriteHeader方法
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// 实现ResponseWriter接口的Write方法
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// 实现ResponseWriter接口的WriteString方法
func (r *responseRecorder) WriteString(s string) (int, error) {
	r.body.WriteString(s)
	return r.ResponseWriter.WriteString(s)
}
