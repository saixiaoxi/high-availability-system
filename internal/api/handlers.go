package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/high-availability-system/internal/models"
	"github.com/yourusername/high-availability-system/internal/service"
)

// Handler 封装所有API处理函数
type Handler struct {
	service *service.Service
}

// NewHandler 创建新的API处理器
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes 注册所有API路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// API版本组
	v1 := router.Group("/api/v1")
	{
		// 产品API
		products := v1.Group("/products")
		{
			products.GET("", h.GetProducts)
			products.GET("/:id", h.GetProduct)
			products.POST("", h.CreateProduct)
			products.PUT("/:id", h.UpdateProduct)
			products.DELETE("/:id", h.DeleteProduct)
		}

		// 订单API
		orders := v1.Group("/orders")
		{
			orders.GET("", h.GetOrders)
			orders.GET("/:id", h.GetOrder)
			orders.POST("", h.CreateOrder)
		}
	}
}

// GetProducts 获取所有产品
func (h *Handler) GetProducts(c *gin.Context) {
	// 从服务层获取产品
	products, err := h.service.GetAllProducts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve products",
		})
		return
	}

	c.JSON(http.StatusOK, products)
}

// GetProduct 获取单个产品
func (h *Handler) GetProduct(c *gin.Context) {
	id := c.Param("id")

	// 从服务层获取产品
	product, err := h.service.GetProductByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Product with ID %s not found", id),
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

// CreateProduct 创建新产品
func (h *Handler) CreateProduct(c *gin.Context) {
	var product models.Product

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product data",
		})
		return
	}

	// 设置创建时间
	product.CreatedAt = time.Now()

	// 调用服务层创建产品
	createdProduct, err := h.service.CreateProduct(c.Request.Context(), product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create product",
		})
		return
	}

	c.JSON(http.StatusCreated, createdProduct)
}

// UpdateProduct 更新产品
func (h *Handler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product data",
		})
		return
	}

	// 设置ID和更新时间
	product.ID = id
	product.UpdatedAt = time.Now()

	// 调用服务层更新产品
	updatedProduct, err := h.service.UpdateProduct(c.Request.Context(), product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update product",
		})
		return
	}

	c.JSON(http.StatusOK, updatedProduct)
}

// DeleteProduct 删除产品
func (h *Handler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	// 调用服务层删除产品
	err := h.service.DeleteProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete product",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Product with ID %s deleted successfully", id),
	})
}

// GetOrders 获取所有订单
func (h *Handler) GetOrders(c *gin.Context) {
	// 从服务层获取订单
	orders, err := h.service.GetAllOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve orders",
		})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// GetOrder 获取单个订单
func (h *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	// 从服务层获取订单
	order, err := h.service.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Order with ID %s not found", id),
		})
		return
	}

	c.JSON(http.StatusOK, order)
}

// CreateOrder 创建新订单
func (h *Handler) CreateOrder(c *gin.Context) {
	var order models.Order

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order data",
		})
		return
	}

	// 设置创建时间
	order.CreatedAt = time.Now()

	// 调用服务层创建订单
	createdOrder, err := h.service.CreateOrder(c.Request.Context(), order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create order",
		})
		return
	}

	c.JSON(http.StatusCreated, createdOrder)
}
