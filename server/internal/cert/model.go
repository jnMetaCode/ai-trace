package cert

import (
	"time"

	"github.com/ai-trace/server/internal/merkle"
)

// Legacy constants for backward compatibility
// Prefer using EvidenceLevelInternal, EvidenceLevelCompliance, EvidenceLevelLegal
const (
	EvidenceLevelL1 EvidenceLevel = "internal"   // Legacy L1 -> internal
	EvidenceLevelL2 EvidenceLevel = "compliance" // Legacy L2 -> compliance
	EvidenceLevelL3 EvidenceLevel = "legal"      // Legacy L3 -> legal
)

// Certificate 存证证书
type Certificate struct {
	// 存证标识
	CertID        string `json:"cert_id"`
	CertVersion   string `json:"cert_version"`
	SchemaVersion string `json:"schema_version"`

	// 关联的trace
	TraceID string `json:"trace_id"`

	// 事件哈希列表
	EventHashes []string `json:"event_hashes"`

	// Merkle树
	MerkleTree *merkle.Tree `json:"merkle_tree"`
	RootHash   string       `json:"root_hash"`

	// 时间证明
	TimeProof *TimeProof `json:"time_proof,omitempty"`

	// 锚定证明
	AnchorProof *AnchorProof `json:"anchor_proof,omitempty"`

	// 披露策略
	DisclosurePolicy *DisclosurePolicy `json:"disclosure_policy,omitempty"`

	// 元信息
	Metadata *CertMetadata `json:"metadata"`
}

// TimeProof 时间证明
type TimeProof struct {
	ProofType    string    `json:"proof_type"`    // local, ntp, tsa
	Timestamp    time.Time `json:"timestamp"`
	TSAAuthority string    `json:"tsa_authority,omitempty"`
	TSAToken     string    `json:"tsa_token,omitempty"`
	TSACertChain []string  `json:"tsa_cert_chain,omitempty"`
	Signature    string    `json:"signature,omitempty"`
}

// AnchorProof 锚定证明
type AnchorProof struct {
	AnchorType      string           `json:"anchor_type"` // local, worm, blockchain
	AnchorID        string           `json:"anchor_id"`
	StorageProvider string           `json:"storage_provider,omitempty"`
	ObjectKey       string           `json:"object_key,omitempty"`
	AnchorTimestamp time.Time        `json:"anchor_timestamp"`
	Blockchain      *BlockchainProof `json:"blockchain,omitempty"`
}

// BlockchainProof 区块链证明
type BlockchainProof struct {
	ChainID         string `json:"chain_id"`
	TxHash          string `json:"tx_hash"`
	BlockHeight     int64  `json:"block_height"`
	ContractAddress string `json:"contract_address,omitempty"`
}

// DisclosurePolicy 披露策略
type DisclosurePolicy struct {
	PolicyID        string   `json:"policy_id"`
	AllowedFields   []string `json:"allowed_fields"`
	MaskedFields    []string `json:"masked_fields"`
	EncryptedFields []string `json:"encrypted_fields"`
}

// CertMetadata 证书元信息
type CertMetadata struct {
	TenantID      string        `json:"tenant_id"`
	CreatedAt     time.Time     `json:"created_at"`
	CreatedBy     string        `json:"created_by"`
	EvidenceLevel EvidenceLevel `json:"evidence_level"`
	PublicKey     string        `json:"public_key,omitempty"` // Ed25519公钥（用于验证签名）
}

// MinimalDisclosureProof 最小披露证明
type MinimalDisclosureProof struct {
	SchemaVersion string `json:"schema_version"`

	// 证明目标
	CertID   string `json:"cert_id"`
	RootHash string `json:"root_hash"`

	// 披露的事件
	DisclosedEvents []DisclosedEvent `json:"disclosed_events"`

	// Merkle证明
	MerkleProofs []EventMerkleProof `json:"merkle_proofs"`

	// 时间证明
	TimeProof *TimeProof `json:"time_proof,omitempty"`

	// 锚定证明
	AnchorProof *AnchorProof `json:"anchor_proof,omitempty"`

	// 验证指引
	VerificationInstructions *VerificationInstructions `json:"verification_instructions,omitempty"`
}

// DisclosedEvent 披露的事件
type DisclosedEvent struct {
	EventIndex      int                    `json:"event_index"`
	EventType       string                 `json:"event_type"`
	EventHash       string                 `json:"event_hash"`
	DisclosedFields map[string]interface{} `json:"disclosed_fields"`
}

// EventMerkleProof 事件的Merkle证明
type EventMerkleProof struct {
	EventIndex int                `json:"event_index"`
	EventHash  string             `json:"event_hash"`
	ProofPath  []merkle.ProofNode `json:"proof_path"`
}

// VerificationInstructions 验证指引
type VerificationInstructions struct {
	VerifierURL   string `json:"verifier_url"`
	VerifyCommand string `json:"verify_command"`
}
