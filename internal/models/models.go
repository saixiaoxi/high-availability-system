package models

import (
	"time"
)

// Product 表示产品模型
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Inventory   int       `json:"inventory"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// Order 表示订单模型
type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
	Status     string      `json:"status"`
	TotalPrice float64     `json:"total_price"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at,omitempty"`
}

// OrderItem 表示订单项模型
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// Customer 表示客户模型
type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// OrderStatus 定义订单状态常量
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusShipping  = "shipping"
	OrderStatusDelivered = "delivered"
	OrderStatusCancelled = "cancelled"
)

// ApiResponse 通用API响应结构
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}, message string) ApiResponse {
	return ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(errMsg string) ApiResponse {
	return ApiResponse{
		Success: false,
		Error:   errMsg,
	}
}
