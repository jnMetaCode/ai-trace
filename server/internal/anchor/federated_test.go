package anchor

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewFederatedAnchor(t *testing.T) {
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				FederatedNodes:   []string{"http://node1:8006", "http://node2:8006"},
				MinConfirmations: 1,
			},
			wantErr: false,
		},
		{
			name: "empty nodes",
			cfg: &Config{
				FederatedNodes:   []string{},
				MinConfirmations: 1,
			},
			wantErr: true, // ErrNotConfigured when no nodes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa, err := NewFederatedAnchor(tt.cfg, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFederatedAnchor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && fa == nil {
				t.Error("NewFederatedAnchor() returned nil without error")
			}
		})
	}
}

func TestFederatedAnchorNodeID(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	nodeID := fa.GetNodeID()
	if nodeID == "" {
		t.Error("GetNodeID() returned empty string")
	}

	// NodeID should be consistent
	if fa.GetNodeID() != nodeID {
		t.Error("GetNodeID() returned different values on subsequent calls")
	}
}

func TestFederatedAnchorPublicKey(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	pubKey := fa.GetPublicKey()
	if pubKey == "" {
		t.Error("GetPublicKey() returned empty string")
	}

	// Should be valid hex
	_, err = hex.DecodeString(pubKey)
	if err != nil {
		t.Errorf("GetPublicKey() returned invalid hex: %v", err)
	}
}

func TestFederatedAnchorAddNode(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	// Initially one node
	nodes := fa.GetKnownNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	// Add a new node
	fa.AddNode("http://newnode:8006")
	nodes = fa.GetKnownNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	// Adding same node again should not duplicate
	fa.AddNode("http://newnode:8006")
	nodes = fa.GetKnownNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes after duplicate add, got %d", len(nodes))
	}
}

func TestFederatedAnchorTrustManagement(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	// Initially no trusted nodes
	trusted := fa.GetTrustedNodes()
	if len(trusted) != 0 {
		t.Errorf("Expected 0 trusted nodes, got %d", len(trusted))
	}

	// Register a trusted node with valid public key (64 hex chars for Ed25519)
	validPubKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	err = fa.RegisterTrustedNode("node-1", validPubKey)
	if err != nil {
		t.Errorf("RegisterTrustedNode() error = %v", err)
	}

	trusted = fa.GetTrustedNodes()
	if len(trusted) != 1 {
		t.Errorf("Expected 1 trusted node, got %d", len(trusted))
	}

	// Invalid public key should fail
	err = fa.RegisterTrustedNode("node-2", "invalid-hex")
	if err == nil {
		t.Error("RegisterTrustedNode() should fail with invalid hex")
	}

	// Wrong length public key should fail
	err = fa.RegisterTrustedNode("node-3", "0123456789abcdef")
	if err == nil {
		t.Error("RegisterTrustedNode() should fail with wrong length key")
	}

	// Remove trusted node
	fa.RemoveTrustedNode("node-1")
	trusted = fa.GetTrustedNodes()
	if len(trusted) != 0 {
		t.Errorf("Expected 0 trusted nodes after removal, got %d", len(trusted))
	}
}

func TestHandleConfirmRequest(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	tests := []struct {
		name         string
		req          *FederatedAnchorRequest
		wantAccepted bool
	}{
		{
			name: "valid request from untrusted node",
			req: &FederatedAnchorRequest{
				CertID:     "cert-123",
				RootHash:   "abcdef123456",
				OriginNode: "unknown-node",
				Timestamp:  time.Now(),
				Signature:  "",
			},
			wantAccepted: true, // Untrusted nodes are accepted without sig verification
		},
		{
			name: "expired timestamp",
			req: &FederatedAnchorRequest{
				CertID:     "cert-123",
				RootHash:   "abcdef123456",
				OriginNode: "unknown-node",
				Timestamp:  time.Now().Add(-10 * time.Minute),
				Signature:  "",
			},
			wantAccepted: false,
		},
		{
			name: "future timestamp",
			req: &FederatedAnchorRequest{
				CertID:     "cert-123",
				RootHash:   "abcdef123456",
				OriginNode: "unknown-node",
				Timestamp:  time.Now().Add(10 * time.Minute),
				Signature:  "",
			},
			wantAccepted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := fa.HandleConfirmRequest(tt.req)
			if err != nil {
				t.Errorf("HandleConfirmRequest() error = %v", err)
				return
			}
			if resp.Accepted != tt.wantAccepted {
				t.Errorf("HandleConfirmRequest() accepted = %v, want %v, error: %s",
					resp.Accepted, tt.wantAccepted, resp.Error)
			}
		})
	}
}

func TestFederatedAnchorVerify(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	ctx := context.Background()

	// Test with non-federated anchor type
	result := &AnchorResult{
		AnchorID:   "test-anchor",
		AnchorType: AnchorTypeEthereum, // Use Ethereum instead of blockchain
	}
	valid, err := fa.Verify(ctx, result)
	if err == nil {
		// Verify returns error for non-federated type
		if valid {
			t.Error("Verify() should return false for non-federated anchor type")
		}
	}

	// Test with federated anchor type but no confirmations
	result = &AnchorResult{
		AnchorID:   "test-anchor",
		AnchorType: AnchorTypeFederated,
	}
	valid, err = fa.Verify(ctx, result)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}
	// This will fail verification since there are no actual confirmations
	if valid {
		t.Error("Verify() should return false for anchor without confirmations")
	}
}

func TestFederatedAnchorGetAnchorType(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &Config{
		FederatedNodes:   []string{"http://node1:8006"},
		MinConfirmations: 1,
	}

	fa, err := NewFederatedAnchor(cfg, logger)
	if err != nil {
		t.Fatalf("NewFederatedAnchor() error = %v", err)
	}

	if fa.GetAnchorType() != AnchorTypeFederated {
		t.Errorf("GetAnchorType() = %v, want %v", fa.GetAnchorType(), AnchorTypeFederated)
	}
}
