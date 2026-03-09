package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// createTestHandler creates a handler for testing
func createTestHandler() *Handler {
	logger := zap.NewNop().Sugar()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "test",
		},
	}

	return &Handler{
		config: cfg,
		stores: nil,
		logger: logger,
	}
}

// createTestHandlerWithStores creates a handler with mock stores
func createTestHandlerWithStores() *Handler {
	h := createTestHandler()
	h.stores = &store.Stores{
		DB:    nil,
		Redis: nil,
		Minio: nil,
	}
	return h
}

// TestHealth tests the basic health endpoint
func TestHealth(t *testing.T) {
	h := createTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)

	h.Health(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("status: got %v, want 'healthy'", response["status"])
	}

	if response["version"] != "0.2.0" {
		t.Errorf("version: got %v, want '0.2.0'", response["version"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("response should include timestamp")
	}
}

// TestLive tests the liveness probe endpoint
func TestLive(t *testing.T) {
	h := createTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/live", nil)

	h.Live(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["alive"] != true {
		t.Error("alive should be true")
	}
}

// TestReadyWithoutStores tests readiness when stores are nil
func TestReadyWithoutStores(t *testing.T) {
	h := createTestHandler()
	h.stores = nil

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ready", nil)

	h.Ready(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["ready"] != false {
		t.Error("ready should be false")
	}
}

// TestReadyWithoutDB tests readiness when DB is nil
func TestReadyWithoutDB(t *testing.T) {
	h := createTestHandlerWithStores()
	h.stores.DB = nil

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/ready", nil)

	h.Ready(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// TestHealthDetailedWithoutStores tests detailed health without stores
func TestHealthDetailedWithoutStores(t *testing.T) {
	h := createTestHandler()
	h.stores = nil

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health/detailed", nil)

	h.HealthDetailed(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != HealthStatusUnhealthy {
		t.Errorf("status: got %v, want %v", response.Status, HealthStatusUnhealthy)
	}

	// All dependencies should be unhealthy
	for name, dep := range response.Dependencies {
		if dep.Status == HealthStatusHealthy {
			t.Errorf("dependency %s should not be healthy without stores", name)
		}
	}
}

// TestHealthDetailedWithEmptyStores tests detailed health with nil DB/Redis/MinIO
func TestHealthDetailedWithEmptyStores(t *testing.T) {
	h := createTestHandlerWithStores()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health/detailed", nil)

	h.HealthDetailed(c)

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// PostgreSQL should be unhealthy (nil)
	if pgStatus, ok := response.Dependencies["postgresql"]; ok {
		if pgStatus.Status != HealthStatusUnhealthy {
			t.Errorf("postgresql status: got %v, want %v", pgStatus.Status, HealthStatusUnhealthy)
		}
	} else {
		t.Error("postgresql dependency should be present")
	}

	// Redis should be unhealthy (nil)
	if redisStatus, ok := response.Dependencies["redis"]; ok {
		if redisStatus.Status != HealthStatusUnhealthy {
			t.Errorf("redis status: got %v, want %v", redisStatus.Status, HealthStatusUnhealthy)
		}
	} else {
		t.Error("redis dependency should be present")
	}

	// MinIO should be degraded (optional)
	if minioStatus, ok := response.Dependencies["minio"]; ok {
		if minioStatus.Status != HealthStatusDegraded {
			t.Errorf("minio status: got %v, want %v", minioStatus.Status, HealthStatusDegraded)
		}
	} else {
		t.Error("minio dependency should be present")
	}
}

// TestHealthStatusConstants tests health status constants
func TestHealthStatusConstants(t *testing.T) {
	statuses := []HealthStatus{
		HealthStatusHealthy,
		HealthStatusDegraded,
		HealthStatusUnhealthy,
	}

	seen := make(map[HealthStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Error("health status should not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate health status: %s", s)
		}
		seen[s] = true
	}
}

// TestDependencyStatusJSON tests DependencyStatus JSON serialization
func TestDependencyStatusJSON(t *testing.T) {
	status := DependencyStatus{
		Status:    HealthStatusHealthy,
		Latency:   "10ms",
		Message:   "Connected",
		LastCheck: time.Now(),
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DependencyStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != status.Status {
		t.Errorf("status: got %v, want %v", decoded.Status, status.Status)
	}
	if decoded.Latency != status.Latency {
		t.Errorf("latency: got %v, want %v", decoded.Latency, status.Latency)
	}
	if decoded.Message != status.Message {
		t.Errorf("message: got %v, want %v", decoded.Message, status.Message)
	}
}

// TestHealthResponseJSON tests HealthResponse JSON serialization
func TestHealthResponseJSON(t *testing.T) {
	response := HealthResponse{
		Status:  HealthStatusDegraded,
		Version: "0.1.0",
		Uptime:  "1h30m",
		Dependencies: map[string]DependencyStatus{
			"postgresql": {Status: HealthStatusHealthy},
			"redis":      {Status: HealthStatusDegraded},
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("status: got %v, want %v", decoded.Status, response.Status)
	}
	if decoded.Version != response.Version {
		t.Errorf("version: got %v, want %v", decoded.Version, response.Version)
	}
	if len(decoded.Dependencies) != len(response.Dependencies) {
		t.Errorf("dependencies count: got %d, want %d",
			len(decoded.Dependencies), len(response.Dependencies))
	}
}

// TestHealthResponseOmitEmpty tests that empty fields are omitted
func TestHealthResponseOmitEmpty(t *testing.T) {
	response := HealthResponse{
		Status:    HealthStatusHealthy,
		Version:   "0.1.0",
		Timestamp: time.Now(),
		// Uptime and Dependencies are empty
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Empty uptime should be omitted
	if containsSubstring(jsonStr, `"uptime":""`) {
		t.Error("empty uptime should be omitted")
	}

	// Empty dependencies should be omitted
	if containsSubstring(jsonStr, `"dependencies":null`) {
		t.Error("null dependencies should be omitted")
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGettingStarted tests the getting-started endpoint
func TestGettingStarted(t *testing.T) {
	h := createTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/getting-started", nil)

	h.GettingStarted(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Check welcome message
	if _, ok := response["welcome"]; !ok {
		t.Error("response should include welcome message")
	}

	// Check version
	if response["version"] != "0.2.0" {
		t.Errorf("version: got %v, want '0.2.0'", response["version"])
	}

	// Check quick_start exists and has steps
	quickStart, ok := response["quick_start"].([]interface{})
	if !ok {
		t.Error("response should include quick_start array")
	} else if len(quickStart) < 3 {
		t.Errorf("quick_start should have at least 3 steps, got %d", len(quickStart))
	}

	// Check api_endpoints exists
	if _, ok := response["api_endpoints"]; !ok {
		t.Error("response should include api_endpoints")
	}

	// Check environment_variables exists
	if _, ok := response["environment_variables"]; !ok {
		t.Error("response should include environment_variables")
	}

	// Check resources exists
	if _, ok := response["resources"]; !ok {
		t.Error("response should include resources")
	}
}
