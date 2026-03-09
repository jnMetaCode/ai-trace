package merkle

import (
	"strings"
	"testing"
)

func TestNewTree(t *testing.T) {
	tests := []struct {
		name    string
		leaves  []string
		wantErr bool
	}{
		{
			name:    "single leaf",
			leaves:  []string{"sha256:abc123"},
			wantErr: false,
		},
		{
			name:    "two leaves",
			leaves:  []string{"sha256:abc123", "sha256:def456"},
			wantErr: false,
		},
		{
			name:    "multiple leaves",
			leaves:  []string{"sha256:a", "sha256:b", "sha256:c", "sha256:d"},
			wantErr: false,
		},
		{
			name:    "odd number of leaves",
			leaves:  []string{"sha256:a", "sha256:b", "sha256:c"},
			wantErr: false,
		},
		{
			name:    "empty leaves",
			leaves:  []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := NewTree(tt.leaves)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tree == nil {
					t.Error("NewTree() returned nil tree")
					return
				}
				if tree.Root == "" {
					t.Error("NewTree() returned empty root")
				}
				if !strings.HasPrefix(tree.Root, "sha256:") {
					t.Errorf("NewTree() root should start with 'sha256:', got %s", tree.Root)
				}
			}
		})
	}
}

func TestTreeGetProof(t *testing.T) {
	leaves := []string{
		"sha256:leaf0",
		"sha256:leaf1",
		"sha256:leaf2",
		"sha256:leaf3",
	}

	tree, err := NewTree(leaves)
	if err != nil {
		t.Fatalf("NewTree() error = %v", err)
	}

	tests := []struct {
		name    string
		index   int
		wantErr bool
	}{
		{"first leaf", 0, false},
		{"last leaf", 3, false},
		{"middle leaf", 1, false},
		{"negative index", -1, true},
		{"out of range", 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proof, err := tree.GetProof(tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProof() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if proof == nil {
					t.Error("GetProof() returned nil proof")
					return
				}
				if proof.Root != tree.Root {
					t.Errorf("GetProof() root = %v, want %v", proof.Root, tree.Root)
				}
				if proof.LeafIndex != tt.index {
					t.Errorf("GetProof() index = %v, want %v", proof.LeafIndex, tt.index)
				}
			}
		})
	}
}

func TestVerifyProof(t *testing.T) {
	leaves := []string{
		"sha256:leaf0",
		"sha256:leaf1",
		"sha256:leaf2",
		"sha256:leaf3",
	}

	tree, err := NewTree(leaves)
	if err != nil {
		t.Fatalf("NewTree() error = %v", err)
	}

	// Test all leaves can be verified
	for i := 0; i < len(leaves); i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Errorf("GetProof(%d) error = %v", i, err)
			continue
		}

		if !VerifyProof(proof) {
			t.Errorf("VerifyProof(%d) returned false, expected true", i)
		}
	}
}

func TestVerifyProofTampered(t *testing.T) {
	leaves := []string{
		"sha256:leaf0",
		"sha256:leaf1",
		"sha256:leaf2",
		"sha256:leaf3",
	}

	tree, err := NewTree(leaves)
	if err != nil {
		t.Fatalf("NewTree() error = %v", err)
	}

	proof, err := tree.GetProof(0)
	if err != nil {
		t.Fatalf("GetProof() error = %v", err)
	}

	// Tamper with proof
	tamperedProof := &Proof{
		LeafIndex: proof.LeafIndex,
		LeafHash:  "sha256:tampered",
		Path:      proof.Path,
		Root:      proof.Root,
	}

	if VerifyProof(tamperedProof) {
		t.Error("VerifyProof() should return false for tampered proof")
	}
}

func TestTreeGetRoot(t *testing.T) {
	leaves := []string{"sha256:a", "sha256:b"}
	tree, err := NewTree(leaves)
	if err != nil {
		t.Fatalf("NewTree() error = %v", err)
	}

	root := tree.GetRoot()
	if root == "" {
		t.Error("GetRoot() returned empty string")
	}
	if root != tree.Root {
		t.Errorf("GetRoot() = %v, want %v", root, tree.Root)
	}
}

func TestTreeGetLeafCount(t *testing.T) {
	tests := []struct {
		name     string
		leaves   []string
		expected int
	}{
		{"one leaf", []string{"a"}, 1},
		{"two leaves", []string{"a", "b"}, 2},
		{"four leaves", []string{"a", "b", "c", "d"}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, _ := NewTree(tt.leaves)
			if count := tree.GetLeafCount(); count != tt.expected {
				t.Errorf("GetLeafCount() = %v, want %v", count, tt.expected)
			}
		})
	}
}

func TestTreeGetHeight(t *testing.T) {
	tests := []struct {
		name           string
		leaves         []string
		expectedHeight int
	}{
		{"one leaf", []string{"a"}, 1},
		{"two leaves", []string{"a", "b"}, 2},
		{"three leaves", []string{"a", "b", "c"}, 3},
		{"four leaves", []string{"a", "b", "c", "d"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, _ := NewTree(tt.leaves)
			if height := tree.GetHeight(); height != tt.expectedHeight {
				t.Errorf("GetHeight() = %v, want %v", height, tt.expectedHeight)
			}
		})
	}
}

func TestHashPairConsistency(t *testing.T) {
	// Same inputs should produce same output
	left := "sha256:left"
	right := "sha256:right"

	hash1 := hashPair(left, right)
	hash2 := hashPair(left, right)

	if hash1 != hash2 {
		t.Errorf("hashPair() not consistent: %v != %v", hash1, hash2)
	}

	// Different order should produce different output
	hashReversed := hashPair(right, left)
	if hash1 == hashReversed {
		t.Error("hashPair() should produce different output for different order")
	}
}

func BenchmarkNewTree(b *testing.B) {
	leaves := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		leaves[i] = "sha256:benchmark_leaf_hash"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewTree(leaves)
	}
}

func BenchmarkGetProof(b *testing.B) {
	leaves := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		leaves[i] = "sha256:benchmark_leaf_hash"
	}
	tree, _ := NewTree(leaves)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GetProof(i % 1000)
	}
}

func BenchmarkVerifyProof(b *testing.B) {
	leaves := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		leaves[i] = "sha256:benchmark_leaf_hash"
	}
	tree, _ := NewTree(leaves)
	proof, _ := tree.GetProof(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyProof(proof)
	}
}
