package event

import (
	"encoding/json"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventTypeInput     EventType = "INPUT"
	EventTypeModel     EventType = "MODEL"
	EventTypeRetrieval EventType = "RETRIEVAL"
	EventTypeToolCall  EventType = "TOOL_CALL"
	EventTypeChunk     EventType = "CHUNK"    // 流式chunk事件
	EventTypeOutput    EventType = "OUTPUT"
	EventTypePostEdit  EventType = "POST_EDIT"
)

// Event 事件基础结构
type Event struct {
	// 事件标识
	EventID       string    `json:"event_id"`
	TraceID       string    `json:"trace_id"`
	ParentEventID string    `json:"parent_event_id,omitempty"`
	PrevEventHash string    `json:"prev_event_hash,omitempty"`

	// DAG 支持：多前驱事件（用于并行事件）
	// 如果 PrevEventHashes 非空，则表示该事件有多个前驱（DAG 结构）
	// 如果为空，则退化为单链结构，使用 PrevEventHash
	PrevEventHashes []string `json:"prev_event_hashes,omitempty"`

	// 事件元信息
	EventType EventType `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int       `json:"sequence"`

	// 租户与用户
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// 业务上下文
	Context EventContext `json:"context,omitempty"`

	// 事件载荷（根据event_type不同）
	Payload json.RawMessage `json:"payload"`

	// 哈希
	PayloadHash string `json:"payload_hash"`
	EventHash   string `json:"event_hash"`
}

// EventContext 事件上下文
type EventContext struct {
	BusinessID   string `json:"business_id,omitempty"`
	BusinessType string `json:"business_type,omitempty"`
	Department   string `json:"department,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	ClientType   string `json:"client_type,omitempty"`
}

// InputPayload INPUT事件载荷
type InputPayload struct {
	PromptHash         string       `json:"prompt_hash"`
	PromptLength       int          `json:"prompt_length"`
	PromptEncryptedRef string       `json:"prompt_encrypted_ref,omitempty"`
	Attachments        []Attachment `json:"attachments,omitempty"`
	RequestParams      RequestParams `json:"request_params,omitempty"`
}

// Attachment 附件信息
type Attachment struct {
	Type         string `json:"type"`
	Hash         string `json:"hash"`
	SizeBytes    int64  `json:"size_bytes"`
	EncryptedRef string `json:"encrypted_ref,omitempty"`
}

