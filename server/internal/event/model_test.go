package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventType(t *testing.T) {
	tests := []struct {
		name     string
		typ      EventType
		expected string
	}{
		{"INPUT", EventTypeInput, "INPUT"},
		{"MODEL", EventTypeModel, "MODEL"},
		{"RETRIEVAL", EventTypeRetrieval, "RETRIEVAL"},
		{"TOOL_CALL", EventTypeToolCall, "TOOL_CALL"},
		{"OUTPUT", EventTypeOutput, "OUTPUT"},
		{"POST_EDIT", EventTypePostEdit, "POST_EDIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.expected {
				t.Errorf("EventType = %v, want %v", tt.typ, tt.expected)
			}
		})
	}
}

func TestEventSerialization(t *testing.T) {
	event := Event{
		EventID:   "evt_test123",
		TraceID:   "trc_abc",
		EventType: EventTypeInput,
		Timestamp: time.Now(),
		Sequence:  1,
		TenantID:  "default",
		UserID:    "user1",
		Context: EventContext{
			BusinessID: "biz_123",
		},
		Payload:     json.RawMessage(`{"test": "data"}`),
		PayloadHash: "sha256:abc",
		EventHash:   "sha256:def",
	}

	// Serialize
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Deserialize
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify
	if decoded.EventID != event.EventID {
		t.Errorf("EventID = %v, want %v", decoded.EventID, event.EventID)
	}
	if decoded.TraceID != event.TraceID {
		t.Errorf("TraceID = %v, want %v", decoded.TraceID, event.TraceID)
	}
	if decoded.EventType != event.EventType {
		t.Errorf("EventType = %v, want %v", decoded.EventType, event.EventType)
	}
}

func TestInputPayloadSerialization(t *testing.T) {
	payload := InputPayload{
		PromptHash:   "sha256:prompt",
		PromptLength: 100,
		Attachments: []Attachment{
			{
				Type:      "image",
				Hash:      "sha256:img",
				SizeBytes: 1024,
			},
		},
		RequestParams: RequestParams{
			ModelRequested: "gpt-4",
			Temperature:    0.7,
			MaxTokens:      1000,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal InputPayload: %v", err)
	}

	var decoded InputPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal InputPayload: %v", err)
	}

	if decoded.PromptHash != payload.PromptHash {
		t.Errorf("PromptHash = %v, want %v", decoded.PromptHash, payload.PromptHash)
	}
	if decoded.PromptLength != payload.PromptLength {
		t.Errorf("PromptLength = %v, want %v", decoded.PromptLength, payload.PromptLength)
	}
	if len(decoded.Attachments) != 1 {
		t.Errorf("Attachments count = %v, want 1", len(decoded.Attachments))
	}
}

func TestModelPayloadSerialization(t *testing.T) {
	payload := ModelPayload{
		ModelID:       "gpt-4",
		ModelVersion:  "v1",
		ModelProvider: "openai",
		ActualParams: ActualParams{
			Temperature: 0.7,
			TopP:        1.0,
			MaxTokens:   1000,
			Seed:        12345,
		},
		ParamsHash:       "sha256:params",
		SystemPromptHash: "sha256:system",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal ModelPayload: %v", err)
	}

	var decoded ModelPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ModelPayload: %v", err)
	}

	if decoded.ModelID != payload.ModelID {
		t.Errorf("ModelID = %v, want %v", decoded.ModelID, payload.ModelID)
	}
	if decoded.ActualParams.Temperature != payload.ActualParams.Temperature {
		t.Errorf("Temperature = %v, want %v", decoded.ActualParams.Temperature, payload.ActualParams.Temperature)
	}
}

func TestOutputPayloadSerialization(t *testing.T) {
	payload := OutputPayload{
		OutputHash:   "sha256:output",
		OutputLength: 500,
		Usage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
		FinishReason: "stop",
		LatencyMs:    1500,
		SafetyCheck: SafetyCheck{
			Passed:            true,
			FlaggedCategories: []string{},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal OutputPayload: %v", err)
	}

	var decoded OutputPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal OutputPayload: %v", err)
	}

	if decoded.Usage.TotalTokens != payload.Usage.TotalTokens {
		t.Errorf("TotalTokens = %v, want %v", decoded.Usage.TotalTokens, payload.Usage.TotalTokens)
	}
	if decoded.LatencyMs != payload.LatencyMs {
		t.Errorf("LatencyMs = %v, want %v", decoded.LatencyMs, payload.LatencyMs)
	}
}

