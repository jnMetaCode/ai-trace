// Package cache 提供分布式缓存实现
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrCacheMiss    = errors.New("cache: key not found")
	ErrCacheExpired = errors.New("cache: key expired")
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string, dest interface{}) error

	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Exists 检查 key 是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// TTL 获取剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Clear 清空所有缓存（谨慎使用）
	Clear(ctx context.Context) error

	// Close 关闭缓存连接
	Close() error
}

// ============================================================================
// Redis 缓存实现
// ============================================================================

// RedisCache Redis 缓存
type RedisCache struct {
	client    *redis.Client
	keyPrefix string
}

// RedisCacheConfig Redis 缓存配置
type RedisCacheConfig struct {
	Addr      string
	Password  string
	DB        int
	KeyPrefix string
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(cfg RedisCacheConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

// prefixKey 添加 key 前缀
func (c *RedisCache) prefixKey(key string) string {
	if c.keyPrefix == "" {
		return key
	}
	return c.keyPrefix + ":" + key
}

// Get 获取缓存值
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, c.prefixKey(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("redis get error: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	return nil
}

// Set 设置缓存值
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	return c.client.Set(ctx, c.prefixKey(key), data, ttl).Err()
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.prefixKey(key)).Err()
}

// Exists 检查 key 是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.prefixKey(key)).Result()
	return n > 0, err
}

// TTL 获取剩余过期时间
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.prefixKey(key)).Result()
}

// Clear 清空所有缓存
func (c *RedisCache) Clear(ctx context.Context) error {
	if c.keyPrefix == "" {
		return c.client.FlushDB(ctx).Err()
	}

	// 只删除带前缀的 key
	pattern := c.prefixKey("*")
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Close 关闭连接
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// ============================================================================
// 内存缓存实现（用于测试或单机部署）
// ============================================================================

// MemoryCache 内存缓存
type MemoryCache struct {
	data map[string]*cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		data: make(map[string]*cacheItem),
	}

	// 启动过期清理协程
	go c.cleanupLoop()

	return c
}

// cleanupLoop 定期清理过期项
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期项
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.data, key)
		}
	}
}

// Get 获取缓存值
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return ErrCacheMiss
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.Delete(ctx, key)
		return ErrCacheExpired
	}

	return json.Unmarshal(item.value, dest)
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.data[key] = &cacheItem{
		value:     data,
		expiresAt: expiresAt,
	}
	c.mu.Unlock()

	return nil
}

// Delete 删除缓存
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	return nil
}

// Exists 检查 key 是否存在
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return false, nil
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return false, nil
	}

	return true, nil
}

// TTL 获取剩余过期时间
func (c *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return -1, nil
	}

	if item.expiresAt.IsZero() {
		return -1, nil // 永不过期
	}

	remaining := time.Until(item.expiresAt)
	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}

// Clear 清空所有缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	c.data = make(map[string]*cacheItem)
	c.mu.Unlock()
	return nil
}

// Close 关闭缓存
func (c *MemoryCache) Close() error {
	return nil
}

// ============================================================================
// 多级缓存（L1 内存 + L2 Redis）
// ============================================================================

// TwoLevelCache 两级缓存
type TwoLevelCache struct {
	l1        *MemoryCache
	l2        Cache
	l1TTL     time.Duration
	l1MaxSize int
}

// TwoLevelCacheConfig 两级缓存配置
type TwoLevelCacheConfig struct {
	L1TTL     time.Duration
	L1MaxSize int
}

// NewTwoLevelCache 创建两级缓存
func NewTwoLevelCache(l2 Cache, cfg TwoLevelCacheConfig) *TwoLevelCache {
	return &TwoLevelCache{
		l1:        NewMemoryCache(),
		l2:        l2,
		l1TTL:     cfg.L1TTL,
		l1MaxSize: cfg.L1MaxSize,
	}
}

// Get 获取缓存值（先 L1，再 L2）
func (c *TwoLevelCache) Get(ctx context.Context, key string, dest interface{}) error {
	// 先查 L1
	if err := c.l1.Get(ctx, key, dest); err == nil {
		return nil
	}

	// 再查 L2
	if err := c.l2.Get(ctx, key, dest); err != nil {
		return err
	}

	// 回填 L1
	c.l1.Set(ctx, key, dest, c.l1TTL)

	return nil
}

// Set 设置缓存值（同时写入 L1 和 L2）
func (c *TwoLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 写入 L1（使用较短的 TTL）
	l1TTL := c.l1TTL
	if ttl < l1TTL {
		l1TTL = ttl
	}
	c.l1.Set(ctx, key, value, l1TTL)

	// 写入 L2
	return c.l2.Set(ctx, key, value, ttl)
}

// Delete 删除缓存（同时删除 L1 和 L2）
func (c *TwoLevelCache) Delete(ctx context.Context, key string) error {
	c.l1.Delete(ctx, key)
	return c.l2.Delete(ctx, key)
}

// Exists 检查 key 是否存在
func (c *TwoLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	if exists, _ := c.l1.Exists(ctx, key); exists {
		return true, nil
	}
	return c.l2.Exists(ctx, key)
}

// TTL 获取剩余过期时间（从 L2 获取）
func (c *TwoLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.l2.TTL(ctx, key)
}

// Clear 清空所有缓存
func (c *TwoLevelCache) Clear(ctx context.Context) error {
	c.l1.Clear(ctx)
	return c.l2.Clear(ctx)
}

// Close 关闭缓存
func (c *TwoLevelCache) Close() error {
	c.l1.Close()
	return c.l2.Close()
}

// ============================================================================
// 缓存键生成器
// ============================================================================

// KeyBuilder 缓存键构建器
type KeyBuilder struct {
	namespace string
}

// NewKeyBuilder 创建键构建器
func NewKeyBuilder(namespace string) *KeyBuilder {
	return &KeyBuilder{namespace: namespace}
}

// Build 构建缓存键
func (b *KeyBuilder) Build(parts ...string) string {
	key := b.namespace
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// ProofKey 生成证明缓存键
func (b *KeyBuilder) ProofKey(proofHash string) string {
	return b.Build("proof", proofHash)
}

// EventKey 生成事件缓存键
func (b *KeyBuilder) EventKey(eventID string) string {
	return b.Build("event", eventID)
}

// CertKey 生成证书缓存键
func (b *KeyBuilder) CertKey(certID string) string {
	return b.Build("cert", certID)
}

// TraceKey 生成追踪缓存键
func (b *KeyBuilder) TraceKey(traceID string) string {
	return b.Build("trace", traceID)
}
