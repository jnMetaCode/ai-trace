package zkp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Service ZKP 服务
type Service struct {
	prover   *Prover
	verifier *Verifier
	logger   *zap.SugaredLogger
	cache    map[string]*Proof // 证明缓存
	cacheMux sync.RWMutex
}

// NewService 创建 ZKP 服务
func NewService(logger *zap.SugaredLogger) *Service {
	return &Service{
		prover:   NewProver(),
		verifier: NewVerifier(),
		logger:   logger,
		cache:    make(map[string]*Proof),
	}
}

// Initialize 初始化服务（预编译电路）
func (s *Service) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing ZKP service...")

	// 预编译常用电路
	proofTypes := []ProofType{
		ProofTypeHashPreimage,
		ProofTypeContentOwnership,
		ProofTypeFingerprintVerify,
		ProofTypeMerkleProof,
	}

	for _, pt := range proofTypes {
		s.logger.Infof("Setting up circuit for %s...", pt)
		if err := s.prover.Setup(pt); err != nil {
			s.logger.Warnf("Failed to setup %s circuit: %v", pt, err)
			continue
		}

		// 同步验证密钥到验证器
		vk, err := s.prover.GetVerifyingKey(pt)
		if err == nil {
			s.verifier.SetVerifyingKey(pt, vk)
		}
	}

	s.logger.Info("ZKP service initialized")
	return nil
}

// ProveContentOwnership 证明内容所有权
type ContentOwnershipRequest struct {
	Content         []byte `json:"content"`
	OwnerID         string `json:"owner_id"`
	Timestamp       int64  `json:"timestamp"`
	BeforeTimestamp int64  `json:"before_timestamp"`
}

