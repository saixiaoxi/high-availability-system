package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/high-availability-system/internal/monitors"
)

// MonitoringMiddleware 创建一个用于记录API指标的中间件
func MonitoringMiddleware(monitor *monitors.MonitorWithFallback) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 继续处理请求
		c.Next()

		// 记录请求持续时间
		duration := time.Since(start)
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		// 获取状态码和请求方法
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		// 创建标签
		labels := map[string]string{
			"path":   path,
			"method": method,
			"status": status,
		}

		// 不阻塞请求处理
		go func() {
			ctx := c.Request.Context()

			// 记录请求计数
			_ = monitor.Counter(ctx, "http_requests_total", 1, labels)

			// 记录请求持续时间
			_ = monitor.Histogram(ctx, "http_request_duration_seconds", duration.Seconds(), labels)

			// 记录当前活动请求（在实际应用中需要更复杂的机制）
			_ = monitor.Gauge(ctx, "http_requests_active", 0, labels)
		}()
	}
}

// ErrorMonitoring 创建一个用于记录API错误的中间件
func ErrorMonitoring(monitor *monitors.MonitorWithFallback) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			// 获取路径和错误信息
			path := c.FullPath()
			if path == "" {
				path = "unknown"
			}

			// 对于每个错误增加计数
			for _, err := range c.Errors {
				errType := fmt.Sprintf("%T", err.Err)

				// 创建标签
				labels := map[string]string{
					"path":       path,
					"method":     c.Request.Method,
					"error":      err.Error(),
					"error_type": errType,
				}

				// 记录错误指标
				go func() {
					ctx := c.Request.Context()
					_ = monitor.Counter(ctx, "http_errors_total", 1, labels)
				}()
			}
		}
	}
}

// HealthCheckHandler 创建健康检查处理函数
func HealthCheckHandler(monitor *monitors.MonitorWithFallback) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查监控系统状态
		isHealthy, _ := monitor.IsHealthy(c)

		if isHealthy {
			c.JSON(200, gin.H{
				"status": "UP",
				"details": gin.H{
					"monitoring": "UP",
				},
			})
		} else {
			c.JSON(200, gin.H{
				"status": "UP",
				"details": gin.H{
					"monitoring": "DOWN",
					"notes":      "Using fallback strategy for monitoring",
				},
			})
		}
	}
}
