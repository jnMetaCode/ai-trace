// Package fingerprint 提供 LLM 推理行为指纹能力
// 用于捕获和验证 AI 生成内容的行为特征
package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// InferenceFingerprint 推理行为指纹（4层结构）
type InferenceFingerprint struct {
	// Layer 1: 统计特征（所有 API 可获取）
	Statistical *StatisticalFeatures `json:"statistical"`

	// Layer 2: Token 概率特征（需要 logprobs 参数）
	TokenProbs *TokenProbFeatures `json:"token_probs,omitempty"`

	// Layer 3: 模型内部特征（仅本地模型如 Ollama）
	ModelInternal *ModelInternalFeatures `json:"model_internal,omitempty"`

	// Layer 4: 语义特征
	Semantic *SemanticFeatures `json:"semantic"`

	// 元信息
	ModelID         string    `json:"model_id"`
	ModelProvider   string    `json:"model_provider"`
	GeneratedAt     time.Time `json:"generated_at"`
	FingerprintHash string    `json:"fingerprint_hash"` // 综合指纹哈希
}

// StatisticalFeatures Layer 1: 统计特征
type StatisticalFeatures struct {
	// 基础统计
	TotalTokens       int     `json:"total_tokens"`
	PromptTokens      int     `json:"prompt_tokens"`
	CompletionTokens  int     `json:"completion_tokens"`
	TotalCharacters   int     `json:"total_characters"`

	// 时间特征
	TotalLatencyMs    int64   `json:"total_latency_ms"`
	FirstTokenMs      int64   `json:"first_token_ms"`      // 首 token 延迟
	TokensPerSecond   float64 `json:"tokens_per_second"`   // 生成速度
	AvgTokenLatencyMs float64 `json:"avg_token_latency_ms"` // 平均每 token 延迟

	// 流式特征
	ChunkCount        int     `json:"chunk_count"`
	AvgChunkSize      float64 `json:"avg_chunk_size"`
	ChunkSizeVariance float64 `json:"chunk_size_variance"`

	// 结束特征
	FinishReason      string  `json:"finish_reason"`
}

// TokenProbFeatures Layer 2: Token 概率特征
type TokenProbFeatures struct {
	// 概率分布
	AvgLogProb        float64   `json:"avg_log_prob"`         // 平均 log 概率
	LogProbVariance   float64   `json:"log_prob_variance"`    // log 概率方差
	MinLogProb        float64   `json:"min_log_prob"`         // 最小 log 概率
	MaxLogProb        float64   `json:"max_log_prob"`         // 最大 log 概率

	// 熵特征
	AvgEntropy        float64   `json:"avg_entropy"`          // 平均熵
	EntropyVariance   float64   `json:"entropy_variance"`     // 熵方差
	EntropyDistribution []float64 `json:"entropy_distribution"` // 熵分布（分桶）

	// 低概率 token 分析
	LowProbTokenRatio float64   `json:"low_prob_token_ratio"` // 低概率 token 比例
	LowProbPositions  []int     `json:"low_prob_positions"`   // 低概率 token 位置（采样）

	// Top-k 分析
	AvgTopKProb       float64   `json:"avg_top_k_prob"`       // 平均 top-k 概率
	TopKConcentration float64   `json:"top_k_concentration"`  // top-k 集中度
}

// ModelInternalFeatures Layer 3: 模型内部特征（仅本地模型）
type ModelInternalFeatures struct {
	// Attention 特征
	AttentionPatternHash string  `json:"attention_pattern_hash,omitempty"`
	AvgAttentionEntropy  float64 `json:"avg_attention_entropy,omitempty"`

	// Hidden State 特征
	HiddenStateNorm      float64 `json:"hidden_state_norm,omitempty"`
	HiddenStateVariance  float64 `json:"hidden_state_variance,omitempty"`

	// 推理时间序列
	TokenLatencyPattern  []int64 `json:"token_latency_pattern,omitempty"` // 采样的 token 延迟模式

	// 模型特定信息
	ModelWeightsHash     string  `json:"model_weights_hash,omitempty"`
	QuantizationType     string  `json:"quantization_type,omitempty"`
	ContextLength        int     `json:"context_length,omitempty"`
}

