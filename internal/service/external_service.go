package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// MockExternalService 实现了外部服务接口的模拟版本
// 在真实环境中，这将调用实际的外部支付和通知服务
type MockExternalService struct {
	paymentServiceURL      string
	notificationServiceURL string
	client                 *http.Client
	failureRate            float64 // 0.0 - 1.0 之间，表示模拟失败的概率
}

// NewMockExternalService 创建一个新的模拟外部服务
func NewMockExternalService(paymentURL, notificationURL string, timeout time.Duration, failureRate float64) *MockExternalService {
	return &MockExternalService{
		paymentServiceURL:      paymentURL,
		notificationServiceURL: notificationURL,
		client: &http.Client{
			Timeout: timeout,
		},
		failureRate: failureRate,
	}
}

// ProcessPayment 模拟处理支付
func (m *MockExternalService) ProcessPayment(ctx context.Context, orderID string, amount float64) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 继续处理
	}

	// 模拟网络延迟
	delay := rand.Intn(500) + 100
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// 模拟随机失败
	if rand.Float64() < m.failureRate {
		return simulateExternalServiceError()
	}

	// 实际应用中，这里将发送HTTP请求到实际的支付服务
	// req, err := http.NewRequestWithContext(ctx, "POST", m.paymentServiceURL, body)
	// resp, err := m.client.Do(req)
	// 处理响应...

	fmt.Printf("Payment processed for order %s: $%.2f\n", orderID, amount)
	return nil
}

// SendNotification 模拟发送通知
func (m *MockExternalService) SendNotification(ctx context.Context, customerID, message string) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 继续处理
	}

	// 模拟网络延迟
	delay := rand.Intn(300) + 50
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// 模拟随机失败
	if rand.Float64() < m.failureRate {
		return simulateExternalServiceError()
	}

	// 实际应用中，这里将发送HTTP请求到实际的通知服务
	// req, err := http.NewRequestWithContext(ctx, "POST", m.notificationServiceURL, body)
	// resp, err := m.client.Do(req)
	// 处理响应...

	fmt.Printf("Notification sent to customer %s: %s\n", customerID, message)
	return nil
}

// 模拟可能的外部服务错误类型
func simulateExternalServiceError() error {
	errors := []error{
		errors.New("connection refused"),
		errors.New("timeout exceeded"),
		errors.New("internal server error"),
		errors.New("service unavailable"),
		errors.New("bad gateway"),
	}

	return errors[rand.Intn(len(errors))]
}
