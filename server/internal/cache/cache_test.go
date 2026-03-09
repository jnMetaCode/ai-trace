package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Test Set and Get
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := testStruct{Name: "test", Value: 42}
	err := cache.Set(ctx, "key1", original, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	var retrieved testStruct
	err = cache.Get(ctx, "key1", &retrieved)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != original.Name || retrieved.Value != original.Value {
		t.Errorf("Retrieved value doesn't match: got %+v, want %+v", retrieved, original)
	}
}

func TestMemoryCacheMiss(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "nonexistent", &result)
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Set with short TTL
	err := cache.Set(ctx, "expiring", "value", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist initially
	exists, _ := cache.Exists(ctx, "expiring")
	if !exists {
		t.Error("Key should exist before expiration")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	var result string
	err = cache.Get(ctx, "expiring", &result)
	if err != ErrCacheExpired && err != ErrCacheMiss {
		t.Errorf("Expected cache miss/expired, got %v", err)
	}
}

func TestMemoryCacheDelete(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "delete-me", "value", time.Hour)

	// Verify it exists
	exists, _ := cache.Exists(ctx, "delete-me")
	if !exists {
		t.Error("Key should exist before deletion")
	}

	// Delete
	err := cache.Delete(ctx, "delete-me")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	exists, _ = cache.Exists(ctx, "delete-me")
	if exists {
		t.Error("Key should not exist after deletion")
	}
}

func TestMemoryCacheTTL(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "ttl-test", "value", time.Second)

	ttl, err := cache.TTL(ctx, "ttl-test")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}

	// TTL should be close to 1 second
	if ttl < 900*time.Millisecond || ttl > time.Second {
		t.Errorf("Expected TTL around 1s, got %v", ttl)
	}

	// Non-existent key
	ttl, _ = cache.TTL(ctx, "nonexistent")
	if ttl != -1 {
		t.Errorf("Expected -1 for nonexistent key, got %v", ttl)
	}
}

func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", time.Hour)
	cache.Set(ctx, "key2", "value2", time.Hour)
	cache.Set(ctx, "key3", "value3", time.Hour)

	// Clear all
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// All should be gone
	for _, key := range []string{"key1", "key2", "key3"} {
		exists, _ := cache.Exists(ctx, key)
		if exists {
			t.Errorf("Key %s should not exist after clear", key)
		}
	}
}

func TestMemoryCacheNoExpiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	// Set with zero TTL (no expiration)
	cache.Set(ctx, "permanent", "value", 0)

	ttl, _ := cache.TTL(ctx, "permanent")
	if ttl != -1 {
		t.Errorf("Expected -1 (no expiration), got %v", ttl)
	}

	// Should still exist
	exists, _ := cache.Exists(ctx, "permanent")
	if !exists {
		t.Error("Permanent key should exist")
	}
}

func TestKeyBuilder(t *testing.T) {
	kb := NewKeyBuilder("ai-trace")

	// Test Build
	key := kb.Build("proof", "abc123")
	if key != "ai-trace:proof:abc123" {
		t.Errorf("Expected ai-trace:proof:abc123, got %s", key)
	}

	// Test specific key generators
	proofKey := kb.ProofKey("hash123")
	if proofKey != "ai-trace:proof:hash123" {
		t.Errorf("Expected ai-trace:proof:hash123, got %s", proofKey)
	}

	eventKey := kb.EventKey("evt_001")
	if eventKey != "ai-trace:event:evt_001" {
		t.Errorf("Expected ai-trace:event:evt_001, got %s", eventKey)
	}

	certKey := kb.CertKey("cert_001")
	if certKey != "ai-trace:cert:cert_001" {
		t.Errorf("Expected ai-trace:cert:cert_001, got %s", certKey)
	}

	traceKey := kb.TraceKey("trc_001")
	if traceKey != "ai-trace:trace:trc_001" {
		t.Errorf("Expected ai-trace:trace:trc_001, got %s", traceKey)
	}
}

func TestTwoLevelCache(t *testing.T) {
	l2 := NewMemoryCache()
	defer l2.Close()

	cache := NewTwoLevelCache(l2, TwoLevelCacheConfig{
		L1TTL:     100 * time.Millisecond,
		L1MaxSize: 100,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set value (goes to both L1 and L2)
	type data struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	original := data{ID: 1, Name: "test"}
	err := cache.Set(ctx, "key1", original, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get should hit L1
	var result data
	err = cache.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result.ID != original.ID || result.Name != original.Name {
		t.Errorf("Got %+v, want %+v", result, original)
	}

	// Wait for L1 to expire
	time.Sleep(150 * time.Millisecond)

	// Should still get from L2 (and refill L1)
	var result2 data
	err = cache.Get(ctx, "key1", &result2)
	if err != nil {
		t.Fatalf("Get from L2 failed: %v", err)
	}

	if result2.ID != original.ID {
		t.Errorf("L2 value mismatch")
	}
}

func TestTwoLevelCacheDelete(t *testing.T) {
	l2 := NewMemoryCache()
	defer l2.Close()

	cache := NewTwoLevelCache(l2, TwoLevelCacheConfig{
		L1TTL:     time.Hour,
		L1MaxSize: 100,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "delete-test", "value", time.Hour)

	// Delete should remove from both levels
	cache.Delete(ctx, "delete-test")

	// Should be gone from L2
	exists, _ := l2.Exists(ctx, "delete-test")
	if exists {
		t.Error("Key should be deleted from L2")
	}
}

func TestCacheInterface(t *testing.T) {
	// Verify both implementations satisfy the interface
	var _ Cache = (*MemoryCache)(nil)
	var _ Cache = (*RedisCache)(nil)
	var _ Cache = (*TwoLevelCache)(nil)
}

func BenchmarkMemoryCacheGet(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	cache.Set(ctx, "bench-key", "bench-value", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		cache.Get(ctx, "bench-key", &result)
	}
}

func BenchmarkMemoryCacheSet(b *testing.B) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "bench-key", "bench-value", time.Hour)
	}
}
