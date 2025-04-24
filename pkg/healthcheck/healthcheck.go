package healthcheck

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// Status 表示健康检查状态
type Status string

const (
	// StatusUp 表示检查通过
	StatusUp Status = "UP"
	// StatusDown 表示检查失败
	StatusDown Status = "DOWN"
)

// Check 是一个健康检查接口
type Check interface {
	// Name 返回检查的名称
	Name() string
	// Execute 执行检查并返回结果
	Execute(ctx context.Context) (Status, error)
}

// Result 表示单个健康检查的结果
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// AggregateResult 表示所有健康检查的聚合结果
type AggregateResult struct {
	Status  Status            `json:"status"`
	Details map[string]Result `json:"details"`
}

// Checker 是健康检查管理器
type Checker struct {
	checks []Check
	mutex  sync.RWMutex
}

// NewChecker 创建新的健康检查管理器
func NewChecker() *Checker {
	return &Checker{
		checks: []Check{},
	}
}

// AddCheck 添加一个健康检查
func (c *Checker) AddCheck(check Check) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.checks = append(c.checks, check)
}

// RunChecks 执行所有健康检查
func (c *Checker) RunChecks(ctx context.Context) AggregateResult {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	aggregateResult := AggregateResult{
		Status:  StatusUp,
		Details: make(map[string]Result),
	}

	// 如果没有配置检查项，返回UP状态
	if len(c.checks) == 0 {
		return aggregateResult
	}

	// 执行所有检查
	for _, check := range c.checks {
		status, err := check.Execute(ctx)
		result := Result{
			Name:   check.Name(),
			Status: status,
		}

		if err != nil {
			result.Error = err.Error()
		}

		aggregateResult.Details[check.Name()] = result

		// 如果任何一个检查失败，设置总体状态为DOWN
		if status == StatusDown {
			aggregateResult.Status = StatusDown
		}
	}

	return aggregateResult
}

// HTTPCheck 实现了对HTTP服务的健康检查
type HTTPCheck struct {
	name    string
	url     string
	timeout time.Duration
	client  *http.Client
}

// NewHTTPCheck 创建一个新的HTTP健康检查
func NewHTTPCheck(name, url string, timeout time.Duration) *HTTPCheck {
	return &HTTPCheck{
		name:    name,
		url:     url,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name 返回检查名称
func (h *HTTPCheck) Name() string {
	return h.name
}

// Execute 执行HTTP健康检查
func (h *HTTPCheck) Execute(ctx context.Context) (Status, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
	if err != nil {
		return StatusDown, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return StatusDown, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StatusDown, errors.New("unexpected status code: " + resp.Status)
	}

	return StatusUp, nil
}

// CustomCheck 实现了一个自定义函数健康检查
type CustomCheck struct {
	name string
	fn   func(ctx context.Context) (Status, error)
}

// NewCustomCheck 创建一个自定义函数健康检查
func NewCustomCheck(name string, fn func(ctx context.Context) (Status, error)) *CustomCheck {
	return &CustomCheck{
		name: name,
		fn:   fn,
	}
}

// Name 返回检查名称
func (c *CustomCheck) Name() string {
	return c.name
}

// Execute 执行自定义函数检查
func (c *CustomCheck) Execute(ctx context.Context) (Status, error) {
	return c.fn(ctx)
}
