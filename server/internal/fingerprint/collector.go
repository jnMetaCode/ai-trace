package fingerprint

import (
	"time"
)

// DefaultCollector 默认指纹采集器（整合 4 层）
type DefaultCollector struct {
	statistical *StatisticalCollector
	logProbs    *LogProbsCollector
	ollama      *OllamaCollector
	semantic    *SemanticCollector
}

// NewDefaultCollector 创建默认采集器
func NewDefaultCollector() *DefaultCollector {
	return &DefaultCollector{
		statistical: NewStatisticalCollector(),
		logProbs:    NewLogProbsCollector(),
		ollama:      NewOllamaCollector(),
		semantic:    NewSemanticCollector(),
	}
}

// Collect 采集完整的 4 层指纹
func (c *DefaultCollector) Collect(data *CollectionData) (*InferenceFingerprint, error) {
	fp := &InferenceFingerprint{
		ModelID:       data.ModelID,
		ModelProvider: data.ModelProvider,
		GeneratedAt:   time.Now(),
	}

	// Layer 1: 统计特征（总是采集）
	statistical, err := c.statistical.Collect(data)
	if err != nil {
		return nil, err
	}
	fp.Statistical = statistical

	// Layer 2: Token 概率特征（如果有 logprobs）
	if len(data.LogProbs) > 0 || len(data.TopLogProbs) > 0 {
		tokenProbs, err := c.logProbs.Collect(data)
		if err != nil {
			return nil, err
		}
		fp.TokenProbs = tokenProbs
	}

	// Layer 3: 模型内部特征（仅 Ollama）
	// 注意：需要外部传入 OllamaCollectionData

	// Layer 4: 语义特征（总是采集）
	semantic, err := c.semantic.Collect(data)
	if err != nil {
		return nil, err
	}
	fp.Semantic = semantic

	// 计算综合指纹哈希
	fp.FingerprintHash = fp.ComputeFingerprintHash()

	return fp, nil
}

// CollectStatistical 仅采集统计特征
func (c *DefaultCollector) CollectStatistical(data *CollectionData) (*StatisticalFeatures, error) {
	return c.statistical.Collect(data)
}

// CollectTokenProbs 仅采集 Token 概率特征
func (c *DefaultCollector) CollectTokenProbs(data *CollectionData) (*TokenProbFeatures, error) {
	return c.logProbs.Collect(data)
}

// CollectModelInternal 采集模型内部特征（仅 Ollama）
func (c *DefaultCollector) CollectModelInternal(data *CollectionDataOllama) (*ModelInternalFeatures, error) {
	return c.ollama.Collect(data)
}

// CollectSemantic 仅采集语义特征
func (c *DefaultCollector) CollectSemantic(data *CollectionData) (*SemanticFeatures, error) {
	return c.semantic.Collect(data)
}

// CollectWithOllama 采集完整指纹（包含 Ollama 特有特征）
func (c *DefaultCollector) CollectWithOllama(data *CollectionDataOllama) (*InferenceFingerprint, error) {
	// 先采集基础特征
	fp, err := c.Collect(data.CollectionData)
	if err != nil {
		return nil, err
	}

	// 添加 Ollama 特有特征
	modelInternal, err := c.ollama.Collect(data)
	if err != nil {
		return nil, err
	}
	fp.ModelInternal = modelInternal

	// 重新计算指纹哈希
	fp.FingerprintHash = fp.ComputeFingerprintHash()

	return fp, nil
}

// QuickFingerprint 快速指纹（仅 Layer 1 和 Layer 4）
// 适用于不需要详细 logprobs 的场景
func (c *DefaultCollector) QuickFingerprint(data *CollectionData) (*InferenceFingerprint, error) {
	fp := &InferenceFingerprint{
		ModelID:       data.ModelID,
		ModelProvider: data.ModelProvider,
		GeneratedAt:   time.Now(),
	}

	// Layer 1
	statistical, err := c.statistical.Collect(data)
	if err != nil {
		return nil, err
	}
	fp.Statistical = statistical

	// Layer 4
	semantic, err := c.semantic.Collect(data)
	if err != nil {
		return nil, err
	}
	fp.Semantic = semantic

	fp.FingerprintHash = fp.ComputeFingerprintHash()

	return fp, nil
}

