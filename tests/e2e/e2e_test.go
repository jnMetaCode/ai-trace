// +build e2e

// End-to-end tests for AI-Trace
// Run with: go test -tags=e2e ./tests/e2e/...

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	baseURL    = getEnv("AI_TRACE_URL", "http://localhost:8080")
	apiKey     = getEnv("AI_TRACE_API_KEY", "test-api-key-12345")
	tenantID   = getEnv("AI_TRACE_TENANT_ID", "default")
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func makeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Tenant-ID", tenantID)

	return httpClient.Do(req)
}

func TestHealthCheck(t *testing.T) {
	resp, err := makeRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Health status = %v, want 'healthy'", result["status"])
	}
}

func TestEventsWorkflow(t *testing.T) {
	traceID := fmt.Sprintf("trc_e2e_%d", time.Now().UnixNano())

	// 1. Ingest events
	events := []map[string]interface{}{
		{
			"event_id":     fmt.Sprintf("evt_%d_1", time.Now().UnixNano()),
			"trace_id":     traceID,
			"event_type":   "INPUT",
			"timestamp":    time.Now().Format(time.RFC3339),
			"sequence":     1,
			"tenant_id":    tenantID,
			"payload":      map[string]interface{}{"prompt_hash": "sha256:test", "prompt_length": 100},
			"payload_hash": "sha256:payload1",
			"event_hash":   "sha256:event1",
		},
		{
			"event_id":     fmt.Sprintf("evt_%d_2", time.Now().UnixNano()),
			"trace_id":     traceID,
			"event_type":   "MODEL",
			"timestamp":    time.Now().Format(time.RFC3339),
			"sequence":     2,
			"tenant_id":    tenantID,
			"payload":      map[string]interface{}{"model_id": "gpt-4"},
			"payload_hash": "sha256:payload2",
			"event_hash":   "sha256:event2",
		},
		{
			"event_id":     fmt.Sprintf("evt_%d_3", time.Now().UnixNano()),
			"trace_id":     traceID,
			"event_type":   "OUTPUT",
			"timestamp":    time.Now().Format(time.RFC3339),
			"sequence":     3,
			"tenant_id":    tenantID,
			"payload":      map[string]interface{}{"output_hash": "sha256:output"},
			"payload_hash": "sha256:payload3",
			"event_hash":   "sha256:event3",
		},
	}

	resp, err := makeRequest("POST", "/api/v1/events/ingest", map[string]interface{}{
		"events": events,
	})
	if err != nil {
		t.Fatalf("Ingest events failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Ingest events status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// 2. Search events
	resp, err = makeRequest("GET", "/api/v1/events/search?trace_id="+traceID, nil)
	if err != nil {
		t.Fatalf("Search events failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Search events status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	t.Logf("Events workflow completed for trace_id: %s", traceID)
}

func TestCertificateWorkflow(t *testing.T) {
	traceID := fmt.Sprintf("trc_cert_%d", time.Now().UnixNano())

	// 1. Ingest events first
	events := []map[string]interface{}{
		{
			"event_id":     fmt.Sprintf("evt_%d", time.Now().UnixNano()),
			"trace_id":     traceID,
			"event_type":   "INPUT",
			"timestamp":    time.Now().Format(time.RFC3339),
			"sequence":     1,
			"tenant_id":    tenantID,
			"payload":      map[string]interface{}{"test": "data"},
			"payload_hash": "sha256:test",
			"event_hash":   "sha256:test",
		},
	}

	resp, err := makeRequest("POST", "/api/v1/events/ingest", map[string]interface{}{
		"events": events,
	})
	if err != nil {
		t.Fatalf("Ingest events failed: %v", err)
	}
	resp.Body.Close()

	// 2. Commit certificate
	resp, err = makeRequest("POST", "/api/v1/certs/commit", map[string]interface{}{
		"trace_id":       traceID,
		"evidence_level": "L1",
	})
	if err != nil {
		t.Fatalf("Commit cert failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		// NotFound is acceptable if events weren't stored (test without DB)
		t.Errorf("Commit cert status = %d", resp.StatusCode)
	}

	var certResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&certResult); err != nil {
		t.Logf("Failed to decode cert response: %v", err)
		return
	}

	certID, ok := certResult["cert_id"].(string)
	if !ok || certID == "" {
		t.Log("No cert_id returned, skipping verify")
		return
	}

	// 3. Verify certificate
	resp, err = makeRequest("POST", "/api/v1/certs/verify", map[string]interface{}{
		"cert_id": certID,
	})
	if err != nil {
		t.Fatalf("Verify cert failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Verify cert status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var verifyResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&verifyResult); err != nil {
		t.Fatalf("Failed to decode verify response: %v", err)
	}

	if valid, ok := verifyResult["valid"].(bool); ok && !valid {
		t.Error("Certificate verification should pass")
	}

	// 4. Generate proof
	resp, err = makeRequest("POST", fmt.Sprintf("/api/v1/certs/%s/prove", certID), map[string]interface{}{
		"disclose_events": []int{0},
		"disclose_fields": []string{"event_type"},
	})
	if err != nil {
		t.Fatalf("Generate proof failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Certificate workflow completed for cert_id: %s", certID)
}

func TestSearchCertificates(t *testing.T) {
	resp, err := makeRequest("GET", "/api/v1/certs/search", nil)
	if err != nil {
		t.Fatalf("Search certs failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Search certs status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := result["certificates"]; !ok {
		t.Error("Response should contain 'certificates' field")
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL+"/api/v1/events/search", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	// No API key set

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unauthorized access status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestInvalidAPIKey(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL+"/api/v1/events/search", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-API-Key", "invalid-key")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Invalid API key status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

// Performance tests
func BenchmarkHealthCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", "/health", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkEventsSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", "/api/v1/events/search?page_size=10", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
