// Package queue 提供异步工作器
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WorkerPool 工作池
type WorkerPool struct {
	queue    Queue
	logger   *zap.SugaredLogger
	handlers map[string]Handler
	mu       sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorkerPool 创建工作池
func NewWorkerPool(queue Queue, logger *zap.SugaredLogger) *WorkerPool {
	return &WorkerPool{
		queue:    queue,
		logger:   logger,
		handlers: make(map[string]Handler),
	}
}

// RegisterHandler 注册处理器
func (wp *WorkerPool) RegisterHandler(topic string, handler Handler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.handlers[topic] = handler
	wp.queue.Subscribe(topic, wp.wrapHandler(topic, handler))

	wp.logger.Infow("Handler registered", "topic", topic)
}

// wrapHandler 包装处理器添加日志和错误处理
func (wp *WorkerPool) wrapHandler(topic string, handler Handler) Handler {
	return func(ctx context.Context, msg *Message) error {
		start := time.Now()

		wp.logger.Debugw("Processing message",
			"topic", topic,
			"message_id", msg.ID,
			"retry", msg.Retries,
		)

		err := handler(ctx, msg)

		duration := time.Since(start)

		if err != nil {
			wp.logger.Warnw("Message processing failed",
				"topic", topic,
				"message_id", msg.ID,
				"error", err,
				"duration", duration,
				"retry", msg.Retries,
			)
		} else {
			wp.logger.Debugw("Message processed",
				"topic", topic,
				"message_id", msg.ID,
				"duration", duration,
			)
		}

		return err
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.ctx, wp.cancel = context.WithCancel(ctx)

	if err := wp.queue.Start(wp.ctx); err != nil {
		return fmt.Errorf("failed to start queue: %w", err)
	}

	wp.logger.Info("Worker pool started")
	return nil
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() error {
	if wp.cancel != nil {
		wp.cancel()
	}

	if err := wp.queue.Stop(); err != nil {
		return fmt.Errorf("failed to stop queue: %w", err)
	}

	wp.logger.Info("Worker pool stopped")
	return nil
}

// Publish 发布消息
func (wp *WorkerPool) Publish(ctx context.Context, topic string, payload interface{}) error {
	return wp.queue.Publish(ctx, topic, payload)
}

// Stats 获取统计
func (wp *WorkerPool) Stats() QueueStats {
	return wp.queue.Stats()
}

// ============================================================================
// 事件处理器工厂
// ============================================================================

// EventStoreHandler 创建事件存储处理器
func EventStoreHandler(storeFn func(ctx context.Context, msg *EventStoreMessage) error) Handler {
	return func(ctx context.Context, msg *Message) error {
		data, err := ParseMessage[EventStoreMessage](msg)
		if err != nil {
			return fmt.Errorf("failed to parse event store message: %w", err)
		}
		return storeFn(ctx, data)
	}
}

// CertCommitHandler 创建证书提交处理器
func CertCommitHandler(commitFn func(ctx context.Context, msg *CertCommitMessage) error) Handler {
	return func(ctx context.Context, msg *Message) error {
		data, err := ParseMessage[CertCommitMessage](msg)
		if err != nil {
			return fmt.Errorf("failed to parse cert commit message: %w", err)
		}
		return commitFn(ctx, data)
	}
}

// BlockchainAnchorHandler 创建区块链锚定处理器
func BlockchainAnchorHandler(anchorFn func(ctx context.Context, msg *BlockchainAnchorMessage) error) Handler {
	return func(ctx context.Context, msg *Message) error {
		data, err := ParseMessage[BlockchainAnchorMessage](msg)
		if err != nil {
			return fmt.Errorf("failed to parse blockchain anchor message: %w", err)
		}
		return anchorFn(ctx, data)
	}
}

// FingerprintComputeHandler 创建指纹计算处理器
func FingerprintComputeHandler(computeFn func(ctx context.Context, msg *FingerprintComputeMessage) error) Handler {
	return func(ctx context.Context, msg *Message) error {
		data, err := ParseMessage[FingerprintComputeMessage](msg)
		if err != nil {
			return fmt.Errorf("failed to parse fingerprint compute message: %w", err)
		}
		return computeFn(ctx, data)
	}
}

// ============================================================================
// 批量处理器
// ============================================================================

// BatchProcessor 批量处理器
type BatchProcessor[T any] struct {
	queue       Queue
	topic       string
	batchSize   int
	flushInterval time.Duration
	processFn   func(ctx context.Context, batch []T) error
	logger      *zap.SugaredLogger

	buffer    []T
	bufferMu  sync.Mutex
	lastFlush time.Time

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BatchProcessorConfig 批量处理器配置
type BatchProcessorConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor[T any](
	queue Queue,
	topic string,
	config BatchProcessorConfig,
	processFn func(ctx context.Context, batch []T) error,
	logger *zap.SugaredLogger,
) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		queue:         queue,
		topic:         topic,
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		processFn:     processFn,
		logger:        logger,
		buffer:        make([]T, 0, config.BatchSize),
		lastFlush:     time.Now(),
	}
}

// Start 启动批量处理器
func (bp *BatchProcessor[T]) Start(ctx context.Context) error {
	bp.ctx, bp.cancel = context.WithCancel(ctx)

	// 订阅消息
	bp.queue.Subscribe(bp.topic, bp.handleMessage)

	// 启动定时刷新
	bp.wg.Add(1)
	go bp.flushLoop()

	return nil
}

// Stop 停止批量处理器
func (bp *BatchProcessor[T]) Stop() error {
	if bp.cancel != nil {
		bp.cancel()
	}

	bp.wg.Wait()

	// 处理剩余的消息
	bp.flush()

	return nil
}

// handleMessage 处理单条消息
func (bp *BatchProcessor[T]) handleMessage(ctx context.Context, msg *Message) error {
	var item T
	if err := json.Unmarshal(msg.Payload, &item); err != nil {
		return err
	}

	bp.bufferMu.Lock()
	bp.buffer = append(bp.buffer, item)
	shouldFlush := len(bp.buffer) >= bp.batchSize
	bp.bufferMu.Unlock()

	if shouldFlush {
		bp.flush()
	}

	return nil
}

// flushLoop 定时刷新循环
func (bp *BatchProcessor[T]) flushLoop() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-ticker.C:
			bp.flush()
		}
	}
}

