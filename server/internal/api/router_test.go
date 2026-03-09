package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/gateway"
	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// createTestRouter creates a router for testing
func createTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "test",
		},
		Auth: config.AuthConfig{
			APIKeys: []string{"test-api-key", "another-key"},
		},
		Features: config.FeatureConfig{
			Metrics: false,
			Reports: false,
		},
		Minio: config.MinioConfig{
			Bucket: "test-bucket",
		},
	}

	stores := &store.Stores{
		DB:    nil,
		Redis: nil,
		Minio: nil,
	}

	logger := zap.NewNop().Sugar()
	gw := gateway.New(cfg.Gateway, stores, logger)

	return NewRouter(cfg, stores, gw, logger)
}

// TestNewRouter tests router creation
func TestNewRouter(t *testing.T) {
	router := createTestRouter()

	if router == nil {
		t.Fatal("router should not be nil")
	}
}

// TestHealthRoutes tests health check routes
func TestHealthRoutes(t *testing.T) {
	router := createTestRouter()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"health", "/health", http.StatusOK},
		{"live", "/live", http.StatusOK},
		{"ready without db", "/ready", http.StatusServiceUnavailable},
		{"health detailed without db", "/health/detailed", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestAuthMiddlewareWithAPIKey tests API key authentication
func TestAuthMiddlewareWithAPIKey(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{"test-api-key", "another-key"}

	tests := []struct {
		name      string
		apiKey    string
		wantAbort bool
	}{
		{"valid key", "test-api-key", false},
		{"another valid key", "another-key", false},
		{"invalid key", "wrong-key", true},
		{"empty key", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				c.Request.Header.Set("X-API-Key", tt.apiKey)
			}

			h.AuthMiddleware()(c)

			if tt.wantAbort {
				if w.Code != http.StatusUnauthorized {
					t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
				}
			} else {
				if w.Code == http.StatusUnauthorized {
					t.Error("should not get 401 with valid API key")
				}
			}
		})
	}
}

// TestAuthMiddlewareWithBearerToken tests Bearer token authentication
func TestAuthMiddlewareWithBearerToken(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{"test-api-key"}

	tests := []struct {
		name       string
		authHeader string
		wantAbort  bool
	}{
		{"valid bearer", "Bearer test-api-key", false},
		{"invalid bearer", "Bearer wrong-key", true},
		{"bearer no space", "Bearertest-api-key", true},
		{"bearer only", "Bearer ", true},
		{"just Bearer", "Bearer", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Authorization", tt.authHeader)

			h.AuthMiddleware()(c)

			gotAbort := w.Code == http.StatusUnauthorized
			if gotAbort != tt.wantAbort {
				t.Errorf("unauthorized: got %v, want %v (status %d)",
					gotAbort, tt.wantAbort, w.Code)
			}
		})
	}
}

// TestAuthMiddlewareNoKeys tests auth when no API keys configured
func TestAuthMiddlewareNoKeys(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{} // No API keys

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	// No API key header set

	h.AuthMiddleware()(c)

	// Without API keys configured, all requests should pass auth
	if w.Code == http.StatusUnauthorized {
		t.Error("should not require auth when no API keys configured")
	}
}

// TestTenantIDHeader tests X-Tenant-ID header handling
func TestTenantIDHeader(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{"test-key"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-API-Key", "test-key")
	c.Request.Header.Set("X-Tenant-ID", "custom-tenant")

	h.AuthMiddleware()(c)

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		t.Fatal("tenant_id should be set in context")
	}
	if tenantID != "custom-tenant" {
		t.Errorf("tenant_id: got %v, want 'custom-tenant'", tenantID)
	}
}

// TestTenantIDDefault tests default tenant ID
func TestTenantIDDefault(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{"test-key"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-API-Key", "test-key")
	// No X-Tenant-ID header

	h.AuthMiddleware()(c)

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		t.Fatal("tenant_id should be set in context")
	}
	if tenantID != "default" {
		t.Errorf("tenant_id: got %v, want 'default'", tenantID)
	}
}

