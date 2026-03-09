package aitrace

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// CertsService handles certificate operations.
type CertsService struct {
	client *Client
}

// Commit commits a certificate for a trace.
//
// Evidence levels:
//   - L1: Basic attestation with Merkle tree and timestamp
//   - L2: WORM (Write Once Read Many) storage for legal compliance
//   - L3: Blockchain anchor for maximum tamper-proof guarantee
//
// Example:
//
//	cert, err := client.Certs.Commit(ctx, "trace-123", aitrace.EvidenceLevelL2)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Certificate ID: %s\n", cert.CertID)
//	fmt.Printf("Root Hash: %s\n", cert.RootHash)
func (s *CertsService) Commit(ctx context.Context, traceID string, evidenceLevel string) (*Certificate, error) {
	// Input validation
	if traceID == "" {
		return nil, &APIError{Code: "invalid_request", Message: "trace_id is required", StatusCode: 400}
	}
	if !IsValidEvidenceLevel(evidenceLevel) {
		return nil, &APIError{Code: "invalid_request", Message: "invalid evidence level: must be L1, L2, or L3", StatusCode: 400}
	}

	req := CommitRequest{
		TraceID:       traceID,
		EvidenceLevel: evidenceLevel,
	}

	respBody, err := s.client.post(ctx, "/api/v1/certs/commit", req, nil)
	if err != nil {
		return nil, err
	}

	var cert Certificate
	if err := json.Unmarshal(respBody, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &cert, nil
}

// Verify verifies a certificate's integrity.
//
// You can verify by either cert_id or root_hash.
//
// Example:
//
//	result, err := client.Certs.Verify(ctx, aitrace.VerifyRequest{
//	    CertID: "cert-123",
//	})
//	if result.Valid {
//	    fmt.Println("Certificate is valid!")
//	}
func (s *CertsService) Verify(ctx context.Context, req VerifyRequest) (*VerificationResult, error) {
	// Input validation
	if req.CertID == "" && req.RootHash == "" {
		return nil, &APIError{Code: "invalid_request", Message: "either cert_id or root_hash is required", StatusCode: 400}
	}

	respBody, err := s.client.post(ctx, "/api/v1/certs/verify", req, nil)
	if err != nil {
		return nil, err
	}

	var result VerificationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// VerifyByCertID verifies a certificate by its ID.
func (s *CertsService) VerifyByCertID(ctx context.Context, certID string) (*VerificationResult, error) {
	return s.Verify(ctx, VerifyRequest{CertID: certID})
}

// VerifyByRootHash verifies a certificate by its root hash.
func (s *CertsService) VerifyByRootHash(ctx context.Context, rootHash string) (*VerificationResult, error) {
	return s.Verify(ctx, VerifyRequest{RootHash: rootHash})
}

// Search searches for certificates.
//
// Example:
//
//	resp, err := client.Certs.Search(ctx, 1, 20)
//	for _, cert := range resp.Certificates {
//	    fmt.Printf("Cert: %s, Trace: %s\n", cert.CertID, cert.TraceID)
//	}
func (s *CertsService) Search(ctx context.Context, page, pageSize int) (*CertSearchResponse, error) {
	params := make(map[string]string)
	if page > 0 {
		params["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		params["page_size"] = strconv.Itoa(pageSize)
	}

	respBody, err := s.client.get(ctx, "/api/v1/certs/search", params)
	if err != nil {
		return nil, err
	}

	var resp CertSearchResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// Get retrieves a certificate by ID.
func (s *CertsService) Get(ctx context.Context, certID string) (*Certificate, error) {
	if certID == "" {
		return nil, &APIError{Code: "invalid_request", Message: "cert_id is required", StatusCode: 400}
	}

	respBody, err := s.client.get(ctx, fmt.Sprintf("/api/v1/certs/%s", certID), nil)
	if err != nil {
		return nil, err
	}

	var cert Certificate
	if err := json.Unmarshal(respBody, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &cert, nil
}

// Prove generates a minimal disclosure proof.
//
// This allows you to prove that certain events exist in a certificate
// without revealing all events. Useful for privacy-preserving verification.
//
// Example:
//
//	proof, err := client.Certs.Prove(ctx, "cert-123", aitrace.ProveRequest{
//	    DiscloseEvents: []int{0, 2},  // Only disclose events at index 0 and 2
//	    DiscloseFields: []string{"prompt", "response"},  // Only disclose these fields
//	})
func (s *CertsService) Prove(ctx context.Context, certID string, req ProveRequest) (*ProofResponse, error) {
	respBody, err := s.client.post(ctx, fmt.Sprintf("/api/v1/certs/%s/prove", certID), req, nil)
	if err != nil {
		return nil, err
	}

	var proof ProofResponse
	if err := json.Unmarshal(respBody, &proof); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &proof, nil
}

// ProveWithIndices is a convenience method to create a proof with event indices.
func (s *CertsService) ProveWithIndices(ctx context.Context, certID string, indices ...int) (*ProofResponse, error) {
	return s.Prove(ctx, certID, ProveRequest{
		DiscloseEvents: indices,
	})
}

// VerifyProof verifies a minimal disclosure proof against a root hash.
//
// This allows third parties to verify that disclosed events are part
// of the original certificate without accessing all events.
func (s *CertsService) VerifyProof(ctx context.Context, proof *ProofResponse) (bool, error) {
	// Verify the proof locally using Merkle proofs
	// This is a client-side verification that doesn't require an API call

	if proof == nil || len(proof.MerkleProofs) == 0 {
		return false, fmt.Errorf("invalid proof")
	}

	// For each disclosed event, verify its Merkle proof
	for i, merkleProof := range proof.MerkleProofs {
		if i >= len(proof.DiscloseEvents) {
			continue
		}

		// Compute the event hash
		event := proof.DiscloseEvents[i]
		eventHash, err := computeEventHash(&event)
		if err != nil {
			return false, fmt.Errorf("failed to compute event hash: %w", err)
		}

		// Verify Merkle path
		valid := verifyMerkleProof(eventHash, proof.RootHash, merkleProof.Siblings, merkleProof.Direction)
		if !valid {
			return false, nil
		}
	}

	return true, nil
}

// computeEventHash computes the hash of an event (simplified).
func computeEventHash(event *Event) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	return HashContent(string(data)), nil
}

// verifyMerkleProof verifies a Merkle proof.
func verifyMerkleProof(leafHash, rootHash string, siblings []string, direction []int) bool {
	current := leafHash

	for i, sibling := range siblings {
		if i >= len(direction) {
			return false
		}

		if direction[i] == 0 {
			// Sibling is on the right
			current = HashContent(current + sibling)
		} else {
			// Sibling is on the left
			current = HashContent(sibling + current)
		}
	}

	return current == rootHash
}

// CertificateWithEvents holds a certificate with its events.
type CertificateWithEvents struct {
	Certificate *Certificate
	Events      []Event
}

// GetWithEvents retrieves a certificate along with its events.
func (s *CertsService) GetWithEvents(ctx context.Context, certID string) (*CertificateWithEvents, error) {
	cert, err := s.Get(ctx, certID)
	if err != nil {
		return nil, err
	}

	events, err := s.client.Events.GetByTrace(ctx, cert.TraceID)
	if err != nil {
		return nil, err
	}

	return &CertificateWithEvents{
		Certificate: cert,
		Events:      events,
	}, nil
}

// CommitOptions holds options for certificate commitment.
type CommitOptions struct {
	EvidenceLevel string
	Metadata      map[string]interface{}
}

// CommitWithOptions commits a certificate with additional options.
func (s *CertsService) CommitWithOptions(ctx context.Context, traceID string, opts CommitOptions) (*Certificate, error) {
	if opts.EvidenceLevel == "" {
		opts.EvidenceLevel = EvidenceLevelL1
	}

	return s.Commit(ctx, traceID, opts.EvidenceLevel)
}
