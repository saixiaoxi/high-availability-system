package monitors

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMonitor 实现了基于Prometheus的监控
type PrometheusMonitor struct {
	registry      *prometheus.Registry
	counters      map[string]*prometheus.CounterVec
	gauges        map[string]*prometheus.GaugeVec
	histograms    map[string]*prometheus.HistogramVec
	mutex         sync.RWMutex
	endpoint      string
	server        *http.Server
	serverStarted bool
}

// NewPrometheusMonitor 创建新的Prometheus监控
func NewPrometheusMonitor(endpoint string) *PrometheusMonitor {
	// 创建一个自定义的注册表
	registry := prometheus.NewRegistry()

	// 使用默认注册表注册收集器
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	return &PrometheusMonitor{
		registry:   registry,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		endpoint:   endpoint,
	}
}

// StartServer 启动Prometheus HTTP服务器以暴露指标
func (p *PrometheusMonitor) StartServer(port string) error {
	if p.serverStarted {
		return errors.New("prometheus metrics server already started")
	}

	// 创建HTTP处理器
	mux := http.NewServeMux()
	mux.Handle(p.endpoint, promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))

	// 创建服务器
	p.server = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// 启动服务器
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 记录错误，但不要中断程序
			println("prometheus metrics server error:", err.Error())
		}
	}()

	p.serverStarted = true
	return nil
}

// StopServer 停止Prometheus HTTP服务器
func (p *PrometheusMonitor) StopServer(ctx context.Context) error {
	if p.server != nil && p.serverStarted {
		p.serverStarted = false
		return p.server.Shutdown(ctx)
	}
	return nil
}

// getOrCreateCounter 获取或创建计数器
func (p *PrometheusMonitor) getOrCreateCounter(name string, labels map[string]string) (*prometheus.CounterVec, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 检查是否已经存在
	if counter, ok := p.counters[name]; ok {
		return counter, nil
	}

	// 提取标签键
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}

	// 创建新的计数器
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: name,
		},
		labelKeys,
	)

	// 注册到Prometheus
	if err := p.registry.Register(counter); err != nil {
		// 如果已注册，尝试从现有计数器中获取
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if counterVec, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
				p.counters[name] = counterVec
				return counterVec, nil
			}
		}
		return nil, err
	}

	p.counters[name] = counter
	return counter, nil
}

// getOrCreateGauge 获取或创建仪表
func (p *PrometheusMonitor) getOrCreateGauge(name string, labels map[string]string) (*prometheus.GaugeVec, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 检查是否已经存在
	if gauge, ok := p.gauges[name]; ok {
		return gauge, nil
	}

	// 提取标签键
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}

	// 创建新的仪表
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: name,
		},
		labelKeys,
	)

	// 注册到Prometheus
	if err := p.registry.Register(gauge); err != nil {
		// 如果已注册，尝试从现有仪表中获取
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if gaugeVec, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
				p.gauges[name] = gaugeVec
				return gaugeVec, nil
			}
		}
		return nil, err
	}

	p.gauges[name] = gauge
	return gauge, nil
}

// getOrCreateHistogram 获取或创建直方图
func (p *PrometheusMonitor) getOrCreateHistogram(name string, labels map[string]string) (*prometheus.HistogramVec, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 检查是否已经存在
	if histogram, ok := p.histograms[name]; ok {
		return histogram, nil
	}

	// 提取标签键
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}

	// 创建新的直方图
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    name,
			Buckets: prometheus.DefBuckets,
		},
		labelKeys,
	)

	// 注册到Prometheus
	if err := p.registry.Register(histogram); err != nil {
		// 如果已注册，尝试从现有直方图中获取
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if histogramVec, ok := are.ExistingCollector.(*prometheus.HistogramVec); ok {
				p.histograms[name] = histogramVec
				return histogramVec, nil
			}
		}
		return nil, err
	}

	p.histograms[name] = histogram
	return histogram, nil
}

// Counter 实现Monitor接口的Counter方法
func (p *PrometheusMonitor) Counter(ctx context.Context, name string, value float64, labels map[string]string) error {
	counter, err := p.getOrCreateCounter(name, labels)
	if err != nil {
		return err
	}

	counter.With(labels).Add(value)
	return nil
}

// Gauge 实现Monitor接口的Gauge方法
func (p *PrometheusMonitor) Gauge(ctx context.Context, name string, value float64, labels map[string]string) error {
	gauge, err := p.getOrCreateGauge(name, labels)
	if err != nil {
		return err
	}

	gauge.With(labels).Set(value)
	return nil
}

// Histogram 实现Monitor接口的Histogram方法
func (p *PrometheusMonitor) Histogram(ctx context.Context, name string, value float64, labels map[string]string) error {
	histogram, err := p.getOrCreateHistogram(name, labels)
	if err != nil {
		return err
	}

	histogram.With(labels).Observe(value)
	return nil
}

// IsHealthy 检查Prometheus是否健康
func (p *PrometheusMonitor) IsHealthy(ctx context.Context) (bool, error) {
	if !p.serverStarted {
		return false, errors.New("prometheus metrics server not started")
	}

	// 尝试对度量服务器进行HTTP请求
	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost"+p.server.Addr+p.endpoint, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