// TestUserIDHeader tests X-User-ID header handling
func TestUserIDHeader(t *testing.T) {
	h := createTestHandler()
	h.config.Auth.APIKeys = []string{"test-key"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-API-Key", "test-key")
	c.Request.Header.Set("X-User-ID", "user-123")

	h.AuthMiddleware()(c)

	userID, exists := c.Get("user_id")
	if !exists {
		t.Fatal("user_id should be set in context")
	}
	if userID != "user-123" {
		t.Errorf("user_id: got %v, want 'user-123'", userID)
	}
}

// TestSwaggerRoute tests swagger endpoint
func TestSwaggerRoute(t *testing.T) {
	router := createTestRouter()

	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Swagger should return 200 or redirect
	if w.Code != http.StatusOK && w.Code != http.StatusMovedPermanently &&
		w.Code != http.StatusFound {
		t.Errorf("swagger status: got %d, want 200/301/302", w.Code)
	}
}

// TestAPIv1RoutesExist tests that v1 API routes are registered
func TestAPIv1RoutesExist(t *testing.T) {
	router := createTestRouter()

	// Get all registered routes
	routes := router.Routes()

	expectedPaths := []string{
		"/api/v1/chat/completions",
		"/api/v1/events/ingest",
		"/api/v1/events/search",
		"/api/v1/events/:event_id",
		"/api/v1/certs/commit",
		"/api/v1/certs/verify",
		"/api/v1/certs/search",
		"/api/v1/certs/:cert_id",
		"/api/v1/certs/:cert_id/prove",
		"/api/v1/reports/generate",
		"/api/v1/fingerprints/:trace_id",
		"/api/v1/fingerprints/compare",
		"/api/v1/fingerprints/verify",
		"/api/v1/decrypt",
		"/api/v1/decrypt/audit",
	}

	registeredPaths := make(map[string]bool)
	for _, r := range routes {
		registeredPaths[r.Path] = true
	}

	for _, path := range expectedPaths {
		if !registeredPaths[path] {
			t.Errorf("route %s should be registered", path)
		}
	}
}

// TestReleaseModeRouter tests router in release mode
func TestReleaseModeRouter(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Mode: "release",
		},
		Auth:     config.AuthConfig{},
		Features: config.FeatureConfig{},
		Minio:    config.MinioConfig{Bucket: "test"},
	}

	stores := &store.Stores{}
	logger := zap.NewNop().Sugar()
	gw := gateway.New(cfg.Gateway, stores, logger)

	router := NewRouter(cfg, stores, gw, logger)

	if router == nil {
		t.Fatal("router should not be nil in release mode")
	}

	// Health should still work
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// TestCORSHeaders tests CORS middleware
func TestCORSHeaders(t *testing.T) {
	router := createTestRouter()

	req := httptest.NewRequest("OPTIONS", "/api/v1/events/search", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should have CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS Allow-Origin header should be set")
	}
}

// TestRequestIDMiddleware tests request ID generation
func TestRequestIDMiddleware(t *testing.T) {
	router := createTestRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Request ID should be in response header
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID header should be set")
	}
}

// TestSecurityHeaders tests security headers
func TestSecurityHeaders(t *testing.T) {
	router := createTestRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	expectedHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
	}

	for _, header := range expectedHeaders {
		if w.Header().Get(header) == "" {
			t.Errorf("security header %s should be set", header)
		}
	}
}

// TestHandlerStruct tests Handler struct fields
func TestHandlerStruct(t *testing.T) {
	h := createTestHandler()

	if h.config == nil {
		t.Error("config should not be nil")
	}
	if h.logger == nil {
		t.Error("logger should not be nil")
	}
}

// TestHealthResponseStatusCodes tests HTTP status codes for health
func TestHealthResponseStatusCodes(t *testing.T) {
	router := createTestRouter()

	// Basic health should always be 200
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if w.Code != http.StatusOK {
		t.Errorf("health status: got %d, want %d", w.Code, http.StatusOK)
	}

	// Detailed health should be 503 when deps unavailable
	req = httptest.NewRequest("GET", "/health/detailed", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("detailed health status: got %d, want %d",
			w.Code, http.StatusServiceUnavailable)
	}
}
