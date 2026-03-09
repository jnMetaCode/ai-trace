package fingerprint

import (
	"math"
)

// StatisticalCollector Layer 1 统计特征采集器
type StatisticalCollector struct{}

// NewStatisticalCollector 创建统计特征采集器
func NewStatisticalCollector() *StatisticalCollector {
	return &StatisticalCollector{}
}

// Collect 采集统计特征
func (c *StatisticalCollector) Collect(data *CollectionData) (*StatisticalFeatures, error) {
	features := &StatisticalFeatures{
		TotalTokens:      data.TotalTokens,
		PromptTokens:     data.PromptTokens,
		CompletionTokens: data.CompletionTokens,
		TotalCharacters:  len(data.OutputContent),
		FinishReason:     data.FinishReason,
	}

	// 计算时间特征
	totalLatency := data.EndTime.Sub(data.StartTime).Milliseconds()
	features.TotalLatencyMs = totalLatency

	if !data.FirstTokenAt.IsZero() {
		features.FirstTokenMs = data.FirstTokenAt.Sub(data.StartTime).Milliseconds()
	}

	// 计算生成速度
	if totalLatency > 0 && data.CompletionTokens > 0 {
		features.TokensPerSecond = float64(data.CompletionTokens) / (float64(totalLatency) / 1000.0)
		features.AvgTokenLatencyMs = float64(totalLatency) / float64(data.CompletionTokens)
	}

	// 计算流式特征
	features.ChunkCount = len(data.ChunkSizes)
	if features.ChunkCount > 0 {
		features.AvgChunkSize = c.calculateMean(data.ChunkSizes)
		features.ChunkSizeVariance = c.calculateVariance(data.ChunkSizes, features.AvgChunkSize)
	}

	return features, nil
}

// calculateMean 计算平均值
func (c *StatisticalCollector) calculateMean(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// calculateVariance 计算方差
func (c *StatisticalCollector) calculateVariance(values []int, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values))
}

// CalculateTokenLatencyStats 计算 token 延迟统计
func (c *StatisticalCollector) CalculateTokenLatencyStats(latencies []int64) (mean, variance, min, max float64) {
	if len(latencies) == 0 {
		return 0, 0, 0, 0
	}

	// 计算均值
	var sum int64
	minVal := latencies[0]
	maxVal := latencies[0]

	for _, l := range latencies {
		sum += l
		if l < minVal {
			minVal = l
		}
		if l > maxVal {
			maxVal = l
		}
	}

	mean = float64(sum) / float64(len(latencies))
	min = float64(minVal)
	max = float64(maxVal)

	// 计算方差
	var sumSquares float64
	for _, l := range latencies {
		diff := float64(l) - mean
		sumSquares += diff * diff
	}
	variance = sumSquares / float64(len(latencies))

	return mean, variance, min, max
}

// DetectLatencyPattern 检测延迟模式
// 返回: "steady" (稳定), "accelerating" (加速), "decelerating" (减速), "irregular" (不规则)
func (c *StatisticalCollector) DetectLatencyPattern(latencies []int64) string {
	if len(latencies) < 3 {
		return "unknown"
	}

	// 计算趋势
	n := float64(len(latencies))
	var sumX, sumY, sumXY, sumX2 float64

	for i, l := range latencies {
		x := float64(i)
		y := float64(l)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// 线性回归斜率
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 计算变异系数
	mean, variance, _, _ := c.CalculateTokenLatencyStats(latencies)
	cv := 0.0
	if mean > 0 {
		cv = math.Sqrt(variance) / mean
	}

	// 判断模式
	if cv > 0.5 {
		return "irregular"
	}

	if math.Abs(slope) < 0.1*mean {
		return "steady"
	} else if slope < 0 {
		return "accelerating"
	} else {
		return "decelerating"
	}
}
