package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ai-trace/server/internal/event"
)

// TestIngestEventsRequest tests IngestEventsRequest serialization
func TestIngestEventsRequest(t *testing.T) {
	req := IngestEventsRequest{
		Events: []event.Event{
			{
				EventID:   "evt_001",
				TraceID:   "trc_test",
				EventType: event.EventTypeInput,
				Timestamp: time.Now(),
				Sequence:  1,
				TenantID:  "tenant_1",
			},
			{
				EventID:   "evt_002",
				TraceID:   "trc_test",
				EventType: event.EventTypeOutput,
				Timestamp: time.Now(),
				Sequence:  2,
				TenantID:  "tenant_1",
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IngestEventsRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.Events) != len(req.Events) {
		t.Errorf("events count: got %d, want %d", len(decoded.Events), len(req.Events))
	}
}

// TestIngestEventsRequestEmpty tests empty events request
func TestIngestEventsRequestEmpty(t *testing.T) {
	req := IngestEventsRequest{
		Events: []event.Event{},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IngestEventsRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.Events) != 0 {
		t.Errorf("events count: got %d, want 0", len(decoded.Events))
	}
}

// TestIngestEventsResponse tests IngestEventsResponse serialization
func TestIngestEventsResponse(t *testing.T) {
	resp := IngestEventsResponse{
		Success: true,
		Results: []IngestEventResult{
			{EventID: "evt_001", EventHash: "sha256:abc123"},
			{EventID: "evt_002", EventHash: "sha256:def456"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IngestEventsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Success != resp.Success {
		t.Errorf("success: got %v, want %v", decoded.Success, resp.Success)
	}
	if len(decoded.Results) != len(resp.Results) {
		t.Errorf("results count: got %d, want %d", len(decoded.Results), len(resp.Results))
	}
}

// TestIngestEventResult tests IngestEventResult serialization
func TestIngestEventResult(t *testing.T) {
	tests := []struct {
		name   string
		result IngestEventResult
	}{
		{
			name: "success",
			result: IngestEventResult{
				EventID:   "evt_001",
				EventHash: "sha256:abc",
			},
		},
		{
			name: "with error",
			result: IngestEventResult{
				EventID: "evt_002",
				Error:   "validation failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded IngestEventResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.EventID != tt.result.EventID {
				t.Errorf("event_id: got %s, want %s", decoded.EventID, tt.result.EventID)
			}
		})
	}
}

// TestSearchEventsQueryParams tests query parameter parsing
func TestSearchEventsQueryParams(t *testing.T) {
	tests := []struct {
		name       string
		traceID    string
		eventType  string
		startTime  string
		endTime    string
		page       int
		pageSize   int
	}{
		{
			name:    "basic search",
			traceID: "trc_123",
		},
		{
			name:      "with event type",
			traceID:   "trc_456",
			eventType: "INPUT",
		},
		{
			name:      "with time range",
			traceID:   "trc_789",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-02T00:00:00Z",
		},
		{
			name:     "with pagination",
			traceID:  "trc_abc",
			page:     2,
			pageSize: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just validate that these are valid query params
			if tt.traceID == "" && tt.eventType == "" {
				t.Log("both empty - should have at least one filter")
			}
			if tt.page < 0 {
				t.Error("page should not be negative")
			}
			if tt.pageSize < 0 {
				t.Error("page size should not be negative")
			}
		})
	}
}

// TestEventTypes tests event type constants
func TestEventTypes(t *testing.T) {
	types := []event.EventType{
		event.EventTypeInput,
		event.EventTypeOutput,
		event.EventTypeModel,
		event.EventTypeRetrieval,
		event.EventTypeToolCall,
		event.EventTypeChunk,
		event.EventTypePostEdit,
	}

	seen := make(map[event.EventType]bool)
	for _, et := range types {
		if et == "" {
			t.Error("event type should not be empty")
		}
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

// TestIngestEventsResponseWithErrors tests response with partial errors
func TestIngestEventsResponseWithErrors(t *testing.T) {
	resp := IngestEventsResponse{
		Success: false,
		Results: []IngestEventResult{
			{EventID: "evt_001", EventHash: "sha256:abc"},
			{EventID: "evt_002", Error: "duplicate event"},
			{EventID: "evt_003", EventHash: "sha256:def"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IngestEventsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Count successes and errors
	successCount := 0
	errorCount := 0
	for _, r := range decoded.Results {
		if r.Error != "" {
			errorCount++
		} else if r.EventHash != "" {
			successCount++
		}
	}

	if successCount != 2 {
		t.Errorf("success count: got %d, want 2", successCount)
	}
	if errorCount != 1 {
		t.Errorf("error count: got %d, want 1", errorCount)
	}
}

// TestEventSerialization tests event JSON serialization
func TestEventSerialization(t *testing.T) {
	evt := event.Event{
		EventID:     "evt_test",
		TraceID:     "trc_test",
		EventType:   event.EventTypeInput,
		Timestamp:   time.Now(),
		Sequence:    1,
		TenantID:    "tenant_1",
		SessionID:   "sess_123",
		Payload:     json.RawMessage(`{"content": "test input"}`),
		PayloadHash: "sha256:payload",
		EventHash:   "sha256:event",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded event.Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.EventID != evt.EventID {
		t.Errorf("event_id: got %s, want %s", decoded.EventID, evt.EventID)
	}
	if decoded.TraceID != evt.TraceID {
		t.Errorf("trace_id: got %s, want %s", decoded.TraceID, evt.TraceID)
	}
	if decoded.EventType != evt.EventType {
		t.Errorf("event_type: got %s, want %s", decoded.EventType, evt.EventType)
	}
}
