package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// Tree Merkle树
type Tree struct {
	Leaves    []string   `json:"leaves"`
	Nodes     [][]string `json:"nodes"`
	Root      string     `json:"root"`
	Algorithm string     `json:"algorithm"`
}

// Proof Merkle证明
type Proof struct {
	LeafIndex int         `json:"leaf_index"`
	LeafHash  string      `json:"leaf_hash"`
	Path      []ProofNode `json:"path"`
	Root      string      `json:"root"`
}

// ProofNode 证明路径节点
type ProofNode struct {
	Hash     string `json:"hash"`
	Position string `json:"position"` // "left" or "right"
}

// NewTree 创建Merkle树
func NewTree(leaves []string) (*Tree, error) {
	if len(leaves) == 0 {
		return nil, errors.New("leaves cannot be empty")
	}

	tree := &Tree{
		Leaves:    leaves,
		Algorithm: "sha256",
	}

	// 构建树
	tree.Nodes = buildTree(leaves)
	if len(tree.Nodes) > 0 && len(tree.Nodes[len(tree.Nodes)-1]) > 0 {
		tree.Root = tree.Nodes[len(tree.Nodes)-1][0]
	}

	return tree, nil
}

// buildTree 构建Merkle树的所有层
func buildTree(leaves []string) [][]string {
	if len(leaves) == 0 {
		return nil
	}

	levels := [][]string{leaves}
	currentLevel := leaves

	for len(currentLevel) > 1 {
		nextLevel := []string{}

		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			right := left
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			}

			// 计算父节点哈希
			parent := hashPair(left, right)
			nextLevel = append(nextLevel, parent)
		}

		levels = append(levels, nextLevel)
		currentLevel = nextLevel
	}

	return levels
}

// hashPair 计算两个哈希的组合哈希
func hashPair(left, right string) string {
	h := sha256.New()
	h.Write([]byte(left))
	h.Write([]byte(right))
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}

// GetProof 获取指定叶子节点的Merkle证明
func (t *Tree) GetProof(index int) (*Proof, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, errors.New("index out of range")
	}

	proof := &Proof{
		LeafIndex: index,
		LeafHash:  t.Leaves[index],
		Root:      t.Root,
		Path:      []ProofNode{},
	}

	currentIndex := index

	// 遍历每一层（除了根节点层）
	for level := 0; level < len(t.Nodes)-1; level++ {
		levelNodes := t.Nodes[level]

		// 计算兄弟节点索引
		siblingIndex := currentIndex ^ 1 // XOR 1 来获取兄弟节点

		if siblingIndex < len(levelNodes) {
			position := "right"
			if currentIndex%2 == 1 {
				position = "left"
			}

			proof.Path = append(proof.Path, ProofNode{
				Hash:     levelNodes[siblingIndex],
				Position: position,
			})
		}

		// 移动到父节点索引
		currentIndex = currentIndex / 2
	}

	return proof, nil
}

// VerifyProof 验证Merkle证明
func VerifyProof(proof *Proof) bool {
	currentHash := proof.LeafHash

	for _, node := range proof.Path {
		if node.Position == "right" {
			currentHash = hashPair(currentHash, node.Hash)
		} else {
			currentHash = hashPair(node.Hash, currentHash)
		}
	}

	return currentHash == proof.Root
}

// GetRoot 获取根哈希
func (t *Tree) GetRoot() string {
	return t.Root
}

// GetLeafCount 获取叶子节点数量
func (t *Tree) GetLeafCount() int {
	return len(t.Leaves)
}

// GetHeight 获取树高度
func (t *Tree) GetHeight() int {
	return len(t.Nodes)
}
