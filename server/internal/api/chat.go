package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/cert"
	"github.com/ai-trace/server/internal/gateway"
	"github.com/ai-trace/server/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ChatCompletions 聊天完成接口（OpenAI兼容）
// @Summary 聊天完成（OpenAI 兼容）
// @Description 代理 OpenAI/Claude/Ollama 聊天完成请求，同时记录追踪事件
// @Description
// @Description ## API Key 透传
// @Description 用户的 API Key 通过以下方式传递，**不会被存储**：
// @Description - `X-Upstream-API-Key` header（推荐）
// @Description - `Authorization: Bearer sk-...` header（兼容模式）
// @Description
// @Description ## 自定义代理
// @Description 可通过 `X-Upstream-Base-URL` 指定自己的代理服务器，避免 IP 封禁风险
// @Tags Gateway
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param X-Upstream-API-Key header string false "上游 API Key（OpenAI/Claude，透传不存储）"
// @Param X-Upstream-Base-URL header string false "自定义上游代理 URL"
// @Param X-Trace-ID header string false "追踪 ID（可选，自动生成）"
// @Param X-Session-ID header string false "会话 ID"
// @Param X-Business-ID header string false "业务 ID"
// @Param request body gateway.ChatCompletionRequest true "聊天请求"
// @Success 200 {object} gateway.ChatCompletionResponse "聊天响应"
// @Header 200 {string} X-Trace-ID "追踪 ID"
// @Header 200 {string} X-Latency-Ms "延迟毫秒数"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /chat/completions [post]
func (h *Handler) ChatCompletions(c *gin.Context) {
	var req gateway.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// 提取用户的上游API Key（透传到OpenAI/Claude，AI-Trace不存储）
	// 支持两种方式：
	// 1. X-Upstream-API-Key header（推荐，明确区分）
	// 2. 标准 Authorization: Bearer sk-... header（兼容性）
	upstreamAPIKey := c.GetHeader("X-Upstream-API-Key")
	if upstreamAPIKey == "" {
		// 从 Authorization header 提取（如果是 sk- 开头的 OpenAI key）
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			key := auth[7:]
			// 只有 sk- 开头的才是 OpenAI key，否则是 AI-Trace 的 key
			if len(key) > 3 && key[:3] == "sk-" {
				upstreamAPIKey = key
			}
		}
	}

	// 提取用户自定义的上游URL（允许用户使用自己的代理，避免封禁风险）
	// 例如: X-Upstream-Base-URL: https://my-proxy.com/v1
	upstreamBaseURL := c.GetHeader("X-Upstream-Base-URL")

	// 构建追踪上下文
	traceCtx := &gateway.TraceContext{
		TraceID:         c.GetHeader("X-Trace-ID"),
		TenantID:        c.GetString("tenant_id"),
		UserID:          c.GetString("user_id"),
		SessionID:       c.GetHeader("X-Session-ID"),
		BusinessID:      c.GetHeader("X-Business-ID"),
		StartTime:       time.Now(),
		UpstreamAPIKey:  upstreamAPIKey,  // 用户的 OpenAI/Claude Key（透传）
		UpstreamBaseURL: upstreamBaseURL, // 用户自定义的代理URL
	}

	// 代理请求
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.Gateway.Timeout)*time.Second)
	defer cancel()

	result, err := h.gateway.ProxyChatCompletion(ctx, &req, traceCtx)
	if err != nil {
		h.logger.Errorf("Proxy failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 计算 token count 用于自动存证评估
	tokenCount := 0
	if result.Response != nil {
		tokenCount = result.Response.Usage.TotalTokens
	}

	// 异步存储事件并评估自动存证（不阻塞响应）
	go func() {
		for _, evt := range result.Events {
			if err := h.storeEvent(evt); err != nil {
				h.logger.Errorf("Failed to store event: %v", err)
			}
		}

		// 评估是否需要自动生成存证
		if h.autoCertEval != nil {
			evalResult := h.autoCertEval.Evaluate(context.Background(), &cert.TraceContext{
				TraceID:    traceCtx.TraceID,
				Model:      req.Model,
				TokenCount: tokenCount,
				TenantID:   traceCtx.TenantID,
				CreatedAt:  traceCtx.StartTime,
			})

			if evalResult.ShouldCert {
				h.logger.Infof("Auto-cert triggered for trace %s: %s (level: %s)",
					traceCtx.TraceID, evalResult.TriggerReason, evalResult.Level)
				// TODO: 异步触发证书生成
			}
		}
	}()

	// 获取输出事件的 payload hash
	var payloadHash string
	if len(result.Events) > 0 {
		payloadHash = result.Events[len(result.Events)-1].PayloadHash
	}

	// 设置人性化的追踪响应头
	middleware.SetTraceHeaders(c, &middleware.TraceResponseHeaders{
		TraceID:      traceCtx.TraceID,
		EventCount:   len(result.Events),
		PayloadHash:  payloadHash,
		EvidenceHint: middleware.BuildCertHint(traceCtx.TraceID, "internal"),
	})

	// 添加延迟信息
	c.Header("X-Latency-Ms", fmt.Sprintf("%d", result.LatencyMs))

	c.JSON(http.StatusOK, result.Response)
}

// ChatCompletionsStream 流式聊天完成接口（SSE + 增量存证）
// @Summary 流式聊天完成（SSE）
// @Description 代理 LLM 流式请求，每个 chunk 实时存证并返回
// @Description
// @Description ## 流式存证特性
// @Description - 每个 chunk 生成独立的 CHUNK 事件
// @Description - 累积哈希保证完整性（任意 chunk 被篡改可检测）
// @Description - 实时返回 chunk 内容和存证信息
// @Tags Gateway
// @Accept json
// @Produce text/event-stream
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param X-Upstream-API-Key header string false "上游 API Key（OpenAI/Claude，透传不存储）"
// @Param X-Upstream-Base-URL header string false "自定义上游代理 URL"
// @Param X-Trace-ID header string false "追踪 ID（可选，自动生成）"
// @Param X-Session-ID header string false "会话 ID"
// @Param X-Business-ID header string false "业务 ID"
// @Param request body gateway.ChatCompletionRequest true "聊天请求"
// @Success 200 {string} string "SSE stream"
// @Header 200 {string} X-Trace-ID "追踪 ID"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /chat/completions/stream [post]
func (h *Handler) ChatCompletionsStream(c *gin.Context) {
	var req gateway.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// 提取用户的上游API Key
	upstreamAPIKey := c.GetHeader("X-Upstream-API-Key")
	if upstreamAPIKey == "" {
		auth := c.GetHeader("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			key := auth[7:]
			if len(key) > 3 && key[:3] == "sk-" {
				upstreamAPIKey = key
			}
		}
	}

	upstreamBaseURL := c.GetHeader("X-Upstream-Base-URL")

	// 构建追踪上下文
	traceCtx := &gateway.TraceContext{
		TraceID:         c.GetHeader("X-Trace-ID"),
		TenantID:        c.GetString("tenant_id"),
		UserID:          c.GetString("user_id"),
		SessionID:       c.GetHeader("X-Session-ID"),
		BusinessID:      c.GetHeader("X-Business-ID"),
		StartTime:       time.Now(),
		UpstreamAPIKey:  upstreamAPIKey,
		UpstreamBaseURL: upstreamBaseURL,
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // 禁用nginx缓冲

	// 创建上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.config.Gateway.Timeout)*time.Second)
	defer cancel()

	// 用于存储所有事件
	var allEvents []*gateway.ChunkEvent

	// 流式代理
	result, err := h.gateway.ProxyStreamingChat(ctx, &req, traceCtx, func(chunk *gateway.ChunkEvent) {
		// 收集事件用于后续存储
		allEvents = append(allEvents, chunk)

		// 构建SSE响应
		sseData := map[string]interface{}{
			"id":      chunk.Event.EventID,
			"index":   chunk.Index,
			"content": chunk.Content,
			"done":    chunk.Done,
			"attestation": map[string]interface{}{
				"event_hash":      chunk.Event.EventHash,
				"payload_hash":    chunk.Event.PayloadHash,
				"cumulative_hash": "", // 从payload解析
				"timestamp":       chunk.Event.Timestamp,
			},
		}

		// 解析payload获取累积哈希
		var payload map[string]interface{}
		if err := json.Unmarshal(chunk.Event.Payload, &payload); err == nil {
			if cumHash, ok := payload["cumulative_hash"].(string); ok {
				sseData["attestation"].(map[string]interface{})["cumulative_hash"] = cumHash
			}
		}

		// 发送SSE事件
		data, _ := json.Marshal(sseData)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	})

	if err != nil {
		h.logger.Errorf("Stream proxy failed: %v", err)
		// 发送错误事件
		errData, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		})
		fmt.Fprintf(c.Writer, "data: %s\n\n", errData)
		c.Writer.Flush()
		return
	}

	// 发送完成事件
	doneData, _ := json.Marshal(map[string]interface{}{
		"done":     true,
		"trace_id": result.TraceID,
		"session": map[string]interface{}{
			"chunk_count":    result.Session.ChunkCount,
			"total_ms":       result.Session.TotalMs,
			"first_chunk_ms": result.Session.FirstChunkMs,
			"final_hash":     result.Session.FinalHash,
		},
	})
	fmt.Fprintf(c.Writer, "data: %s\n\n", doneData)
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()

	// 异步存储所有事件
	go func() {
		for _, evt := range result.Events {
			if err := h.storeEvent(evt); err != nil {
				h.logger.Errorf("Failed to store event: %v", err)
			}
		}
	}()

	// 设置人性化的追踪响应头
	middleware.SetTraceHeaders(c, &middleware.TraceResponseHeaders{
		TraceID:      result.TraceID,
		EventCount:   len(result.Events),
		EvidenceHint: middleware.BuildCertHint(result.TraceID, "internal"),
	})
}
