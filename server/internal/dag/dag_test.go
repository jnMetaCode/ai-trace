package dag

import (
	"testing"
	"time"

	"github.com/ai-trace/server/internal/event"
)

// createTestEvent creates a test event with computed hash
func createTestEvent(id, traceID, prevHash string, prevHashes []string, ts time.Time, seq int) *event.Event {
	evt := &event.Event{
		EventID:         id,
		TraceID:         traceID,
		PrevEventHash:   prevHash,
		PrevEventHashes: prevHashes,
		EventType:       event.EventTypeInput,
		Timestamp:       ts,
		Sequence:        seq,
		PayloadHash:     "test_payload_hash",
	}
	// Compute the event hash
	v := NewValidator()
	evt.EventHash = v.ComputeEventHash(evt)
	return evt
}

func TestNewDAG(t *testing.T) {
	dag := NewDAG("trace-123")
	if dag == nil {
		t.Fatal("NewDAG returned nil")
	}
	if dag.Size() != 0 {
		t.Errorf("Expected empty DAG, got size %d", dag.Size())
	}
	if len(dag.GetRoots()) != 0 {
		t.Errorf("Expected no roots, got %d", len(dag.GetRoots()))
	}
	if len(dag.GetLeaves()) != 0 {
		t.Errorf("Expected no leaves, got %d", len(dag.GetLeaves()))
	}
}

func TestAddEvent(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Add root event
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	if err := dag.AddEvent(evt1); err != nil {
		t.Fatalf("Failed to add root event: %v", err)
	}

	if dag.Size() != 1 {
		t.Errorf("Expected size 1, got %d", dag.Size())
	}
	if len(dag.GetRoots()) != 1 {
		t.Errorf("Expected 1 root, got %d", len(dag.GetRoots()))
	}
	if len(dag.GetLeaves()) != 1 {
		t.Errorf("Expected 1 leaf, got %d", len(dag.GetLeaves()))
	}
}

func TestAddEventWithPredecessor(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Add root event
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	if err := dag.AddEvent(evt1); err != nil {
		t.Fatalf("Failed to add root event: %v", err)
	}

	// Add child event
	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	if err := dag.AddEvent(evt2); err != nil {
		t.Fatalf("Failed to add child event: %v", err)
	}

	if dag.Size() != 2 {
		t.Errorf("Expected size 2, got %d", dag.Size())
	}
	if len(dag.GetRoots()) != 1 {
		t.Errorf("Expected 1 root, got %d", len(dag.GetRoots()))
	}
	if len(dag.GetLeaves()) != 1 {
		t.Errorf("Expected 1 leaf, got %d", len(dag.GetLeaves()))
	}

	// Verify the connection
	node2, err := dag.GetNode("evt-2")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}
	if len(node2.Predecessors) != 1 {
		t.Errorf("Expected 1 predecessor, got %d", len(node2.Predecessors))
	}
	if node2.Predecessors[0].Event.EventID != "evt-1" {
		t.Errorf("Expected predecessor evt-1, got %s", node2.Predecessors[0].Event.EventID)
	}
}

func TestAddDuplicateEvent(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	if err := dag.AddEvent(evt1); err != nil {
		t.Fatalf("Failed to add event: %v", err)
	}

	// Try to add duplicate
	err := dag.AddEvent(evt1)
	if err != ErrDuplicateEvent {
		t.Errorf("Expected ErrDuplicateEvent, got %v", err)
	}
}

func TestParallelEvents(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Add root event
	root := createTestEvent("root", "trace-123", "", nil, now, 1)
	if err := dag.AddEvent(root); err != nil {
		t.Fatalf("Failed to add root: %v", err)
	}

	// Add two parallel events
	evt1 := createTestEvent("evt-1", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	evt2 := createTestEvent("evt-2", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)

	if err := dag.AddEvent(evt1); err != nil {
		t.Fatalf("Failed to add evt-1: %v", err)
	}
	if err := dag.AddEvent(evt2); err != nil {
		t.Fatalf("Failed to add evt-2: %v", err)
	}

	if dag.Size() != 3 {
		t.Errorf("Expected size 3, got %d", dag.Size())
	}
	if len(dag.GetLeaves()) != 2 {
		t.Errorf("Expected 2 leaves, got %d", len(dag.GetLeaves()))
	}

	// Add merge event
	merge := createTestEvent("merge", "trace-123", "", []string{evt1.EventHash, evt2.EventHash}, now.Add(2*time.Second), 3)
	if err := dag.AddEvent(merge); err != nil {
		t.Fatalf("Failed to add merge: %v", err)
	}

	if dag.Size() != 4 {
		t.Errorf("Expected size 4, got %d", dag.Size())
	}
	if len(dag.GetLeaves()) != 1 {
		t.Errorf("Expected 1 leaf, got %d", len(dag.GetLeaves()))
	}

	// Verify merge node has two predecessors
	mergeNode, _ := dag.GetNode("merge")
	if len(mergeNode.Predecessors) != 2 {
		t.Errorf("Expected 2 predecessors, got %d", len(mergeNode.Predecessors))
	}
}

func TestTopologicalSort(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Create a chain: evt1 -> evt2 -> evt3
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)
	dag.AddEvent(evt3)

	sorted := dag.TopologicalSort()
	if len(sorted) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(sorted))
	}

	// Verify order - predecessors should come before successors
	eventIndex := make(map[string]int)
	for i, evt := range sorted {
		eventIndex[evt.EventID] = i
	}

	if eventIndex["evt-1"] > eventIndex["evt-2"] {
		t.Error("evt-1 should come before evt-2")
	}
	if eventIndex["evt-2"] > eventIndex["evt-3"] {
		t.Error("evt-2 should come before evt-3")
	}
}

