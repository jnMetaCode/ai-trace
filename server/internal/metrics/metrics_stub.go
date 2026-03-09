//go:build !metrics
// +build !metrics

// 此文件在不使用 metrics tag 时编译
// 提供空实现，避免引入 prometheus 依赖

package metrics

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware returns a no-op middleware when metrics are disabled
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// Handler returns a handler that indicates metrics are not enabled
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Metrics not enabled",
			"message": "Build with: go build -tags metrics",
		})
	}
}

// RecordEvent is a no-op when metrics are disabled
func RecordEvent(eventType, tenantID string) {}

// RecordCert is a no-op when metrics are disabled
func RecordCert(evidenceLevel, tenantID string) {}

// RecordLLMRequest is a no-op when metrics are disabled
func RecordLLMRequest(model, provider, status string, duration time.Duration) {}

// RecordLLMTokens is a no-op when metrics are disabled
func RecordLLMTokens(model string, promptTokens, completionTokens int) {}

// SetActiveTraces is a no-op when metrics are disabled
func SetActiveTraces(tenantID string, count float64) {}

// SetStorageUsage is a no-op when metrics are disabled
func SetStorageUsage(storageType string, bytes float64) {}

// RecordAnchorOperation is a no-op when metrics are disabled
func RecordAnchorOperation(anchorType, status string) {}

// IsEnabled returns false when metrics are disabled
func IsEnabled() bool {
	return false
}

// RegisterEndpoint registers the /metrics endpoint (returns unavailable message)
func RegisterEndpoint(r *gin.Engine) {
	r.GET("/metrics", Handler())
}

// HTTPHandler returns a handler that returns an error
func HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Metrics not enabled","message":"Build with: go build -tags metrics"}`))
	})
}
