// Package queue 提供消息队列抽象和实现
// 用于异步处理事件存储、区块链锚定等耗时操作
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrQueueFull     = errors.New("queue: queue is full")
	ErrQueueClosed   = errors.New("queue: queue is closed")
	ErrNoSubscribers = errors.New("queue: no subscribers")
)

// Message 消息结构
type Message struct {
	ID        string          `json:"id"`
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	Retries   int             `json:"retries"`
	MaxRetries int            `json:"max_retries"`
}

// Handler 消息处理函数
type Handler func(ctx context.Context, msg *Message) error

// Queue 消息队列接口
type Queue interface {
	// Publish 发布消息
	Publish(ctx context.Context, topic string, payload interface{}) error

	// Subscribe 订阅主题
	Subscribe(topic string, handler Handler) error

	// Unsubscribe 取消订阅
	Unsubscribe(topic string) error

	// Start 启动队列处理
	Start(ctx context.Context) error

	// Stop 停止队列
	Stop() error

	// Stats 获取统计信息
	Stats() QueueStats
}

// QueueStats 队列统计
type QueueStats struct {
	Published   int64            `json:"published"`
	Processed   int64            `json:"processed"`
	Failed      int64            `json:"failed"`
	Retried     int64            `json:"retried"`
	QueueLength int              `json:"queue_length"`
	TopicStats  map[string]int64 `json:"topic_stats"`
}

// ============================================================================
// 内存队列实现（用于测试和单机部署）
// ============================================================================

// MemoryQueueConfig 内存队列配置
type MemoryQueueConfig struct {
	BufferSize    int
	WorkerCount   int
	MaxRetries    int
	RetryDelay    time.Duration
}

// DefaultMemoryQueueConfig 默认配置
func DefaultMemoryQueueConfig() MemoryQueueConfig {
	return MemoryQueueConfig{
		BufferSize:  10000,
		WorkerCount: 10,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}
}

// MemoryQueue 内存队列
type MemoryQueue struct {
	config     MemoryQueueConfig
	messages   chan *Message
	handlers   map[string]Handler
	handlersMu sync.RWMutex

	// 统计
	stats      QueueStats
	statsMu    sync.RWMutex
	topicStats map[string]int64

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// NewMemoryQueue 创建内存队列
func NewMemoryQueue(config MemoryQueueConfig) *MemoryQueue {
	return &MemoryQueue{
		config:     config,
		messages:   make(chan *Message, config.BufferSize),
		handlers:   make(map[string]Handler),
		topicStats: make(map[string]int64),
	}
}

// Publish 发布消息
func (q *MemoryQueue) Publish(ctx context.Context, topic string, payload interface{}) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.mu.Unlock()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &Message{
		ID:         generateMessageID(),
		Topic:      topic,
		Payload:    payloadBytes,
		Timestamp:  time.Now(),
		MaxRetries: q.config.MaxRetries,
	}

	select {
	case q.messages <- msg:
		q.incrementPublished()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Subscribe 订阅主题
func (q *MemoryQueue) Subscribe(topic string, handler Handler) error {
	q.handlersMu.Lock()
	defer q.handlersMu.Unlock()
	q.handlers[topic] = handler
	return nil
}

// Unsubscribe 取消订阅
func (q *MemoryQueue) Unsubscribe(topic string) error {
	q.handlersMu.Lock()
	defer q.handlersMu.Unlock()
	delete(q.handlers, topic)
	return nil
}

// Start 启动队列处理
func (q *MemoryQueue) Start(ctx context.Context) error {
	q.ctx, q.cancel = context.WithCancel(ctx)

	// 启动 worker
	for i := 0; i < q.config.WorkerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	return nil
}

// Stop 停止队列
func (q *MemoryQueue) Stop() error {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()

	if q.cancel != nil {
		q.cancel()
	}

	// 等待所有 worker 完成
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("timeout waiting for workers to stop")
	}
}

// Stats 获取统计
func (q *MemoryQueue) Stats() QueueStats {
	q.statsMu.RLock()
	defer q.statsMu.RUnlock()

	stats := q.stats
	stats.QueueLength = len(q.messages)
	stats.TopicStats = make(map[string]int64)
	for k, v := range q.topicStats {
		stats.TopicStats[k] = v
	}
	return stats
}

// worker 工作协程
func (q *MemoryQueue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case msg := <-q.messages:
			q.processMessage(msg)
		}
	}
}

