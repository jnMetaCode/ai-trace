package aitrace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")

	if client.apiKey != "test-api-key" {
		t.Errorf("Expected apiKey 'test-api-key', got '%s'", client.apiKey)
	}

	if client.baseURL != "https://api.aitrace.cc" {
		t.Errorf("Expected default baseURL, got '%s'", client.baseURL)
	}

	if client.Chat == nil {
		t.Error("Chat service should not be nil")
	}

	if client.Events == nil {
		t.Error("Events service should not be nil")
	}

	if client.Certs == nil {
		t.Error("Certs service should not be nil")
	}
}

func TestClientOptions(t *testing.T) {
	client := NewClient("test-api-key",
		WithBaseURL("https://custom.example.com"),
		WithUpstreamAPIKey("sk-upstream"),
		WithUpstreamBaseURL("https://upstream.example.com"),
		WithTimeout(30*time.Second),
	)

	if client.baseURL != "https://custom.example.com" {
		t.Errorf("Expected custom baseURL, got '%s'", client.baseURL)
	}

	if client.upstreamAPIKey != "sk-upstream" {
		t.Errorf("Expected upstream API key, got '%s'", client.upstreamAPIKey)
	}

	if client.upstreamBaseURL != "https://upstream.example.com" {
		t.Errorf("Expected upstream base URL, got '%s'", client.upstreamBaseURL)
	}

	if client.timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", client.timeout)
	}
}

func TestChatCreate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("Expected /api/v1/chat/completions, got %s", r.URL.Path)
		}

		if r.Header.Get("X-API-Key") != "test-api-key" {
			t.Errorf("Expected X-API-Key header")
		}

		// Return mock response
		resp := ChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "Hello!",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
			TraceID: "trace-123",
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewClient("test-api-key", WithBaseURL(server.URL))

	// Make request
	resp, err := client.Chat.Create(context.Background(), ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hello!"},
		},
	})

	if err != nil {
		t.Fatalf("Chat.Create failed: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("Expected ID 'chatcmpl-123', got '%s'", resp.ID)
	}

	if resp.TraceID != "trace-123" {
		t.Errorf("Expected trace ID 'trace-123', got '%s'", resp.TraceID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got '%s'", resp.Choices[0].Message.Content)
	}
}

func TestEventsIngest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events/ingest" {
			t.Errorf("Expected /api/v1/events/ingest, got %s", r.URL.Path)
		}

		resp := IngestResponse{
			Ingested: 2,
			EventIDs: []string{"event-1", "event-2"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	events := []Event{
		{
			EventID:   "event-1",
			TraceID:   "trace-123",
			EventType: EventTypeInput,
			Payload:   map[string]interface{}{"prompt": "Hello"},
		},
		{
			EventID:   "event-2",
			TraceID:   "trace-123",
			EventType: EventTypeOutput,
			Payload:   map[string]interface{}{"content": "World"},
		},
	}

	resp, err := client.Events.Ingest(context.Background(), events)
	if err != nil {
		t.Fatalf("Events.Ingest failed: %v", err)
	}

	if resp.Ingested != 2 {
		t.Errorf("Expected 2 ingested, got %d", resp.Ingested)
	}
}

func TestCertsCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/certs/commit" {
			t.Errorf("Expected /api/v1/certs/commit, got %s", r.URL.Path)
		}

		cert := Certificate{
			CertID:        "cert-123",
			TraceID:       "trace-123",
			RootHash:      "abc123",
			EventCount:    5,
			EvidenceLevel: EvidenceLevelL2,
			CreatedAt:     time.Now(),
		}
		json.NewEncoder(w).Encode(cert)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	cert, err := client.Certs.Commit(context.Background(), "trace-123", EvidenceLevelL2)
	if err != nil {
		t.Fatalf("Certs.Commit failed: %v", err)
	}

	if cert.CertID != "cert-123" {
		t.Errorf("Expected cert ID 'cert-123', got '%s'", cert.CertID)
	}

	if cert.EvidenceLevel != EvidenceLevelL2 {
		t.Errorf("Expected evidence level L2, got '%s'", cert.EvidenceLevel)
	}
}

func TestCertsVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/certs/verify" {
			t.Errorf("Expected /api/v1/certs/verify, got %s", r.URL.Path)
		}

		result := VerificationResult{
			Valid: true,
			Checks: map[string]interface{}{
				"merkle_root":   true,
				"timestamp":     true,
				"event_hashes":  true,
			},
		}
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	result, err := client.Certs.VerifyByCertID(context.Background(), "cert-123")
	if err != nil {
		t.Fatalf("Certs.Verify failed: %v", err)
	}

	if !result.Valid {
		t.Error("Expected valid certificate")
	}

	if result.Checks["merkle_root"] != true {
		t.Error("Expected merkle_root check to pass")
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{
			Code:    "invalid_request",
			Message: "Invalid trace ID",
		})
	}))
	defer server.Close()

	client := NewClient("test-api-key", WithBaseURL(server.URL))

	_, err := client.Certs.Commit(context.Background(), "", EvidenceLevelL1)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.Code != "invalid_request" {
		t.Errorf("Expected code 'invalid_request', got '%s'", apiErr.Code)
	}
}

func TestEventBuilder(t *testing.T) {
	event := NewEventBuilder("trace-123", EventTypeInput).
		WithSequence(1).
		AddPayloadField("prompt", "Hello").
		AddPayloadField("model_id", "gpt-4").
		Build()

	if event.TraceID != "trace-123" {
		t.Errorf("Expected trace ID 'trace-123', got '%s'", event.TraceID)
	}

	if event.EventType != EventTypeInput {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeInput, event.EventType)
	}

	if event.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", event.Sequence)
	}

	if event.Payload["prompt"] != "Hello" {
		t.Errorf("Expected prompt 'Hello', got '%v'", event.Payload["prompt"])
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test InputEvent
	inputEvent := InputEvent("trace-123", "Hello", "gpt-4")
	if inputEvent.EventType != EventTypeInput {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeInput, inputEvent.EventType)
	}

	// Test OutputEvent
	outputEvent := OutputEvent("trace-123", "World", 10)
	if outputEvent.EventType != EventTypeOutput {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeOutput, outputEvent.EventType)
	}

	// Test ToolCallEvent
	toolEvent := ToolCallEvent("trace-123", "search", map[string]interface{}{"query": "test"})
	if toolEvent.EventType != EventTypeToolCall {
		t.Errorf("Expected event type '%s', got '%s'", EventTypeToolCall, toolEvent.EventType)
	}
}

func TestHashContent(t *testing.T) {
	hash1 := HashContent("test")
	hash2 := HashContent("test")
	hash3 := HashContent("different")

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}

	if len(hash1) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

func TestIsValidEvidenceLevel(t *testing.T) {
	if !IsValidEvidenceLevel(EvidenceLevelL1) {
		t.Error("L1 should be valid")
	}
	if !IsValidEvidenceLevel(EvidenceLevelL2) {
		t.Error("L2 should be valid")
	}
	if !IsValidEvidenceLevel(EvidenceLevelL3) {
		t.Error("L3 should be valid")
	}
	if IsValidEvidenceLevel("L4") {
		t.Error("L4 should be invalid")
	}
}

func TestIsValidEventType(t *testing.T) {
	if !IsValidEventType(EventTypeInput) {
		t.Error("llm.input should be valid")
	}
	if !IsValidEventType(EventTypeOutput) {
		t.Error("llm.output should be valid")
	}
	if IsValidEventType("invalid.type") {
		t.Error("invalid.type should be invalid")
	}
}
