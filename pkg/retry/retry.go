package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

var (
	// ErrMaxRetriesReached 表示达到最大重试次数
	ErrMaxRetriesReached = errors.New("max retries reached")
)

// Config 保存重试配置
type Config struct {
	MaxAttempts         int           // 最大尝试次数
	InitialInterval     time.Duration // 初始重试间隔
	MaxInterval         time.Duration // 最大重试间隔
	Multiplier          float64       // 退避乘数
	RandomizationFactor float64       // 随机因子 (0-1)
}

// DefaultConfig 返回默认重试配置
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:         3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5,
	}
}

// RetryFunc 是重试的函数类型
type RetryFunc func() error

// RetryFuncContext 是带上下文的重试函数类型
type RetryFuncContext func(context.Context) error

// Do 执行带重试的操作
func Do(fn RetryFunc, config *Config) error {
	var err error
	var nextInterval time.Duration = config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt == config.MaxAttempts {
			break
		}

		// 计算下一次重试的间隔
		nextInterval = calculateNextInterval(nextInterval, attempt, config)
		time.Sleep(nextInterval)
	}

	if err != nil {
		return errors.New(err.Error() + ": " + ErrMaxRetriesReached.Error())
	}

	return err
}

// DoWithContext 执行带上下文和重试的操作
func DoWithContext(ctx context.Context, fn RetryFuncContext, config *Config) error {
	var err error
	var nextInterval time.Duration = config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 继续重试
		}

		err = fn(ctx)
		if err == nil {
			return nil
		}

		if attempt == config.MaxAttempts {
			break
		}

		// 计算下一次重试的间隔
		nextInterval = calculateNextInterval(nextInterval, attempt, config)

		// 带上下文的等待
		timer := time.NewTimer(nextInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// 继续下一次重试
		}
	}

	if err != nil {
		return errors.New(err.Error() + ": " + ErrMaxRetriesReached.Error())
	}

	return err
}

// 计算指数退避的下一个间隔时间
func calculateNextInterval(currentInterval time.Duration, attempt int, config *Config) time.Duration {
	// 应用乘数来实现指数退避
	interval := float64(currentInterval) * config.Multiplier

	// 应用随机因子 (指数退避加抖动)
	delta := config.RandomizationFactor * interval
	minInterval := interval - delta
	maxInterval := interval + delta

	// 获取随机间隔
	interval = minInterval + (rand.Float64() * (maxInterval - minInterval + 1))

	// 确保不超过最大间隔
	interval = math.Min(interval, float64(config.MaxInterval))

	return time.Duration(interval)
}
