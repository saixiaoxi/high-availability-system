package monitors

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrMonitoringSystemUnavailable 表示监控系统不可用
	ErrMonitoringSystemUnavailable = errors.New("monitoring system is unavailable")
)

// MetricType 定义指标类型
type MetricType string

const (
	// CounterType 是一个单调递增的计数器
	CounterType MetricType = "counter"
	// GaugeType 是一个可增可减的值
	GaugeType MetricType = "gauge"
	// HistogramType 对观察值进行采样并统计
	HistogramType MetricType = "histogram"
)

// MetricOption 定义指标选项
type MetricOption struct {
	Name        string            // 指标名称
	Description string            // 指标描述
	Type        MetricType        // 指标类型
	Labels      map[string]string // 标签
}

// Monitor 定义监控系统接口
type Monitor interface {
	// Counter 创建或增加计数器
	Counter(ctx context.Context, name string, value float64, labels map[string]string) error
	// Gauge 设置一个仪表值
	Gauge(ctx context.Context, name string, value float64, labels map[string]string) error
	// Histogram 记录一个直方图观察值
	Histogram(ctx context.Context, name string, value float64, labels map[string]string) error
	// IsHealthy 检查监控系统是否健康
	IsHealthy(ctx context.Context) (bool, error)
}

// FallbackStrategy 定义监控系统失效时的容错策略
type FallbackStrategy interface {
	// HandleFailure 处理监控系统失效的情况
	HandleFailure(ctx context.Context, metricName string, metricType MetricType, value float64, labels map[string]string) error
	// IsEnabled 策略是否启用
	IsEnabled() bool
}

// Monitor 实现：带有容错策略的监控包装器
type MonitorWithFallback struct {
	primaryMonitor    Monitor
	fallbackStrategy  FallbackStrategy
	healthCheckTicker *time.Ticker
	isHealthy         bool
	mutex             sync.RWMutex
	periodicCheck     time.Duration
}

// NewMonitorWithFallback 创建带有容错的监控系统
func NewMonitorWithFallback(primaryMonitor Monitor, fallbackStrategy FallbackStrategy, periodicCheck time.Duration) *MonitorWithFallback {
	m := &MonitorWithFallback{
		primaryMonitor:   primaryMonitor,
		fallbackStrategy: fallbackStrategy,
		isHealthy:        true,
		periodicCheck:    periodicCheck,
	}

	// 定期检查主监控系统的健康状态
	if periodicCheck > 0 {
		m.healthCheckTicker = time.NewTicker(periodicCheck)
		go m.periodicHealthCheck()
	}

	return m
}

// 定期执行健康检查
func (m *MonitorWithFallback) periodicHealthCheck() {
	for range m.healthCheckTicker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		healthy, _ := m.primaryMonitor.IsHealthy(ctx)
		cancel()

		m.mutex.Lock()
		m.isHealthy = healthy
		m.mutex.Unlock()
	}
}

// Counter 增加计数器，如果主系统不可用则使用容错策略
func (m *MonitorWithFallback) Counter(ctx context.Context, name string, value float64, labels map[string]string) error {
	m.mutex.RLock()
	isHealthy := m.isHealthy
	m.mutex.RUnlock()

	// 尝试使用主监控系统
	if isHealthy {
		err := m.primaryMonitor.Counter(ctx, name, value, labels)
		if err == nil {
			return nil
		}

		// 更新健康状态
		m.mutex.Lock()
		m.isHealthy = false
		m.mutex.Unlock()
	}

	// 如果启用了容错策略，使用容错措施
	if m.fallbackStrategy != nil && m.fallbackStrategy.IsEnabled() {
		return m.fallbackStrategy.HandleFailure(ctx, name, CounterType, value, labels)
	}

	return ErrMonitoringSystemUnavailable
}

// Gauge 设置仪表值，如果主系统不可用则使用容错策略
func (m *MonitorWithFallback) Gauge(ctx context.Context, name string, value float64, labels map[string]string) error {
	m.mutex.RLock()
	isHealthy := m.isHealthy
	m.mutex.RUnlock()

	// 尝试使用主监控系统
	if isHealthy {
		err := m.primaryMonitor.Gauge(ctx, name, value, labels)
		if err == nil {
			return nil
		}

		// 更新健康状态
		m.mutex.Lock()
		m.isHealthy = false
		m.mutex.Unlock()
	}

	// 如果启用了容错策略，使用容错措施
	if m.fallbackStrategy != nil && m.fallbackStrategy.IsEnabled() {
		return m.fallbackStrategy.HandleFailure(ctx, name, GaugeType, value, labels)
	}

	return ErrMonitoringSystemUnavailable
}

// Histogram 记录直方图观察值，如果主系统不可用则使用容错策略
func (m *MonitorWithFallback) Histogram(ctx context.Context, name string, value float64, labels map[string]string) error {
	m.mutex.RLock()
	isHealthy := m.isHealthy
	m.mutex.RUnlock()

	// 尝试使用主监控系统
	if isHealthy {
		err := m.primaryMonitor.Histogram(ctx, name, value, labels)
		if err == nil {
			return nil
		}

		// 更新健康状态
		m.mutex.Lock()
		m.isHealthy = false
		m.mutex.Unlock()
	}

	// 如果启用了容错策略，使用容错措施
	if m.fallbackStrategy != nil && m.fallbackStrategy.IsEnabled() {
		return m.fallbackStrategy.HandleFailure(ctx, name, HistogramType, value, labels)
	}

	return ErrMonitoringSystemUnavailable
}

// IsHealthy 检查监控系统是否健康
func (m *MonitorWithFallback) IsHealthy(ctx context.Context) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.isHealthy, nil
}

// Stop 停止定期健康检查
func (m *MonitorWithFallback) Stop() {
	if m.healthCheckTicker != nil {
		m.healthCheckTicker.Stop()
	}
}
