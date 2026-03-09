package zkp

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestNewProver(t *testing.T) {
	p := NewProver()
	if p == nil {
		t.Fatal("NewProver returned nil")
	}
	if p.keys == nil {
		t.Error("Prover keys map should be initialized")
	}
}

func TestNewVerifier(t *testing.T) {
	v := NewVerifier()
	if v == nil {
		t.Fatal("NewVerifier returned nil")
	}
	if v.verifyingKeys == nil {
		t.Error("Verifier keys map should be initialized")
	}
}

func TestProofTypes(t *testing.T) {
	// Verify proof type constants
	types := []ProofType{
		ProofTypeHashPreimage,
		ProofTypeContentOwnership,
		ProofTypeFingerprintVerify,
		ProofTypeMerkleProof,
		ProofTypeSelectiveDisclosure,
	}

	for _, pt := range types {
		if pt == "" {
			t.Error("Proof type should not be empty")
		}
	}

	if ProofTypeHashPreimage != "hash_preimage" {
		t.Errorf("Expected hash_preimage, got %s", ProofTypeHashPreimage)
	}
}

func TestProofToJSON(t *testing.T) {
	proof := &Proof{
		Type:       ProofTypeHashPreimage,
		ProofData:  []byte("test_proof_data"),
		PublicData: []byte("test_public_data"),
		ProofHash:  "test_hash_123",
	}

	data, err := proof.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("ToJSON returned empty data")
	}

	// Verify JSON structure
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if decoded["type"] != "hash_preimage" {
		t.Errorf("Expected type hash_preimage, got %v", decoded["type"])
	}
}

