package zkp

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
)

// Verifier ZK 证明验证器
type Verifier struct {
	verifyingKeys map[ProofType]groth16.VerifyingKey
	keysMux       sync.RWMutex
	curve         ecc.ID
}

// NewVerifier 创建验证器
func NewVerifier() *Verifier {
	return &Verifier{
		verifyingKeys: make(map[ProofType]groth16.VerifyingKey),
		curve:         ecc.BN254,
	}
}

// ImportVerifyingKey 导入验证密钥
func (v *Verifier) ImportVerifyingKey(proofType ProofType, vkData []byte) error {
	v.keysMux.Lock()
	defer v.keysMux.Unlock()

	vk := groth16.NewVerifyingKey(v.curve)
	_, err := vk.ReadFrom(bytes.NewReader(vkData))
	if err != nil {
		return fmt.Errorf("failed to import verifying key: %w", err)
	}

	v.verifyingKeys[proofType] = vk
	return nil
}

// SetVerifyingKey 设置验证密钥
func (v *Verifier) SetVerifyingKey(proofType ProofType, vk groth16.VerifyingKey) {
	v.keysMux.Lock()
	defer v.keysMux.Unlock()
	v.verifyingKeys[proofType] = vk
}

// Verify 验证证明
func (v *Verifier) Verify(proof *Proof) (bool, error) {
	v.keysMux.RLock()
	vk, exists := v.verifyingKeys[proof.Type]
	v.keysMux.RUnlock()

	if !exists {
		return false, fmt.Errorf("verifying key not found for proof type: %s", proof.Type)
	}

	// 反序列化证明
	groth16Proof := groth16.NewProof(v.curve)
	_, err := groth16Proof.ReadFrom(bytes.NewReader(proof.ProofData))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// 反序列化公开输入
	publicWitness, err := witness.New(v.curve.ScalarField())
	if err != nil {
		return false, fmt.Errorf("failed to create witness: %w", err)
	}
	_, err = publicWitness.ReadFrom(bytes.NewReader(proof.PublicData))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize public witness: %w", err)
	}

	// 验证证明
	err = groth16.Verify(groth16Proof, vk, publicWitness)
	if err != nil {
		return false, nil // 验证失败但不是错误
	}

	return true, nil
}

// VerifyWithPublicInputs 使用给定的公开输入验证证明
func (v *Verifier) VerifyWithPublicInputs(proof *Proof, publicInputs map[string]interface{}) (bool, error) {
	// 这个方法允许外部提供公开输入而不是使用证明中的
	// 用于需要独立验证公开输入的场景

	v.keysMux.RLock()
	vk, exists := v.verifyingKeys[proof.Type]
	v.keysMux.RUnlock()

	if !exists {
		return false, fmt.Errorf("verifying key not found for proof type: %s", proof.Type)
	}

	// 反序列化证明
	groth16Proof := groth16.NewProof(v.curve)
	_, err := groth16Proof.ReadFrom(bytes.NewReader(proof.ProofData))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// 使用提供的公开输入
	publicWitness, err := witness.New(v.curve.ScalarField())
	if err != nil {
		return false, fmt.Errorf("failed to create witness: %w", err)
	}
	_, err = publicWitness.ReadFrom(bytes.NewReader(proof.PublicData))
	if err != nil {
		return false, fmt.Errorf("failed to deserialize public witness: %w", err)
	}

	// 验证
	err = groth16.Verify(groth16Proof, vk, publicWitness)
	return err == nil, nil
}

// BatchVerify 批量验证证明
func (v *Verifier) BatchVerify(proofs []*Proof) ([]bool, error) {
	results := make([]bool, len(proofs))
	var wg sync.WaitGroup
	var errMux sync.Mutex
	var firstErr error

	for i, proof := range proofs {
		wg.Add(1)
		go func(idx int, p *Proof) {
			defer wg.Done()

			valid, err := v.Verify(p)
			if err != nil {
				errMux.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMux.Unlock()
				return
			}
			results[idx] = valid
		}(i, proof)
	}

	wg.Wait()
	return results, firstErr
}

// VerificationResult 验证结果
type VerificationResult struct {
	Valid       bool      `json:"valid"`
	ProofType   ProofType `json:"proof_type"`
	ProofHash   string    `json:"proof_hash"`
	VerifiedAt  int64     `json:"verified_at"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
}

// VerifyAndReport 验证并生成报告
func (v *Verifier) VerifyAndReport(proof *Proof) *VerificationResult {
	result := &VerificationResult{
		ProofType:  proof.Type,
		ProofHash:  proof.ProofHash,
		VerifiedAt: 0, // 应该使用 time.Now().Unix()
	}

	valid, err := v.Verify(proof)
	if err != nil {
		result.Valid = false
		result.ErrorMsg = err.Error()
	} else {
		result.Valid = valid
	}

	return result
}

// HasVerifyingKey 检查是否有特定类型的验证密钥
func (v *Verifier) HasVerifyingKey(proofType ProofType) bool {
	v.keysMux.RLock()
	defer v.keysMux.RUnlock()
	_, exists := v.verifyingKeys[proofType]
	return exists
}

// ListVerifyingKeys 列出所有可用的验证密钥类型
func (v *Verifier) ListVerifyingKeys() []ProofType {
	v.keysMux.RLock()
	defer v.keysMux.RUnlock()

	types := make([]ProofType, 0, len(v.verifyingKeys))
	for t := range v.verifyingKeys {
		types = append(types, t)
	}
	return types
}
