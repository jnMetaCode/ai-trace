package anchor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// batchMockAnchorer 模拟锚定器
type batchMockAnchorer struct {
	anchorCount int64
	shouldFail  bool
	delay       time.Duration
}

func (m *batchMockAnchorer) getAnchorCount() int64 {
	return atomic.LoadInt64(&m.anchorCount)
}

func (m *batchMockAnchorer) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error) {
	atomic.AddInt64(&m.anchorCount, 1)

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.shouldFail {
		return nil, ErrTxFailed
	}

	return &AnchorResult{
		AnchorID:    "anchor_" + req.CertID,
		AnchorType:  AnchorTypeEthereum,
		TxHash:      "0x" + req.RootHash[:10],
		BlockNumber: 12345,
		Timestamp:   time.Now(),
	}, nil
}

func (m *batchMockAnchorer) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	return true, nil
}

func (m *batchMockAnchorer) GetAnchorType() AnchorType {
	return AnchorTypeEthereum
}

func (m *batchMockAnchorer) IsAvailable(ctx context.Context) bool {
	return true
}

func TestDefaultBatchAnchorConfig(t *testing.T) {
	config := DefaultBatchAnchorConfig()

	if config.MaxBatchSize != 100 {
		t.Errorf("Expected MaxBatchSize 100, got %d", config.MaxBatchSize)
	}

	if config.FlushInterval != 1*time.Minute {
		t.Errorf("Expected FlushInterval 1m, got %v", config.FlushInterval)
	}

	if config.AnchorTimeout != 5*time.Minute {
		t.Errorf("Expected AnchorTimeout 5m, got %v", config.AnchorTimeout)
	}

	if config.RetryCount != 3 {
		t.Errorf("Expected RetryCount 3, got %d", config.RetryCount)
	}
}

func TestNewBatchAnchorer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := DefaultBatchAnchorConfig()

	ba := NewBatchAnchorer(mock, config, logger)
	if ba == nil {
		t.Fatal("NewBatchAnchorer returned nil")
	}
}

