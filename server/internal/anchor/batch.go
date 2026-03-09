// Package anchor 提供批量区块链锚定功能
// 通过合并多个证书到单一 Merkle 根来节省 gas 费用
package anchor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrBatchFull      = errors.New("batch anchor: batch is full")
	ErrBatcherClosed  = errors.New("batch anchor: batcher is closed")
	ErrAnchorTimeout  = errors.New("batch anchor: anchor timeout")
)

// BatchAnchorConfig 批量锚定配置
type BatchAnchorConfig struct {
	// MaxBatchSize 最大批次大小
	MaxBatchSize int

	// FlushInterval 刷新间隔
	FlushInterval time.Duration

	// AnchorTimeout 锚定超时
	AnchorTimeout time.Duration

	// RetryCount 重试次数
	RetryCount int

	// RetryDelay 重试延迟
	RetryDelay time.Duration
}

// DefaultBatchAnchorConfig 默认配置
func DefaultBatchAnchorConfig() BatchAnchorConfig {
	return BatchAnchorConfig{
		MaxBatchSize:  100,
		FlushInterval: 1 * time.Minute,
		AnchorTimeout: 5 * time.Minute,
		RetryCount:    3,
		RetryDelay:    10 * time.Second,
	}
}

// AnchorItem 待锚定项
type AnchorItem struct {
	CertID       string    `json:"cert_id"`
	RootHash     string    `json:"root_hash"`
	TenantID     string    `json:"tenant_id"`
	Priority     int       `json:"priority"` // 优先级，数字越小越优先
	SubmittedAt  time.Time `json:"submitted_at"`
	ResultChan   chan *BatchAnchorResult `json:"-"`
}

// BatchAnchorResult 批量锚定结果
type BatchAnchorResult struct {
	Success     bool      `json:"success"`
	BatchID     string    `json:"batch_id"`
	BatchRoot   string    `json:"batch_root"`
	TxHash      string    `json:"tx_hash,omitempty"`
	BlockHeight int64     `json:"block_height,omitempty"`
	MerkleProof []string  `json:"merkle_proof"` // 证书在批次中的 Merkle 证明
	ProofIndex  int       `json:"proof_index"`
	Error       string    `json:"error,omitempty"`
	AnchoredAt  time.Time `json:"anchored_at"`
}

// Batch 批次
type Batch struct {
	ID          string        `json:"id"`
	Items       []*AnchorItem `json:"items"`
	MerkleRoot  string        `json:"merkle_root"`
	MerkleNodes [][]string    `json:"merkle_nodes"`
	CreatedAt   time.Time     `json:"created_at"`
	AnchoredAt  time.Time     `json:"anchored_at"`
	TxHash      string        `json:"tx_hash"`
	BlockHeight int64         `json:"block_height"`
	Status      BatchStatus   `json:"status"`
}

// BatchStatus 批次状态
type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "pending"
	BatchStatusAnchoring BatchStatus = "anchoring"
	BatchStatusAnchored  BatchStatus = "anchored"
	BatchStatusFailed    BatchStatus = "failed"
)

// BatchAnchorer 批量锚定器
type BatchAnchorer struct {
	config    BatchAnchorConfig
	anchorer  Anchorer
	logger    *zap.SugaredLogger

	pendingItems []*AnchorItem
	itemsMu      sync.Mutex

	batches   map[string]*Batch
	batchesMu sync.RWMutex

	// 统计
	stats      BatchAnchorStats
	statsMu    sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
}

// BatchAnchorStats 批量锚定统计
type BatchAnchorStats struct {
	TotalItems       int64 `json:"total_items"`
	TotalBatches     int64 `json:"total_batches"`
	SuccessfulItems  int64 `json:"successful_items"`
	FailedItems      int64 `json:"failed_items"`
	PendingItems     int   `json:"pending_items"`
	AverageBatchSize float64 `json:"average_batch_size"`
	GasSaved         int64 `json:"gas_saved"` // 估算节省的 gas（假设单次锚定 100000 gas）
}

// NewBatchAnchorer 创建批量锚定器
func NewBatchAnchorer(anchorer Anchorer, config BatchAnchorConfig, logger *zap.SugaredLogger) *BatchAnchorer {
	return &BatchAnchorer{
		config:       config,
		anchorer:     anchorer,
		logger:       logger,
		pendingItems: make([]*AnchorItem, 0),
		batches:      make(map[string]*Batch),
	}
}

// Start 启动批量锚定器
func (ba *BatchAnchorer) Start(ctx context.Context) error {
	ba.ctx, ba.cancel = context.WithCancel(ctx)

	ba.wg.Add(1)
	go ba.flushLoop()

	ba.logger.Info("Batch anchorer started")
	return nil
}

