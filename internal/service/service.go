package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/saixiaoxi/high-availability-system/internal/models"
	"github.com/saixiaoxi/high-availability-system/pkg/retry"
)

// Service 处理业务逻辑
type Service struct {
	products        map[string]models.Product
	orders          map[string]models.Order
	productsMutex   sync.RWMutex
	ordersMutex     sync.RWMutex
	retryConfig     *retry.Config
	externalService ExternalService
}

// ExternalService 表示外部服务接口
type ExternalService interface {
	ProcessPayment(ctx context.Context, orderID string, amount float64) error
	SendNotification(ctx context.Context, customerID, message string) error
}

// NewService 创建新的服务实例
func NewService(retryConfig *retry.Config, externalService ExternalService) *Service {
	return &Service{
		products:        make(map[string]models.Product),
		orders:          make(map[string]models.Order),
		retryConfig:     retryConfig,
		externalService: externalService,
	}
}

// GetAllProducts 获取所有产品
func (s *Service) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	s.productsMutex.RLock()
	defer s.productsMutex.RUnlock()

	products := make([]models.Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}
	return products, nil
}

// GetProductByID 根据ID获取产品
func (s *Service) GetProductByID(ctx context.Context, id string) (models.Product, error) {
	s.productsMutex.RLock()
	defer s.productsMutex.RUnlock()

	product, exists := s.products[id]
	if !exists {
		return models.Product{}, fmt.Errorf("product with ID %s not found", id)
	}
	return product, nil
}

// CreateProduct 创建新产品
func (s *Service) CreateProduct(ctx context.Context, product models.Product) (models.Product, error) {
	s.productsMutex.Lock()
	defer s.productsMutex.Unlock()

	// 简单的ID生成 (实际应用中应使用UUID等)
	product.ID = fmt.Sprintf("prod-%d", time.Now().UnixNano())
	s.products[product.ID] = product
	return product, nil
}

// UpdateProduct 更新产品
func (s *Service) UpdateProduct(ctx context.Context, product models.Product) (models.Product, error) {
	s.productsMutex.Lock()
	defer s.productsMutex.Unlock()

	_, exists := s.products[product.ID]
	if !exists {
		return models.Product{}, fmt.Errorf("product with ID %s not found", product.ID)
	}

	// 更新产品
	s.products[product.ID] = product
	return product, nil
}

// DeleteProduct 删除产品
func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	s.productsMutex.Lock()
	defer s.productsMutex.Unlock()

	_, exists := s.products[id]
	if !exists {
		return fmt.Errorf("product with ID %s not found", id)
	}

	delete(s.products, id)
	return nil
}

// GetAllOrders 获取所有订单
func (s *Service) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	s.ordersMutex.RLock()
	defer s.ordersMutex.RUnlock()

	orders := make([]models.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

// GetOrderByID 根据ID获取订单
func (s *Service) GetOrderByID(ctx context.Context, id string) (models.Order, error) {
	s.ordersMutex.RLock()
	defer s.ordersMutex.RUnlock()

	order, exists := s.orders[id]
	if !exists {
		return models.Order{}, fmt.Errorf("order with ID %s not found", id)
	}
	return order, nil
}

// CreateOrder 创建新订单，带有重试机制处理外部服务调用
func (s *Service) CreateOrder(ctx context.Context, order models.Order) (models.Order, error) {
	// 计算总价
	var totalPrice float64
	for _, item := range order.Items {
		totalPrice += item.Price * float64(item.Quantity)
	}
	order.TotalPrice = totalPrice

	// 生成订单ID (实际应用中应使用UUID等)
	order.ID = fmt.Sprintf("order-%d", time.Now().UnixNano())
	order.Status = models.OrderStatusPending

	// 处理支付，带有重试机制
	err := retry.DoWithContext(ctx, func(ctx context.Context) error {
		return s.externalService.ProcessPayment(ctx, order.ID, order.TotalPrice)
	}, s.retryConfig)

	if err != nil {
		order.Status = models.OrderStatusCancelled
		// 存储失败的订单以便后续重试
		s.ordersMutex.Lock()
		s.orders[order.ID] = order
		s.ordersMutex.Unlock()
		return order, errors.New("payment processing failed after retries: " + err.Error())
	}

	// 支付成功
	order.Status = models.OrderStatusPaid

	// 存储订单
	s.ordersMutex.Lock()
	s.orders[order.ID] = order
	s.ordersMutex.Unlock()

	// 发送通知，使用重试机制
	go func() {
		notificationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		notificationMessage := fmt.Sprintf("Your order %s has been successfully processed.", order.ID)

		// 使用重试机制发送通知
		_ = retry.DoWithContext(notificationCtx, func(ctx context.Context) error {
			return s.externalService.SendNotification(ctx, order.CustomerID, notificationMessage)
		}, s.retryConfig)
	}()

	return order, nil
}