func TestProofFromJSON(t *testing.T) {
	original := &Proof{
		Type:       ProofTypeContentOwnership,
		ProofData:  []byte("proof_data"),
		PublicData: []byte("public_data"),
		ProofHash:  "hash_456",
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	decoded, err := ProofFromJSON(data)
	if err != nil {
		t.Fatalf("ProofFromJSON failed: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Expected type %s, got %s", original.Type, decoded.Type)
	}
	if decoded.ProofHash != original.ProofHash {
		t.Errorf("Expected hash %s, got %s", original.ProofHash, decoded.ProofHash)
	}
}

func TestProofFromJSONInvalid(t *testing.T) {
	_, err := ProofFromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestComputeProofHash(t *testing.T) {
	data := []byte("test proof data")
	hash := computeProofHash(data)

	if hash == "" {
		t.Error("computeProofHash returned empty string")
	}

	// Verify deterministic
	hash2 := computeProofHash(data)
	if hash != hash2 {
		t.Error("computeProofHash should be deterministic")
	}

	// Different input should give different hash
	hash3 := computeProofHash([]byte("different data"))
	if hash == hash3 {
		t.Error("Different inputs should give different hashes")
	}
}

func TestVerifierHasVerifyingKey(t *testing.T) {
	v := NewVerifier()

	// Initially no keys
	if v.HasVerifyingKey(ProofTypeHashPreimage) {
		t.Error("Should not have key initially")
	}
}

func TestVerifierListVerifyingKeys(t *testing.T) {
	v := NewVerifier()

	keys := v.ListVerifyingKeys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

func TestVerificationResult(t *testing.T) {
	result := &VerificationResult{
		Valid:      true,
		ProofType:  ProofTypeHashPreimage,
		ProofHash:  "test_hash",
		VerifiedAt: 1234567890,
	}

	if !result.Valid {
		t.Error("Expected valid result")
	}
	if result.ProofType != ProofTypeHashPreimage {
		t.Errorf("Expected proof type %s, got %s", ProofTypeHashPreimage, result.ProofType)
	}
}

func TestVerificationResultWithError(t *testing.T) {
	result := &VerificationResult{
		Valid:    false,
		ErrorMsg: "test error message",
	}

	if result.Valid {
		t.Error("Expected invalid result")
	}
	if result.ErrorMsg == "" {
		t.Error("Expected error message")
	}
}

func TestProverSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping setup test in short mode (computationally expensive)")
	}

	p := NewProver()

	// Setup should work for valid proof type
	err := p.Setup(ProofTypeHashPreimage)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Second setup should be no-op
	err = p.Setup(ProofTypeHashPreimage)
	if err != nil {
		t.Fatalf("Second setup failed: %v", err)
	}
}

func TestProverSetupUnknownType(t *testing.T) {
	p := NewProver()

	err := p.Setup(ProofType("unknown_type"))
	if err == nil {
		t.Error("Expected error for unknown proof type")
	}
}

func TestGetVerifyingKeyNotSetup(t *testing.T) {
	p := NewProver()

	_, err := p.GetVerifyingKey(ProofTypeHashPreimage)
	if err == nil {
		t.Error("Expected error when key not setup")
	}
}

// Integration test: Full proof generation and verification cycle
func TestHashPreimageProofCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full proof cycle test in short mode")
	}

	p := NewProver()
	v := NewVerifier()

	// Create test data
	preimage := []byte("test preimage data")

	// Compute expected hash using the same logic as the circuit
	var hash big.Int
	for i := 0; i < 8; i++ {
		if i*32 < len(preimage) {
			end := (i + 1) * 32
			if end > len(preimage) {
				end = len(preimage)
			}
			chunk := preimage[i*32 : end]
			chunkVal := new(big.Int).SetBytes(chunk)
			contrib := new(big.Int).Mul(chunkVal, big.NewInt(int64(i+1)))
			hash.Add(&hash, contrib)
		}
	}

	// Generate proof
	proof, err := p.ProveHashPreimage(preimage, &hash)
	if err != nil {
		t.Fatalf("ProveHashPreimage failed: %v", err)
	}

	if proof == nil {
		t.Fatal("Proof should not be nil")
	}
	if proof.Type != ProofTypeHashPreimage {
		t.Errorf("Expected proof type %s, got %s", ProofTypeHashPreimage, proof.Type)
	}
	if len(proof.ProofData) == 0 {
		t.Error("ProofData should not be empty")
	}
	if len(proof.PublicData) == 0 {
		t.Error("PublicData should not be empty")
	}

	// Export and import verifying key
	vkData, err := p.ExportVerifyingKey(ProofTypeHashPreimage)
	if err != nil {
		t.Fatalf("ExportVerifyingKey failed: %v", err)
	}

	err = v.ImportVerifyingKey(ProofTypeHashPreimage, vkData)
	if err != nil {
		t.Fatalf("ImportVerifyingKey failed: %v", err)
	}

	// Verify proof
	valid, err := v.Verify(proof)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !valid {
		t.Error("Proof should be valid")
	}
}

func TestBatchVerifyEmpty(t *testing.T) {
	v := NewVerifier()

	results, err := v.BatchVerify([]*Proof{})
	if err != nil {
		t.Fatalf("BatchVerify failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestVerifyAndReport(t *testing.T) {
	v := NewVerifier()

	proof := &Proof{
		Type:       ProofTypeHashPreimage,
		ProofData:  []byte("fake proof"),
		PublicData: []byte("fake public"),
		ProofHash:  "test_hash",
	}

	result := v.VerifyAndReport(proof)

	if result == nil {
		t.Fatal("VerifyAndReport returned nil")
	}
	if result.ProofType != ProofTypeHashPreimage {
		t.Errorf("Expected proof type %s, got %s", ProofTypeHashPreimage, result.ProofType)
	}
	if result.ProofHash != "test_hash" {
		t.Errorf("Expected proof hash test_hash, got %s", result.ProofHash)
	}
	// Should have error since no verifying key
	if result.ErrorMsg == "" {
		t.Error("Expected error message since no verifying key")
	}
}

func TestVerifyWithoutKey(t *testing.T) {
	v := NewVerifier()

	proof := &Proof{
		Type:       ProofTypeHashPreimage,
		ProofData:  []byte("test"),
		PublicData: []byte("test"),
	}

	_, err := v.Verify(proof)
	if err == nil {
		t.Error("Expected error when verifying without key")
	}
}