// ProveContentOwnership 生成内容所有权证明
func (s *Service) ProveContentOwnership(ctx context.Context, req *ContentOwnershipRequest) (*Proof, error) {
	// 计算公开哈希
	h := sha256.New()
	h.Write(req.Content)
	h.Write([]byte(req.OwnerID))
	contentHash := h.Sum(nil)

	// 分块内容
	chunks := splitIntoChunks(req.Content, 32)

	// 计算公开哈希承诺
	ownerIDInt := stringToBigInt(req.OwnerID)
	publicHash := computePublicHash(contentHash, ownerIDInt, req.Timestamp)

	proof, err := s.prover.ProveContentOwnership(
		chunks,
		ownerIDInt,
		req.Timestamp,
		publicHash,
		req.BeforeTimestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	// 缓存证明
	s.cacheProof(proof)

	return proof, nil
}

// ProveHashPreimage 证明哈希原像
type HashPreimageRequest struct {
	Preimage []byte `json:"preimage"`
}

// ProveHashPreimage 生成哈希原像证明
func (s *Service) ProveHashPreimage(ctx context.Context, req *HashPreimageRequest) (*Proof, error) {
	// 计算哈希
	hash := computeSimpleHash(req.Preimage)

	proof, err := s.prover.ProveHashPreimage(req.Preimage, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	s.cacheProof(proof)
	return proof, nil
}

// ProveFingerprintMatch 证明指纹匹配
type FingerprintMatchRequest struct {
	StatisticalFeatures []float64 `json:"statistical_features"`
	SemanticFeatures    []float64 `json:"semantic_features"`
	ModelID             string    `json:"model_id"`
	RegisteredHash      string    `json:"registered_hash"`
}

// ProveFingerprintMatch 生成指纹匹配证明
func (s *Service) ProveFingerprintMatch(ctx context.Context, req *FingerprintMatchRequest) (*Proof, error) {
	// 转换特征为 big.Int
	var statFeatures, semFeatures [8]*big.Int
	for i := 0; i < 8; i++ {
		if i < len(req.StatisticalFeatures) {
			statFeatures[i] = floatToBigInt(req.StatisticalFeatures[i])
		} else {
			statFeatures[i] = big.NewInt(0)
		}
		if i < len(req.SemanticFeatures) {
			semFeatures[i] = floatToBigInt(req.SemanticFeatures[i])
		} else {
			semFeatures[i] = big.NewInt(0)
		}
	}

	modelID := stringToBigInt(req.ModelID)
	registeredHash := hexToBigInt(req.RegisteredHash)

	proof, err := s.prover.ProveFingerprintVerification(
		statFeatures,
		semFeatures,
		modelID,
		registeredHash,
		big.NewInt(0), // threshold
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	s.cacheProof(proof)
	return proof, nil
}

// ProveMerkleInclusion 证明 Merkle 包含
type MerkleInclusionRequest struct {
	LeafHash  string   `json:"leaf_hash"`
	Path      []string `json:"path"`
	PathIndex []int    `json:"path_index"`
	RootHash  string   `json:"root_hash"`
}

// ProveMerkleInclusion 生成 Merkle 包含证明
func (s *Service) ProveMerkleInclusion(ctx context.Context, req *MerkleInclusionRequest) (*Proof, error) {
	leaf := hexToBigInt(req.LeafHash)
	root := hexToBigInt(req.RootHash)

	path := make([]*big.Int, len(req.Path))
	for i, p := range req.Path {
		path[i] = hexToBigInt(p)
	}

	proof, err := s.prover.ProveMerkleProof(leaf, path, req.PathIndex, root)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	s.cacheProof(proof)
	return proof, nil
}

// VerifyProof 验证证明
func (s *Service) VerifyProof(ctx context.Context, proof *Proof) (*VerificationResult, error) {
	result := s.verifier.VerifyAndReport(proof)
	result.VerifiedAt = time.Now().Unix()
	return result, nil
}

// VerifyProofByHash 通过哈希验证缓存的证明
func (s *Service) VerifyProofByHash(ctx context.Context, proofHash string) (*VerificationResult, error) {
	s.cacheMux.RLock()
	proof, exists := s.cache[proofHash]
	s.cacheMux.RUnlock()

	if !exists {
		return nil, fmt.Errorf("proof not found in cache: %s", proofHash)
	}

	return s.VerifyProof(ctx, proof)
}

// GetProof 获取缓存的证明
func (s *Service) GetProof(proofHash string) (*Proof, error) {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	proof, exists := s.cache[proofHash]
	if !exists {
		return nil, fmt.Errorf("proof not found: %s", proofHash)
	}

	return proof, nil
}

// ExportVerifyingKeys 导出所有验证密钥
func (s *Service) ExportVerifyingKeys() (map[ProofType][]byte, error) {
	keys := make(map[ProofType][]byte)
	for _, pt := range s.verifier.ListVerifyingKeys() {
		vkData, err := s.prover.ExportVerifyingKey(pt)
		if err != nil {
			continue
		}
		keys[pt] = vkData
	}
	return keys, nil
}

// cacheProof 缓存证明
func (s *Service) cacheProof(proof *Proof) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.cache[proof.ProofHash] = proof
}

// 辅助函数

func splitIntoChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, chunkSize)
		copy(chunk, data[i:end])
		chunks = append(chunks, chunk)
	}
	// 确保至少有 16 个块
	for len(chunks) < 16 {
		chunks = append(chunks, make([]byte, chunkSize))
	}
	return chunks[:16]
}

func stringToBigInt(s string) *big.Int {
	h := sha256.Sum256([]byte(s))
	return new(big.Int).SetBytes(h[:])
}

func hexToBigInt(hexStr string) *big.Int {
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	bytes, _ := hex.DecodeString(hexStr)
	return new(big.Int).SetBytes(bytes)
}

func floatToBigInt(f float64) *big.Int {
	// 乘以 1e9 保留精度
	return big.NewInt(int64(f * 1e9))
}

func computeSimpleHash(data []byte) *big.Int {
	result := big.NewInt(0)
	for i, b := range data {
		result.Add(result, big.NewInt(int64(b)*int64(i+1)))
	}
	return result
}

func computePublicHash(contentHash []byte, ownerID *big.Int, timestamp int64) *big.Int {
	h := sha256.New()
	h.Write(contentHash)
	h.Write(ownerID.Bytes())
	h.Write(big.NewInt(timestamp).Bytes())
	return new(big.Int).SetBytes(h.Sum(nil))
}