// Stop 停止批量锚定器
func (ba *BatchAnchorer) Stop() error {
	ba.itemsMu.Lock()
	ba.closed = true
	ba.itemsMu.Unlock()

	if ba.cancel != nil {
		ba.cancel()
	}

	// 处理剩余的待锚定项
	ba.flush()

	ba.wg.Wait()
	ba.logger.Info("Batch anchorer stopped")
	return nil
}

// Submit 提交待锚定项
func (ba *BatchAnchorer) Submit(item *AnchorItem) error {
	ba.itemsMu.Lock()
	if ba.closed {
		ba.itemsMu.Unlock()
		return ErrBatcherClosed
	}

	item.SubmittedAt = time.Now()
	item.ResultChan = make(chan *BatchAnchorResult, 1)

	ba.pendingItems = append(ba.pendingItems, item)
	shouldFlush := len(ba.pendingItems) >= ba.config.MaxBatchSize
	ba.itemsMu.Unlock()

	ba.statsMu.Lock()
	ba.stats.TotalItems++
	ba.statsMu.Unlock()

	if shouldFlush {
		go ba.flush()
	}

	return nil
}

// SubmitAndWait 提交并等待结果
func (ba *BatchAnchorer) SubmitAndWait(ctx context.Context, item *AnchorItem) (*BatchAnchorResult, error) {
	if err := ba.Submit(item); err != nil {
		return nil, err
	}

	select {
	case result := <-item.ResultChan:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(ba.config.AnchorTimeout):
		return nil, ErrAnchorTimeout
	}
}

// flushLoop 定时刷新循环
func (ba *BatchAnchorer) flushLoop() {
	defer ba.wg.Done()

	ticker := time.NewTicker(ba.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ba.ctx.Done():
			return
		case <-ticker.C:
			ba.flush()
		}
	}
}

// flush 刷新待锚定项
func (ba *BatchAnchorer) flush() {
	ba.itemsMu.Lock()
	if len(ba.pendingItems) == 0 {
		ba.itemsMu.Unlock()
		return
	}

	items := ba.pendingItems
	ba.pendingItems = make([]*AnchorItem, 0)
	ba.itemsMu.Unlock()

	// 按优先级排序
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].SubmittedAt.Before(items[j].SubmittedAt)
	})

	// 创建批次
	batch := ba.createBatch(items)

	// 执行锚定
	ba.anchorBatch(batch)
}

// createBatch 创建批次
func (ba *BatchAnchorer) createBatch(items []*AnchorItem) *Batch {
	batch := &Batch{
		ID:        generateBatchID(),
		Items:     items,
		CreatedAt: time.Now(),
		Status:    BatchStatusPending,
	}

	// 构建 Merkle 树
	leaves := make([]string, len(items))
	for i, item := range items {
		leaves[i] = item.RootHash
	}

	merkleRoot, merkleNodes := buildBatchMerkleTree(leaves)
	batch.MerkleRoot = merkleRoot
	batch.MerkleNodes = merkleNodes

	// 保存批次
	ba.batchesMu.Lock()
	ba.batches[batch.ID] = batch
	ba.batchesMu.Unlock()

	ba.statsMu.Lock()
	ba.stats.TotalBatches++
	ba.statsMu.Unlock()

	ba.logger.Infow("Batch created",
		"batch_id", batch.ID,
		"items_count", len(items),
		"merkle_root", merkleRoot,
	)

	return batch
}

// anchorBatch 执行批次锚定
func (ba *BatchAnchorer) anchorBatch(batch *Batch) {
	batch.Status = BatchStatusAnchoring

	ctx, cancel := context.WithTimeout(ba.ctx, ba.config.AnchorTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= ba.config.RetryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(ba.config.RetryDelay)
			ba.logger.Infow("Retrying batch anchor",
				"batch_id", batch.ID,
				"attempt", attempt,
			)
		}

		// 构建元数据
		metadata, _ := json.Marshal(map[string]interface{}{
			"batch_id":    batch.ID,
			"items_count": len(batch.Items),
		})

		result, err := ba.anchorer.Anchor(ctx, &AnchorRequest{
			CertID:    fmt.Sprintf("batch:%s", batch.ID),
			RootHash:  batch.MerkleRoot,
			Timestamp: batch.CreatedAt,
			Metadata:  string(metadata),
		})

		if err != nil {
			lastErr = err
			continue
		}

		// 成功
		batch.Status = BatchStatusAnchored
		batch.TxHash = result.TxHash
		batch.BlockHeight = int64(result.BlockNumber)
		batch.AnchoredAt = time.Now()

		ba.notifyResults(batch, true, "")
		ba.updateStats(batch, true)

		ba.logger.Infow("Batch anchored successfully",
			"batch_id", batch.ID,
			"tx_hash", result.TxHash,
			"block_number", result.BlockNumber,
			"items_count", len(batch.Items),
		)

		return
	}

	// 失败
	batch.Status = BatchStatusFailed
	ba.notifyResults(batch, false, lastErr.Error())
	ba.updateStats(batch, false)

	ba.logger.Errorw("Batch anchor failed",
		"batch_id", batch.ID,
		"error", lastErr,
	)
}

