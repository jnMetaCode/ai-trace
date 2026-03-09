//go:build metrics
// +build metrics

// 此文件仅在使用 -tags metrics 编译时包含
// 编译命令: go build -tags metrics ./...

package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP 请求计数器
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP 请求延迟
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_trace_http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// 事件处理计数器
	eventsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"event_type", "tenant_id"},
	)

	// 证书生成计数器
	certsGeneratedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_certs_generated_total",
			Help: "Total number of certificates generated",
		},
		[]string{"evidence_level", "tenant_id"},
	)

	// LLM 请求计数器
	llmRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_llm_requests_total",
			Help: "Total number of LLM API requests",
		},
		[]string{"model", "provider", "status"},
	)

	// LLM Token 使用量
	llmTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_llm_tokens_total",
			Help: "Total number of tokens used",
		},
		[]string{"model", "type"}, // type: prompt, completion
	)

	// LLM 请求延迟
	llmRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_trace_llm_request_duration_seconds",
			Help:    "LLM API request latencies in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"model", "provider"},
	)

	// 活跃追踪数
	activeTracesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_trace_active_traces",
			Help: "Number of active traces",
		},
		[]string{"tenant_id"},
	)

	// 存储使用量
	storageUsageBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ai_trace_storage_usage_bytes",
			Help: "Storage usage in bytes",
		},
		[]string{"storage_type"}, // minio, postgres
	)

	// 区块链锚定计数器
	anchorOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_trace_anchor_operations_total",
			Help: "Total number of anchor operations",
		},
		[]string{"anchor_type", "status"}, // ethereum, federated; success, failed
	)
)

// Middleware returns a Gin middleware for collecting HTTP metrics
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// Handler returns the Prometheus metrics handler
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordEvent records an event processing
func RecordEvent(eventType, tenantID string) {
	eventsProcessedTotal.WithLabelValues(eventType, tenantID).Inc()
}

// RecordCert records a certificate generation
func RecordCert(evidenceLevel, tenantID string) {
	certsGeneratedTotal.WithLabelValues(evidenceLevel, tenantID).Inc()
}

// RecordLLMRequest records an LLM API request
func RecordLLMRequest(model, provider, status string, duration time.Duration) {
	llmRequestsTotal.WithLabelValues(model, provider, status).Inc()
	llmRequestDuration.WithLabelValues(model, provider).Observe(duration.Seconds())
}

// RecordLLMTokens records token usage
func RecordLLMTokens(model string, promptTokens, completionTokens int) {
	llmTokensTotal.WithLabelValues(model, "prompt").Add(float64(promptTokens))
	llmTokensTotal.WithLabelValues(model, "completion").Add(float64(completionTokens))
}

// SetActiveTraces sets the number of active traces for a tenant
func SetActiveTraces(tenantID string, count float64) {
	activeTracesGauge.WithLabelValues(tenantID).Set(count)
}

// SetStorageUsage sets the storage usage for a storage type
func SetStorageUsage(storageType string, bytes float64) {
	storageUsageBytes.WithLabelValues(storageType).Set(bytes)
}

// RecordAnchorOperation records an anchor operation
func RecordAnchorOperation(anchorType, status string) {
	anchorOperationsTotal.WithLabelValues(anchorType, status).Inc()
}

// IsEnabled returns true if metrics are enabled
func IsEnabled() bool {
	return true
}

// RegisterEndpoint registers the /metrics endpoint
func RegisterEndpoint(r *gin.Engine) {
	r.GET("/metrics", Handler())
}

// Custom HTTP handler for non-Gin usage
func HTTPHandler() http.Handler {
	return promhttp.Handler()
}
