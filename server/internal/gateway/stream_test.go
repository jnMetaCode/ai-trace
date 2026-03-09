package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/event"
	"github.com/ai-trace/server/internal/store"
	"go.uber.org/zap"
)

// mockSSEServer 创建模拟SSE服务器
func mockSSEServer(chunks []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i, content := range chunks {
			chunk := StreamChunk{
				ID:      fmt.Sprintf("chunk-%d", i),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   "gpt-4",
				Choices: []StreamChoice{
					{
						Index: 0,
						Delta: StreamDelta{
							Content: content,
						},
					},
				},
			}

			// 最后一个chunk设置finish_reason
			if i == len(chunks)-1 {
				chunk.Choices[0].FinishReason = "stop"
			}

			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond) // 模拟延迟
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
}

func TestProxyStreamingChat(t *testing.T) {
	// 创建模拟服务器
	chunks := []string{"Hello", " ", "World", "!"}
	server := mockSSEServer(chunks)
	defer server.Close()

	// 创建logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// 创建Gateway配置
	cfg := config.GatewayConfig{
		Timeout: 30,
		OpenAI: config.OpenAIConfig{
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
	}

	// 创建Gateway
	gw := New(cfg, &store.Stores{}, sugar)

	// 创建请求
	req := &ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Say hello"},
		},
		Stream: true,
	}

	traceCtx := &TraceContext{
		TenantID: "test-tenant",
		UserID:   "test-user",
	}

	// 收集chunks
	var receivedChunks []*ChunkEvent
	callback := func(chunk *ChunkEvent) {
		receivedChunks = append(receivedChunks, chunk)
	}

	// 执行流式请求
	ctx := context.Background()
	result, err := gw.ProxyStreamingChat(ctx, req, traceCtx, callback)

	if err != nil {
		t.Fatalf("ProxyStreamingChat failed: %v", err)
	}

	// 验证结果
	if result.TraceID == "" {
		t.Error("TraceID should not be empty")
	}

	if len(receivedChunks) != len(chunks) {
		t.Errorf("Expected %d chunks, got %d", len(chunks), len(receivedChunks))
	}

	// 验证内容
	expectedContent := "Hello World!"
	if result.TotalContent != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, result.TotalContent)
	}

	// 验证累积哈希链
	if result.Session.FinalHash == "" {
		t.Error("FinalHash should not be empty")
	}

	// 验证事件类型
	for _, evt := range result.Events {
		switch evt.EventType {
		case event.EventTypeInput, event.EventTypeModel, event.EventTypeChunk, event.EventTypeOutput:
			// OK
		default:
			t.Errorf("Unexpected event type: %s", evt.EventType)
		}
	}
}

func TestChunkHashChain(t *testing.T) {
	// 验证累积哈希的正确性
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	cfg := config.GatewayConfig{
		Timeout: 30,
	}

	gw := New(cfg, &store.Stores{}, sugar)

	// 模拟创建多个chunk事件
	traceCtx := &TraceContext{
		TraceID:  "test-trace",
		TenantID: "test-tenant",
	}

	prevHash := "prev-hash"
	cumulativeHash := ""

	chunks := []string{"Hello", " ", "World"}
	var events []*event.Event

	for i, content := range chunks {
		chunkHash := fmt.Sprintf("hash-%d", i)
		if cumulativeHash == "" {
			cumulativeHash = chunkHash
		} else {
			cumulativeHash = fmt.Sprintf("cumulative-%d", i)
		}

		evt := gw.createChunkEvent(
			i,
			content,
			chunkHash,
			cumulativeHash,
			prevHash,
			int64(i*10),
			"",
			"gpt-4",
			traceCtx,
		)
		events = append(events, evt)
		prevHash = evt.EventHash
	}

	// 验证事件链
	for i := 1; i < len(events); i++ {
		if events[i].PrevEventHash != events[i-1].EventHash {
			t.Errorf("Event chain broken at index %d", i)
		}
	}

	// 验证每个事件都有唯一的hash
	hashSet := make(map[string]bool)
	for _, evt := range events {
		if hashSet[evt.EventHash] {
			t.Errorf("Duplicate event hash: %s", evt.EventHash)
		}
		hashSet[evt.EventHash] = true
	}
}

func TestChunkPayload(t *testing.T) {
	// 验证ChunkPayload结构
	payload := event.ChunkPayload{
		ChunkIndex:     0,
		ChunkHash:      "test-hash",
		CumulativeHash: "cumulative-hash",
		ContentLength:  10,
		TokenCount:     5,
		LatencyMs:      100,
		FinishReason:   "",
		Model:          "gpt-4",
	}

	// 序列化
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal ChunkPayload: %v", err)
	}

	// 反序列化
	var decoded event.ChunkPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ChunkPayload: %v", err)
	}

	// 验证字段
	if decoded.ChunkIndex != payload.ChunkIndex {
		t.Errorf("ChunkIndex mismatch: %d != %d", decoded.ChunkIndex, payload.ChunkIndex)
	}
	if decoded.CumulativeHash != payload.CumulativeHash {
		t.Errorf("CumulativeHash mismatch: %s != %s", decoded.CumulativeHash, payload.CumulativeHash)
	}
}

func TestStreamSession(t *testing.T) {
	session := event.StreamSession{
		SessionID:       "ss-123",
		TraceID:         "trc-456",
		StartTime:       time.Now(),
		ChunkCount:      10,
		TotalTokens:     100,
		TotalContentLen: 500,
		FinalHash:       "final-hash",
		FirstChunkMs:    50,
		TotalMs:         1000,
	}

	// 序列化
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal StreamSession: %v", err)
	}

	// 反序列化
	var decoded event.StreamSession
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal StreamSession: %v", err)
	}

	// 验证字段
	if decoded.ChunkCount != session.ChunkCount {
		t.Errorf("ChunkCount mismatch: %d != %d", decoded.ChunkCount, session.ChunkCount)
	}
	if decoded.FinalHash != session.FinalHash {
		t.Errorf("FinalHash mismatch: %s != %s", decoded.FinalHash, session.FinalHash)
	}
}
