package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyProof(t *testing.T) {
	verifier := NewVerifier(false)

	// Test valid proof structure
	proof := &MinimalDisclosureProof{
		SchemaVersion: "0.1",
		CertID:        "cert_test123",
		RootHash:      "sha256:abc123def456",
		MerkleProofs: []EventMerkleProof{
			{
				EventIndex: 0,
				EventHash:  "sha256:leaf0",
				ProofPath: []ProofNode{
					{Hash: "sha256:sibling", Position: "right"},
				},
			},
		},
		TimeProof: &TimeProof{
			ProofType: "local",
			Timestamp: "2024-01-01T00:00:00Z",
		},
		AnchorProof: &AnchorProof{
			AnchorType:      "local",
			AnchorID:        "anchor_123",
			AnchorTimestamp: "2024-01-01T00:00:00Z",
		},
	}

	result := verifier.VerifyProof(proof)

	// Should have checks
	if len(result.Checks) == 0 {
		t.Error("VerifyProof() should return checks")
	}

	// Cert ID should be set
	if result.CertID != proof.CertID {
		t.Errorf("CertID = %v, want %v", result.CertID, proof.CertID)
	}

	// Root hash should be set
	if result.RootHash != proof.RootHash {
		t.Errorf("RootHash = %v, want %v", result.RootHash, proof.RootHash)
	}
}

func TestVerifyProofMissingSchema(t *testing.T) {
	verifier := NewVerifier(false)

	proof := &MinimalDisclosureProof{
		SchemaVersion: "",
		RootHash:      "sha256:abc",
	}

	result := verifier.VerifyProof(proof)

	// Should fail schema check
	var schemaCheck *CheckResult
	for _, check := range result.Checks {
		if check.Name == "Schema Version" {
			schemaCheck = &check
			break
		}
	}

	if schemaCheck == nil {
		t.Fatal("Schema Version check not found")
	}

	if schemaCheck.Passed {
		t.Error("Schema Version check should fail for empty schema")
	}
}

func TestVerifyProofInvalidRootHash(t *testing.T) {
	verifier := NewVerifier(false)

	proof := &MinimalDisclosureProof{
		SchemaVersion: "0.1",
		RootHash:      "invalid_hash", // Not sha256: prefix
	}

	result := verifier.VerifyProof(proof)

	// Should fail root hash check
	var rootHashCheck *CheckResult
	for _, check := range result.Checks {
		if check.Name == "Root Hash Format" {
			rootHashCheck = &check
			break
		}
	}

	if rootHashCheck == nil {
		t.Fatal("Root Hash Format check not found")
	}

	if rootHashCheck.Passed {
		t.Error("Root Hash Format check should fail for invalid hash")
	}
}

func TestVerifyCertificate(t *testing.T) {
	verifier := NewVerifier(false)

	cert := &Certificate{
		CertID:        "cert_test",
		CertVersion:   "1.0",
		SchemaVersion: "0.1",
		TraceID:       "trc_test",
		EventHashes: []string{
			"sha256:event1",
			"sha256:event2",
		},
		RootHash: "sha256:root123",
		TimeProof: &TimeProof{
			ProofType: "local",
			Timestamp: "2024-01-01T00:00:00Z",
		},
		AnchorProof: &AnchorProof{
			AnchorType:      "local",
			AnchorID:        "anchor_test",
			AnchorTimestamp: "2024-01-01T00:00:00Z",
		},
		Metadata: &Metadata{
			TenantID:      "default",
			CreatedAt:     "2024-01-01T00:00:00Z",
			CreatedBy:     "test",
			EvidenceLevel: "L1",
		},
	}

	result := verifier.VerifyCertificate(cert)

	// Should have checks
	if len(result.Checks) == 0 {
		t.Error("VerifyCertificate() should return checks")
	}

	// Event count should be set
	if result.EventCount != 2 {
		t.Errorf("EventCount = %v, want 2", result.EventCount)
	}
}

func TestVerifyCertificateEmpty(t *testing.T) {
	verifier := NewVerifier(false)

	cert := &Certificate{
		CertID:      "",
		EventHashes: []string{},
		RootHash:    "",
	}

	result := verifier.VerifyCertificate(cert)

	// Should fail
	if result.Valid {
		t.Error("VerifyCertificate() should fail for empty certificate")
	}
}

func TestHashPair(t *testing.T) {
	verifier := NewVerifier(false)

	hash := verifier.hashPair("sha256:left", "sha256:right")

	// Should start with sha256:
	if len(hash) < 7 || hash[:7] != "sha256:" {
		t.Errorf("hashPair() should return sha256: prefixed hash, got %v", hash)
	}

	// Should be consistent
	hash2 := verifier.hashPair("sha256:left", "sha256:right")
	if hash != hash2 {
		t.Error("hashPair() should be consistent")
	}

	// Order matters
	hashReversed := verifier.hashPair("sha256:right", "sha256:left")
	if hash == hashReversed {
		t.Error("hashPair() should produce different results for different order")
	}
}

