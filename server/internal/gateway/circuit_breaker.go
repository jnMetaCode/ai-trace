// Package gateway 提供熔断器实现
package gateway

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen    = errors.New("circuit breaker is open")
	ErrTooManyRetries = errors.New("exceeded maximum retry attempts")
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed   CircuitState = iota // 正常状态，允许请求
	StateOpen                          // 熔断状态，拒绝请求
	StateHalfOpen                      // 半开状态，允许探测请求
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// FailureThreshold 触发熔断的连续失败次数
	FailureThreshold int

	// SuccessThreshold 从半开恢复到关闭的连续成功次数
	SuccessThreshold int

	// OpenTimeout 熔断后等待多久进入半开状态
	OpenTimeout time.Duration

	// HalfOpenMaxRequests 半开状态允许的最大请求数
	HalfOpenMaxRequests int
}

// DefaultCircuitBreakerConfig 返回默认配置
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		OpenTimeout:         30 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config CircuitBreakerConfig

	state             CircuitState
	failures          int
	successes         int
	lastFailureTime   time.Time
	halfOpenRequests  int

	mu sync.RWMutex

	// 监控回调
	onStateChange func(from, to CircuitState)
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// SetOnStateChange 设置状态变更回调
func (cb *CircuitBreaker) SetOnStateChange(fn func(from, to CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState 内部获取状态（需已持有锁）
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == StateOpen {
		// 检查是否应该进入半开状态
		if time.Since(cb.lastFailureTime) >= cb.config.OpenTimeout {
			return StateHalfOpen
		}
	}
	return cb.state
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case StateClosed:
		return nil

	case StateOpen:
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			return ErrCircuitOpen
		}
		cb.halfOpenRequests++
		return nil
	}

	return nil
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	state := cb.currentState()
	if state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transition(StateClosed)
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successes = 0

	state := cb.currentState()
	switch state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.transition(StateOpen)
		}
	case StateHalfOpen:
		// 半开状态下的失败直接回到打开状态
		cb.transition(StateOpen)
	}
}

// transition 状态转换
func (cb *CircuitBreaker) transition(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	// 重置计数器
	switch newState {
	case StateClosed:
		cb.failures = 0
		cb.successes = 0
	case StateOpen:
		cb.halfOpenRequests = 0
	case StateHalfOpen:
		cb.successes = 0
		cb.halfOpenRequests = 0
	}

	// 触发回调
	if cb.onStateChange != nil {
		go cb.onStateChange(oldState, newState)
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}

// Stats 获取统计信息
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:            cb.currentState().String(),
		Failures:         cb.failures,
		Successes:        cb.successes,
		LastFailureTime:  cb.lastFailureTime,
		HalfOpenRequests: cb.halfOpenRequests,
	}
}

// CircuitBreakerStats 统计信息
type CircuitBreakerStats struct {
	State            string    `json:"state"`
	Failures         int       `json:"failures"`
	Successes        int       `json:"successes"`
	LastFailureTime  time.Time `json:"last_failure_time"`
	HalfOpenRequests int       `json:"half_open_requests"`
}

// ============================================================================
// 熔断器管理器
// ============================================================================

// CircuitBreakerManager 管理多个熔断器
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewCircuitBreakerManager 创建管理器
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// GetBreaker 获取或创建指定名称的熔断器
func (m *CircuitBreakerManager) GetBreaker(name string) *CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.breakers[name]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if cb, exists = m.breakers[name]; exists {
		return cb
	}

	cb = NewCircuitBreaker(m.config)
	m.breakers[name] = cb
	return cb
}

// AllStats 获取所有熔断器统计
func (m *CircuitBreakerManager) AllStats() map[string]CircuitBreakerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats, len(m.breakers))
	for name, cb := range m.breakers {
		stats[name] = cb.Stats()
	}
	return stats
}

// ============================================================================
// 重试机制
// ============================================================================

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts     int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiplier float64
	RetryableErrors []error
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// Retry 执行带重试的操作
func Retry[T any](ctx context.Context, config RetryConfig, operation func() (T, error)) (T, error) {
	var result T
	var lastErr error

	backoff := config.InitialBackoff

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, lastErr = operation()
		if lastErr == nil {
			return result, nil
		}

		// 检查是否可重试
		if !isRetryable(lastErr, config.RetryableErrors) {
			return result, lastErr
		}

		// 最后一次尝试不需要等待
		if attempt == config.MaxAttempts {
			break
		}

		// 等待退避时间
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(backoff):
		}

		// 增加退避时间
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	return result, ErrTooManyRetries
}

// isRetryable 检查错误是否可重试
func isRetryable(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		// 默认所有错误都可重试
		return true
	}

	for _, retryable := range retryableErrors {
		if errors.Is(err, retryable) {
			return true
		}
	}
	return false
}

// ============================================================================
// 带熔断和重试的执行器
// ============================================================================

// ResilientExecutor 弹性执行器（熔断 + 重试）
type ResilientExecutor struct {
	breaker     *CircuitBreaker
	retryConfig RetryConfig
}

// NewResilientExecutor 创建弹性执行器
func NewResilientExecutor(breaker *CircuitBreaker, retryConfig RetryConfig) *ResilientExecutor {
	return &ResilientExecutor{
		breaker:     breaker,
		retryConfig: retryConfig,
	}
}

// Execute 执行带熔断和重试的操作
func (e *ResilientExecutor) Execute(ctx context.Context, operation func() error) error {
	// 检查熔断器
	if err := e.breaker.Allow(); err != nil {
		return err
	}

	// 带重试执行
	_, err := Retry(ctx, e.retryConfig, func() (struct{}, error) {
		return struct{}{}, operation()
	})

	// 记录结果
	if err != nil {
		e.breaker.RecordFailure()
	} else {
		e.breaker.RecordSuccess()
	}

	return err
}

// ExecuteWithResult 执行带返回值的操作
func ExecuteWithResult[T any](ctx context.Context, executor *ResilientExecutor, operation func() (T, error)) (T, error) {
	var zero T

	// 检查熔断器
	if err := executor.breaker.Allow(); err != nil {
		return zero, err
	}

	// 带重试执行
	result, err := Retry(ctx, executor.retryConfig, operation)

	// 记录结果
	if err != nil {
		executor.breaker.RecordFailure()
	} else {
		executor.breaker.RecordSuccess()
	}

	return result, err
}
