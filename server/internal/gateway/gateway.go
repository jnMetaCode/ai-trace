package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/event"
	"github.com/ai-trace/server/internal/store"
	"github.com/ai-trace/server/pkg/hash"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Gateway LLM代理网关
type Gateway struct {
	config  config.GatewayConfig
	stores  *store.Stores
	logger  *zap.SugaredLogger
	client  *http.Client
}

// New 创建Gateway实例
func New(cfg config.GatewayConfig, stores *store.Stores, logger *zap.SugaredLogger) *Gateway {
	return &Gateway{
		config: cfg,
		stores: stores,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// ChatCompletionRequest OpenAI聊天完成请求
type ChatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Seed        int64           `json:"seed,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse OpenAI聊天完成响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TraceContext 追踪上下文
type TraceContext struct {
	TraceID    string
	TenantID   string
	UserID     string
	SessionID  string
	BusinessID string
	StartTime  time.Time
	// 用户提供的上游API Key（透传，不存储）
	UpstreamAPIKey string
	// 用户自定义的上游URL（允许用户使用自己的代理）
	UpstreamBaseURL string
}

// ProxyResult 代理结果
type ProxyResult struct {
	Response     *ChatCompletionResponse
	Events       []*event.Event
	Error        error
	LatencyMs    int64
}

// ProxyChatCompletion 代理聊天完成请求
func (g *Gateway) ProxyChatCompletion(ctx context.Context, req *ChatCompletionRequest, traceCtx *TraceContext) (*ProxyResult, error) {
	startTime := time.Now()
	result := &ProxyResult{
		Events: make([]*event.Event, 0),
	}

	// 生成trace_id
	if traceCtx.TraceID == "" {
		traceCtx.TraceID = fmt.Sprintf("trc_%s", uuid.New().String()[:8])
	}

	// 1. 创建INPUT事件
	inputEvent := g.createInputEvent(req, traceCtx)
	result.Events = append(result.Events, inputEvent)

	// 2. 创建MODEL事件
	modelEvent := g.createModelEvent(req, traceCtx, inputEvent.EventHash)
	result.Events = append(result.Events, modelEvent)

	// 3. 发送请求到上游
	var upstreamURL string
	var headers map[string]string

	if g.isOllamaModel(req.Model) {
		upstreamURL = fmt.Sprintf("%s/api/chat", g.config.Ollama.BaseURL)
		headers = map[string]string{
			"Content-Type": "application/json",
		}
	} else if g.isClaudeModel(req.Model) {
		// Claude API
		baseURL := "https://api.anthropic.com/v1"
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/messages", baseURL)
		apiKey := traceCtx.UpstreamAPIKey
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		}
	} else if g.isDeepSeekModel(req.Model) {
		// DeepSeek API (OpenAI 兼容)
		baseURL := "https://api.deepseek.com/v1"
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/chat/completions", baseURL)
		apiKey := traceCtx.UpstreamAPIKey
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
	} else {
		// OpenAI API（默认）
		// 优先使用用户提供的上游URL（允许用户用自己的代理）
		baseURL := g.config.OpenAI.BaseURL
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/chat/completions", baseURL)
		// 优先使用用户提供的API Key（透传），否则使用服务端默认Key
		apiKey := traceCtx.UpstreamAPIKey
		if apiKey == "" {
			apiKey = g.config.OpenAI.APIKey
		}
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to marshal request: %w", err)
		return result, result.Error
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result, result.Error
	}

	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	// 发送请求
	resp, err := g.client.Do(httpReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read response: %w", err)
		return result, result.Error
	}

	// 解析响应
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		result.Error = fmt.Errorf("failed to unmarshal response: %w", err)
		return result, result.Error
	}

	result.Response = &chatResp
	result.LatencyMs = time.Since(startTime).Milliseconds()

	// 4. 创建OUTPUT事件
	outputEvent := g.createOutputEvent(&chatResp, traceCtx, modelEvent.EventHash, result.LatencyMs)
	result.Events = append(result.Events, outputEvent)

	return result, nil
}

// createInputEvent 创建INPUT事件
func (g *Gateway) createInputEvent(req *ChatCompletionRequest, traceCtx *TraceContext) *event.Event {
	// 提取用户输入
	var userPrompt string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userPrompt += msg.Content + "\n"
		}
	}

	payload := event.InputPayload{
		PromptHash:   hash.SHA256(userPrompt),
		PromptLength: len(userPrompt),
		RequestParams: event.RequestParams{
			ModelRequested: req.Model,
			Temperature:    req.Temperature,
			MaxTokens:      req.MaxTokens,
			TopP:           req.TopP,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		g.logger.Warnf("Failed to marshal input payload: %v", err)
		payloadBytes = []byte("{}")
	}
	payloadHash := hash.SHA256Bytes(payloadBytes)

	evt := &event.Event{
		EventID:     fmt.Sprintf("evt_%s", uuid.New().String()[:12]),
		TraceID:     traceCtx.TraceID,
		EventType:   event.EventTypeInput,
		Timestamp:   time.Now(),
		Sequence:    1,
		TenantID:    traceCtx.TenantID,
		UserID:      traceCtx.UserID,
		SessionID:   traceCtx.SessionID,
		Context: event.EventContext{
			BusinessID: traceCtx.BusinessID,
		},
		Payload:     payloadBytes,
		PayloadHash: payloadHash,
	}

	// 计算event_hash
	evt.EventHash = g.calculateEventHash(evt)

	return evt
}

// createModelEvent 创建MODEL事件
func (g *Gateway) createModelEvent(req *ChatCompletionRequest, traceCtx *TraceContext, prevHash string) *event.Event {
	provider := "openai"
	if g.isOllamaModel(req.Model) {
		provider = "ollama"
	} else if g.isClaudeModel(req.Model) {
		provider = "anthropic"
	} else if g.isDeepSeekModel(req.Model) {
		provider = "deepseek"
	}

	payload := event.ModelPayload{
		ModelID:       req.Model,
		ModelProvider: provider,
		ActualParams: event.ActualParams{
			Temperature: req.Temperature,
			TopP:        req.TopP,
			MaxTokens:   req.MaxTokens,
			Seed:        req.Seed,
		},
	}

	paramsBytes, err := json.Marshal(payload.ActualParams)
	if err != nil {
		g.logger.Warnf("Failed to marshal model params: %v", err)
		paramsBytes = []byte("{}")
	}
	payload.ParamsHash = hash.SHA256Bytes(paramsBytes)

	// 提取system prompt
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			payload.SystemPromptHash = hash.SHA256(msg.Content)
			break
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		g.logger.Warnf("Failed to marshal model payload: %v", err)
		payloadBytes = []byte("{}")
	}
	payloadHash := hash.SHA256Bytes(payloadBytes)

	evt := &event.Event{
		EventID:       fmt.Sprintf("evt_%s", uuid.New().String()[:12]),
		TraceID:       traceCtx.TraceID,
		PrevEventHash: prevHash,
		EventType:     event.EventTypeModel,
		Timestamp:     time.Now(),
		Sequence:      2,
		TenantID:      traceCtx.TenantID,
		UserID:        traceCtx.UserID,
		Payload:       payloadBytes,
		PayloadHash:   payloadHash,
	}

	evt.EventHash = g.calculateEventHash(evt)

	return evt
}

// createOutputEvent 创建OUTPUT事件
func (g *Gateway) createOutputEvent(resp *ChatCompletionResponse, traceCtx *TraceContext, prevHash string, latencyMs int64) *event.Event {
	var outputContent string
	var finishReason string

	if len(resp.Choices) > 0 {
		outputContent = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
	}

	payload := event.OutputPayload{
		OutputHash:   hash.SHA256(outputContent),
		OutputLength: len(outputContent),
		Usage: event.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		FinishReason: finishReason,
		LatencyMs:    latencyMs,
		SafetyCheck: event.SafetyCheck{
			Passed: true,
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		g.logger.Warnf("Failed to marshal output payload: %v", err)
		payloadBytes = []byte("{}")
	}
	payloadHash := hash.SHA256Bytes(payloadBytes)

	evt := &event.Event{
		EventID:       fmt.Sprintf("evt_%s", uuid.New().String()[:12]),
		TraceID:       traceCtx.TraceID,
		PrevEventHash: prevHash,
		EventType:     event.EventTypeOutput,
		Timestamp:     time.Now(),
		Sequence:      3,
		TenantID:      traceCtx.TenantID,
		UserID:        traceCtx.UserID,
		Payload:       payloadBytes,
		PayloadHash:   payloadHash,
	}

	evt.EventHash = g.calculateEventHash(evt)

	return evt
}

// calculateEventHash 计算事件哈希
func (g *Gateway) calculateEventHash(evt *event.Event) string {
	// 组合关键字段计算哈希
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%s",
		evt.EventID,
		evt.TraceID,
		evt.EventType,
		evt.Timestamp.Format(time.RFC3339Nano),
		evt.Sequence,
		evt.PayloadHash,
	)

	if evt.PrevEventHash != "" {
		data = fmt.Sprintf("%s|%s", data, evt.PrevEventHash)
	}

	return hash.SHA256(data)
}

// isOllamaModel 判断是否是Ollama模型
func (g *Gateway) isOllamaModel(model string) bool {
	ollamaModels := []string{"llama", "mistral", "qwen", "codellama", "phi"}
	modelLower := strings.ToLower(model)
	for _, m := range ollamaModels {
		if strings.Contains(modelLower, m) {
			return true
		}
	}
	return false
}

// isClaudeModel 判断是否是Claude模型
func (g *Gateway) isClaudeModel(model string) bool {
	claudeModels := []string{"claude"}
	modelLower := strings.ToLower(model)
	for _, m := range claudeModels {
		if strings.Contains(modelLower, m) {
			return true
		}
	}
	return false
}

// isDeepSeekModel 判断是否是DeepSeek模型
func (g *Gateway) isDeepSeekModel(model string) bool {
	deepseekModels := []string{"deepseek"}
	modelLower := strings.ToLower(model)
	for _, m := range deepseekModels {
		if strings.Contains(modelLower, m) {
			return true
		}
	}
	return false
}