// flush 刷新缓冲区
func (bp *BatchProcessor[T]) flush() {
	bp.bufferMu.Lock()
	if len(bp.buffer) == 0 {
		bp.bufferMu.Unlock()
		return
	}

	batch := bp.buffer
	bp.buffer = make([]T, 0, bp.batchSize)
	bp.lastFlush = time.Now()
	bp.bufferMu.Unlock()

	ctx, cancel := context.WithTimeout(bp.ctx, 30*time.Second)
	defer cancel()

	if err := bp.processFn(ctx, batch); err != nil {
		bp.logger.Errorw("Batch processing failed",
			"topic", bp.topic,
			"batch_size", len(batch),
			"error", err,
		)
	} else {
		bp.logger.Debugw("Batch processed",
			"topic", bp.topic,
			"batch_size", len(batch),
		)
	}
}

// ============================================================================
// 延迟队列
// ============================================================================

// DelayedMessage 延迟消息
type DelayedMessage struct {
	Message   *Message
	ExecuteAt time.Time
}

// DelayedQueue 延迟队列
type DelayedQueue struct {
	queue    Queue
	messages []*DelayedMessage
	mu       sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDelayedQueue 创建延迟队列
func NewDelayedQueue(queue Queue) *DelayedQueue {
	return &DelayedQueue{
		queue:    queue,
		messages: make([]*DelayedMessage, 0),
	}
}

// PublishDelayed 发布延迟消息
func (dq *DelayedQueue) PublishDelayed(ctx context.Context, topic string, payload interface{}, delay time.Duration) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := &Message{
		ID:        generateMessageID(),
		Topic:     topic,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}

	dq.mu.Lock()
	dq.messages = append(dq.messages, &DelayedMessage{
		Message:   msg,
		ExecuteAt: time.Now().Add(delay),
	})
	dq.mu.Unlock()

	return nil
}

// Start 启动延迟队列处理
func (dq *DelayedQueue) Start(ctx context.Context) error {
	dq.ctx, dq.cancel = context.WithCancel(ctx)

	dq.wg.Add(1)
	go dq.processLoop()

	return nil
}

// Stop 停止延迟队列
func (dq *DelayedQueue) Stop() error {
	if dq.cancel != nil {
		dq.cancel()
	}
	dq.wg.Wait()
	return nil
}

// processLoop 处理循环
func (dq *DelayedQueue) processLoop() {
	defer dq.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-dq.ctx.Done():
			return
		case <-ticker.C:
			dq.processDelayed()
		}
	}
}

// processDelayed 处理到期的延迟消息
func (dq *DelayedQueue) processDelayed() {
	now := time.Now()

	dq.mu.Lock()
	var remaining []*DelayedMessage
	var ready []*DelayedMessage

	for _, dm := range dq.messages {
		if now.After(dm.ExecuteAt) {
			ready = append(ready, dm)
		} else {
			remaining = append(remaining, dm)
		}
	}

	dq.messages = remaining
	dq.mu.Unlock()

	// 发布到期的消息
	for _, dm := range ready {
		dq.queue.Publish(dq.ctx, dm.Message.Topic, dm.Message.Payload)
	}
}
