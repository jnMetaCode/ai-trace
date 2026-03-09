package zkp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// ProofType 证明类型
type ProofType string

const (
	ProofTypeHashPreimage       ProofType = "hash_preimage"
	ProofTypeContentOwnership   ProofType = "content_ownership"
	ProofTypeFingerprintVerify  ProofType = "fingerprint_verify"
	ProofTypeMerkleProof        ProofType = "merkle_proof"
	ProofTypeSelectiveDisclosure ProofType = "selective_disclosure"
)

// Proof ZK 证明结构
type Proof struct {
	Type       ProofType `json:"type"`
	ProofData  []byte    `json:"proof_data"`
	PublicData []byte    `json:"public_data"`
	ProofHash  string    `json:"proof_hash"`
}

// ProverKeys 证明密钥
type ProverKeys struct {
	ProvingKey   groth16.ProvingKey
	VerifyingKey groth16.VerifyingKey
	R1CS         constraint.ConstraintSystem
}

// Prover ZK 证明生成器
type Prover struct {
	keys    map[ProofType]*ProverKeys
	keysMux sync.RWMutex
	curve   ecc.ID
}

// NewProver 创建证明生成器
func NewProver() *Prover {
	return &Prover{
		keys:  make(map[ProofType]*ProverKeys),
		curve: ecc.BN254, // 使用 BN254 曲线
	}
}

// Setup 设置电路（生成密钥）
func (p *Prover) Setup(proofType ProofType) error {
	p.keysMux.Lock()
	defer p.keysMux.Unlock()

	if _, exists := p.keys[proofType]; exists {
		return nil // 已经设置过
	}

	var circuit frontend.Circuit
	switch proofType {
	case ProofTypeHashPreimage:
		circuit = &HashPreimageCircuit{}
	case ProofTypeContentOwnership:
		circuit = &ContentOwnershipCircuit{}
	case ProofTypeFingerprintVerify:
		circuit = &FingerprintVerificationCircuit{}
	case ProofTypeMerkleProof:
		circuit = &MerkleProofCircuit{}
	case ProofTypeSelectiveDisclosure:
		circuit = &SelectiveDisclosureCircuit{}
	default:
		return fmt.Errorf("unknown proof type: %s", proofType)
	}

	// 编译电路
	r1cs, err := frontend.Compile(p.curve.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		return fmt.Errorf("failed to compile circuit: %w", err)
	}

	// 生成密钥
	pk, vk, err := groth16.Setup(r1cs)
	if err != nil {
		return fmt.Errorf("failed to setup keys: %w", err)
	}

	p.keys[proofType] = &ProverKeys{
		ProvingKey:   pk,
		VerifyingKey: vk,
		R1CS:         r1cs,
	}

	return nil
}

