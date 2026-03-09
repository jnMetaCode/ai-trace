package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryQueue(t *testing.T) {
	config := MemoryQueueConfig{
		BufferSize:  100,
		WorkerCount: 2,
		MaxRetries:  2,
		RetryDelay:  10 * time.Millisecond,
	}

	q := NewMemoryQueue(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register handler
	var processed int64
	q.Subscribe("test-topic", func(ctx context.Context, msg *Message) error {
		atomic.AddInt64(&processed, 1)
		return nil
	})

	// Start queue
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}

	// Publish messages
	for i := 0; i < 10; i++ {
		err := q.Publish(ctx, "test-topic", map[string]int{"value": i})
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check stats
	stats := q.Stats()
	if stats.Published != 10 {
		t.Errorf("Expected 10 published, got %d", stats.Published)
	}

	if atomic.LoadInt64(&processed) != 10 {
		t.Errorf("Expected 10 processed, got %d", processed)
	}

	// Stop queue
	if err := q.Stop(); err != nil {
		t.Fatalf("Failed to stop queue: %v", err)
	}
}

func TestMemoryQueueRetry(t *testing.T) {
	config := MemoryQueueConfig{
		BufferSize:  100,
		WorkerCount: 1,
		MaxRetries:  2,
		RetryDelay:  10 * time.Millisecond,
	}

	q := NewMemoryQueue(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handler that fails first time
	var attempts int64
	q.Subscribe("retry-topic", func(ctx context.Context, msg *Message) error {
		count := atomic.AddInt64(&attempts, 1)
		if count < 2 {
			return context.DeadlineExceeded // Simulate failure
		}
		return nil
	})

	q.Start(ctx)
	defer q.Stop()

	q.Publish(ctx, "retry-topic", "test")

	// Wait for retries
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt64(&attempts) < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", attempts)
	}
}

func TestMemoryQueueUnsubscribe(t *testing.T) {
	config := DefaultMemoryQueueConfig()
	q := NewMemoryQueue(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processed int64
	q.Subscribe("unsub-topic", func(ctx context.Context, msg *Message) error {
		atomic.AddInt64(&processed, 1)
		return nil
	})

	q.Start(ctx)
	defer q.Stop()

	// Publish before unsubscribe
	q.Publish(ctx, "unsub-topic", "test1")
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe
	q.Unsubscribe("unsub-topic")

	// Publish after unsubscribe
	q.Publish(ctx, "unsub-topic", "test2")
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&processed) != 1 {
		t.Errorf("Expected 1 processed (before unsub), got %d", processed)
	}
}

func TestParseMessage(t *testing.T) {
	type testPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	msg := &Message{
		Payload: []byte(`{"name":"test","value":42}`),
	}

	result, err := ParseMessage[testPayload](msg)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("Unexpected result: %+v", result)
	}
}

func TestEventStoreMessage(t *testing.T) {
	msg := EventStoreMessage{
		TraceID:  "trace-123",
		TenantID: "tenant-456",
	}

	if msg.TraceID != "trace-123" {
		t.Error("TraceID mismatch")
	}
}

func TestCertCommitMessage(t *testing.T) {
	msg := CertCommitMessage{
		TraceID:       "trace-123",
		TenantID:      "tenant-456",
		EvidenceLevel: "L2",
	}

	if msg.EvidenceLevel != "L2" {
		t.Error("EvidenceLevel mismatch")
	}
}

func TestTopicConstants(t *testing.T) {
	topics := []string{
		TopicEventStore,
		TopicCertCommit,
		TopicBlockchainAnchor,
		TopicFingerprintCompute,
		TopicZKPGenerate,
		TopicAuditLog,
	}

	for _, topic := range topics {
		if topic == "" {
			t.Error("Topic should not be empty")
		}
	}
}

func TestQueueStats(t *testing.T) {
	config := DefaultMemoryQueueConfig()
	q := NewMemoryQueue(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.Subscribe("stats-topic", func(ctx context.Context, msg *Message) error {
		return nil
	})

	q.Start(ctx)
	defer q.Stop()

	// Initial stats
	stats := q.Stats()
	if stats.Published != 0 {
		t.Error("Initial published should be 0")
	}

	// Publish and check
	q.Publish(ctx, "stats-topic", "test")
	time.Sleep(50 * time.Millisecond)

	stats = q.Stats()
	if stats.Published != 1 {
		t.Errorf("Expected 1 published, got %d", stats.Published)
	}
}

func TestQueueFull(t *testing.T) {
	config := MemoryQueueConfig{
		BufferSize:  1,
		WorkerCount: 0, // No workers to process
		MaxRetries:  0,
	}

	q := NewMemoryQueue(config)

	ctx := context.Background()

	// First message should succeed
	err := q.Publish(ctx, "full-topic", "test1")
	if err != nil {
		t.Fatalf("First publish should succeed: %v", err)
	}

	// Second message should fail (queue full, no workers)
	err = q.Publish(ctx, "full-topic", "test2")
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

func TestQueueClosed(t *testing.T) {
	config := DefaultMemoryQueueConfig()
	q := NewMemoryQueue(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q.Start(ctx)
	q.Stop()

	err := q.Publish(ctx, "closed-topic", "test")
	if err != ErrQueueClosed {
		t.Errorf("Expected ErrQueueClosed, got %v", err)
	}
}
