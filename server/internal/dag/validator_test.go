package dag

import (
	"testing"
	"time"

	"github.com/ai-trace/server/internal/event"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
}

func TestComputeEventHash(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	evt := &event.Event{
		EventID:     "evt-123",
		TraceID:     "trace-456",
		EventType:   event.EventTypeInput,
		Timestamp:   now,
		Sequence:    1,
		PayloadHash: "payload_hash_123",
	}

	hash1 := v.ComputeEventHash(evt)
	hash2 := v.ComputeEventHash(evt)

	if hash1 != hash2 {
		t.Error("ComputeEventHash should be deterministic")
	}
	if hash1 == "" {
		t.Error("ComputeEventHash should return non-empty hash")
	}
	if len(hash1) != 64 {
		t.Errorf("Expected SHA256 hash length 64, got %d", len(hash1))
	}
}

func TestVerifyEventHash(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	evt := &event.Event{
		EventID:     "evt-123",
		TraceID:     "trace-456",
		EventType:   event.EventTypeInput,
		Timestamp:   now,
		Sequence:    1,
		PayloadHash: "payload_hash_123",
	}

	// Set correct hash
	evt.EventHash = v.ComputeEventHash(evt)

	if !v.VerifyEventHash(evt) {
		t.Error("VerifyEventHash should return true for valid hash")
	}

	// Test with wrong hash
	evt.EventHash = "invalid_hash"
	if v.VerifyEventHash(evt) {
		t.Error("VerifyEventHash should return false for invalid hash")
	}
}

func TestVerifyDAG(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	dag := NewDAG("trace-123")

	// Create valid chain
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	report := v.VerifyDAG(dag)

	if !report.Valid {
		t.Errorf("Expected valid DAG, got errors: %v", report.Errors)
	}
	if report.TotalNodes != 2 {
		t.Errorf("Expected 2 nodes, got %d", report.TotalNodes)
	}
	if report.TraceID != "trace-123" {
		t.Errorf("Expected TraceID trace-123, got %s", report.TraceID)
	}
}

func TestVerifyMergePoint(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	// Create two predecessor events
	pred1 := &event.Event{
		EventID:   "pred-1",
		EventHash: "hash_pred_1",
	}
	pred2 := &event.Event{
		EventID:   "pred-2",
		EventHash: "hash_pred_2",
	}

	// Create merge event with correct predecessors
	merge := &event.Event{
		EventID:         "merge",
		PrevEventHashes: []string{"hash_pred_1", "hash_pred_2"},
		Timestamp:       now,
	}

	if !v.VerifyMergePoint(merge, []*event.Event{pred1, pred2}) {
		t.Error("VerifyMergePoint should return true for valid merge")
	}

	// Test with wrong predecessors
	wrong := []*event.Event{
		{EventID: "wrong-1", EventHash: "wrong_hash"},
	}
	if v.VerifyMergePoint(merge, wrong) {
		t.Error("VerifyMergePoint should return false for invalid merge")
	}

	// Test non-merge event (single predecessor)
	nonMerge := &event.Event{
		EventID:       "single",
		PrevEventHash: "hash_pred_1",
	}
	if !v.VerifyMergePoint(nonMerge, []*event.Event{pred1}) {
		t.Error("VerifyMergePoint should return true for non-merge event")
	}
}

func TestValidateCausalOrder(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	// Create events in correct order
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)

	// Valid order
	validOrder := []*event.Event{evt1, evt2, evt3}
	if !v.ValidateCausalOrder(validOrder) {
		t.Error("ValidateCausalOrder should return true for valid order")
	}

	// Invalid order (predecessor comes after)
	invalidOrder := []*event.Event{evt2, evt1, evt3}
	if v.ValidateCausalOrder(invalidOrder) {
		t.Error("ValidateCausalOrder should return false for invalid order")
	}
}

func TestComputeCausalHash(t *testing.T) {
	v := NewValidator()
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)

	// Test empty slice
	emptyHash := v.ComputeCausalHash([]*event.Event{})
	if emptyHash != "" {
		t.Error("ComputeCausalHash should return empty string for empty input")
	}

	// Test with events
	hash1 := v.ComputeCausalHash([]*event.Event{evt1, evt2})
	hash2 := v.ComputeCausalHash([]*event.Event{evt2, evt1}) // Different input order

	// Should produce same hash (sorted internally by timestamp)
	if hash1 != hash2 {
		t.Error("ComputeCausalHash should be deterministic regardless of input order")
	}
	if hash1 == "" {
		t.Error("ComputeCausalHash should return non-empty hash")
	}
}

func TestBuildDAGFromEvents(t *testing.T) {
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)

	// Build DAG from events (in arbitrary order)
	dag, err := BuildDAGFromEvents("trace-123", []*event.Event{evt3, evt1, evt2})
	if err != nil {
		t.Fatalf("BuildDAGFromEvents failed: %v", err)
	}

	if dag.Size() != 3 {
		t.Errorf("Expected size 3, got %d", dag.Size())
	}

	// Verify structure
	node3, _ := dag.GetNode("evt-3")
	if len(node3.Predecessors) != 1 {
		t.Errorf("Expected 1 predecessor, got %d", len(node3.Predecessors))
	}
}

func TestGetPrevHashes(t *testing.T) {
	// Test with PrevEventHashes
	evt1 := &event.Event{
		PrevEventHashes: []string{"hash1", "hash2"},
	}
	hashes := getPrevHashes(evt1)
	if len(hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(hashes))
	}

	// Test with single PrevEventHash
	evt2 := &event.Event{
		PrevEventHash: "single_hash",
	}
	hashes = getPrevHashes(evt2)
	if len(hashes) != 1 || hashes[0] != "single_hash" {
		t.Error("Expected single hash")
	}

	// Test with no prev hash
	evt3 := &event.Event{}
	hashes = getPrevHashes(evt3)
	if len(hashes) != 0 {
		t.Error("Expected no hashes")
	}
}

func TestValidationReport(t *testing.T) {
	report := &ValidationReport{
		TraceID:    "trace-123",
		Valid:      true,
		TotalNodes: 5,
		Errors:     []ValidationError{},
	}

	if !report.Valid {
		t.Error("Expected valid report")
	}
	if report.TotalNodes != 5 {
		t.Errorf("Expected 5 nodes, got %d", report.TotalNodes)
	}

	// Test with errors
	report.Errors = append(report.Errors, ValidationError{
		EventID: "evt-1",
		Type:    "invalid_hash",
		Message: "test error",
	})
	report.Valid = len(report.Errors) == 0

	if report.Valid {
		t.Error("Expected invalid report when errors present")
	}
}