// BuildCollectionData 从流式会话数据构建采集数据
func BuildCollectionData(
	modelID, modelProvider string,
	promptContent, outputContent string,
	promptTokens, completionTokens int,
	startTime, endTime, firstTokenAt time.Time,
	chunkSizes []int,
	chunkLatencies []int64,
	finishReason string,
) *CollectionData {
	return &CollectionData{
		ModelID:          modelID,
		ModelProvider:    modelProvider,
		PromptContent:    promptContent,
		OutputContent:    outputContent,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		StartTime:        startTime,
		EndTime:          endTime,
		FirstTokenAt:     firstTokenAt,
		ChunkSizes:       chunkSizes,
		ChunkLatencies:   chunkLatencies,
		FinishReason:     finishReason,
	}
}

// CompareFingerprints 比较两个指纹的相似度
func CompareFingerprints(fp1, fp2 *InferenceFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	var totalScore float64
	var weightSum float64

	// 比较统计特征（权重 0.3）
	if fp1.Statistical != nil && fp2.Statistical != nil {
		statScore := compareStatistical(fp1.Statistical, fp2.Statistical)
		totalScore += statScore * 0.3
		weightSum += 0.3
	}

	// 比较 Token 概率特征（权重 0.3）
	if fp1.TokenProbs != nil && fp2.TokenProbs != nil {
		probScore := compareTokenProbs(fp1.TokenProbs, fp2.TokenProbs)
		totalScore += probScore * 0.3
		weightSum += 0.3
	}

	// 比较语义特征（权重 0.4）
	if fp1.Semantic != nil && fp2.Semantic != nil {
		semScore := compareSemantic(fp1.Semantic, fp2.Semantic)
		totalScore += semScore * 0.4
		weightSum += 0.4
	}

	if weightSum == 0 {
		return 0
	}

	return totalScore / weightSum
}

func compareStatistical(s1, s2 *StatisticalFeatures) float64 {
	// 比较关键统计指标
	score := 0.0

	// Token 速度相似度
	if s1.TokensPerSecond > 0 && s2.TokensPerSecond > 0 {
		ratio := s1.TokensPerSecond / s2.TokensPerSecond
		if ratio > 1 {
			ratio = 1 / ratio
		}
		score += ratio * 0.5
	}

	// 首 token 延迟相似度
	if s1.FirstTokenMs > 0 && s2.FirstTokenMs > 0 {
		ratio := float64(s1.FirstTokenMs) / float64(s2.FirstTokenMs)
		if ratio > 1 {
			ratio = 1 / ratio
		}
		score += ratio * 0.5
	}

	return score
}

func compareTokenProbs(t1, t2 *TokenProbFeatures) float64 {
	score := 0.0

	// 平均 log prob 相似度
	diff := t1.AvgLogProb - t2.AvgLogProb
	if diff < 0 {
		diff = -diff
	}
	if diff < 1 {
		score += (1 - diff) * 0.5
	}

	// 熵相似度
	diff = t1.AvgEntropy - t2.AvgEntropy
	if diff < 0 {
		diff = -diff
	}
	if diff < 1 {
		score += (1 - diff) * 0.5
	}

	return score
}

func compareSemantic(s1, s2 *SemanticFeatures) float64 {
	score := 0.0

	// 词汇多样性相似度
	diff := s1.VocabularyDiversity - s2.VocabularyDiversity
	if diff < 0 {
		diff = -diff
	}
	score += (1 - diff) * 0.3

	// 平均句长相似度
	if s1.AvgSentenceLength > 0 && s2.AvgSentenceLength > 0 {
		ratio := s1.AvgSentenceLength / s2.AvgSentenceLength
		if ratio > 1 {
			ratio = 1 / ratio
		}
		score += ratio * 0.3
	}

	// 复杂度相似度
	diff = s1.TextComplexity - s2.TextComplexity
	if diff < 0 {
		diff = -diff
	}
	if diff < 20 {
		score += (1 - diff/20) * 0.4
	}

	return score
}