// RequestParams 请求参数
type RequestParams struct {
	ModelRequested string  `json:"model_requested,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	MaxTokens      int     `json:"max_tokens,omitempty"`
	TopP           float64 `json:"top_p,omitempty"`
}

// ModelPayload MODEL事件载荷
type ModelPayload struct {
	ModelID             string       `json:"model_id"`
	ModelVersion        string       `json:"model_version,omitempty"`
	ModelProvider       string       `json:"model_provider"`
	ActualParams        ActualParams `json:"actual_params"`
	ParamsHash          string       `json:"params_hash"`
	SystemPromptHash    string       `json:"system_prompt_hash,omitempty"`
	SystemPromptVersion string       `json:"system_prompt_version,omitempty"`
	ModelWeightsHash    string       `json:"model_weights_hash,omitempty"`
	Quantization        string       `json:"quantization,omitempty"`
}

// ActualParams 实际使用的参数
type ActualParams struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Seed        int64   `json:"seed,omitempty"`
}

// RetrievalPayload RETRIEVAL事件载荷
type RetrievalPayload struct {
	QueryHash            string           `json:"query_hash"`
	KnowledgeBaseID      string           `json:"knowledge_base_id"`
	KnowledgeBaseVersion string           `json:"knowledge_base_version,omitempty"`
	RetrievedChunks      []RetrievedChunk `json:"retrieved_chunks"`
	RetrievalMethod      string           `json:"retrieval_method,omitempty"`
	TopK                 int              `json:"top_k,omitempty"`
	RerankApplied        bool             `json:"rerank_applied,omitempty"`
}

// RetrievedChunk 检索到的文档块
type RetrievedChunk struct {
	ChunkID          string  `json:"chunk_id"`
	DocID            string  `json:"doc_id"`
	ChunkHash        string  `json:"chunk_hash"`
	SimilarityScore  float64 `json:"similarity_score"`
	ChunkEncryptedRef string `json:"chunk_encrypted_ref,omitempty"`
}

// ToolCallPayload TOOL_CALL事件载荷
type ToolCallPayload struct {
	ToolName              string `json:"tool_name"`
	ToolVersion           string `json:"tool_version,omitempty"`
	ToolArgsHash          string `json:"tool_args_hash"`
	ToolArgsEncryptedRef  string `json:"tool_args_encrypted_ref,omitempty"`
	ToolResultHash        string `json:"tool_result_hash"`
	ToolResultEncryptedRef string `json:"tool_result_encrypted_ref,omitempty"`
	ToolStatus            string `json:"tool_status"`
	ToolLatencyMs         int64  `json:"tool_latency_ms"`
	ErrorCode             string `json:"error_code,omitempty"`
	ErrorMessage          string `json:"error_message,omitempty"`
}

// OutputPayload OUTPUT事件载荷
type OutputPayload struct {
	OutputHash         string      `json:"output_hash"`
	OutputLength       int         `json:"output_length"`
	OutputEncryptedRef string      `json:"output_encrypted_ref,omitempty"`
	Usage              TokenUsage  `json:"usage"`
	FinishReason       string      `json:"finish_reason"`
	LatencyMs          int64       `json:"latency_ms"`
	PerceptualHash     string      `json:"perceptual_hash,omitempty"`
	SafetyCheck        SafetyCheck `json:"safety_check,omitempty"`

	// 推理行为指纹（4层结构）
	InferenceFingerprint json.RawMessage `json:"inference_fingerprint,omitempty"`
	FingerprintHash      string          `json:"fingerprint_hash,omitempty"`
}

// TokenUsage Token使用统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SafetyCheck 安全检查结果
type SafetyCheck struct {
	Passed            bool     `json:"passed"`
	FlaggedCategories []string `json:"flagged_categories,omitempty"`
}

// PostEditPayload POST_EDIT事件载荷
type PostEditPayload struct {
	OriginalOutputEventID    string `json:"original_output_event_id"`
	OriginalOutputHash       string `json:"original_output_hash"`
	EditedOutputHash         string `json:"edited_output_hash"`
	EditedOutputEncryptedRef string `json:"edited_output_encrypted_ref,omitempty"`
	EditType                 string `json:"edit_type"`
	EditorID                 string `json:"editor_id,omitempty"`
	EditReason               string `json:"edit_reason,omitempty"`
}

// ChunkPayload CHUNK事件载荷（流式增量存证）
type ChunkPayload struct {
	ChunkIndex     int    `json:"chunk_index"`               // chunk序号（从0开始）
	ChunkHash      string `json:"chunk_hash"`                // 当前chunk内容哈希
	CumulativeHash string `json:"cumulative_hash"`           // 累积哈希（链式验证）
	ContentLength  int    `json:"content_length"`            // 当前chunk内容长度
	TokenCount     int    `json:"token_count,omitempty"`     // token数量（如果可获取）
	LatencyMs      int64  `json:"latency_ms"`                // 距上一个chunk的延迟
	FinishReason   string `json:"finish_reason,omitempty"`   // 结束原因（最后一个chunk）
	Model          string `json:"model,omitempty"`           // 模型标识
}

// StreamSession 流式会话信息（用于聚合chunk）
type StreamSession struct {
	SessionID       string    `json:"session_id"`
	TraceID         string    `json:"trace_id"`
	StartTime       time.Time `json:"start_time"`
	ChunkCount      int       `json:"chunk_count"`
	TotalTokens     int       `json:"total_tokens"`
	TotalContentLen int       `json:"total_content_len"`
	FinalHash       string    `json:"final_hash"`       // 最终累积哈希
	FirstChunkMs    int64     `json:"first_chunk_ms"`   // 首chunk延迟
	TotalMs         int64     `json:"total_ms"`         // 总耗时
}
