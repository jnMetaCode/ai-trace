package gateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-trace/server/internal/event"
	"github.com/ai-trace/server/internal/fingerprint"
	"github.com/ai-trace/server/pkg/hash"
	"github.com/google/uuid"
)

// StreamChunk 流式响应chunk
type StreamChunk struct {
	ID           string `json:"id"`
	Object       string `json:"object"`
	Created      int64  `json:"created"`
	Model        string `json:"model"`
	Choices      []StreamChoice `json:"choices"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// StreamChoice 流式选择
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// StreamDelta 增量内容
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ChunkEvent 用于回调的chunk事件
type ChunkEvent struct {
	Event     *event.Event      `json:"event"`
	Content   string            `json:"content"`
	Index     int               `json:"index"`
	Done      bool              `json:"done"`
	RawChunk  *StreamChunk      `json:"raw_chunk,omitempty"`
}

// StreamResult 流式代理结果
type StreamResult struct {
	TraceID         string          `json:"trace_id"`
	Events          []*event.Event  `json:"events"`
	Session         *event.StreamSession `json:"session"`
	TotalContent    string          `json:"total_content"`
	Error           error           `json:"error,omitempty"`
}

// ChunkCallback chunk回调函数类型
type ChunkCallback func(*ChunkEvent)

// ProxyStreamingChat 代理流式聊天请求
func (g *Gateway) ProxyStreamingChat(
	ctx context.Context,
	req *ChatCompletionRequest,
	traceCtx *TraceContext,
	callback ChunkCallback,
) (*StreamResult, error) {
	startTime := time.Now()
	result := &StreamResult{
		TraceID: traceCtx.TraceID,
		Events:  make([]*event.Event, 0),
	}

	// 生成trace_id
	if traceCtx.TraceID == "" {
		traceCtx.TraceID = fmt.Sprintf("trc_%s", uuid.New().String()[:8])
		result.TraceID = traceCtx.TraceID
	}

	// 强制开启流式
	req.Stream = true

	// 1. 创建INPUT事件
	inputEvent := g.createInputEvent(req, traceCtx)
	result.Events = append(result.Events, inputEvent)

	// 2. 创建MODEL事件
	modelEvent := g.createModelEvent(req, traceCtx, inputEvent.EventHash)
	result.Events = append(result.Events, modelEvent)

	// 3. 构建上游请求
	upstreamURL, headers := g.buildUpstreamRequest(req, traceCtx)

	// 序列化请求
	reqBody, err := g.marshalRequest(req)
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

	// 4. 发送请求
	resp, err := g.client.Do(httpReq)
	if err != nil {
		result.Error = fmt.Errorf("failed to send request: %w", err)
		return result, result.Error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("upstream error: %d - %s", resp.StatusCode, string(body))
		return result, result.Error
	}

	// 5. 初始化流式会话
	session := &event.StreamSession{
		SessionID: fmt.Sprintf("ss_%s", uuid.New().String()[:8]),
		TraceID:   traceCtx.TraceID,
		StartTime: startTime,
	}

	// 6. 处理SSE流
	var cumulativeHash string
	var totalContent strings.Builder
	var chunkEvents []*event.Event
	var prevChunkHash string
	var firstChunkTime time.Time
	var chunkSizes []int        // 用于指纹采集
	var chunkLatencies []int64  // 用于指纹采集
	var lastChunkTime time.Time
	var modelName string        // 从响应中获取模型名

	reader := bufio.NewReader(resp.Body)
	chunkIndex := 0
	prevEventHash := modelEvent.EventHash

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			g.logger.Warnf("Error reading stream: %v", err)
			break
		}

		// 解析SSE行
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// 跳过非data行
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))

		// 检查是否结束
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		// 解析chunk
		var chunk StreamChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			g.logger.Warnf("Failed to parse chunk: %v", err)
			continue
		}

		// 提取内容
		var content string
		var finishReason string
		if len(chunk.Choices) > 0 {
			content = chunk.Choices[0].Delta.Content
			finishReason = chunk.Choices[0].FinishReason
		}

		// 跳过空内容（除非是结束chunk）
		if content == "" && finishReason == "" {
			continue
		}

		// 记录首chunk时间
		now := time.Now()
		if chunkIndex == 0 {
			firstChunkTime = now
			lastChunkTime = now
			session.FirstChunkMs = firstChunkTime.Sub(startTime).Milliseconds()
		}

		// 记录模型名（从响应获取）
		if modelName == "" && chunk.Model != "" {
			modelName = chunk.Model
		}

		// 计算累积哈希（链式）
		chunkHash := hash.SHA256(content)
		if cumulativeHash == "" {
			cumulativeHash = chunkHash
		} else {
			cumulativeHash = hash.SHA256(cumulativeHash + chunkHash)
		}

		// 累积内容
		totalContent.WriteString(content)

		// 记录 chunk 大小和延迟（用于指纹）
		chunkSizes = append(chunkSizes, len(content))
		var latencyMs int64
		if chunkIndex == 0 {
			latencyMs = session.FirstChunkMs
		} else {
			latencyMs = now.Sub(lastChunkTime).Milliseconds()
		}
		chunkLatencies = append(chunkLatencies, latencyMs)
		lastChunkTime = now

		// 创建CHUNK事件
		chunkEvent := g.createChunkEvent(
			chunkIndex,
			content,
			chunkHash,
			cumulativeHash,
			prevEventHash,
			latencyMs,
			finishReason,
			chunk.Model,
			traceCtx,
		)
		chunkEvents = append(chunkEvents, chunkEvent)
		prevEventHash = chunkEvent.EventHash
		prevChunkHash = chunkHash

		// 回调
		if callback != nil {
			callback(&ChunkEvent{
				Event:    chunkEvent,
				Content:  content,
				Index:    chunkIndex,
				Done:     finishReason != "",
				RawChunk: &chunk,
			})
		}

		chunkIndex++

		// 检查是否结束
		if finishReason != "" {
			break
		}
	}

	// 7. 完成会话统计
	session.ChunkCount = chunkIndex
	session.TotalContentLen = totalContent.Len()
	session.FinalHash = cumulativeHash
	session.TotalMs = time.Since(startTime).Milliseconds()
	result.Session = session
	result.TotalContent = totalContent.String()

	// 提取 prompt 内容
	var promptContent string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			promptContent += msg.Content + "\n"
		}
	}

	// 确定模型提供商
	provider := "openai"
	if g.isOllamaModel(req.Model) {
		provider = "ollama"
	} else if g.isClaudeModel(req.Model) {
		provider = "anthropic"
	} else if g.isDeepSeekModel(req.Model) {
		provider = "deepseek"
	}

	// 8. 创建最终OUTPUT事件（含指纹）
	endTime := time.Now()
	outputData := &StreamOutputData{
		TotalContent:        totalContent.String(),
		FinalCumulativeHash: cumulativeHash,
		PrevEventHash:       prevEventHash,
		Session:             session,
		TraceCtx:            traceCtx,
		ModelID:             modelName,
		ModelProvider:       provider,
		PromptContent:       promptContent,
		ChunkSizes:          chunkSizes,
		ChunkLatencies:      chunkLatencies,
		StartTime:           startTime,
		EndTime:             endTime,
		FirstTokenAt:        firstChunkTime,
	}
	outputEvent := g.createStreamOutputEvent(outputData)
	result.Events = append(result.Events, outputEvent)
	result.Events = append(result.Events, chunkEvents...)

	_ = prevChunkHash // 可用于额外验证

	return result, nil
}

// buildUpstreamRequest 构建上游请求URL和headers
func (g *Gateway) buildUpstreamRequest(req *ChatCompletionRequest, traceCtx *TraceContext) (string, map[string]string) {
	var upstreamURL string
	var headers map[string]string

	if g.isOllamaModel(req.Model) {
		upstreamURL = fmt.Sprintf("%s/api/chat", g.config.Ollama.BaseURL)
		headers = map[string]string{
			"Content-Type": "application/json",
		}
	} else if g.isClaudeModel(req.Model) {
		baseURL := "https://api.anthropic.com/v1"
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/messages", baseURL)
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         traceCtx.UpstreamAPIKey,
			"anthropic-version": "2023-06-01",
		}
	} else if g.isDeepSeekModel(req.Model) {
		baseURL := "https://api.deepseek.com/v1"
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/chat/completions", baseURL)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", traceCtx.UpstreamAPIKey),
		}
	} else {
		// OpenAI API
		baseURL := g.config.OpenAI.BaseURL
		if traceCtx.UpstreamBaseURL != "" {
			baseURL = traceCtx.UpstreamBaseURL
		}
		upstreamURL = fmt.Sprintf("%s/chat/completions", baseURL)
		apiKey := traceCtx.UpstreamAPIKey
		if apiKey == "" {
			apiKey = g.config.OpenAI.APIKey
		}
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
	}

	return upstreamURL, headers
}

// marshalRequest 序列化请求（处理不同模型的格式差异）
func (g *Gateway) marshalRequest(req *ChatCompletionRequest) ([]byte, error) {
	// 对于Ollama，需要转换格式
	if g.isOllamaModel(req.Model) {
		ollamaReq := map[string]interface{}{
			"model":  req.Model,
			"stream": true,
			"messages": req.Messages,
		}
		if req.Temperature > 0 {
			ollamaReq["options"] = map[string]interface{}{
				"temperature": req.Temperature,
			}
		}
		return json.Marshal(ollamaReq)
	}

	// 对于Claude，需要转换格式
	if g.isClaudeModel(req.Model) {
		claudeReq := map[string]interface{}{
			"model":      req.Model,
			"max_tokens": req.MaxTokens,
			"stream":     true,
		}
		// 转换消息格式
		var messages []map[string]string
		for _, msg := range req.Messages {
			if msg.Role == "system" {
				claudeReq["system"] = msg.Content
			} else {
				messages = append(messages, map[string]string{
					"role":    msg.Role,
					"content": msg.Content,
				})
			}
		}
		claudeReq["messages"] = messages
		return json.Marshal(claudeReq)
	}

	// OpenAI/DeepSeek 兼容格式
	return json.Marshal(req)
}

// createChunkEvent 创建CHUNK事件
func (g *Gateway) createChunkEvent(
	index int,
	content string,
	chunkHash string,
	cumulativeHash string,
	prevEventHash string,
	latencyMs int64,
	finishReason string,
	model string,
	traceCtx *TraceContext,
) *event.Event {
	payload := event.ChunkPayload{
		ChunkIndex:     index,
		ChunkHash:      chunkHash,
		CumulativeHash: cumulativeHash,
		ContentLength:  len(content),
		LatencyMs:      latencyMs,
		FinishReason:   finishReason,
		Model:          model,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		// 如果序列化失败，使用空 payload
		payloadBytes = []byte("{}")
	}
	payloadHash := hash.SHA256Bytes(payloadBytes)

	evt := &event.Event{
		EventID:       fmt.Sprintf("evt_%s", uuid.New().String()[:12]),
		TraceID:       traceCtx.TraceID,
		PrevEventHash: prevEventHash,
		EventType:     event.EventTypeChunk,
		Timestamp:     time.Now(),
		Sequence:      index + 3, // INPUT=1, MODEL=2, CHUNK从3开始
		TenantID:      traceCtx.TenantID,
		UserID:        traceCtx.UserID,
		SessionID:     traceCtx.SessionID,
		Payload:       payloadBytes,
		PayloadHash:   payloadHash,
	}

	evt.EventHash = g.calculateEventHash(evt)
	return evt
}

// StreamOutputData 创建流式 OUTPUT 事件所需的数据
type StreamOutputData struct {
	TotalContent        string
	FinalCumulativeHash string
	PrevEventHash       string
	Session             *event.StreamSession
	TraceCtx            *TraceContext
	// 用于指纹采集的额外数据
	ModelID       string
	ModelProvider string
	PromptContent string
	ChunkSizes    []int
	ChunkLatencies []int64
	StartTime     time.Time
	EndTime       time.Time
	FirstTokenAt  time.Time
}

// createStreamOutputEvent 创建流式OUTPUT事件（含推理行为指纹）
func (g *Gateway) createStreamOutputEvent(data *StreamOutputData) *event.Event {
	payload := event.OutputPayload{
		OutputHash:   hash.SHA256(data.TotalContent),
		OutputLength: len(data.TotalContent),
		Usage: event.TokenUsage{
			CompletionTokens: data.Session.TotalTokens,
		},
		FinishReason: "stop",
		LatencyMs:    data.Session.TotalMs,
		SafetyCheck: event.SafetyCheck{
			Passed: true,
		},
	}

	// 采集推理行为指纹
	collector := fingerprint.NewDefaultCollector()
	fpData := fingerprint.BuildCollectionData(
		data.ModelID,
		data.ModelProvider,
		data.PromptContent,
		data.TotalContent,
		0, // prompt tokens (如果可获取)
		data.Session.TotalTokens,
		data.StartTime,
		data.EndTime,
		data.FirstTokenAt,
		data.ChunkSizes,
		data.ChunkLatencies,
		"stop",
	)

	fp, fpErr := collector.QuickFingerprint(fpData)
	if fpErr == nil && fp != nil {
		if fpBytes, marshalErr := json.Marshal(fp); marshalErr == nil {
			payload.InferenceFingerprint = fpBytes
			payload.FingerprintHash = fp.FingerprintHash
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		payloadBytes = []byte("{}")
	}
	payloadHash := hash.SHA256Bytes(payloadBytes)

	evt := &event.Event{
		EventID:       fmt.Sprintf("evt_%s", uuid.New().String()[:12]),
		TraceID:       data.TraceCtx.TraceID,
		PrevEventHash: data.PrevEventHash,
		EventType:     event.EventTypeOutput,
		Timestamp:     time.Now(),
		Sequence:      data.Session.ChunkCount + 3, // 最后一个序号
		TenantID:      data.TraceCtx.TenantID,
		UserID:        data.TraceCtx.UserID,
		SessionID:     data.TraceCtx.SessionID,
		Payload:       payloadBytes,
		PayloadHash:   payloadHash,
	}

	evt.EventHash = g.calculateEventHash(evt)
	return evt
}
