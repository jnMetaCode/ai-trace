package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// VerifyResult verification result
type VerifyResult struct {
	Valid      bool          `json:"valid"`
	CertID     string        `json:"cert_id,omitempty"`
	RootHash   string        `json:"root_hash,omitempty"`
	EventCount int           `json:"event_count,omitempty"`
	Checks     []CheckResult `json:"checks"`
}

// CheckResult single check result
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// Certificate certificate structure
type Certificate struct {
	CertID        string      `json:"cert_id"`
	CertVersion   string      `json:"cert_version"`
	SchemaVersion string      `json:"schema_version"`
	TraceID       string      `json:"trace_id"`
	EventHashes   []string    `json:"event_hashes"`
	MerkleTree    *MerkleTree `json:"merkle_tree,omitempty"`
	RootHash      string      `json:"root_hash"`
	TimeProof     *TimeProof  `json:"time_proof,omitempty"`
	AnchorProof   *AnchorProof `json:"anchor_proof,omitempty"`
	Metadata      *Metadata   `json:"metadata,omitempty"`
}

// MerkleTree merkle tree structure
type MerkleTree struct {
	Leaves    []string   `json:"leaves"`
	Nodes     [][]string `json:"nodes"`
	Root      string     `json:"root"`
	Algorithm string     `json:"algorithm"`
}

// TimeProof time proof
type TimeProof struct {
	ProofType    string `json:"proof_type"`
	Timestamp    string `json:"timestamp"`
	TSAAuthority string `json:"tsa_authority,omitempty"`
	Signature    string `json:"signature,omitempty"`
}

// AnchorProof anchor proof
type AnchorProof struct {
	AnchorType      string           `json:"anchor_type"`
	AnchorID        string           `json:"anchor_id"`
	StorageProvider string           `json:"storage_provider,omitempty"`
	AnchorTimestamp string           `json:"anchor_timestamp"`
	Blockchain      *BlockchainProof `json:"blockchain,omitempty"`
}

// BlockchainProof blockchain proof
type BlockchainProof struct {
	ChainID     string `json:"chain_id"`
	TxHash      string `json:"tx_hash"`
	BlockHeight int64  `json:"block_height"`
}

// Metadata certificate metadata
type Metadata struct {
	TenantID      string `json:"tenant_id"`
	CreatedAt     string `json:"created_at"`
	CreatedBy     string `json:"created_by"`
	EvidenceLevel string `json:"evidence_level"`
}

// MinimalDisclosureProof minimal disclosure proof
type MinimalDisclosureProof struct {
	SchemaVersion   string             `json:"schema_version"`
	CertID          string             `json:"cert_id"`
	RootHash        string             `json:"root_hash"`
	DisclosedEvents []DisclosedEvent   `json:"disclosed_events"`
	MerkleProofs    []EventMerkleProof `json:"merkle_proofs"`
	TimeProof       *TimeProof         `json:"time_proof,omitempty"`
	AnchorProof     *AnchorProof       `json:"anchor_proof,omitempty"`
}

// DisclosedEvent disclosed event
type DisclosedEvent struct {
	EventIndex      int                    `json:"event_index"`
	EventType       string                 `json:"event_type"`
	EventHash       string                 `json:"event_hash"`
	DisclosedFields map[string]interface{} `json:"disclosed_fields"`
}

// EventMerkleProof event merkle proof
type EventMerkleProof struct {
	EventIndex int         `json:"event_index"`
	EventHash  string      `json:"event_hash"`
	ProofPath  []ProofNode `json:"proof_path"`
}

// ProofNode proof path node
type ProofNode struct {
	Hash     string `json:"hash"`
	Position string `json:"position"` // "left" or "right"
}

// Verifier verifier instance
type Verifier struct {
	verbose bool
}

// NewVerifier create new verifier
func NewVerifier(verbose bool) *Verifier {
	return &Verifier{verbose: verbose}
}

