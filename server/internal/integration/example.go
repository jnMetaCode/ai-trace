// Package integration 提供各模块的集成示例
// 展示如何将消息队列、批量锚定、增量Merkle树等组件结合使用
package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-trace/server/internal/anchor"
	"github.com/ai-trace/server/internal/merkle"
	"github.com/ai-trace/server/internal/queue"
	"go.uber.org/zap"
)

// TraceProcessorConfig 追踪处理器配置
type TraceProcessorConfig struct {
	QueueConfig     queue.MemoryQueueConfig
	BatchConfig     anchor.BatchAnchorConfig
	MerkleMaxDepth  int
}

// DefaultTraceProcessorConfig 默认配置
func DefaultTraceProcessorConfig() TraceProcessorConfig {
	return TraceProcessorConfig{
		QueueConfig:    queue.DefaultMemoryQueueConfig(),
		BatchConfig:    anchor.DefaultBatchAnchorConfig(),
		MerkleMaxDepth: 32,
	}
}

// TraceProcessor 追踪处理器
// 集成消息队列、批量锚定和增量Merkle树
type TraceProcessor struct {
	queue         queue.Queue
	workerPool    *queue.WorkerPool
	batchAnchorer *anchor.BatchAnchorer
	merkleTree    *merkle.IncrementalTree
	logger        *zap.SugaredLogger
	config        TraceProcessorConfig
}

// NewTraceProcessor 创建追踪处理器
func NewTraceProcessor(
	anchorer anchor.Anchorer,
	config TraceProcessorConfig,
	logger *zap.SugaredLogger,
) *TraceProcessor {
	// 创建消息队列
	q := queue.NewMemoryQueue(config.QueueConfig)

	// 创建工作池
	workerPool := queue.NewWorkerPool(q, logger)

	// 创建批量锚定器
	batchAnchorer := anchor.NewBatchAnchorer(anchorer, config.BatchConfig, logger)

	// 创建增量Merkle树
	merkleTree := merkle.NewIncrementalTree(merkle.IncrementalTreeConfig{
		MaxDepth: config.MerkleMaxDepth,
	})

	return &TraceProcessor{
		queue:         q,
		workerPool:    workerPool,
		batchAnchorer: batchAnchorer,
		merkleTree:    merkleTree,
		logger:        logger,
		config:        config,
	}
}

// Start 启动处理器
func (tp *TraceProcessor) Start(ctx context.Context) error {
	// 注册事件处理器
	tp.workerPool.RegisterHandler(queue.TopicEventStore, tp.handleEventStore)
	tp.workerPool.RegisterHandler(queue.TopicCertCommit, tp.handleCertCommit)
	tp.workerPool.RegisterHandler(queue.TopicBlockchainAnchor, tp.handleBlockchainAnchor)

	// 启动工作池
	if err := tp.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// 启动批量锚定器
	if err := tp.batchAnchorer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start batch anchorer: %w", err)
	}

	tp.logger.Info("Trace processor started")
	return nil
}

// Stop 停止处理器
func (tp *TraceProcessor) Stop() error {
	if err := tp.workerPool.Stop(); err != nil {
		tp.logger.Warnf("Error stopping worker pool: %v", err)
	}

	if err := tp.batchAnchorer.Stop(); err != nil {
		tp.logger.Warnf("Error stopping batch anchorer: %v", err)
	}

	tp.logger.Info("Trace processor stopped")
	return nil
}