// processMessage 处理消息
func (q *MemoryQueue) processMessage(msg *Message) {
	q.handlersMu.RLock()
	handler, exists := q.handlers[msg.Topic]
	q.handlersMu.RUnlock()

	if !exists {
		// 没有处理器，丢弃消息
		return
	}

	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
	defer cancel()

	err := handler(ctx, msg)
	if err != nil {
		q.handleFailure(msg, err)
	} else {
		q.incrementProcessed()
		q.incrementTopicStats(msg.Topic)
	}
}

// handleFailure 处理失败消息
func (q *MemoryQueue) handleFailure(msg *Message, err error) {
	msg.Retries++

	if msg.Retries <= msg.MaxRetries {
		// 重试
		q.incrementRetried()
		time.Sleep(q.config.RetryDelay * time.Duration(msg.Retries))

		select {
		case q.messages <- msg:
		default:
			q.incrementFailed()
		}
	} else {
		q.incrementFailed()
	}
}

// 统计辅助函数
func (q *MemoryQueue) incrementPublished() {
	q.statsMu.Lock()
	q.stats.Published++
	q.statsMu.Unlock()
}

func (q *MemoryQueue) incrementProcessed() {
	q.statsMu.Lock()
	q.stats.Processed++
	q.statsMu.Unlock()
}

func (q *MemoryQueue) incrementFailed() {
	q.statsMu.Lock()
	q.stats.Failed++
	q.statsMu.Unlock()
}

func (q *MemoryQueue) incrementRetried() {
	q.statsMu.Lock()
	q.stats.Retried++
	q.statsMu.Unlock()
}

func (q *MemoryQueue) incrementTopicStats(topic string) {
	q.statsMu.Lock()
	q.topicStats[topic]++
	q.statsMu.Unlock()
}

// ============================================================================
// 预定义主题
// ============================================================================

const (
	// TopicEventStore 事件存储
	TopicEventStore = "ai-trace.events.store"

	// TopicCertCommit 证书提交
	TopicCertCommit = "ai-trace.certs.commit"

	// TopicBlockchainAnchor 区块链锚定
	TopicBlockchainAnchor = "ai-trace.anchor.blockchain"

	// TopicFingerprintCompute 指纹计算
	TopicFingerprintCompute = "ai-trace.fingerprint.compute"

	// TopicZKPGenerate ZKP 生成
	TopicZKPGenerate = "ai-trace.zkp.generate"

	// TopicAuditLog 审计日志
	TopicAuditLog = "ai-trace.audit.log"
)

// ============================================================================
// 消息类型定义
// ============================================================================

// EventStoreMessage 事件存储消息
type EventStoreMessage struct {
	TraceID  string          `json:"trace_id"`
	TenantID string          `json:"tenant_id"`
	Events   json.RawMessage `json:"events"`
}

// CertCommitMessage 证书提交消息
type CertCommitMessage struct {
	TraceID       string `json:"trace_id"`
	TenantID      string `json:"tenant_id"`
	EvidenceLevel string `json:"evidence_level"`
	RequestID     string `json:"request_id"`
}

// BlockchainAnchorMessage 区块链锚定消息
type BlockchainAnchorMessage struct {
	CertID    string `json:"cert_id"`
	RootHash  string `json:"root_hash"`
	ChainType string `json:"chain_type"`
	Priority  int    `json:"priority"`
}

// FingerprintComputeMessage 指纹计算消息
type FingerprintComputeMessage struct {
	TraceID       string `json:"trace_id"`
	TenantID      string `json:"tenant_id"`
	ModelID       string `json:"model_id"`
	OutputContent string `json:"output_content"`
}

// ============================================================================
// 辅助函数
// ============================================================================

func generateMessageID() string {
	return time.Now().Format("20060102150405.000000000")
}

// ParseMessage 解析消息 payload
func ParseMessage[T any](msg *Message) (*T, error) {
	var result T
	if err := json.Unmarshal(msg.Payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
