package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// LocalLoggingFallback 实现了基于本地日志记录的容错策略
type LocalLoggingFallback struct {
	enabled bool
	logger  *logrus.Logger
	buffer  []MetricData
	mutex   sync.Mutex
	maxSize int
}

// MetricData 表示要记录的指标数据
type MetricData struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// NewLocalLoggingFallback 创建一个新的基于本地日志的容错策略
func NewLocalLoggingFallback(enabled bool, logPath string, maxBufferSize int) *LocalLoggingFallback {
	logger := logrus.New()

	// 配置日志输出
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.Warnf("Failed to log to file, using default stderr: %v", err)
		}
	}

	// 设置日志格式
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &LocalLoggingFallback{
		enabled: enabled,
		logger:  logger,
		buffer:  make([]MetricData, 0, maxBufferSize),
		maxSize: maxBufferSize,
	}
}

// IsEnabled 检查策略是否启用
func (l *LocalLoggingFallback) IsEnabled() bool {
	return l.enabled
}

// HandleFailure 实现失效处理
func (l *LocalLoggingFallback) HandleFailure(ctx context.Context, metricName string, metricType MetricType, value float64, labels map[string]string) error {
	if !l.enabled {
		return nil
	}

	// 创建指标数据
	metricData := MetricData{
		Name:      metricName,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	// 将指标添加到缓冲区
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 如果缓冲区已满，记录日志并清空
	if len(l.buffer) >= l.maxSize {
		l.flushBuffer()
	}

	l.buffer = append(l.buffer, metricData)
	return nil
}

// flushBuffer 将缓冲区数据写入日志
func (l *LocalLoggingFallback) flushBuffer() {
	if len(l.buffer) == 0 {
		return
	}

	// 将缓冲区转为JSON
	data, err := json.Marshal(l.buffer)
	if err != nil {
		l.logger.Errorf("Failed to marshal metrics buffer: %v", err)
		return
	}

	// 记录到日志
	l.logger.WithField("metrics_count", len(l.buffer)).Info(string(data))

	// 清空缓冲区
	l.buffer = l.buffer[:0]
}

// Flush 强制刷新缓冲区
func (l *LocalLoggingFallback) Flush() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.flushBuffer()
}

// Stop 停止并刷新所有指标
func (l *LocalLoggingFallback) Stop() {
	l.Flush()
}

// PeriodicFlusher 定期执行缓冲区刷新
type PeriodicFlusher struct {
	fallback  *LocalLoggingFallback
	interval  time.Duration
	ticker    *time.Ticker
	stopChan  chan struct{}
	stopOnce  sync.Once
	isRunning bool
}

// NewPeriodicFlusher 创建一个新的定期刷新器
func NewPeriodicFlusher(fallback *LocalLoggingFallback, interval time.Duration) *PeriodicFlusher {
	return &PeriodicFlusher{
		fallback:  fallback,
		interval:  interval,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// Start 开始定期刷新
func (p *PeriodicFlusher) Start() {
	if p.isRunning {
		return
	}

	p.ticker = time.NewTicker(p.interval)
	p.isRunning = true

	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.fallback.Flush()
			case <-p.stopChan:
				return
			}
		}
	}()
}

// Stop 停止定期刷新
func (p *PeriodicFlusher) Stop() {
	p.stopOnce.Do(func() {
		if p.isRunning {
			close(p.stopChan)
			p.ticker.Stop()
			p.isRunning = false

			// 最后刷新一次
			p.fallback.Flush()
		}
	})
}

// FlushOnShutdown 在程序关闭时确保数据被刷新
func FlushOnShutdown(fallback *LocalLoggingFallback) {
	// 记录当前时间以计算持续时间
	startTime := time.Now()

	// 在退出时刷新数据
	fallback.Flush()

	// 记录刷新持续时间
	duration := time.Since(startTime)
	fmt.Printf("Metrics flushed in %v\n", duration)
}