// VerifyProofFile verify proof file
func (v *Verifier) VerifyProofFile(filePath string) (*VerifyResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var proof MinimalDisclosureProof
	if err := json.Unmarshal(data, &proof); err != nil {
		return nil, fmt.Errorf("failed to parse proof JSON: %w", err)
	}

	return v.VerifyProof(&proof), nil
}

// VerifyCertFile verify certificate file
func (v *Verifier) VerifyCertFile(filePath string) (*VerifyResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(data, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse certificate JSON: %w", err)
	}

	return v.VerifyCertificate(&cert), nil
}

// VerifyProof verify minimal disclosure proof
func (v *Verifier) VerifyProof(proof *MinimalDisclosureProof) *VerifyResult {
	result := &VerifyResult{
		Valid:    true,
		CertID:   proof.CertID,
		RootHash: proof.RootHash,
		Checks:   []CheckResult{},
	}

	// Check 1: Schema version
	schemaCheck := CheckResult{Name: "Schema Version", Passed: true}
	if proof.SchemaVersion == "" {
		schemaCheck.Passed = false
		schemaCheck.Message = "Missing schema version"
		result.Valid = false
	} else {
		schemaCheck.Message = proof.SchemaVersion
	}
	result.Checks = append(result.Checks, schemaCheck)

	// Check 2: Root hash format
	rootHashCheck := CheckResult{Name: "Root Hash Format", Passed: true}
	if !strings.HasPrefix(proof.RootHash, "sha256:") {
		rootHashCheck.Passed = false
		rootHashCheck.Message = "Invalid root hash format"
		result.Valid = false
	} else {
		rootHashCheck.Message = "Valid SHA256 format"
	}
	result.Checks = append(result.Checks, rootHashCheck)

	// Check 3: Merkle proofs
	merkleCheck := CheckResult{Name: "Merkle Proofs", Passed: true}
	if len(proof.MerkleProofs) == 0 {
		merkleCheck.Passed = false
		merkleCheck.Message = "No merkle proofs provided"
		result.Valid = false
	} else {
		validProofs := 0
		for _, mp := range proof.MerkleProofs {
			if v.verifyMerkleProof(mp.EventHash, mp.ProofPath, proof.RootHash) {
				validProofs++
			}
		}
		if validProofs == len(proof.MerkleProofs) {
			merkleCheck.Message = fmt.Sprintf("All %d proofs verified", validProofs)
		} else {
			merkleCheck.Passed = false
			merkleCheck.Message = fmt.Sprintf("%d/%d proofs failed", len(proof.MerkleProofs)-validProofs, len(proof.MerkleProofs))
			result.Valid = false
		}
	}
	result.Checks = append(result.Checks, merkleCheck)

	// Check 4: Time proof
	timeCheck := CheckResult{Name: "Time Proof", Passed: true}
	if proof.TimeProof == nil {
		timeCheck.Message = "No time proof (optional)"
	} else {
		timeCheck.Message = fmt.Sprintf("%s @ %s", proof.TimeProof.ProofType, proof.TimeProof.Timestamp)
	}
	result.Checks = append(result.Checks, timeCheck)

	// Check 5: Anchor proof
	anchorCheck := CheckResult{Name: "Anchor Proof", Passed: true}
	if proof.AnchorProof == nil {
		anchorCheck.Message = "No anchor proof (optional)"
	} else {
		anchorCheck.Message = fmt.Sprintf("%s: %s", proof.AnchorProof.AnchorType, proof.AnchorProof.AnchorID)
	}
	result.Checks = append(result.Checks, anchorCheck)

	result.EventCount = len(proof.MerkleProofs)
	return result
}

