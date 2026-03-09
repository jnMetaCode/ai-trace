// Package gateway 提供 LLM 代理网关功能
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// LLMProvider LLM 提供商接口
// 实现开闭原则：新增提供商只需实现此接口并注册
type LLMProvider interface {
	// Name 返回提供商名称
	Name() string

	// Match 判断模型是否属于此提供商
	Match(model string) bool

	// Priority 返回匹配优先级（数字越小优先级越高）
	Priority() int

	// BuildRequest 构建 HTTP 请求
	BuildRequest(ctx context.Context, baseReq *ChatCompletionRequest, traceCtx *TraceContext) (*http.Request, error)

	// ParseResponse 解析响应
	ParseResponse(resp *http.Response) (*ChatCompletionResponse, error)

	// SupportsStreaming 是否支持流式响应
	SupportsStreaming() bool
}

// ProviderRegistry 提供商注册表
type ProviderRegistry struct {
	providers []LLMProvider
	mu        sync.RWMutex
}

// NewProviderRegistry 创建注册表
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make([]LLMProvider, 0),
	}
}

// Register 注册提供商
func (r *ProviderRegistry) Register(provider LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = append(r.providers, provider)

	// 按优先级排序
	for i := len(r.providers) - 1; i > 0; i-- {
		if r.providers[i].Priority() < r.providers[i-1].Priority() {
			r.providers[i], r.providers[i-1] = r.providers[i-1], r.providers[i]
		}
	}
}

// GetProvider 获取匹配的提供商
func (r *ProviderRegistry) GetProvider(model string) LLMProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.providers {
		if p.Match(model) {
			return p
		}
	}
	return nil
}

// ListProviders 列出所有提供商
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}

// ============================================================================
// 内置提供商实现
// ============================================================================

// OpenAIProvider OpenAI 提供商
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
}

func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

func (p *OpenAIProvider) Name() string     { return "openai" }
func (p *OpenAIProvider) Priority() int    { return 100 } // 最低优先级（默认）
func (p *OpenAIProvider) SupportsStreaming() bool { return true }

func (p *OpenAIProvider) Match(model string) bool {
	// OpenAI 作为默认提供商，匹配所有未被其他提供商匹配的模型
	prefixes := []string{"gpt-", "o1-", "text-", "davinci", "curie", "babbage", "ada"}
	modelLower := strings.ToLower(model)
	for _, prefix := range prefixes {
		if strings.HasPrefix(modelLower, prefix) {
			return true
		}
	}
	return false
}

func (p *OpenAIProvider) BuildRequest(ctx context.Context, req *ChatCompletionRequest, traceCtx *TraceContext) (*http.Request, error) {
	baseURL := p.BaseURL
	if traceCtx.UpstreamBaseURL != "" {
		baseURL = traceCtx.UpstreamBaseURL
	}
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(baseURL, "/"))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	apiKey := traceCtx.UpstreamAPIKey
	if apiKey == "" {
		apiKey = p.APIKey
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return httpReq, nil
}

func (p *OpenAIProvider) ParseResponse(resp *http.Response) (*ChatCompletionResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

// ClaudeProvider Claude/Anthropic 提供商
type ClaudeProvider struct {
	BaseURL string
}

func NewClaudeProvider(baseURL string) *ClaudeProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &ClaudeProvider{BaseURL: baseURL}
}

func (p *ClaudeProvider) Name() string     { return "anthropic" }
func (p *ClaudeProvider) Priority() int    { return 10 }
func (p *ClaudeProvider) SupportsStreaming() bool { return true }

func (p *ClaudeProvider) Match(model string) bool {
	return strings.Contains(strings.ToLower(model), "claude")
}

func (p *ClaudeProvider) BuildRequest(ctx context.Context, req *ChatCompletionRequest, traceCtx *TraceContext) (*http.Request, error) {
	baseURL := p.BaseURL
	if traceCtx.UpstreamBaseURL != "" {
		baseURL = traceCtx.UpstreamBaseURL
	}
	url := fmt.Sprintf("%s/messages", strings.TrimSuffix(baseURL, "/"))

	// 转换为 Claude API 格式
	claudeReq := convertToClaudeRequest(req)
	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", traceCtx.UpstreamAPIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	return httpReq, nil
}

func (p *ClaudeProvider) ParseResponse(resp *http.Response) (*ChatCompletionResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析 Claude 响应并转换为标准格式
	return convertFromClaudeResponse(body)
}

// DeepSeekProvider DeepSeek 提供商
type DeepSeekProvider struct {
	BaseURL string
}

func NewDeepSeekProvider(baseURL string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	return &DeepSeekProvider{BaseURL: baseURL}
}

func (p *DeepSeekProvider) Name() string     { return "deepseek" }
func (p *DeepSeekProvider) Priority() int    { return 20 }
func (p *DeepSeekProvider) SupportsStreaming() bool { return true }