func TestBuildMerkleRoot(t *testing.T) {
	verifier := NewVerifier(false)

	tests := []struct {
		name   string
		leaves []string
	}{
		{"single leaf", []string{"sha256:a"}},
		{"two leaves", []string{"sha256:a", "sha256:b"}},
		{"four leaves", []string{"sha256:a", "sha256:b", "sha256:c", "sha256:d"}},
		{"odd leaves", []string{"sha256:a", "sha256:b", "sha256:c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := verifier.buildMerkleRoot(tt.leaves)
			if root == "" {
				t.Error("buildMerkleRoot() returned empty string")
			}
			if len(root) < 7 || root[:7] != "sha256:" {
				t.Errorf("buildMerkleRoot() should return sha256: prefixed hash, got %v", root)
			}
		})
	}
}

func TestBuildMerkleRootEmpty(t *testing.T) {
	verifier := NewVerifier(false)

	root := verifier.buildMerkleRoot([]string{})
	if root != "" {
		t.Errorf("buildMerkleRoot() should return empty string for empty leaves, got %v", root)
	}
}

func TestBuildMerkleRootConsistency(t *testing.T) {
	verifier := NewVerifier(false)

	leaves := []string{"sha256:a", "sha256:b", "sha256:c"}

	root1 := verifier.buildMerkleRoot(leaves)
	root2 := verifier.buildMerkleRoot(leaves)

	if root1 != root2 {
		t.Error("buildMerkleRoot() should be consistent")
	}
}

func TestVerifyMerkleProof(t *testing.T) {
	verifier := NewVerifier(false)

	// Build a simple tree
	leaves := []string{"sha256:a", "sha256:b"}
	root := verifier.buildMerkleRoot(leaves)

	// Create proof for first leaf
	path := []ProofNode{
		{Hash: "sha256:b", Position: "right"},
	}

	// The proof should match
	calculatedRoot := verifier.hashPair("sha256:a", "sha256:b")

	valid := verifier.verifyMerkleProof("sha256:a", path, calculatedRoot)
	if !valid {
		t.Errorf("verifyMerkleProof() should return true, root=%v, calculated=%v", root, calculatedRoot)
	}
}

func TestVerifyMerkleProofInvalid(t *testing.T) {
	verifier := NewVerifier(false)

	path := []ProofNode{
		{Hash: "sha256:sibling", Position: "right"},
	}

	// Wrong root should fail
	valid := verifier.verifyMerkleProof("sha256:leaf", path, "sha256:wrong_root")
	if valid {
		t.Error("verifyMerkleProof() should return false for wrong root")
	}
}

func TestVerifyProofFile(t *testing.T) {
	verifier := NewVerifier(false)

	// Create temp file
	tmpDir := t.TempDir()
	proofFile := filepath.Join(tmpDir, "proof.json")

	proofJSON := `{
		"schema_version": "0.1",
		"cert_id": "cert_file_test",
		"root_hash": "sha256:abc123",
		"merkle_proofs": [],
		"disclosed_events": []
	}`

	if err := os.WriteFile(proofFile, []byte(proofJSON), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := verifier.VerifyProofFile(proofFile)
	if err != nil {
		t.Fatalf("VerifyProofFile() error = %v", err)
	}

	if result.CertID != "cert_file_test" {
		t.Errorf("CertID = %v, want cert_file_test", result.CertID)
	}
}

func TestVerifyProofFileNotFound(t *testing.T) {
	verifier := NewVerifier(false)

	_, err := verifier.VerifyProofFile("/nonexistent/path/proof.json")
	if err == nil {
		t.Error("VerifyProofFile() should return error for nonexistent file")
	}
}

func TestVerifyProofFileInvalidJSON(t *testing.T) {
	verifier := NewVerifier(false)

	// Create temp file with invalid JSON
	tmpDir := t.TempDir()
	proofFile := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(proofFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := verifier.VerifyProofFile(proofFile)
	if err == nil {
		t.Error("VerifyProofFile() should return error for invalid JSON")
	}
}

func TestVerifyCertFile(t *testing.T) {
	verifier := NewVerifier(false)

	// Create temp file
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.json")

	certJSON := `{
		"cert_id": "cert_file_test",
		"cert_version": "1.0",
		"schema_version": "0.1",
		"trace_id": "trc_test",
		"event_hashes": ["sha256:a", "sha256:b"],
		"root_hash": "sha256:root123",
		"metadata": {
			"tenant_id": "default",
			"created_at": "2024-01-01T00:00:00Z",
			"created_by": "test",
			"evidence_level": "L1"
		}
	}`

	if err := os.WriteFile(certFile, []byte(certJSON), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := verifier.VerifyCertFile(certFile)
	if err != nil {
		t.Fatalf("VerifyCertFile() error = %v", err)
	}

	if result.EventCount != 2 {
		t.Errorf("EventCount = %v, want 2", result.EventCount)
	}
}

func BenchmarkVerifyMerkleProof(b *testing.B) {
	verifier := NewVerifier(false)

	path := make([]ProofNode, 20) // Simulate tree height of 20
	for i := range path {
		path[i] = ProofNode{Hash: "sha256:benchmark_hash", Position: "right"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.verifyMerkleProof("sha256:leaf", path, "sha256:root")
	}
}

func BenchmarkBuildMerkleRoot(b *testing.B) {
	verifier := NewVerifier(false)

	leaves := make([]string, 1000)
	for i := range leaves {
		leaves[i] = "sha256:benchmark_leaf"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.buildMerkleRoot(leaves)
	}
}
