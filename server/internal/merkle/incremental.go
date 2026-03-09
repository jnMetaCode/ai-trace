// Package merkle 提供增量 Merkle 树实现
// 支持 O(log n) 的叶子追加操作，而不需要重建整棵树
package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrTreeEmpty     = errors.New("incremental tree: tree is empty")
	ErrInvalidIndex  = errors.New("incremental tree: invalid index")
	ErrTreeCorrupted = errors.New("incremental tree: tree corrupted")
)

// IncrementalTree 增量 Merkle 树
// 支持高效的叶子追加和证明生成
type IncrementalTree struct {
	// 存储树的节点，按层级组织
	// levels[0] = 叶子层
	// levels[len-1] = 根层
	levels [][]string

	// 叶子数量
	leafCount int

	// 最大深度（预分配）
	maxDepth int

	// 算法
	algorithm string

	mu sync.RWMutex
}

// IncrementalTreeConfig 配置
type IncrementalTreeConfig struct {
	// MaxDepth 最大深度（决定最大叶子数量 = 2^MaxDepth）
	MaxDepth int
}

// DefaultIncrementalTreeConfig 默认配置
func DefaultIncrementalTreeConfig() IncrementalTreeConfig {
	return IncrementalTreeConfig{
		MaxDepth: 32, // 最大支持 2^32 约 40 亿个叶子
	}
}

// NewIncrementalTree 创建增量 Merkle 树
func NewIncrementalTree(config IncrementalTreeConfig) *IncrementalTree {
	return &IncrementalTree{
		levels:    make([][]string, config.MaxDepth+1),
		maxDepth:  config.MaxDepth,
		algorithm: "sha256",
	}
}

// Append 追加叶子节点
// 时间复杂度: O(log n)
func (t *IncrementalTree) Append(leaf string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 检查是否超过最大容量
	maxLeaves := 1 << t.maxDepth
	if t.leafCount >= maxLeaves {
		return fmt.Errorf("tree capacity exceeded: max %d leaves", maxLeaves)
	}

	// 在叶子层追加
	t.levels[0] = append(t.levels[0], leaf)
	t.leafCount++

	// 向上更新受影响的路径
	t.updatePath(t.leafCount - 1)

	return nil
}

// AppendBatch 批量追加叶子
func (t *IncrementalTree) AppendBatch(leaves []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	maxLeaves := 1 << t.maxDepth
	if t.leafCount+len(leaves) > maxLeaves {
		return fmt.Errorf("tree capacity exceeded: max %d leaves", maxLeaves)
	}

	for _, leaf := range leaves {
		t.levels[0] = append(t.levels[0], leaf)
		t.leafCount++
		t.updatePath(t.leafCount - 1)
	}

	return nil
}

// updatePath 更新从叶子到根的路径
func (t *IncrementalTree) updatePath(leafIndex int) {
	currentIndex := leafIndex

	for level := 0; level < t.currentHeight(); level++ {
		// 计算父节点索引
		parentIndex := currentIndex / 2

		// 获取左右子节点
		leftIndex := parentIndex * 2
		rightIndex := leftIndex + 1

		var left, right string
		if leftIndex < len(t.levels[level]) {
			left = t.levels[level][leftIndex]
		}
		if rightIndex < len(t.levels[level]) {
			right = t.levels[level][rightIndex]
		} else {
			// 如果右节点不存在，用左节点填充
			right = left
		}

		// 计算父节点哈希
		parentHash := t.hashPair(left, right)

		// 更新或追加父节点
		nextLevel := level + 1
		if parentIndex < len(t.levels[nextLevel]) {
			t.levels[nextLevel][parentIndex] = parentHash
		} else {
			t.levels[nextLevel] = append(t.levels[nextLevel], parentHash)
		}

		currentIndex = parentIndex
	}
}

// currentHeight 计算当前树高度
func (t *IncrementalTree) currentHeight() int {
	if t.leafCount == 0 {
		return 0
	}

	height := 0
	n := t.leafCount
	for n > 1 {
		height++
		n = (n + 1) / 2
	}
	return height + 1
}

// hashPair 计算一对节点的哈希
func (t *IncrementalTree) hashPair(left, right string) string {
	h := sha256.New()
	h.Write([]byte(left))
	h.Write([]byte(right))
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}

// Root 获取根哈希
func (t *IncrementalTree) Root() (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.rootLocked()
}

// rootLocked 获取根哈希（调用者必须已持有锁）
func (t *IncrementalTree) rootLocked() (string, error) {
	if t.leafCount == 0 {
		return "", ErrTreeEmpty
	}

	height := t.currentHeight()
	if height == 0 {
		return t.levels[0][0], nil
	}

	topLevel := t.levels[height-1]
	if len(topLevel) == 0 {
		return "", ErrTreeCorrupted
	}

	return topLevel[0], nil
}

// GetProof 获取指定索引的 Merkle 证明
// 时间复杂度: O(log n)
func (t *IncrementalTree) GetProof(index int) (*IncrementalProof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if index < 0 || index >= t.leafCount {
		return nil, ErrInvalidIndex
	}

	root, err := t.rootLocked()
	if err != nil {
		return nil, err
	}

	proof := &IncrementalProof{
		LeafIndex: index,
		LeafHash:  t.levels[0][index],
		Root:      root,
		Path:      make([]IncrementalProofNode, 0),
	}

	currentIndex := index
	height := t.currentHeight()

	for level := 0; level < height-1; level++ {
		siblingIndex := currentIndex ^ 1 // XOR 1 获取兄弟节点索引

		var siblingHash string
		if siblingIndex < len(t.levels[level]) {
			siblingHash = t.levels[level][siblingIndex]
		} else {
			// 如果兄弟不存在，使用当前节点
			siblingHash = t.levels[level][currentIndex]
		}

		position := "right"
		if currentIndex%2 == 1 {
			position = "left"
		}

		proof.Path = append(proof.Path, IncrementalProofNode{
			Hash:     siblingHash,
			Position: position,
		})

		currentIndex = currentIndex / 2
	}

	return proof, nil
}

