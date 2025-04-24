package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/yourusername/high-availability-system/internal/api"
	"github.com/yourusername/high-availability-system/internal/middleware"
	"github.com/yourusername/high-availability-system/internal/monitors"
	"github.com/yourusername/high-availability-system/internal/service"
	"github.com/yourusername/high-availability-system/pkg/healthcheck"
	"github.com/yourusername/high-availability-system/pkg/retry"
)

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 设置Gin模式
	gin.SetMode(viper.GetString("server.mode"))

	// 创建重试配置
	retryConfig := &retry.Config{
		MaxAttempts:         viper.GetInt("retry.max_attempts"),
		InitialInterval:     viper.GetDuration("retry.initial_interval"),
		MaxInterval:         viper.GetDuration("retry.max_interval"),
		Multiplier:          viper.GetFloat64("retry.multiplier"),
		RandomizationFactor: viper.GetFloat64("retry.randomization_factor"),
	}

	// 创建外部服务客户端
	externalService := service.NewMockExternalService(
		"http://payment-service:8080",
		"http://notification-service:8080",
		5*time.Second,
		0.3, // 30%的模拟失败率
	)

	// 创建业务服务
	svc := service.NewService(retryConfig, externalService)

	// 创建API处理器
	handler := api.NewHandler(svc)

	// 创建Prometheus监控
	prometheusMonitor := monitors.NewPrometheusMonitor(viper.GetString("monitoring.prometheus.endpoint"))
	err := prometheusMonitor.StartServer(":9090")
	if err != nil {
		log.Printf("Warning: Failed to start Prometheus metrics server: %v", err)
	}

	// 创建监控容错策略
	loggingFallback := monitors.NewLocalLoggingFallback(
		viper.GetBool("monitoring.fallback.enabled"),
		"logs/metrics.log",
		1000,
	)

	// 创建带容错机制的监控
	monitor := monitors.NewMonitorWithFallback(
		prometheusMonitor,
		loggingFallback,
		viper.GetDuration("monitoring.fallback.periodic_check"),
	)

	// 创建定期刷新器
	flusher := monitors.NewPeriodicFlusher(loggingFallback, 30*time.Second)
	flusher.Start()

	// 创建健康检查
	healthChecker := healthcheck.NewChecker()

	// 添加HTTP健康检查
	healthChecker.AddCheck(healthcheck.NewHTTPCheck(
		"payment-service",
		"http://payment-service:8080/health",
		5*time.Second,
	))

	// 创建Gin路由
	router := gin.New()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(middleware.MonitoringMiddleware(monitor))
	router.Use(middleware.ErrorMonitoring(monitor))
	router.Use(middleware.RetryMiddleware(nil)) // 使用默认配置

	// 注册API路由
	handler.RegisterRoutes(router)

	// 注册健康检查和指标端点
	router.GET("/health", middleware.HealthCheckHandler(monitor))

	// 启动HTTP服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("server.port")),
		Handler: router,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// 关闭监控
	if err := prometheusMonitor.StopServer(ctx); err != nil {
		log.Printf("Error stopping Prometheus server: %v", err)
	}

	// 停止定期刷新
	flusher.Stop()

	// 最后一次刷新指标
	monitors.FlushOnShutdown(loggingFallback)

	log.Println("Server exited properly")
}

// 加载配置文件
func loadConfig() error {
	// 设置配置文件的位置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	// 设置默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.timeout", "10s")

	viper.SetDefault("retry.max_attempts", 3)
	viper.SetDefault("retry.initial_interval", "100ms")
	viper.SetDefault("retry.max_interval", "1s")
	viper.SetDefault("retry.multiplier", 2.0)
	viper.SetDefault("retry.randomization_factor", 0.5)

	viper.SetDefault("monitoring.prometheus.enabled", true)
	viper.SetDefault("monitoring.prometheus.endpoint", "/metrics")
	viper.SetDefault("monitoring.fallback.enabled", true)
	viper.SetDefault("monitoring.fallback.local_logging", true)
	viper.SetDefault("monitoring.fallback.periodic_check", "30s")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果找不到配置文件，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		log.Println("Config file not found, using defaults")
	}

	return nil
}