// SemanticFeatures Layer 4: 语义特征
type SemanticFeatures struct {
	// 文本复杂度
	TextComplexity      float64 `json:"text_complexity"`       // Flesch-Kincaid 等
	ReadabilityScore    float64 `json:"readability_score"`     // 可读性评分

	// 词汇特征
	VocabularyDiversity float64 `json:"vocabulary_diversity"`  // Type-Token Ratio
	UniqueWordRatio     float64 `json:"unique_word_ratio"`     // 唯一词比例
	AvgWordLength       float64 `json:"avg_word_length"`       // 平均词长

	// 句子特征
	SentenceCount       int     `json:"sentence_count"`
	AvgSentenceLength   float64 `json:"avg_sentence_length"`   // 平均句长（词数）
	SentenceLenVariance float64 `json:"sentence_len_variance"` // 句长方差

	// 结构特征
	ParagraphCount      int     `json:"paragraph_count"`
	HasCodeBlocks       bool    `json:"has_code_blocks"`
	HasLists            bool    `json:"has_lists"`
	HasLinks            bool    `json:"has_links"`

	// 语义向量（可选）
	EmbeddingHash       string  `json:"embedding_hash,omitempty"` // 语义向量哈希
}

// Collector 指纹采集器接口
type Collector interface {
	// CollectStatistical 采集统计特征
	CollectStatistical(data *CollectionData) (*StatisticalFeatures, error)

	// CollectTokenProbs 采集 Token 概率特征
	CollectTokenProbs(data *CollectionData) (*TokenProbFeatures, error)

	// CollectModelInternal 采集模型内部特征（仅本地模型）
	CollectModelInternal(data *CollectionData) (*ModelInternalFeatures, error)

	// CollectSemantic 采集语义特征
	CollectSemantic(data *CollectionData) (*SemanticFeatures, error)

	// Collect 采集完整指纹
	Collect(data *CollectionData) (*InferenceFingerprint, error)
}

// CollectionData 采集所需的原始数据
type CollectionData struct {
	// 模型信息
	ModelID       string `json:"model_id"`
	ModelProvider string `json:"model_provider"` // openai, anthropic, ollama, deepseek

	// 内容
	PromptContent  string `json:"prompt_content"`
	OutputContent  string `json:"output_content"`

	// Token 统计
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	// 时间信息
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	FirstTokenAt  time.Time `json:"first_token_at"`

	// 流式信息
	ChunkSizes    []int   `json:"chunk_sizes"`
	ChunkLatencies []int64 `json:"chunk_latencies"` // 每个 chunk 的延迟 ms

	// Logprobs（如果可用）
	LogProbs      []float64 `json:"log_probs,omitempty"`
	TopLogProbs   [][]TopLogProb `json:"top_log_probs,omitempty"`

	// 结束原因
	FinishReason  string `json:"finish_reason"`
}

// TopLogProb top-k logprob 条目
type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"log_prob"`
}

// ComputeFingerprintHash 计算指纹综合哈希
func (f *InferenceFingerprint) ComputeFingerprintHash() string {
	// 序列化关键字段
	data := map[string]interface{}{
		"model_id":       f.ModelID,
		"model_provider": f.ModelProvider,
		"statistical":    f.Statistical,
		"semantic":       f.Semantic,
	}

	if f.TokenProbs != nil {
		data["token_probs"] = f.TokenProbs
	}
	if f.ModelInternal != nil {
		data["model_internal"] = f.ModelInternal
	}

	jsonBytes, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// Verify 验证指纹完整性
func (f *InferenceFingerprint) Verify() bool {
	computed := f.ComputeFingerprintHash()
	return computed == f.FingerprintHash
}

// ToJSON 转换为 JSON
func (f *InferenceFingerprint) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

// FromJSON 从 JSON 解析
func FromJSON(data []byte) (*InferenceFingerprint, error) {
	var f InferenceFingerprint
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}
