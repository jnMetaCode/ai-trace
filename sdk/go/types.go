// Package aitrace provides a Go SDK for the AI-Trace platform.
//
// AI-Trace is an enterprise AI decision auditing and tamper-proof attestation platform.
// This SDK allows you to:
//   - Record AI inference events with full audit trails
//   - Generate tamper-proof certificates for AI decisions
//   - Verify attestation integrity
//   - Create minimal disclosure proofs
//
// Example usage:
//
//	client := aitrace.NewClient("your-api-key",
//	    aitrace.WithBaseURL("https://api.aitrace.cc"),
//	    aitrace.WithUpstreamAPIKey("sk-your-openai-key"),
//	)
//
//	// Create chat completion with attestation
//	resp, err := client.Chat.Create(ctx, aitrace.ChatRequest{
//	    Model: "gpt-4",
//	    Messages: []aitrace.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	})
//
//	// Commit certificate
//	cert, err := client.Certs.Commit(ctx, resp.TraceID, "L2")
package aitrace

import (
	"time"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	N           int       `json:"n,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`

	// AI-Trace specific fields
	TraceID    string `json:"-"` // Set via header
	SessionID  string `json:"-"` // Set via header
	BusinessID string `json:"-"` // Set via header
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`

	// AI-Trace specific fields
	TraceID string `json:"trace_id,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Event represents an AI-Trace event.
type Event struct {
	EventID       string                 `json:"event_id"`
	TraceID       string                 `json:"trace_id"`
	EventType     string                 `json:"event_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Sequence      int                    `json:"sequence"`
	Payload       map[string]interface{} `json:"payload"`
	EventHash     string                 `json:"event_hash,omitempty"`
	PrevEventHash string                 `json:"prev_event_hash,omitempty"`
	PayloadHash   string                 `json:"payload_hash,omitempty"`
}

// IngestRequest represents an event ingest request.
type IngestRequest struct {
	Events []Event `json:"events"`
}

// IngestResponse represents an event ingest response.
type IngestResponse struct {
	Ingested int      `json:"ingested"`
	EventIDs []string `json:"event_ids"`
}

// EventSearchRequest represents an event search request.
type EventSearchRequest struct {
	TraceID   string `json:"trace_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

// EventSearchResponse represents an event search response.
type EventSearchResponse struct {
	Events     []Event `json:"events"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// Certificate represents an attestation certificate.
type Certificate struct {
	CertID        string      `json:"cert_id"`
	TraceID       string      `json:"trace_id"`
	RootHash      string      `json:"root_hash"`
	EventCount    int         `json:"event_count"`
	EvidenceLevel string      `json:"evidence_level"`
	CreatedAt     time.Time   `json:"created_at"`
	TimeProof     *TimeProof  `json:"time_proof,omitempty"`
	AnchorProof   *AnchorProof `json:"anchor_proof,omitempty"`
}

// TimeProof represents a timestamp proof.
type TimeProof struct {
	Timestamp   time.Time `json:"timestamp"`
	TimestampID string    `json:"timestamp_id"`
	TSAName     string    `json:"tsa_name,omitempty"`
	TSAHash     string    `json:"tsa_hash,omitempty"`
}

// AnchorProof represents a blockchain anchor proof.
type AnchorProof struct {
	Network       string `json:"network"`
	TransactionID string `json:"transaction_id"`
	BlockNumber   uint64 `json:"block_number"`
	BlockHash     string `json:"block_hash"`
	Timestamp     int64  `json:"timestamp"`
}

// CommitRequest represents a certificate commit request.
type CommitRequest struct {
	TraceID       string `json:"trace_id"`
	EvidenceLevel string `json:"evidence_level"`
}

// VerifyRequest represents a certificate verification request.
type VerifyRequest struct {
	CertID   string `json:"cert_id,omitempty"`
	RootHash string `json:"root_hash,omitempty"`
}

// VerificationResult represents a verification result.
type VerificationResult struct {
	Valid       bool                   `json:"valid"`
	Checks      map[string]interface{} `json:"checks"`
	Certificate *Certificate           `json:"certificate,omitempty"`
}

// ProveRequest represents a minimal disclosure proof request.
type ProveRequest struct {
	DiscloseEvents []int    `json:"disclose_events"`
	DiscloseFields []string `json:"disclose_fields,omitempty"`
}

// ProofResponse represents a minimal disclosure proof response.
type ProofResponse struct {
	CertID         string                   `json:"cert_id"`
	RootHash       string                   `json:"root_hash"`
	DiscloseEvents []Event                  `json:"disclosed_events"`
	MerkleProofs   []MerkleProof            `json:"merkle_proofs"`
	Metadata       map[string]interface{}   `json:"metadata,omitempty"`
}

// MerkleProof represents a Merkle proof for an event.
type MerkleProof struct {
	EventIndex int      `json:"event_index"`
	Siblings   []string `json:"siblings"`
	Direction  []int    `json:"direction"`
}

// CertSearchResponse represents a certificate search response.
type CertSearchResponse struct {
	Certificates []Certificate `json:"certificates"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
	TotalPages   int           `json:"total_pages"`
}

// APIError represents an API error response.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// EvidenceLevel constants.
const (
	EvidenceLevelL1 = "L1" // Basic: Merkle tree + timestamp
	EvidenceLevelL2 = "L2" // WORM storage
	EvidenceLevelL3 = "L3" // Blockchain anchor
)

// EventType constants.
const (
	EventTypeInput      = "llm.input"
	EventTypeOutput     = "llm.output"
	EventTypeChunk      = "llm.chunk"
	EventTypeToolCall   = "llm.tool_call"
	EventTypeToolResult = "llm.tool_result"
	EventTypeError      = "llm.error"
)