// notifyResults 通知所有项的结果
func (ba *BatchAnchorer) notifyResults(batch *Batch, success bool, errMsg string) {
	for i, item := range batch.Items {
		proof := getBatchMerkleProof(batch.MerkleNodes, i)

		result := &BatchAnchorResult{
			Success:     success,
			BatchID:     batch.ID,
			BatchRoot:   batch.MerkleRoot,
			TxHash:      batch.TxHash,
			BlockHeight: batch.BlockHeight,
			MerkleProof: proof,
			ProofIndex:  i,
			Error:       errMsg,
			AnchoredAt:  batch.AnchoredAt,
		}

		select {
		case item.ResultChan <- result:
		default:
			// 接收方可能已经超时
		}
	}
}

// updateStats 更新统计
func (ba *BatchAnchorer) updateStats(batch *Batch, success bool) {
	ba.statsMu.Lock()
	defer ba.statsMu.Unlock()

	if success {
		ba.stats.SuccessfulItems += int64(len(batch.Items))
		// 假设单次锚定 100000 gas，批量锚定 120000 gas
		ba.stats.GasSaved += int64((len(batch.Items)-1) * 100000)
	} else {
		ba.stats.FailedItems += int64(len(batch.Items))
	}

	if ba.stats.TotalBatches > 0 {
		ba.stats.AverageBatchSize = float64(ba.stats.TotalItems) / float64(ba.stats.TotalBatches)
	}
}

// Stats 获取统计
func (ba *BatchAnchorer) Stats() BatchAnchorStats {
	ba.statsMu.RLock()
	defer ba.statsMu.RUnlock()

	stats := ba.stats
	ba.itemsMu.Lock()
	stats.PendingItems = len(ba.pendingItems)
	ba.itemsMu.Unlock()

	return stats
}

// GetBatch 获取批次信息
func (ba *BatchAnchorer) GetBatch(batchID string) (*Batch, bool) {
	ba.batchesMu.RLock()
	defer ba.batchesMu.RUnlock()
	batch, ok := ba.batches[batchID]
	return batch, ok
}

// ============================================================================
// 辅助函数
// ============================================================================

func generateBatchID() string {
	return "batch_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond) // 确保随机性
	}
	return string(b)
}

// buildBatchMerkleTree 构建批次 Merkle 树
func buildBatchMerkleTree(leaves []string) (string, [][]string) {
	if len(leaves) == 0 {
		return "", nil
	}

	// 填充到 2 的幂次
	for len(leaves)&(len(leaves)-1) != 0 {
		leaves = append(leaves, leaves[len(leaves)-1])
	}

	nodes := [][]string{leaves}
	current := leaves

	for len(current) > 1 {
		var next []string
		for i := 0; i < len(current); i += 2 {
			combined := current[i] + current[i+1]
			hash := sha256.Sum256([]byte(combined))
			next = append(next, hex.EncodeToString(hash[:]))
		}
		nodes = append(nodes, next)
		current = next
	}

	return current[0], nodes
}

// getBatchMerkleProof 获取批次 Merkle 证明
func getBatchMerkleProof(nodes [][]string, index int) []string {
	if len(nodes) == 0 {
		return nil
	}

	var proof []string
	for level := 0; level < len(nodes)-1; level++ {
		siblingIndex := index ^ 1
		if siblingIndex < len(nodes[level]) {
			proof = append(proof, nodes[level][siblingIndex])
		}
		index = index / 2
	}

	return proof
}

// VerifyBatchProof 验证批次 Merkle 证明
func VerifyBatchProof(rootHash, itemHash string, proof []string, index int) bool {
	current := itemHash

	for _, sibling := range proof {
		var combined string
		if index%2 == 0 {
			combined = current + sibling
		} else {
			combined = sibling + current
		}
		hash := sha256.Sum256([]byte(combined))
		current = hex.EncodeToString(hash[:])
		index = index / 2
	}

	return current == rootHash
}