func (p *DeepSeekProvider) Match(model string) bool {
	return strings.Contains(strings.ToLower(model), "deepseek")
}

func (p *DeepSeekProvider) BuildRequest(ctx context.Context, req *ChatCompletionRequest, traceCtx *TraceContext) (*http.Request, error) {
	baseURL := p.BaseURL
	if traceCtx.UpstreamBaseURL != "" {
		baseURL = traceCtx.UpstreamBaseURL
	}
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(baseURL, "/"))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", traceCtx.UpstreamAPIKey))

	return httpReq, nil
}

func (p *DeepSeekProvider) ParseResponse(resp *http.Response) (*ChatCompletionResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	return &chatResp, nil
}

// OllamaProvider Ollama 本地模型提供商
type OllamaProvider struct {
	BaseURL string
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{BaseURL: baseURL}
}

func (p *OllamaProvider) Name() string     { return "ollama" }
func (p *OllamaProvider) Priority() int    { return 15 }
func (p *OllamaProvider) SupportsStreaming() bool { return true }

func (p *OllamaProvider) Match(model string) bool {
	ollamaModels := []string{"llama", "mistral", "qwen", "codellama", "phi", "gemma", "mixtral"}
	modelLower := strings.ToLower(model)
	for _, m := range ollamaModels {
		if strings.Contains(modelLower, m) {
			return true
		}
	}
	return false
}

func (p *OllamaProvider) BuildRequest(ctx context.Context, req *ChatCompletionRequest, traceCtx *TraceContext) (*http.Request, error) {
	url := fmt.Sprintf("%s/api/chat", strings.TrimSuffix(p.BaseURL, "/"))

	// 转换为 Ollama 格式
	ollamaReq := convertToOllamaRequest(req)
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (p *OllamaProvider) ParseResponse(resp *http.Response) (*ChatCompletionResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return convertFromOllamaResponse(body)
}

// ============================================================================
// 辅助函数
// ============================================================================

// convertToClaudeRequest 转换为 Claude API 格式
func convertToClaudeRequest(req *ChatCompletionRequest) map[string]interface{} {
	messages := make([]map[string]string, 0)
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	claudeReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": req.MaxTokens,
	}

	if systemPrompt != "" {
		claudeReq["system"] = systemPrompt
	}
	if req.Temperature > 0 {
		claudeReq["temperature"] = req.Temperature
	}

	return claudeReq
}

// convertFromClaudeResponse 从 Claude 响应转换
func convertFromClaudeResponse(body []byte) (*ChatCompletionResponse, error) {
	var claudeResp struct {
		ID           string `json:"id"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, err
	}

	var content string
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &ChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Model:   claudeResp.Model,
		Choices: []Choice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: content},
			FinishReason: claudeResp.StopReason,
		}},
		Usage: Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}, nil
}

// convertToOllamaRequest 转换为 Ollama API 格式
func convertToOllamaRequest(req *ChatCompletionRequest) map[string]interface{} {
	messages := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	return map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
		"options": map[string]interface{}{
			"temperature": req.Temperature,
			"top_p":       req.TopP,
		},
	}
}

// convertFromOllamaResponse 从 Ollama 响应转换
func convertFromOllamaResponse(body []byte) (*ChatCompletionResponse, error) {
	var ollamaResp struct {
		Model   string `json:"model"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done               bool  `json:"done"`
		TotalDuration      int64 `json:"total_duration"`
		PromptEvalCount    int   `json:"prompt_eval_count"`
		EvalCount          int   `json:"eval_count"`
	}

	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, err
	}

	finishReason := "stop"
	if !ollamaResp.Done {
		finishReason = "length"
	}

	return &ChatCompletionResponse{
		ID:      fmt.Sprintf("ollama-%d", ollamaResp.TotalDuration),
		Object:  "chat.completion",
		Model:   ollamaResp.Model,
		Choices: []Choice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: ollamaResp.Message.Content},
			FinishReason: finishReason,
		}},
		Usage: Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// DefaultRegistry 创建默认的提供商注册表
func DefaultRegistry(cfg GatewayConfig) *ProviderRegistry {
	registry := NewProviderRegistry()

	// 注册所有内置提供商
	registry.Register(NewClaudeProvider(""))
	registry.Register(NewOllamaProvider(cfg.Ollama.BaseURL))
	registry.Register(NewDeepSeekProvider(""))
	registry.Register(NewOpenAIProvider(cfg.OpenAI.BaseURL, cfg.OpenAI.APIKey))

	return registry
}

// GatewayConfig 网关配置（简化版，用于 provider.go）
type GatewayConfig struct {
	OpenAI struct {
		BaseURL string
		APIKey  string
	}
	Ollama struct {
		BaseURL string
	}
	Timeout    int
	MaxRetries int
}