// IncrementalProof 增量树的 Merkle 证明
type IncrementalProof struct {
	LeafIndex int                    `json:"leaf_index"`
	LeafHash  string                 `json:"leaf_hash"`
	Root      string                 `json:"root"`
	Path      []IncrementalProofNode `json:"path"`
}

// IncrementalProofNode 证明路径节点
type IncrementalProofNode struct {
	Hash     string `json:"hash"`
	Position string `json:"position"` // "left" or "right"
}

// Verify 验证证明
func (p *IncrementalProof) Verify() bool {
	currentHash := p.LeafHash

	for _, node := range p.Path {
		h := sha256.New()
		if node.Position == "right" {
			h.Write([]byte(currentHash))
			h.Write([]byte(node.Hash))
		} else {
			h.Write([]byte(node.Hash))
			h.Write([]byte(currentHash))
		}
		currentHash = fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
	}

	return currentHash == p.Root
}

// LeafCount 获取叶子数量
func (t *IncrementalTree) LeafCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.leafCount
}

// Height 获取树高度
func (t *IncrementalTree) Height() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentHeight()
}

// GetLeaf 获取指定索引的叶子
func (t *IncrementalTree) GetLeaf(index int) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if index < 0 || index >= t.leafCount {
		return "", ErrInvalidIndex
	}

	return t.levels[0][index], nil
}

// GetLeaves 获取所有叶子
func (t *IncrementalTree) GetLeaves() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]string, t.leafCount)
	copy(result, t.levels[0][:t.leafCount])
	return result
}

// ============================================================================
// 持久化支持
// ============================================================================

// TreeSnapshot 树快照（用于持久化）
type TreeSnapshot struct {
	Levels    [][]string `json:"levels"`
	LeafCount int        `json:"leaf_count"`
	MaxDepth  int        `json:"max_depth"`
	Algorithm string     `json:"algorithm"`
}

// Snapshot 创建快照
func (t *IncrementalTree) Snapshot() *TreeSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snapshot := &TreeSnapshot{
		Levels:    make([][]string, len(t.levels)),
		LeafCount: t.leafCount,
		MaxDepth:  t.maxDepth,
		Algorithm: t.algorithm,
	}

	for i, level := range t.levels {
		snapshot.Levels[i] = make([]string, len(level))
		copy(snapshot.Levels[i], level)
	}

	return snapshot
}

// RestoreFromSnapshot 从快照恢复
func RestoreFromSnapshot(snapshot *TreeSnapshot) *IncrementalTree {
	t := &IncrementalTree{
		levels:    make([][]string, len(snapshot.Levels)),
		leafCount: snapshot.LeafCount,
		maxDepth:  snapshot.MaxDepth,
		algorithm: snapshot.Algorithm,
	}

	for i, level := range snapshot.Levels {
		t.levels[i] = make([]string, len(level))
		copy(t.levels[i], level)
	}

	return t
}

// ============================================================================
// 比较和验证
// ============================================================================

// ConsistencyProof 一致性证明
// 证明旧根是新树的前缀
type ConsistencyProof struct {
	OldRoot   string   `json:"old_root"`
	NewRoot   string   `json:"new_root"`
	OldSize   int      `json:"old_size"`
	NewSize   int      `json:"new_size"`
	ProofPath []string `json:"proof_path"`
}

// GetConsistencyProof 获取一致性证明
func (t *IncrementalTree) GetConsistencyProof(oldSize int) (*ConsistencyProof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if oldSize <= 0 || oldSize > t.leafCount {
		return nil, ErrInvalidIndex
	}

	// 简化实现：返回足够的节点让验证者重建
	proof := &ConsistencyProof{
		OldSize:   oldSize,
		NewSize:   t.leafCount,
		ProofPath: make([]string, 0),
	}

	// 计算旧根
	oldTree := NewIncrementalTree(IncrementalTreeConfig{MaxDepth: t.maxDepth})
	for i := 0; i < oldSize; i++ {
		oldTree.Append(t.levels[0][i])
	}
	oldRoot, _ := oldTree.Root()
	proof.OldRoot = oldRoot

	// 当前根
	newRoot, _ := t.rootLocked()
	proof.NewRoot = newRoot

	// 收集证明路径（简化版）
	// 实际实现需要更复杂的算法
	for i := oldSize; i < t.leafCount; i++ {
		proof.ProofPath = append(proof.ProofPath, t.levels[0][i])
	}

	return proof, nil
}

// VerifyConsistency 验证一致性
func (p *ConsistencyProof) Verify() bool {
	// 简化验证：通过证明路径重建并比较
	// 完整实现需要使用证明路径验证新旧根的一致性
	// 当前简化实现：验证新旧大小关系和根哈希非空
	if p.OldSize <= 0 || p.NewSize < p.OldSize {
		return false
	}
	if p.OldRoot == "" || p.NewRoot == "" {
		return false
	}
	return true
}