// ProveHashPreimage 生成哈希原像证明
func (p *Prover) ProveHashPreimage(preimage []byte, hash *big.Int) (*Proof, error) {
	if err := p.Setup(ProofTypeHashPreimage); err != nil {
		return nil, err
	}

	p.keysMux.RLock()
	keys := p.keys[ProofTypeHashPreimage]
	p.keysMux.RUnlock()

	// 准备 witness
	var preimageVars [8]frontend.Variable
	for i := 0; i < 8; i++ {
		if i*32 < len(preimage) {
			end := (i + 1) * 32
			if end > len(preimage) {
				end = len(preimage)
			}
			chunk := preimage[i*32 : end]
			preimageVars[i] = new(big.Int).SetBytes(chunk)
		} else {
			preimageVars[i] = big.NewInt(0)
		}
	}

	assignment := &HashPreimageCircuit{
		Preimage:       preimageVars,
		PreimageLength: len(preimage),
		Hash:           hash,
	}

	// 生成 witness
	witness, err := frontend.NewWitness(assignment, p.curve.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	// 生成证明
	proof, err := groth16.Prove(keys.R1CS, keys.ProvingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	// 序列化证明
	var proofBuf bytes.Buffer
	_, err = proof.WriteTo(&proofBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	// 获取公开输入
	publicWitness, err := witness.Public()
	if err != nil {
		return nil, fmt.Errorf("failed to get public witness: %w", err)
	}

	var publicBuf bytes.Buffer
	_, err = publicWitness.WriteTo(&publicBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize public witness: %w", err)
	}

	return &Proof{
		Type:       ProofTypeHashPreimage,
		ProofData:  proofBuf.Bytes(),
		PublicData: publicBuf.Bytes(),
		ProofHash:  computeProofHash(proofBuf.Bytes()),
	}, nil
}

// ProveContentOwnership 生成内容所有权证明
func (p *Prover) ProveContentOwnership(
	contentChunks [][]byte,
	ownerID *big.Int,
	timestamp int64,
	publicHash *big.Int,
	beforeTimestamp int64,
) (*Proof, error) {
	if err := p.Setup(ProofTypeContentOwnership); err != nil {
		return nil, err
	}

	p.keysMux.RLock()
	keys := p.keys[ProofTypeContentOwnership]
	p.keysMux.RUnlock()

	// 准备 content chunks
	var chunkVars [16]frontend.Variable
	contentHash := big.NewInt(0)
	for i := 0; i < 16; i++ {
		if i < len(contentChunks) {
			chunkVal := new(big.Int).SetBytes(contentChunks[i])
			chunkVars[i] = chunkVal
			// 计算内容哈希
			contentHash.Add(contentHash, new(big.Int).Mul(chunkVal, big.NewInt(int64(i+1))))
		} else {
			chunkVars[i] = big.NewInt(0)
		}
	}

	assignment := &ContentOwnershipCircuit{
		ContentHash:     contentHash,
		ContentChunks:   chunkVars,
		Timestamp:       big.NewInt(timestamp),
		OwnerID:         ownerID,
		PublicHash:      publicHash,
		BeforeTimestamp: big.NewInt(beforeTimestamp),
	}

	witness, err := frontend.NewWitness(assignment, p.curve.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	proof, err := groth16.Prove(keys.R1CS, keys.ProvingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	var proofBuf bytes.Buffer
	_, _ = proof.WriteTo(&proofBuf)

	publicWitness, _ := witness.Public()
	var publicBuf bytes.Buffer
	_, _ = publicWitness.WriteTo(&publicBuf)

	return &Proof{
		Type:       ProofTypeContentOwnership,
		ProofData:  proofBuf.Bytes(),
		PublicData: publicBuf.Bytes(),
		ProofHash:  computeProofHash(proofBuf.Bytes()),
	}, nil
}

// ProveFingerprintVerification 生成指纹验证证明
func (p *Prover) ProveFingerprintVerification(
	statisticalFeatures [8]*big.Int,
	semanticFeatures [8]*big.Int,
	modelID *big.Int,
	registeredFingerprint *big.Int,
	threshold *big.Int,
) (*Proof, error) {
	if err := p.Setup(ProofTypeFingerprintVerify); err != nil {
		return nil, err
	}

	p.keysMux.RLock()
	keys := p.keys[ProofTypeFingerprintVerify]
	p.keysMux.RUnlock()

	var statVars, semVars [8]frontend.Variable
	for i := 0; i < 8; i++ {
		if statisticalFeatures[i] != nil {
			statVars[i] = statisticalFeatures[i]
		} else {
			statVars[i] = big.NewInt(0)
		}
		if semanticFeatures[i] != nil {
			semVars[i] = semanticFeatures[i]
		} else {
			semVars[i] = big.NewInt(0)
		}
	}

	assignment := &FingerprintVerificationCircuit{
		StatisticalFeatures:   statVars,
		SemanticFeatures:      semVars,
		ModelID:               modelID,
		RegisteredFingerprint: registeredFingerprint,
		Threshold:             threshold,
	}

	witness, err := frontend.NewWitness(assignment, p.curve.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	proof, err := groth16.Prove(keys.R1CS, keys.ProvingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	var proofBuf bytes.Buffer
	_, _ = proof.WriteTo(&proofBuf)

	publicWitness, _ := witness.Public()
	var publicBuf bytes.Buffer
	_, _ = publicWitness.WriteTo(&publicBuf)

	return &Proof{
		Type:       ProofTypeFingerprintVerify,
		ProofData:  proofBuf.Bytes(),
		PublicData: publicBuf.Bytes(),
		ProofHash:  computeProofHash(proofBuf.Bytes()),
	}, nil
}

// ProveMerkleProof 生成 Merkle 证明
func (p *Prover) ProveMerkleProof(
	leaf *big.Int,
	path []*big.Int,
	pathIndex []int,
	root *big.Int,
) (*Proof, error) {
	if err := p.Setup(ProofTypeMerkleProof); err != nil {
		return nil, err
	}

	p.keysMux.RLock()
	keys := p.keys[ProofTypeMerkleProof]
	p.keysMux.RUnlock()

	var pathVars, pathIndexVars [32]frontend.Variable
	depth := len(path)
	for i := 0; i < 32; i++ {
		if i < len(path) && path[i] != nil {
			pathVars[i] = path[i]
			pathIndexVars[i] = pathIndex[i]
		} else {
			pathVars[i] = big.NewInt(0)
			pathIndexVars[i] = big.NewInt(0)
		}
	}

	assignment := &MerkleProofCircuit{
		Leaf:      leaf,
		Path:      pathVars,
		PathIndex: pathIndexVars,
		Depth:     depth,
		Root:      root,
	}

	witness, err := frontend.NewWitness(assignment, p.curve.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	proof, err := groth16.Prove(keys.R1CS, keys.ProvingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	var proofBuf bytes.Buffer
	_, _ = proof.WriteTo(&proofBuf)

	publicWitness, _ := witness.Public()
	var publicBuf bytes.Buffer
	_, _ = publicWitness.WriteTo(&publicBuf)

	return &Proof{
		Type:       ProofTypeMerkleProof,
		ProofData:  proofBuf.Bytes(),
		PublicData: publicBuf.Bytes(),
		ProofHash:  computeProofHash(proofBuf.Bytes()),
	}, nil
}

// GetVerifyingKey 获取验证密钥
func (p *Prover) GetVerifyingKey(proofType ProofType) (groth16.VerifyingKey, error) {
	p.keysMux.RLock()
	defer p.keysMux.RUnlock()

	keys, exists := p.keys[proofType]
	if !exists {
		return nil, fmt.Errorf("keys not found for proof type: %s", proofType)
	}

	return keys.VerifyingKey, nil
}

// ExportVerifyingKey 导出验证密钥（用于分发）
func (p *Prover) ExportVerifyingKey(proofType ProofType) ([]byte, error) {
	vk, err := p.GetVerifyingKey(proofType)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = vk.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to export verifying key: %w", err)
	}

	return buf.Bytes(), nil
}

// computeProofHash 计算证明哈希
func computeProofHash(proofData []byte) string {
	// 使用简单的哈希（实际应使用 SHA256）
	hash := big.NewInt(0)
	for i, b := range proofData {
		hash.Add(hash, big.NewInt(int64(b)*int64(i+1)))
	}
	return fmt.Sprintf("%x", hash)
}

// ToJSON 将证明转换为 JSON
func (p *Proof) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON 从 JSON 解析证明
func ProofFromJSON(data []byte) (*Proof, error) {
	var proof Proof
	if err := json.Unmarshal(data, &proof); err != nil {
		return nil, err
	}
	return &proof, nil
}