func TestGetPath(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Create chain
	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)
	dag.AddEvent(evt3)

	// Get path from evt-1 to evt-3
	path, err := dag.GetPath("evt-1", "evt-3")
	if err != nil {
		t.Fatalf("Failed to get path: %v", err)
	}

	if len(path) != 3 {
		t.Errorf("Expected path length 3, got %d", len(path))
	}
}

func TestGetAncestors(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)
	dag.AddEvent(evt3)

	ancestors, err := dag.GetAncestors("evt-3")
	if err != nil {
		t.Fatalf("Failed to get ancestors: %v", err)
	}

	if len(ancestors) != 2 {
		t.Errorf("Expected 2 ancestors, got %d", len(ancestors))
	}
}

func TestGetDescendants(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	evt3 := createTestEvent("evt-3", "trace-123", evt2.EventHash, nil, now.Add(2*time.Second), 3)
	dag.AddEvent(evt3)

	descendants, err := dag.GetDescendants("evt-1")
	if err != nil {
		t.Fatalf("Failed to get descendants: %v", err)
	}

	if len(descendants) != 2 {
		t.Errorf("Expected 2 descendants, got %d", len(descendants))
	}
}

func TestComputeMergeHash(t *testing.T) {
	dag := NewDAG("trace-123")

	// Test deterministic hash computation
	hashes1 := []string{"hash-a", "hash-b", "hash-c"}
	hashes2 := []string{"hash-c", "hash-b", "hash-a"} // Different order

	result1 := dag.ComputeMergeHash(hashes1)
	result2 := dag.ComputeMergeHash(hashes2)

	// Should produce same hash regardless of input order
	if result1 != result2 {
		t.Error("Merge hash should be deterministic regardless of input order")
	}
}

func TestVerify(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	result, err := dag.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid DAG, got invalid: %v", result.InvalidNodes)
	}
	if result.TotalNodes != 2 {
		t.Errorf("Expected 2 nodes, got %d", result.TotalNodes)
	}
}

func TestToJSON(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	evt2 := createTestEvent("evt-2", "trace-123", evt1.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt2)

	export := dag.ToJSON()
	if export == nil {
		t.Fatal("ToJSON returned nil")
	}
	if export.TraceID != "trace-123" {
		t.Errorf("Expected TraceID trace-123, got %s", export.TraceID)
	}
	if len(export.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(export.Nodes))
	}
	if len(export.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(export.Edges))
	}
}

func TestGetEventsByDepth(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	// Create structure with multiple depths
	root := createTestEvent("root", "trace-123", "", nil, now, 1)
	dag.AddEvent(root)

	evt1 := createTestEvent("evt-1", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	evt2 := createTestEvent("evt-2", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt1)
	dag.AddEvent(evt2)

	merge := createTestEvent("merge", "trace-123", "", []string{evt1.EventHash, evt2.EventHash}, now.Add(2*time.Second), 3)
	dag.AddEvent(merge)

	byDepth := dag.GetEventsByDepth()

	if len(byDepth[0]) != 1 {
		t.Errorf("Expected 1 event at depth 0, got %d", len(byDepth[0]))
	}
	if len(byDepth[1]) != 2 {
		t.Errorf("Expected 2 events at depth 1, got %d", len(byDepth[1]))
	}
	if len(byDepth[2]) != 1 {
		t.Errorf("Expected 1 event at depth 2, got %d", len(byDepth[2]))
	}
}

func TestGetParallelEvents(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	root := createTestEvent("root", "trace-123", "", nil, now, 1)
	dag.AddEvent(root)

	// Add parallel events at same depth
	evt1 := createTestEvent("evt-1", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	evt2 := createTestEvent("evt-2", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	evt3 := createTestEvent("evt-3", "trace-123", root.EventHash, nil, now.Add(time.Second), 2)
	dag.AddEvent(evt1)
	dag.AddEvent(evt2)
	dag.AddEvent(evt3)

	parallel := dag.GetParallelEvents()

	if len(parallel) < 1 {
		t.Error("Expected at least one group of parallel events")
	}
	found := false
	for _, group := range parallel {
		if len(group) == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find a group with 3 parallel events")
	}
}

func TestGetNodeNotFound(t *testing.T) {
	dag := NewDAG("trace-123")

	_, err := dag.GetNode("nonexistent")
	if err != ErrEventNotFound {
		t.Errorf("Expected ErrEventNotFound, got %v", err)
	}
}

func TestGetPathNotFound(t *testing.T) {
	dag := NewDAG("trace-123")
	now := time.Now()

	evt1 := createTestEvent("evt-1", "trace-123", "", nil, now, 1)
	dag.AddEvent(evt1)

	_, err := dag.GetPath("evt-1", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent destination")
	}

	_, err = dag.GetPath("nonexistent", "evt-1")
	if err == nil {
		t.Error("Expected error for nonexistent source")
	}
}
