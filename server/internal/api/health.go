package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// DependencyStatus 依赖服务状态
type DependencyStatus struct {
	Status   HealthStatus `json:"status"`
	Latency  string       `json:"latency,omitempty"`
	Message  string       `json:"message,omitempty"`
	LastCheck time.Time   `json:"last_check"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status       HealthStatus                `json:"status"`
	Version      string                      `json:"version"`
	Uptime       string                      `json:"uptime,omitempty"`
	Dependencies map[string]DependencyStatus `json:"dependencies,omitempty"`
	Timestamp    time.Time                   `json:"timestamp"`
}

var startTime = time.Now()

// Health 基础健康检查
// @Summary 健康检查
// @Description 返回服务的健康状态
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "健康状态"
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	// 确定部署模式
	deployMode := "standard"
	if h.config != nil && h.config.IsSimpleMode() {
		deployMode = "simple"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"version":     "0.2.0",
		"deploy_mode": deployMode,
		"timestamp":   time.Now(),
		"links": map[string]string{
			"docs":        "/swagger/index.html",
			"health":      "/health/detailed",
			"getting_started": "/api/v1/getting-started",
		},
	})
}

// HealthDetailed 详细健康检查
// @Summary 详细健康检查
// @Description 返回服务及所有依赖服务的健康状态
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse "详细健康状态"
// @Failure 503 {object} HealthResponse "服务不健康"
// @Router /health/detailed [get]
func (h *Handler) HealthDetailed(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:       HealthStatusHealthy,
		Version:      "0.2.0",
		Uptime:       time.Since(startTime).String(),
		Dependencies: make(map[string]DependencyStatus),
		Timestamp:    time.Now(),
	}

	// 检查 PostgreSQL
	response.Dependencies["postgresql"] = h.checkPostgres(ctx)

	// 检查 Redis
	response.Dependencies["redis"] = h.checkRedis(ctx)

	// 检查 MinIO
	response.Dependencies["minio"] = h.checkMinio(ctx)

	// 计算总体状态
	unhealthyCount := 0
	degradedCount := 0
	for _, dep := range response.Dependencies {
		if dep.Status == HealthStatusUnhealthy {
			unhealthyCount++
		} else if dep.Status == HealthStatusDegraded {
			degradedCount++
		}
	}

	// 确定总体健康状态
	if unhealthyCount > 0 {
		response.Status = HealthStatusUnhealthy
	} else if degradedCount > 0 {
		response.Status = HealthStatusDegraded
	}

	// 返回适当的 HTTP 状态码
	httpStatus := http.StatusOK
	if response.Status == HealthStatusUnhealthy {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, response)
}

// checkPostgres 检查 PostgreSQL 连接
func (h *Handler) checkPostgres(ctx context.Context) DependencyStatus {
	status := DependencyStatus{
		LastCheck: time.Now(),
	}

	if h.stores == nil || h.stores.DB == nil {
		status.Status = HealthStatusUnhealthy
		status.Message = "Database connection not initialized"
		return status
	}

	start := time.Now()
	err := h.stores.DB.Ping(ctx)
	latency := time.Since(start)
	status.Latency = latency.String()

	if err != nil {
		status.Status = HealthStatusUnhealthy
		status.Message = "Failed to ping database"
		return status
	}

	// 延迟超过 100ms 视为降级
	if latency > 100*time.Millisecond {
		status.Status = HealthStatusDegraded
		status.Message = "High latency detected"
		return status
	}

	status.Status = HealthStatusHealthy
	status.Message = "Connected"
	return status
}

// checkRedis 检查 Redis 连接
func (h *Handler) checkRedis(ctx context.Context) DependencyStatus {
	status := DependencyStatus{
		LastCheck: time.Now(),
	}

	if h.stores == nil || h.stores.Redis == nil {
		status.Status = HealthStatusUnhealthy
		status.Message = "Redis connection not initialized"
		return status
	}

	start := time.Now()
	err := h.stores.Redis.Ping(ctx).Err()
	latency := time.Since(start)
	status.Latency = latency.String()

	if err != nil {
		status.Status = HealthStatusUnhealthy
		status.Message = "Failed to ping Redis"
		return status
	}

	// 延迟超过 50ms 视为降级
	if latency > 50*time.Millisecond {
		status.Status = HealthStatusDegraded
		status.Message = "High latency detected"
		return status
	}

	status.Status = HealthStatusHealthy
	status.Message = "Connected"
	return status
}

// checkMinio 检查 MinIO 连接
func (h *Handler) checkMinio(ctx context.Context) DependencyStatus {
	status := DependencyStatus{
		LastCheck: time.Now(),
	}

	if h.stores == nil || h.stores.Minio == nil {
		status.Status = HealthStatusDegraded
		status.Message = "MinIO connection not initialized (optional)"
		return status
	}

	start := time.Now()
	_, err := h.stores.Minio.ListBuckets(ctx)
	latency := time.Since(start)
	status.Latency = latency.String()

	if err != nil {
		status.Status = HealthStatusUnhealthy
		status.Message = "Failed to list buckets"
		return status
	}

	// 延迟超过 200ms 视为降级
	if latency > 200*time.Millisecond {
		status.Status = HealthStatusDegraded
		status.Message = "High latency detected"
		return status
	}

	status.Status = HealthStatusHealthy
	status.Message = "Connected"
	return status
}

// Ready 就绪检查（用于 Kubernetes readiness probe）
// @Summary 就绪检查
// @Description 检查服务是否准备好接收流量
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "服务就绪"
// @Failure 503 {object} map[string]interface{} "服务未就绪"
// @Router /ready [get]
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 检查核心依赖
	if h.stores == nil || h.stores.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"message": "Database not available",
		})
		return
	}

	// 验证数据库连接
	if err := h.stores.DB.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"message": "Database ping failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// Live 存活检查（用于 Kubernetes liveness probe）
// @Summary 存活检查
// @Description 检查服务是否存活
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "服务存活"
// @Router /live [get]
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

// GettingStarted 快速入门指南
// @Summary 快速入门
// @Description 返回 AI-Trace 快速入门指南，帮助新用户了解如何使用平台
// @Tags System
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Success 200 {object} map[string]interface{} "快速入门指南"
// @Security ApiKeyAuth
// @Router /getting-started [get]
func (h *Handler) GettingStarted(c *gin.Context) {
	// 确定部署模式和基础 URL 提示
	deployMode := "standard"
	if h.config != nil && h.config.IsSimpleMode() {
		deployMode = "simple"
	}

	guide := map[string]interface{}{
		"welcome": "Welcome to AI-Trace - Enterprise AI Decision Audit Platform",
		"version": "0.2.0",
		"deploy_mode": deployMode,
		"quick_start": []map[string]interface{}{
			{
				"step":        1,
				"title":       "Make an AI Request",
				"description": "Send a request through AI-Trace gateway to automatically trace AI decisions",
				"example": `curl -X POST "$AI_TRACE_URL/api/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $AI_TRACE_KEY" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}'`,
				"note": "The response header X-AI-Trace-ID contains your trace ID",
			},
			{
				"step":        2,
				"title":       "Generate Certificate",
				"description": "Create a tamper-proof certificate for your AI trace",
				"example": `curl -X POST "$AI_TRACE_URL/api/v1/certs/commit" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $AI_TRACE_KEY" \
  -d '{"trace_id":"YOUR_TRACE_ID","evidence_level":"internal"}'`,
				"evidence_levels": map[string]string{
					"internal":   "Fast, Ed25519 signed - ideal for development and internal audit",
					"compliance": "WORM storage + TSA timestamp - for SOC2/GDPR/HIPAA compliance",
					"legal":      "Blockchain anchored - for legal disputes and court evidence",
				},
			},
			{
				"step":        3,
				"title":       "Verify Certificate",
				"description": "Verify a certificate's integrity and authenticity",
				"example": `curl -X POST "$AI_TRACE_URL/api/v1/certs/verify" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $AI_TRACE_KEY" \
  -d '{"cert_id":"YOUR_CERT_ID"}'`,
			},
		},
		"api_endpoints": map[string]interface{}{
			"gateway": map[string]string{
				"POST /api/v1/chat/completions": "OpenAI-compatible chat endpoint with automatic tracing",
			},
			"events": map[string]string{
				"POST /api/v1/events/ingest": "Batch ingest events (for SDK integration)",
				"GET /api/v1/events/search":  "Search events by trace_id, event_type, time range",
				"GET /api/v1/events/:id":     "Get single event details",
			},
			"certificates": map[string]string{
				"POST /api/v1/certs/commit":    "Generate certificate for a trace",
				"POST /api/v1/certs/verify":    "Verify certificate integrity",
				"GET /api/v1/certs/search":     "List certificates",
				"GET /api/v1/certs/:id":        "Get certificate details",
				"POST /api/v1/certs/:id/prove": "Generate minimal disclosure proof",
			},
		},
		"environment_variables": map[string]string{
			"AI_TRACE_URL": "Base URL of your AI-Trace server (e.g., http://localhost:8080)",
			"AI_TRACE_KEY": "Your API key for authentication",
		},
		"resources": map[string]string{
			"api_docs":    "/swagger/index.html",
			"health":      "/health/detailed",
			"github":      "https://github.com/ai-trace/server",
		},
	}

	c.JSON(http.StatusOK, guide)
}
