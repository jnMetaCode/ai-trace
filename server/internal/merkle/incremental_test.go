package merkle

import (
	"testing"
)

func TestNewIncrementalTree(t *testing.T) {
	config := DefaultIncrementalTreeConfig()
	tree := NewIncrementalTree(config)

	if tree == nil {
		t.Fatal("NewIncrementalTree returned nil")
	}

	if tree.LeafCount() != 0 {
		t.Errorf("Expected 0 leaves, got %d", tree.LeafCount())
	}

	if tree.Height() != 0 {
		t.Errorf("Expected height 0, got %d", tree.Height())
	}
}

func TestIncrementalTreeAppend(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	// Append single leaf
	err := tree.Append("leaf1")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if tree.LeafCount() != 1 {
		t.Errorf("Expected 1 leaf, got %d", tree.LeafCount())
	}

	// Get root
	root, err := tree.Root()
	if err != nil {
		t.Fatalf("Root failed: %v", err)
	}
	if root == "" {
		t.Error("Root should not be empty")
	}

	// Append more leaves
	for i := 2; i <= 10; i++ {
		err := tree.Append("leaf" + string(rune('0'+i)))
		if err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	if tree.LeafCount() != 10 {
		t.Errorf("Expected 10 leaves, got %d", tree.LeafCount())
	}
}

func TestIncrementalTreeAppendBatch(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	leaves := []string{"a", "b", "c", "d", "e"}
	err := tree.AppendBatch(leaves)
	if err != nil {
		t.Fatalf("AppendBatch failed: %v", err)
	}

	if tree.LeafCount() != 5 {
		t.Errorf("Expected 5 leaves, got %d", tree.LeafCount())
	}

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("Root failed: %v", err)
	}
	if root == "" {
		t.Error("Root should not be empty")
	}
}

func TestIncrementalTreeGetProof(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	leaves := []string{"leaf0", "leaf1", "leaf2", "leaf3"}
	for _, leaf := range leaves {
		tree.Append(leaf)
	}

	// Get proof for each leaf
	for i := 0; i < len(leaves); i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Fatalf("GetProof(%d) failed: %v", i, err)
		}

		if proof.LeafIndex != i {
			t.Errorf("Expected leaf index %d, got %d", i, proof.LeafIndex)
		}

		if proof.LeafHash != leaves[i] {
			t.Errorf("Expected leaf hash %s, got %s", leaves[i], proof.LeafHash)
		}

		// Verify proof
		if !proof.Verify() {
			t.Errorf("Proof verification failed for index %d", i)
		}
	}
}

func TestIncrementalTreeProofVerification(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	// Add leaves
	for i := 0; i < 8; i++ {
		tree.Append("data" + string(rune('A'+i)))
	}

	root, _ := tree.Root()

	// Get and verify proof for middle element
	proof, err := tree.GetProof(4)
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}

	if proof.Root != root {
		t.Error("Proof root doesn't match tree root")
	}

	if !proof.Verify() {
		t.Error("Valid proof failed verification")
	}

	// Tamper with proof
	tamperedProof := *proof
	tamperedProof.LeafHash = "tampered"
	if tamperedProof.Verify() {
		t.Error("Tampered proof should fail verification")
	}
}

func TestIncrementalTreeGetLeaf(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	leaves := []string{"first", "second", "third"}
	for _, leaf := range leaves {
		tree.Append(leaf)
	}

	for i, expected := range leaves {
		got, err := tree.GetLeaf(i)
		if err != nil {
			t.Fatalf("GetLeaf(%d) failed: %v", i, err)
		}
		if got != expected {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	}

	// Test out of range
	_, err := tree.GetLeaf(10)
	if err != ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex, got %v", err)
	}

	_, err = tree.GetLeaf(-1)
	if err != ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex for negative index, got %v", err)
	}
}

func TestIncrementalTreeGetLeaves(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	expected := []string{"a", "b", "c", "d"}
	tree.AppendBatch(expected)

	got := tree.GetLeaves()
	if len(got) != len(expected) {
		t.Fatalf("Expected %d leaves, got %d", len(expected), len(got))
	}

	for i, v := range expected {
		if got[i] != v {
			t.Errorf("Leaf %d: expected %s, got %s", i, v, got[i])
		}
	}
}

func TestIncrementalTreeSnapshot(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	leaves := []string{"snap1", "snap2", "snap3", "snap4"}
	tree.AppendBatch(leaves)

	originalRoot, _ := tree.Root()
	originalCount := tree.LeafCount()

	// Create snapshot
	snapshot := tree.Snapshot()

	if snapshot.LeafCount != originalCount {
		t.Errorf("Snapshot leaf count mismatch: expected %d, got %d", originalCount, snapshot.LeafCount)
	}

	// Restore from snapshot
	restored := RestoreFromSnapshot(snapshot)

	restoredRoot, _ := restored.Root()
	if restoredRoot != originalRoot {
		t.Error("Restored tree root doesn't match original")
	}

	if restored.LeafCount() != originalCount {
		t.Errorf("Restored leaf count mismatch: expected %d, got %d", originalCount, restored.LeafCount())
	}

	// Verify leaves
	for i, leaf := range leaves {
		got, err := restored.GetLeaf(i)
		if err != nil {
			t.Fatalf("GetLeaf(%d) failed: %v", i, err)
		}
		if got != leaf {
			t.Errorf("Leaf %d: expected %s, got %s", i, leaf, got)
		}
	}
}

