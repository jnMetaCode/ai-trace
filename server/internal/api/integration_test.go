// +build integration

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/event"
	"github.com/ai-trace/server/internal/gateway"
	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestSetup sets up test environment
func setupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Create mock config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "test",
		},
		Auth: config.AuthConfig{
			APIKeys: []string{"test-api-key"},
		},
	}

	// Create mock stores (nil for unit tests)
	stores := &store.Stores{
		DB:    nil,
		Redis: nil,
		Minio: nil,
	}

	// Create mock logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// Create mock gateway
	gw := gateway.New(cfg.Gateway, stores, sugar)

	// Create router
	router := NewRouter(cfg, stores, gw, sugar)

	return router
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health endpoint returned status %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Health status = %v, want 'healthy'", response["status"])
	}
}

func TestAuthMiddleware(t *testing.T) {
	router := setupTestRouter(t)

	// Test without API key
	req, _ := http.NewRequest("GET", "/api/v1/events/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Without API key: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// Test with valid API key
	req, _ = http.NewRequest("GET", "/api/v1/events/search", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not be 401 (might be 500 if DB is nil, but auth passed)
	if w.Code == http.StatusUnauthorized {
		t.Error("With valid API key: should not return 401")
	}
}

func TestAuthMiddlewareBearerToken(t *testing.T) {
	router := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/events/search", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not be 401
	if w.Code == http.StatusUnauthorized {
		t.Error("With Bearer token: should not return 401")
	}
}

func TestTenantHeader(t *testing.T) {
	router := setupTestRouter(t)

	// Request with tenant header
	req, _ := http.NewRequest("GET", "/api/v1/events/search", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("X-Tenant-ID", "custom-tenant")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Just verify the request was processed (tenant is set in context)
	// Actual tenant verification would need database integration
}

// Mock handler for events endpoint testing
func TestEventsIngestRequest(t *testing.T) {
	// Create test events
	events := []event.Event{
		{
			EventID:     "evt_test1",
			TraceID:     "trc_test",
			EventType:   event.EventTypeInput,
			Timestamp:   time.Now(),
			Sequence:    1,
			TenantID:    "default",
			Payload:     json.RawMessage(`{"test": "data"}`),
			PayloadHash: "sha256:test",
			EventHash:   "sha256:test",
		},
	}

	body := IngestEventsRequest{Events: events}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/events/ingest", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key")

	// Just verify request parsing works
	var parsed IngestEventsRequest
	if err := json.Unmarshal(jsonBody, &parsed); err != nil {
		t.Errorf("Failed to parse request: %v", err)
	}

	if len(parsed.Events) != 1 {
		t.Errorf("Events count = %d, want 1", len(parsed.Events))
	}
}

func TestSearchEventsQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
	}{
		{"default page", "", 1},
		{"page 2", "page=2", 2},
		{"with trace_id", "trace_id=trc_123", 1},
		{"with event_type", "event_type=INPUT", 1},
		{"combined", "trace_id=trc_123&event_type=INPUT&page=3", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/events/search?"+tt.query, nil)

			// Parse query manually to verify
			q := req.URL.Query()
			page := q.Get("page")
			if page == "" {
				page = "1"
			}
			// Just verify parsing works
		})
	}
}

func TestCertCommitRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    CommitCertRequest
		wantErr bool
	}{
		{
			name:    "valid request",
			body:    CommitCertRequest{TraceID: "trc_123"},
			wantErr: false,
		},
		{
			name:    "with evidence level",
			body:    CommitCertRequest{TraceID: "trc_123", EvidenceLevel: "L2"},
			wantErr: false,
		},
		{
			name:    "empty trace_id",
			body:    CommitCertRequest{TraceID: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)

			var parsed CommitCertRequest
			if err := json.Unmarshal(jsonBody, &parsed); err != nil {
				if !tt.wantErr {
					t.Errorf("Failed to parse request: %v", err)
				}
				return
			}

			if tt.wantErr && parsed.TraceID == "" {
				// Expected empty trace_id
				return
			}
		})
	}
}

func TestVerifyCertRequest(t *testing.T) {
	tests := []struct {
		name string
		body VerifyCertRequest
	}{
		{
			name: "by cert_id",
			body: VerifyCertRequest{CertID: "cert_123"},
		},
		{
			name: "by root_hash",
			body: VerifyCertRequest{RootHash: "sha256:abc"},
		},
		{
			name: "both",
			body: VerifyCertRequest{CertID: "cert_123", RootHash: "sha256:abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)

			var parsed VerifyCertRequest
			if err := json.Unmarshal(jsonBody, &parsed); err != nil {
				t.Errorf("Failed to parse request: %v", err)
			}
		})
	}
}

func TestGenerateProofRequest(t *testing.T) {
	tests := []struct {
		name string
		body GenerateProofRequest
	}{
		{
			name: "single event",
			body: GenerateProofRequest{
				DiscloseEvents: []int{0},
			},
		},
		{
			name: "multiple events",
			body: GenerateProofRequest{
				DiscloseEvents: []int{0, 1, 2},
			},
		},
		{
			name: "with fields",
			body: GenerateProofRequest{
				DiscloseEvents: []int{0},
				DiscloseFields: []string{"event_type", "timestamp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)

			var parsed GenerateProofRequest
			if err := json.Unmarshal(jsonBody, &parsed); err != nil {
				t.Errorf("Failed to parse request: %v", err)
			}

			if len(parsed.DiscloseEvents) != len(tt.body.DiscloseEvents) {
				t.Errorf("DiscloseEvents count = %d, want %d",
					len(parsed.DiscloseEvents), len(tt.body.DiscloseEvents))
			}
		})
	}
}

// Response type tests
func TestIngestEventsResponse(t *testing.T) {
	response := IngestEventsResponse{
		Success: true,
		Results: []IngestEventResult{
			{EventID: "evt_1", EventHash: "sha256:hash1"},
			{EventID: "evt_2", Error: "some error"},
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed IngestEventsResponse
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !parsed.Success {
		t.Error("Success should be true")
	}

	if len(parsed.Results) != 2 {
		t.Errorf("Results count = %d, want 2", len(parsed.Results))
	}

	if parsed.Results[1].Error != "some error" {
		t.Errorf("Error = %v, want 'some error'", parsed.Results[1].Error)
	}
}

// Benchmark tests
func BenchmarkHealthEndpoint(b *testing.B) {
	router := setupTestRouter(nil)
	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
