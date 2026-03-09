// AI-Trace Go Example
// This example demonstrates using AI-Trace from Go applications
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "http://localhost:8006"
	apiKey  = "test-api-key-12345"
)

// Trace represents an AI decision trace
type Trace struct {
	TraceID    string                 `json:"trace_id"`
	TenantID   string                 `json:"tenant_id"`
	Name       string                 `json:"name"`
	CreatedAt  time.Time              `json:"created_at"`
	EventCount int                    `json:"event_count"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Event represents an event within a trace
type Event struct {
	EventID   string                 `json:"event_id"`
	TraceID   string                 `json:"trace_id"`
	EventType string                 `json:"event_type"`
	Sequence  int                    `json:"sequence"`
	Hash      string                 `json:"hash"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// Certificate represents an attestation certificate
type Certificate struct {
	CertID        string    `json:"cert_id"`
	TraceID       string    `json:"trace_id"`
	EvidenceLevel string    `json:"evidence_level"`
	RootHash      string    `json:"root_hash"`
	EventCount    int       `json:"event_count"`
	Signature     string    `json:"signature"`
	CreatedAt     time.Time `json:"created_at"`
}

// VerificationResult represents certificate verification result
type VerificationResult struct {
	Valid          bool   `json:"valid"`
	CertID         string `json:"cert_id"`
	HashValid      bool   `json:"hash_valid"`
	SignatureValid bool   `json:"signature_valid"`
	TimestampValid bool   `json:"timestamp_valid"`
}

// Client wraps HTTP client for AI-Trace API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new AI-Trace client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// CreateTrace creates a new trace
func (c *Client) CreateTrace(name string, metadata map[string]interface{}) (*Trace, error) {
	body := map[string]interface{}{
		"name":      name,
		"tenant_id": "default",
		"metadata":  metadata,
	}

	respBody, err := c.doRequest("POST", "/api/v1/traces", body)
	if err != nil {
		return nil, err
	}

	var trace Trace
	if err := json.Unmarshal(respBody, &trace); err != nil {
		return nil, err
	}

	return &trace, nil
}

// AddEvent adds an event to a trace
func (c *Client) AddEvent(traceID, eventType string, payload map[string]interface{}) (*Event, error) {
	body := map[string]interface{}{
		"trace_id":   traceID,
		"event_type": eventType,
		"payload":    payload,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	respBody, err := c.doRequest("POST", "/api/v1/events/ingest", body)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(respBody, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

// CommitCertificate commits a trace to a certificate
func (c *Client) CommitCertificate(traceID, evidenceLevel string) (*Certificate, error) {
	body := map[string]interface{}{
		"trace_id":       traceID,
		"evidence_level": evidenceLevel,
	}

	respBody, err := c.doRequest("POST", "/api/v1/certs/commit", body)
	if err != nil {
		return nil, err
	}

	var cert Certificate
	if err := json.Unmarshal(respBody, &cert); err != nil {
		return nil, err
	}

	return &cert, nil
}

// VerifyCertificate verifies a certificate
func (c *Client) VerifyCertificate(certID string) (*VerificationResult, error) {
	body := map[string]interface{}{
		"cert_id": certID,
	}

	respBody, err := c.doRequest("POST", "/api/v1/certs/verify", body)
	if err != nil {
		return nil, err
	}

	var result VerificationResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("AI-Trace Go Example")
	fmt.Println("==============================================")

	client := NewClient(baseURL, apiKey)

	// Step 1: Create a trace
	fmt.Println("\n1. Creating trace...")
	trace, err := client.CreateTrace("Go Example Trace", map[string]interface{}{
		"language": "go",
		"version":  "1.21",
	})
	if err != nil {
		fmt.Printf("Error creating trace: %v\n", err)
		return
	}
	fmt.Printf("   Trace ID: %s\n", trace.TraceID)

	// Step 2: Add input event
	fmt.Println("\n2. Adding input event...")
	inputEvent, err := client.AddEvent(trace.TraceID, "input", map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello from Go!"},
		},
	})
	if err != nil {
		fmt.Printf("Error adding event: %v\n", err)
		return
	}
	fmt.Printf("   Event ID: %s\n", inputEvent.EventID)
	fmt.Printf("   Hash: %s...\n", inputEvent.Hash[:20])

	// Step 3: Add output event
	fmt.Println("\n3. Adding output event...")
	outputEvent, err := client.AddEvent(trace.TraceID, "output", map[string]interface{}{
		"response": "Hello! I received your message from Go.",
		"model":    "gpt-4",
	})
	if err != nil {
		fmt.Printf("Error adding event: %v\n", err)
		return
	}
	fmt.Printf("   Event ID: %s\n", outputEvent.EventID)

	// Step 4: Commit certificate
	fmt.Println("\n4. Committing certificate...")
	cert, err := client.CommitCertificate(trace.TraceID, "internal") // or "compliance", "legal"
	if err != nil {
		fmt.Printf("Error committing certificate: %v\n", err)
		return
	}
	fmt.Printf("   Certificate ID: %s\n", cert.CertID)
	fmt.Printf("   Root Hash: %s...\n", cert.RootHash[:20])
	fmt.Printf("   Event Count: %d\n", cert.EventCount)

	// Step 5: Verify certificate
	fmt.Println("\n5. Verifying certificate...")
	result, err := client.VerifyCertificate(cert.CertID)
	if err != nil {
		fmt.Printf("Error verifying certificate: %v\n", err)
		return
	}
	fmt.Printf("   Valid: %v\n", result.Valid)
	fmt.Printf("   Hash Valid: %v\n", result.HashValid)
	fmt.Printf("   Signature Valid: %v\n", result.SignatureValid)

	fmt.Println("\n==============================================")
	fmt.Println("Example completed successfully!")
	fmt.Println("==============================================")
}
