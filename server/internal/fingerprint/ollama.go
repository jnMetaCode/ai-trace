package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
)

// OllamaCollector Layer 3 Ollama 模型内部特征采集器
type OllamaCollector struct {
	// 采样的 token 延迟模式数量
	MaxLatencyPatternSize int
}

// NewOllamaCollector 创建 Ollama 特征采集器
func NewOllamaCollector() *OllamaCollector {
	return &OllamaCollector{
		MaxLatencyPatternSize: 50,
	}
}

// OllamaResponse Ollama API 响应结构
type OllamaResponse struct {
	Model           string `json:"model"`
	CreatedAt       string `json:"created_at"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	Context         []int  `json:"context,omitempty"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	LoadDuration    int64  `json:"load_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
	EvalDuration    int64  `json:"eval_duration,omitempty"`
}

// OllamaModelInfo Ollama 模型信息
type OllamaModelInfo struct {
	Name       string            `json:"name"`
	Size       int64             `json:"size"`
	Digest     string            `json:"digest"`
	Details    OllamaModelDetails `json:"details"`
}

// OllamaModelDetails 模型详情
type OllamaModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// CollectionDataOllama Ollama 特有的采集数据
type CollectionDataOllama struct {
	*CollectionData

	// Ollama 特有字段
	ModelInfo         *OllamaModelInfo `json:"model_info,omitempty"`
	TotalDuration     int64            `json:"total_duration"`
	LoadDuration      int64            `json:"load_duration"`
	PromptEvalDuration int64           `json:"prompt_eval_duration"`
	EvalDuration      int64            `json:"eval_duration"`
	ContextTokens     []int            `json:"context_tokens,omitempty"`
}

// Collect 采集 Ollama 模型内部特征
func (c *OllamaCollector) Collect(data *CollectionDataOllama) (*ModelInternalFeatures, error) {
	if data == nil {
		return nil, nil
	}

	features := &ModelInternalFeatures{}

	// 模型权重哈希（来自 Ollama 的 digest）
	if data.ModelInfo != nil {
		features.ModelWeightsHash = data.ModelInfo.Digest
		features.QuantizationType = data.ModelInfo.Details.QuantizationLevel
	}

	// 上下文长度
	if len(data.ContextTokens) > 0 {
		features.ContextLength = len(data.ContextTokens)
	}

	// Token 延迟模式
	if len(data.ChunkLatencies) > 0 {
		features.TokenLatencyPattern = c.sampleLatencyPattern(data.ChunkLatencies)
	}

	// 计算 Attention 模式哈希（基于上下文 token 序列）
	if len(data.ContextTokens) > 0 {
		features.AttentionPatternHash = c.computeContextHash(data.ContextTokens)
		features.AvgAttentionEntropy = c.estimateAttentionEntropy(data.ContextTokens)
	}

	// 估算 Hidden State 特征（基于延迟模式）
	if len(data.ChunkLatencies) > 0 {
		features.HiddenStateNorm = c.estimateHiddenStateNorm(data.ChunkLatencies)
		features.HiddenStateVariance = c.estimateHiddenStateVariance(data.ChunkLatencies)
	}

	return features, nil
}

// sampleLatencyPattern 采样延迟模式
func (c *OllamaCollector) sampleLatencyPattern(latencies []int64) []int64 {
	if len(latencies) <= c.MaxLatencyPatternSize {
		return latencies
	}

	// 均匀采样
	step := len(latencies) / c.MaxLatencyPatternSize
	pattern := make([]int64, 0, c.MaxLatencyPatternSize)

	for i := 0; i < len(latencies); i += step {
		pattern = append(pattern, latencies[i])
		if len(pattern) >= c.MaxLatencyPatternSize {
			break
		}
	}

	return pattern
}

// computeContextHash 计算上下文哈希
func (c *OllamaCollector) computeContextHash(contextTokens []int) string {
	// 采样 token 序列计算哈希
	sampleSize := 100
	if len(contextTokens) < sampleSize {
		sampleSize = len(contextTokens)
	}

	// 取首尾和中间的 token
	sample := make([]int, 0, sampleSize)

	// 首部
	headSize := sampleSize / 3
	for i := 0; i < headSize && i < len(contextTokens); i++ {
		sample = append(sample, contextTokens[i])
	}

	// 中部
	midStart := (len(contextTokens) - sampleSize/3) / 2
	for i := midStart; i < midStart+sampleSize/3 && i < len(contextTokens); i++ {
		sample = append(sample, contextTokens[i])
	}

	// 尾部
	tailStart := len(contextTokens) - sampleSize/3
	if tailStart < 0 {
		tailStart = 0
	}
	for i := tailStart; i < len(contextTokens); i++ {
		sample = append(sample, contextTokens[i])
	}

	data, _ := json.Marshal(sample)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:16]) // 截取前 16 字节
}

// estimateAttentionEntropy 估算 Attention 熵
func (c *OllamaCollector) estimateAttentionEntropy(contextTokens []int) float64 {
	if len(contextTokens) == 0 {
		return 0
	}

	// 基于 token 分布估算熵
	tokenCounts := make(map[int]int)
	for _, t := range contextTokens {
		tokenCounts[t]++
	}

	total := float64(len(contextTokens))
	var entropy float64

	for _, count := range tokenCounts {
		p := float64(count) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// estimateHiddenStateNorm 估算 Hidden State 范数
func (c *OllamaCollector) estimateHiddenStateNorm(latencies []int64) float64 {
	if len(latencies) == 0 {
		return 0
	}

	// 基于延迟的变化率估算
	var sumSquares float64
	for _, l := range latencies {
		sumSquares += float64(l * l)
	}

	return math.Sqrt(sumSquares / float64(len(latencies)))
}

// estimateHiddenStateVariance 估算 Hidden State 方差
func (c *OllamaCollector) estimateHiddenStateVariance(latencies []int64) float64 {
	if len(latencies) == 0 {
		return 0
	}

	// 计算均值
	var sum int64
	for _, l := range latencies {
		sum += l
	}
	mean := float64(sum) / float64(len(latencies))

	// 计算方差
	var sumSquares float64
	for _, l := range latencies {
		diff := float64(l) - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(latencies))
}

// ParseOllamaTimings 从 Ollama 响应解析时间信息
func ParseOllamaTimings(resp *OllamaResponse) map[string]float64 {
	timings := make(map[string]float64)

	if resp.TotalDuration > 0 {
		timings["total_ms"] = float64(resp.TotalDuration) / 1e6
	}
	if resp.LoadDuration > 0 {
		timings["load_ms"] = float64(resp.LoadDuration) / 1e6
	}
	if resp.PromptEvalDuration > 0 {
		timings["prompt_eval_ms"] = float64(resp.PromptEvalDuration) / 1e6
	}
	if resp.EvalDuration > 0 {
		timings["eval_ms"] = float64(resp.EvalDuration) / 1e6
	}

	// 计算速率
	if resp.EvalDuration > 0 && resp.EvalCount > 0 {
		timings["tokens_per_second"] = float64(resp.EvalCount) / (float64(resp.EvalDuration) / 1e9)
	}

	return timings
}

// DetectQuantization 检测量化类型
func DetectQuantization(modelName string, digest string) string {
	// 常见量化类型
	quantTypes := []string{"Q4_0", "Q4_1", "Q5_0", "Q5_1", "Q8_0", "F16", "F32"}

	for _, qt := range quantTypes {
		if containsIgnoreCase(modelName, qt) {
			return qt
		}
	}

	return "unknown"
}

// containsIgnoreCase 忽略大小写检查是否包含子串
func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, i int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		if toLower(s[i+j]) != toLower(substr[j]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
