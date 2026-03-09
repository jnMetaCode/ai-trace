package gateway

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		OpenTimeout:         100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	}

	cb := NewCircuitBreaker(config)

	// Initial state should be closed
	if cb.State() != StateClosed {
		t.Errorf("Expected initial state Closed, got %s", cb.State())
	}

	// Record successes - should stay closed
	cb.RecordSuccess()
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed after successes, got %s", cb.State())
	}

	// Record failures up to threshold
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed with failures < threshold, got %s", cb.State())
	}

	// One more failure should open the circuit
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Errorf("Expected state Open after threshold failures, got %s", cb.State())
	}

	// Should not allow requests when open
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)
	if cb.State() != StateHalfOpen {
		t.Errorf("Expected state HalfOpen after timeout, got %s", cb.State())
	}

	// Should allow limited requests in half-open
	if err := cb.Allow(); err != nil {
		t.Errorf("Expected Allow() to succeed in half-open, got %v", err)
	}

	// Success in half-open
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should transition back to closed
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed after successes in half-open, got %s", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		OpenTimeout:         50 * time.Millisecond,
		HalfOpenMaxRequests: 3,
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	// Failure in half-open should go back to open
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("Expected state Open after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// Create some state
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Errorf("Expected state Open, got %s", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed after reset, got %s", cb.State())
	}

	stats := cb.Stats()
	if stats.Failures != 0 || stats.Successes != 0 {
		t.Errorf("Expected zeroed stats after reset")
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager(DefaultCircuitBreakerConfig())

	// Get or create breakers
	cb1 := manager.GetBreaker("service-a")
	cb2 := manager.GetBreaker("service-b")
	cb3 := manager.GetBreaker("service-a") // Should return same instance

	if cb1 == cb2 {
		t.Error("Different services should have different breakers")
	}

	if cb1 != cb3 {
		t.Error("Same service should return same breaker")
	}

	// Record failures on one
	cb1.RecordFailure()
	cb1.RecordFailure()
	cb1.RecordFailure()
	cb1.RecordFailure()
	cb1.RecordFailure()

	// Check stats
	stats := manager.AllStats()
	if stats["service-a"].State != "open" {
		t.Errorf("Expected service-a to be open, got %s", stats["service-a"].State)
	}
	if stats["service-b"].State != "closed" {
		t.Errorf("Expected service-b to be closed, got %s", stats["service-b"].State)
	}
}

func TestRetry(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	// Test successful operation
	attempts := 0
	result, err := Retry(context.Background(), config, func() (int, error) {
		attempts++
		return 42, nil
	})

	if err != nil || result != 42 || attempts != 1 {
		t.Errorf("Expected single successful attempt, got attempts=%d, result=%d, err=%v", attempts, result, err)
	}

	// Test operation that succeeds on third attempt
	attempts = 0
	result, err = Retry(context.Background(), config, func() (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("temporary error")
		}
		return 100, nil
	})

	if err != nil || result != 100 || attempts != 3 {
		t.Errorf("Expected success on third attempt, got attempts=%d, result=%d, err=%v", attempts, result, err)
	}

	// Test operation that always fails
	attempts = 0
	_, err = Retry(context.Background(), config, func() (int, error) {
		attempts++
		return 0, errors.New("permanent error")
	})

	if err != ErrTooManyRetries || attempts != 3 {
		t.Errorf("Expected ErrTooManyRetries after 3 attempts, got attempts=%d, err=%v", attempts, err)
	}
}

func TestRetryWithContext(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       10,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	_, err := Retry(ctx, config, func() (int, error) {
		attempts++
		return 0, errors.New("always fail")
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Should have been cancelled before all attempts
	if attempts >= 10 {
		t.Errorf("Expected fewer than 10 attempts due to context cancellation, got %d", attempts)
	}
}

func TestResilientExecutor(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	})

	retryConfig := RetryConfig{
		MaxAttempts:       2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	executor := NewResilientExecutor(cb, retryConfig)

	// Successful operations
	for i := 0; i < 5; i++ {
		err := executor.Execute(context.Background(), func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected success, got %v", err)
		}
	}

	// Failing operations should eventually open the circuit
	for i := 0; i < 10; i++ {
		executor.Execute(context.Background(), func() error {
			return errors.New("fail")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected circuit to be open after many failures, got %s", cb.State())
	}

	// New operations should fail with circuit open
	err := executor.Execute(context.Background(), func() error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}