// ProcessEvent 处理追踪事件
// 将事件添加到Merkle树并异步发送到队列
func (tp *TraceProcessor) ProcessEvent(ctx context.Context, traceID, eventData string) error {
	// 将事件哈希添加到增量Merkle树 - O(log n) 操作
	if err := tp.merkleTree.Append(eventData); err != nil {
		return fmt.Errorf("failed to append to merkle tree: %w", err)
	}

	// 异步发送到事件存储队列
	err := tp.workerPool.Publish(ctx, queue.TopicEventStore, &queue.EventStoreMessage{
		TraceID:  traceID,
		TenantID: "default",
		Events:   []byte(eventData),
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// CommitCertificate 提交证书到区块链
// 使用批量锚定来节省gas费用
func (tp *TraceProcessor) CommitCertificate(ctx context.Context, certID, tenantID string) (*anchor.BatchAnchorResult, error) {
	// 获取当前Merkle根
	root, err := tp.merkleTree.Root()
	if err != nil {
		return nil, fmt.Errorf("failed to get merkle root: %w", err)
	}

	// 提交到批量锚定器
	item := &anchor.AnchorItem{
		CertID:   certID,
		RootHash: root,
		TenantID: tenantID,
		Priority: 1, // 正常优先级
	}

	result, err := tp.batchAnchorer.SubmitAndWait(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to anchor certificate: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("anchor failed: %s", result.Error)
	}

	return result, nil
}

// GetMerkleProof 获取指定事件的Merkle证明
func (tp *TraceProcessor) GetMerkleProof(eventIndex int) (*merkle.IncrementalProof, error) {
	return tp.merkleTree.GetProof(eventIndex)
}

// GetStats 获取统计信息
func (tp *TraceProcessor) GetStats() ProcessorStats {
	return ProcessorStats{
		QueueStats:     tp.workerPool.Stats(),
		AnchorStats:    tp.batchAnchorer.Stats(),
		MerkleLeafCount: tp.merkleTree.LeafCount(),
		MerkleHeight:    tp.merkleTree.Height(),
	}
}

// ProcessorStats 处理器统计
type ProcessorStats struct {
	QueueStats      queue.QueueStats
	AnchorStats     anchor.BatchAnchorStats
	MerkleLeafCount int
	MerkleHeight    int
}

// 内部处理器
func (tp *TraceProcessor) handleEventStore(ctx context.Context, msg *queue.Message) error {
	data, err := queue.ParseMessage[queue.EventStoreMessage](msg)
	if err != nil {
		return err
	}

	tp.logger.Debugw("Storing event",
		"trace_id", data.TraceID,
		"tenant_id", data.TenantID,
	)

	// 这里实际实现会调用存储服务
	return nil
}

func (tp *TraceProcessor) handleCertCommit(ctx context.Context, msg *queue.Message) error {
	data, err := queue.ParseMessage[queue.CertCommitMessage](msg)
	if err != nil {
		return err
	}

	tp.logger.Debugw("Committing certificate",
		"trace_id", data.TraceID,
		"evidence_level", data.EvidenceLevel,
	)

	// 这里实际实现会调用证书服务
	return nil
}

func (tp *TraceProcessor) handleBlockchainAnchor(ctx context.Context, msg *queue.Message) error {
	data, err := queue.ParseMessage[queue.BlockchainAnchorMessage](msg)
	if err != nil {
		return err
	}

	tp.logger.Debugw("Anchoring to blockchain",
		"cert_id", data.CertID,
		"chain_type", data.ChainType,
	)

	// 提交到批量锚定器
	item := &anchor.AnchorItem{
		CertID:   data.CertID,
		RootHash: data.RootHash,
		Priority: data.Priority,
	}

	return tp.batchAnchorer.Submit(item)
}

// ============================================================================
// 使用示例
// ============================================================================

// ExampleUsage 展示如何使用TraceProcessor
func ExampleUsage() {
	// 这是一个示例，展示集成使用方式
	logger := zap.NewNop().Sugar()

	// 创建模拟锚定器（实际使用时替换为真实锚定器）
	// anchorer := anchor.NewEthereumAnchor(config, logger)

	// 创建处理器
	config := DefaultTraceProcessorConfig()
	// processor := NewTraceProcessor(anchorer, config, logger)

	ctx := context.Background()

	// 启动处理器
	// processor.Start(ctx)
	// defer processor.Stop()

	// 处理事件
	// processor.ProcessEvent(ctx, "trace-123", "event-data-hash")

	// 提交证书
	// result, _ := processor.CommitCertificate(ctx, "cert-456", "tenant-789")

	// 获取证明
	// proof, _ := processor.GetMerkleProof(0)

	// 获取统计
	// stats := processor.GetStats()

	_ = logger
	_ = config
	_ = ctx
}

// ============================================================================
// 高级用法：批处理器
// ============================================================================

// BatchEventProcessor 批量事件处理器
// 用于高吞吐量场景，将多个事件批量处理
type BatchEventProcessor struct {
	processor     *TraceProcessor
	batchSize     int
	flushInterval time.Duration
	events        []string
	logger        *zap.SugaredLogger
}

// NewBatchEventProcessor 创建批量事件处理器
func NewBatchEventProcessor(
	processor *TraceProcessor,
	batchSize int,
	flushInterval time.Duration,
	logger *zap.SugaredLogger,
) *BatchEventProcessor {
	return &BatchEventProcessor{
		processor:     processor,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		events:        make([]string, 0, batchSize),
		logger:        logger,
	}
}

// AddEvent 添加事件到批次
func (bep *BatchEventProcessor) AddEvent(eventHash string) {
	bep.events = append(bep.events, eventHash)
	if len(bep.events) >= bep.batchSize {
		bep.Flush()
	}
}

// Flush 刷新批次到Merkle树
func (bep *BatchEventProcessor) Flush() {
	if len(bep.events) == 0 {
		return
	}

	// 批量添加到Merkle树
	if err := bep.processor.merkleTree.AppendBatch(bep.events); err != nil {
		bep.logger.Errorw("Failed to append batch", "error", err)
		return
	}

	bep.logger.Infow("Batch flushed",
		"count", len(bep.events),
		"total_leaves", bep.processor.merkleTree.LeafCount(),
	)

	bep.events = bep.events[:0]
}