func TestBatchAnchorerSubmit(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := BatchAnchorConfig{
		MaxBatchSize:  10,
		FlushInterval: 100 * time.Millisecond,
		AnchorTimeout: 5 * time.Second,
		RetryCount:    1,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	// Submit an item
	item := &AnchorItem{
		CertID:   "cert-1",
		RootHash: "hash-1-abcdefghij",
		TenantID: "tenant-1",
		Priority: 1,
	}

	err := ba.Submit(item)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	stats := ba.Stats()
	if stats.TotalItems != 1 {
		t.Errorf("Expected TotalItems 1, got %d", stats.TotalItems)
	}
}

func TestBatchAnchorerSubmitAndWait(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := BatchAnchorConfig{
		MaxBatchSize:  5,
		FlushInterval: 50 * time.Millisecond,
		AnchorTimeout: 5 * time.Second,
		RetryCount:    1,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	item := &AnchorItem{
		CertID:   "cert-wait",
		RootHash: "hash-wait-12345",
		TenantID: "tenant-wait",
		Priority: 0,
	}

	result, err := ba.SubmitAndWait(ctx, item)
	if err != nil {
		t.Fatalf("SubmitAndWait failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	if result.BatchRoot == "" {
		t.Error("BatchRoot should not be empty")
	}
}

func TestBatchAnchorerBatching(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := BatchAnchorConfig{
		MaxBatchSize:  3,
		FlushInterval: 1 * time.Second,
		AnchorTimeout: 5 * time.Second,
		RetryCount:    1,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	// Submit 3 items to trigger batch
	for i := 0; i < 3; i++ {
		item := &AnchorItem{
			CertID:   "batch-cert-" + string(rune('0'+i)),
			RootHash: "hash-" + string(rune('A'+i)) + "-1234567890",
			TenantID: "tenant-batch",
			Priority: i,
		}
		ba.Submit(item)
	}

	// Wait for batch processing
	time.Sleep(300 * time.Millisecond)

	if mock.getAnchorCount() != 1 {
		t.Errorf("Expected 1 anchor call (batched), got %d", mock.getAnchorCount())
	}

	stats := ba.Stats()
	if stats.TotalBatches != 1 {
		t.Errorf("Expected 1 batch, got %d", stats.TotalBatches)
	}
}

func TestBatchAnchorerClosed(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := DefaultBatchAnchorConfig()

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	ba.Stop()

	item := &AnchorItem{
		CertID:   "cert-closed",
		RootHash: "hash-closed",
		TenantID: "tenant-closed",
	}

	err := ba.Submit(item)
	if err != ErrBatcherClosed {
		t.Errorf("Expected ErrBatcherClosed, got %v", err)
	}
}

func TestBatchAnchorerRetry(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{shouldFail: true}
	config := BatchAnchorConfig{
		MaxBatchSize:  1,
		FlushInterval: 50 * time.Millisecond,
		AnchorTimeout: 5 * time.Second,
		RetryCount:    2,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	item := &AnchorItem{
		CertID:   "cert-retry",
		RootHash: "hash-retry-123456",
		TenantID: "tenant-retry",
	}

	ba.Submit(item)

	// Wait for retries
	time.Sleep(300 * time.Millisecond)

	// Should have attempted 1 + 2 retries = 3 times
	if mock.getAnchorCount() != 3 {
		t.Errorf("Expected 3 anchor attempts, got %d", mock.getAnchorCount())
	}

	stats := ba.Stats()
	if stats.FailedItems != 1 {
		t.Errorf("Expected 1 failed item, got %d", stats.FailedItems)
	}
}

func TestBatchAnchorerStats(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := BatchAnchorConfig{
		MaxBatchSize:  2,
		FlushInterval: 500 * time.Millisecond, // Longer interval to control batching
		AnchorTimeout: 5 * time.Second,
		RetryCount:    1,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	// Submit first batch (triggers when hits MaxBatchSize=2)
	for i := 0; i < 2; i++ {
		item := &AnchorItem{
			CertID:   "cert-stats-" + string(rune('0'+i)),
			RootHash: "hash-stats-" + string(rune('A'+i)) + "-123",
			TenantID: "tenant-stats",
		}
		ba.Submit(item)
	}

	// Wait for first batch to process
	time.Sleep(100 * time.Millisecond)

	// Submit second batch
	for i := 2; i < 4; i++ {
		item := &AnchorItem{
			CertID:   "cert-stats-" + string(rune('0'+i)),
			RootHash: "hash-stats-" + string(rune('A'+i)) + "-123",
			TenantID: "tenant-stats",
		}
		ba.Submit(item)
	}

	// Wait for second batch to process
	time.Sleep(200 * time.Millisecond)

	stats := ba.Stats()

	if stats.TotalItems != 4 {
		t.Errorf("Expected TotalItems 4, got %d", stats.TotalItems)
	}

	if stats.SuccessfulItems != 4 {
		t.Errorf("Expected SuccessfulItems 4, got %d", stats.SuccessfulItems)
	}

	if stats.TotalBatches != 2 {
		t.Errorf("Expected TotalBatches 2, got %d", stats.TotalBatches)
	}

	// Gas saved calculation: (items-1) * 100000 per batch
	// 2 batches with 2 items each: 2 * (2-1) * 100000 = 200000
	expectedGasSaved := int64(200000)
	if stats.GasSaved != expectedGasSaved {
		t.Errorf("Expected GasSaved %d, got %d", expectedGasSaved, stats.GasSaved)
	}
}

func TestBatchAnchorerGetBatch(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mock := &batchMockAnchorer{}
	config := BatchAnchorConfig{
		MaxBatchSize:  1,
		FlushInterval: 50 * time.Millisecond,
		AnchorTimeout: 5 * time.Second,
		RetryCount:    1,
		RetryDelay:    10 * time.Millisecond,
	}

	ba := NewBatchAnchorer(mock, config, logger)
	ctx := context.Background()

	ba.Start(ctx)
	defer ba.Stop()

	item := &AnchorItem{
		CertID:   "cert-getbatch",
		RootHash: "hash-getbatch-12345",
		TenantID: "tenant-getbatch",
	}

	ba.Submit(item)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Check that we have at least one batch
	stats := ba.Stats()
	if stats.TotalBatches < 1 {
		t.Fatal("Expected at least 1 batch")
	}
}

func TestBuildBatchMerkleTree(t *testing.T) {
	tests := []struct {
		name   string
		leaves []string
	}{
		{"empty", []string{}},
		{"single", []string{"a"}},
		{"two", []string{"a", "b"}},
		{"three", []string{"a", "b", "c"}},
		{"four", []string{"a", "b", "c", "d"}},
		{"five", []string{"a", "b", "c", "d", "e"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, nodes := buildBatchMerkleTree(tc.leaves)

			if len(tc.leaves) == 0 {
				if root != "" || nodes != nil {
					t.Error("Empty leaves should return empty root and nil nodes")
				}
				return
			}

			if root == "" {
				t.Error("Root should not be empty")
			}

			if len(nodes) == 0 {
				t.Error("Nodes should not be empty")
			}

			// First level should contain leaves (possibly padded)
			if len(nodes[0]) < len(tc.leaves) {
				t.Errorf("First level should have at least %d nodes, got %d", len(tc.leaves), len(nodes[0]))
			}

			// Last level should have single root
			if len(nodes[len(nodes)-1]) != 1 {
				t.Errorf("Last level should have 1 node (root), got %d", len(nodes[len(nodes)-1]))
			}
		})
	}
}

func TestGetBatchMerkleProof(t *testing.T) {
	leaves := []string{"a", "b", "c", "d"}
	root, nodes := buildBatchMerkleTree(leaves)

	for i := 0; i < len(leaves); i++ {
		proof := getBatchMerkleProof(nodes, i)

		// Verify proof
		if !VerifyBatchProof(root, leaves[i], proof, i) {
			t.Errorf("Proof verification failed for index %d", i)
		}
	}
}

func TestVerifyBatchProof(t *testing.T) {
	leaves := []string{"item1", "item2", "item3", "item4"}
	root, nodes := buildBatchMerkleTree(leaves)

	tests := []struct {
		name     string
		root     string
		item     string
		proof    []string
		index    int
		expected bool
	}{
		{
			name:     "valid proof index 0",
			root:     root,
			item:     leaves[0],
			proof:    getBatchMerkleProof(nodes, 0),
			index:    0,
			expected: true,
		},
		{
			name:     "valid proof index 3",
			root:     root,
			item:     leaves[3],
			proof:    getBatchMerkleProof(nodes, 3),
			index:    3,
			expected: true,
		},
		{
			name:     "invalid item",
			root:     root,
			item:     "invalid",
			proof:    getBatchMerkleProof(nodes, 0),
			index:    0,
			expected: false,
		},
		{
			name:     "invalid root",
			root:     "invalidroot",
			item:     leaves[0],
			proof:    getBatchMerkleProof(nodes, 0),
			index:    0,
			expected: false,
		},
		{
			name:     "wrong index",
			root:     root,
			item:     leaves[0],
			proof:    getBatchMerkleProof(nodes, 0),
			index:    1, // Wrong index
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := VerifyBatchProof(tc.root, tc.item, tc.proof, tc.index)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestAnchorItem(t *testing.T) {
	item := AnchorItem{
		CertID:      "cert-123",
		RootHash:    "hash-456",
		TenantID:    "tenant-789",
		Priority:    1,
		SubmittedAt: time.Now(),
	}

	if item.CertID != "cert-123" {
		t.Error("CertID mismatch")
	}
	if item.Priority != 1 {
		t.Error("Priority mismatch")
	}
}

func TestBatchAnchorResult(t *testing.T) {
	result := BatchAnchorResult{
		Success:     true,
		BatchID:     "batch-123",
		BatchRoot:   "root-456",
		TxHash:      "0xabc",
		BlockHeight: 12345,
		MerkleProof: []string{"proof1", "proof2"},
		ProofIndex:  0,
		AnchoredAt:  time.Now(),
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.BatchID != "batch-123" {
		t.Error("BatchID mismatch")
	}
	if len(result.MerkleProof) != 2 {
		t.Error("MerkleProof length mismatch")
	}
}

func TestBatchStatus(t *testing.T) {
	statuses := []BatchStatus{
		BatchStatusPending,
		BatchStatusAnchoring,
		BatchStatusAnchored,
		BatchStatusFailed,
	}

	expected := []string{"pending", "anchoring", "anchored", "failed"}

	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("Status %d: expected %s, got %s", i, expected[i], s)
		}
	}
}

func TestBatchErrors(t *testing.T) {
	if ErrBatchFull.Error() != "batch anchor: batch is full" {
		t.Error("ErrBatchFull message mismatch")
	}
	if ErrBatcherClosed.Error() != "batch anchor: batcher is closed" {
		t.Error("ErrBatcherClosed message mismatch")
	}
	if ErrAnchorTimeout.Error() != "batch anchor: anchor timeout" {
		t.Error("ErrAnchorTimeout message mismatch")
	}
}