func TestRetrievalPayloadSerialization(t *testing.T) {
	payload := RetrievalPayload{
		QueryHash:       "sha256:query",
		KnowledgeBaseID: "kb_123",
		RetrievedChunks: []RetrievedChunk{
			{
				ChunkID:         "chunk_1",
				DocID:           "doc_1",
				ChunkHash:       "sha256:chunk",
				SimilarityScore: 0.95,
			},
		},
		RetrievalMethod: "cosine",
		TopK:            5,
		RerankApplied:   true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal RetrievalPayload: %v", err)
	}

	var decoded RetrievalPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal RetrievalPayload: %v", err)
	}

	if len(decoded.RetrievedChunks) != 1 {
		t.Errorf("RetrievedChunks count = %v, want 1", len(decoded.RetrievedChunks))
	}
	if decoded.RetrievedChunks[0].SimilarityScore != 0.95 {
		t.Errorf("SimilarityScore = %v, want 0.95", decoded.RetrievedChunks[0].SimilarityScore)
	}
}

func TestToolCallPayloadSerialization(t *testing.T) {
	payload := ToolCallPayload{
		ToolName:       "search",
		ToolVersion:    "1.0",
		ToolArgsHash:   "sha256:args",
		ToolResultHash: "sha256:result",
		ToolStatus:     "success",
		ToolLatencyMs:  500,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCallPayload: %v", err)
	}

	var decoded ToolCallPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ToolCallPayload: %v", err)
	}

	if decoded.ToolName != payload.ToolName {
		t.Errorf("ToolName = %v, want %v", decoded.ToolName, payload.ToolName)
	}
	if decoded.ToolStatus != payload.ToolStatus {
		t.Errorf("ToolStatus = %v, want %v", decoded.ToolStatus, payload.ToolStatus)
	}
}

func TestPostEditPayloadSerialization(t *testing.T) {
	payload := PostEditPayload{
		OriginalOutputEventID: "evt_orig",
		OriginalOutputHash:    "sha256:orig",
		EditedOutputHash:      "sha256:edited",
		EditType:              "correction",
		EditorID:              "user_editor",
		EditReason:            "Fixed typo",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal PostEditPayload: %v", err)
	}

	var decoded PostEditPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PostEditPayload: %v", err)
	}

	if decoded.EditType != payload.EditType {
		t.Errorf("EditType = %v, want %v", decoded.EditType, payload.EditType)
	}
	if decoded.EditReason != payload.EditReason {
		t.Errorf("EditReason = %v, want %v", decoded.EditReason, payload.EditReason)
	}
}

func TestEventContextSerialization(t *testing.T) {
	ctx := EventContext{
		BusinessID:   "biz_123",
		BusinessType: "customer_support",
		Department:   "Sales",
		ClientIP:     "192.168.1.1",
		ClientType:   "web",
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal EventContext: %v", err)
	}

	var decoded EventContext
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal EventContext: %v", err)
	}

	if decoded.BusinessID != ctx.BusinessID {
		t.Errorf("BusinessID = %v, want %v", decoded.BusinessID, ctx.BusinessID)
	}
	if decoded.Department != ctx.Department {
		t.Errorf("Department = %v, want %v", decoded.Department, ctx.Department)
	}
}

func BenchmarkEventSerialization(b *testing.B) {
	event := Event{
		EventID:     "evt_benchmark",
		TraceID:     "trc_bench",
		EventType:   EventTypeInput,
		Timestamp:   time.Now(),
		Sequence:    1,
		TenantID:    "default",
		Payload:     json.RawMessage(`{"test": "benchmark data"}`),
		PayloadHash: "sha256:bench",
		EventHash:   "sha256:bench",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(event)
	}
}
