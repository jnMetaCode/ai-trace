package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	// Check security headers
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range headers {
		if got := w.Header().Get(header); got != expectedValue {
			t.Errorf("Header %s = %q, want %q", header, got, expectedValue)
		}
	}
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Test preflight request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS request status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Check CORS headers - with * in AllowOrigins, it echoes back the actual origin
	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}

	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods header not set")
	}

	// Test regular request
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET request status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequestID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		c.String(http.StatusOK, requestID)
	})

	// Test without existing request ID
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Response should contain a request ID
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header not set")
	}

	if w.Body.String() == "" {
		t.Error("request_id not set in context")
	}

	// Test with existing request ID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Request-ID"); got != "custom-request-id" {
		t.Errorf("X-Request-ID = %q, want %q", got, "custom-request-id")
	}
}

func TestRateLimiter(t *testing.T) {
	cfg := RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(cfg)

	// Should allow initial burst
	for i := 0; i < 2; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should block after burst exceeded (without waiting for token refill)
	if limiter.Allow("test-key") {
		t.Error("Request should be blocked after burst exceeded")
	}

	// Different key should have its own bucket
	if !limiter.Allow("other-key") {
		t.Error("Request from different key should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         2,
	}
	limiter := NewRateLimiter(cfg)

	r := gin.New()
	r.Use(RateLimit(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Rate limited request status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	cfg := RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		CleanupInterval:   100 * time.Millisecond,
	}
	limiter := NewRateLimiter(cfg)

	// Create some buckets
	limiter.Allow("key1")
	limiter.Allow("key2")
	limiter.Allow("key3")

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Cleanup should have removed inactive buckets
	// (Note: This is a basic test, actual cleanup logic depends on implementation)

	// New requests should still work
	if !limiter.Allow("key4") {
		t.Error("Request should be allowed after cleanup")
	}
}

func TestRateLimiterAPIKeyPriority(t *testing.T) {
	cfg := RateLimiterConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
	}
	limiter := NewRateLimiter(cfg)

	r := gin.New()
	r.Use(RateLimit(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request with API key
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-API-Key", "test-api-key")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request with API key status = %d, want %d", w.Code, http.StatusOK)
	}

	// Same IP without API key should have different bucket
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request without API key status = %d, want %d", w.Code, http.StatusOK)
	}
}