func TestIncrementalTreeEmptyRoot(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	_, err := tree.Root()
	if err != ErrTreeEmpty {
		t.Errorf("Expected ErrTreeEmpty, got %v", err)
	}
}

func TestIncrementalTreeSingleLeafRoot(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())
	tree.Append("only-leaf")

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("Root failed: %v", err)
	}

	// Single leaf should be the root
	if root != "only-leaf" {
		t.Errorf("Single leaf root should be the leaf itself, got %s", root)
	}
}

func TestIncrementalTreeHeight(t *testing.T) {
	tests := []struct {
		leafCount int
		expected  int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 3},
		{5, 4},
		{8, 4},
		{9, 5},
		{16, 5},
	}

	for _, tc := range tests {
		tree := NewIncrementalTree(DefaultIncrementalTreeConfig())
		for i := 0; i < tc.leafCount; i++ {
			tree.Append("leaf")
		}

		height := tree.Height()
		if height != tc.expected {
			t.Errorf("For %d leaves, expected height %d, got %d", tc.leafCount, tc.expected, height)
		}
	}
}

func TestIncrementalTreeConsistencyProof(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	// Add initial leaves
	for i := 0; i < 4; i++ {
		tree.Append("leaf" + string(rune('A'+i)))
	}
	oldSize := tree.LeafCount()

	// Add more leaves
	for i := 4; i < 8; i++ {
		tree.Append("leaf" + string(rune('A'+i)))
	}

	// Get consistency proof
	proof, err := tree.GetConsistencyProof(oldSize)
	if err != nil {
		t.Fatalf("GetConsistencyProof failed: %v", err)
	}

	if proof.OldSize != oldSize {
		t.Errorf("Expected old size %d, got %d", oldSize, proof.OldSize)
	}

	if proof.NewSize != tree.LeafCount() {
		t.Errorf("Expected new size %d, got %d", tree.LeafCount(), proof.NewSize)
	}

	if proof.OldRoot == "" {
		t.Error("Old root should not be empty")
	}

	if proof.NewRoot == "" {
		t.Error("New root should not be empty")
	}

	// Verify consistency
	if !proof.Verify() {
		t.Error("Consistency verification failed")
	}
}

func TestIncrementalTreeConsistencyProofInvalidSize(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())
	tree.AppendBatch([]string{"a", "b", "c", "d"})

	// Invalid old size (0)
	_, err := tree.GetConsistencyProof(0)
	if err != ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex for size 0, got %v", err)
	}

	// Invalid old size (larger than current)
	_, err = tree.GetConsistencyProof(10)
	if err != ErrInvalidIndex {
		t.Errorf("Expected ErrInvalidIndex for size > current, got %v", err)
	}
}

func TestConsistencyProofVerify(t *testing.T) {
	tests := []struct {
		name     string
		proof    ConsistencyProof
		expected bool
	}{
		{
			name: "valid proof",
			proof: ConsistencyProof{
				OldSize: 4,
				NewSize: 8,
				OldRoot: "root1",
				NewRoot: "root2",
			},
			expected: true,
		},
		{
			name: "invalid old size",
			proof: ConsistencyProof{
				OldSize: 0,
				NewSize: 8,
				OldRoot: "root1",
				NewRoot: "root2",
			},
			expected: false,
		},
		{
			name: "new size smaller than old",
			proof: ConsistencyProof{
				OldSize: 8,
				NewSize: 4,
				OldRoot: "root1",
				NewRoot: "root2",
			},
			expected: false,
		},
		{
			name: "empty old root",
			proof: ConsistencyProof{
				OldSize: 4,
				NewSize: 8,
				OldRoot: "",
				NewRoot: "root2",
			},
			expected: false,
		},
		{
			name: "empty new root",
			proof: ConsistencyProof{
				OldSize: 4,
				NewSize: 8,
				OldRoot: "root1",
				NewRoot: "",
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.proof.Verify() != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, !tc.expected)
			}
		})
	}
}

func TestIncrementalTreeConcurrency(t *testing.T) {
	tree := NewIncrementalTree(DefaultIncrementalTreeConfig())

	// Pre-populate with some data
	for i := 0; i < 10; i++ {
		tree.Append("initial" + string(rune('0'+i)))
	}

	done := make(chan bool)

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			tree.Root()
			tree.LeafCount()
			tree.Height()
		}
		done <- true
	}()

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			tree.Append("concurrent" + string(rune('0'+i%10)))
		}
		done <- true
	}()

	// Concurrent proofs
	go func() {
		for i := 0; i < 100; i++ {
			if tree.LeafCount() > 0 {
				tree.GetProof(0)
			}
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify tree is still consistent
	if tree.LeafCount() < 10 {
		t.Error("Tree should have at least 10 leaves")
	}

	root, err := tree.Root()
	if err != nil {
		t.Fatalf("Root failed after concurrent access: %v", err)
	}
	if root == "" {
		t.Error("Root should not be empty")
	}
}

func TestDefaultIncrementalTreeConfig(t *testing.T) {
	config := DefaultIncrementalTreeConfig()

	if config.MaxDepth != 32 {
		t.Errorf("Expected MaxDepth 32, got %d", config.MaxDepth)
	}
}