// VerifyCertificate verify certificate
func (v *Verifier) VerifyCertificate(cert *Certificate) *VerifyResult {
	result := &VerifyResult{
		Valid:      true,
		CertID:     cert.CertID,
		RootHash:   cert.RootHash,
		EventCount: len(cert.EventHashes),
		Checks:     []CheckResult{},
	}

	// Check 1: Certificate ID
	certIDCheck := CheckResult{Name: "Certificate ID", Passed: true}
	if cert.CertID == "" {
		certIDCheck.Passed = false
		certIDCheck.Message = "Missing certificate ID"
		result.Valid = false
	} else {
		certIDCheck.Message = cert.CertID
	}
	result.Checks = append(result.Checks, certIDCheck)

	// Check 2: Event hashes
	eventCheck := CheckResult{Name: "Event Hashes", Passed: true}
	if len(cert.EventHashes) == 0 {
		eventCheck.Passed = false
		eventCheck.Message = "No event hashes"
		result.Valid = false
	} else {
		eventCheck.Message = fmt.Sprintf("%d events", len(cert.EventHashes))
	}
	result.Checks = append(result.Checks, eventCheck)

	// Check 3: Merkle tree integrity
	merkleCheck := CheckResult{Name: "Merkle Tree Integrity", Passed: true}
	if cert.MerkleTree != nil {
		// Rebuild and compare
		rebuiltRoot := v.buildMerkleRoot(cert.EventHashes)
		if rebuiltRoot == cert.RootHash {
			merkleCheck.Message = "Root hash verified"
		} else {
			merkleCheck.Passed = false
			merkleCheck.Message = "Root hash mismatch"
			result.Valid = false
		}
	} else {
		// Just verify root hash format
		if strings.HasPrefix(cert.RootHash, "sha256:") {
			merkleCheck.Message = "Root hash format valid"
		} else {
			merkleCheck.Passed = false
			merkleCheck.Message = "Invalid root hash format"
			result.Valid = false
		}
	}
	result.Checks = append(result.Checks, merkleCheck)

	// Check 4: Time proof
	timeCheck := CheckResult{Name: "Time Proof", Passed: true}
	if cert.TimeProof == nil {
		timeCheck.Message = "No time proof"
	} else {
		timeCheck.Message = fmt.Sprintf("%s @ %s", cert.TimeProof.ProofType, cert.TimeProof.Timestamp)
	}
	result.Checks = append(result.Checks, timeCheck)

	// Check 5: Anchor proof
	anchorCheck := CheckResult{Name: "Anchor Proof", Passed: true}
	if cert.AnchorProof == nil {
		anchorCheck.Message = "No anchor proof"
	} else {
		anchorCheck.Message = fmt.Sprintf("%s: %s", cert.AnchorProof.AnchorType, cert.AnchorProof.AnchorID)
	}
	result.Checks = append(result.Checks, anchorCheck)

	// Check 6: Evidence level
	levelCheck := CheckResult{Name: "Evidence Level", Passed: true}
	if cert.Metadata != nil {
		levelCheck.Message = cert.Metadata.EvidenceLevel
	} else {
		levelCheck.Message = "Unknown"
	}
	result.Checks = append(result.Checks, levelCheck)

	return result
}

// verifyMerkleProof verify single merkle proof
func (v *Verifier) verifyMerkleProof(leafHash string, path []ProofNode, rootHash string) bool {
	currentHash := leafHash

	for _, node := range path {
		if node.Position == "right" {
			currentHash = v.hashPair(currentHash, node.Hash)
		} else {
			currentHash = v.hashPair(node.Hash, currentHash)
		}
	}

	return currentHash == rootHash
}

// buildMerkleRoot build merkle root from leaves
func (v *Verifier) buildMerkleRoot(leaves []string) string {
	if len(leaves) == 0 {
		return ""
	}

	currentLevel := leaves
	for len(currentLevel) > 1 {
		nextLevel := []string{}
		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			right := left
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			}
			nextLevel = append(nextLevel, v.hashPair(left, right))
		}
		currentLevel = nextLevel
	}

	return currentLevel[0]
}

// hashPair hash two values
func (v *Verifier) hashPair(left, right string) string {
	h := sha256.New()
	h.Write([]byte(left))
	h.Write([]byte(right))
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}
