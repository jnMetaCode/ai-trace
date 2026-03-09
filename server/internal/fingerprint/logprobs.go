package fingerprint

import (
	"math"
	"sort"
)

// LogProbsCollector Layer 2 Token 概率特征采集器
type LogProbsCollector struct {
	// 低概率阈值（log prob < threshold 视为低概率）
	LowProbThreshold float64
	// 熵分布桶数
	EntropyBuckets int
	// 采样的低概率位置数量
	MaxLowProbPositions int
}

// NewLogProbsCollector 创建 Token 概率特征采集器
func NewLogProbsCollector() *LogProbsCollector {
	return &LogProbsCollector{
		LowProbThreshold:    -5.0, // log prob < -5 视为低概率
		EntropyBuckets:      10,
		MaxLowProbPositions: 20,
	}
}

// Collect 采集 Token 概率特征
func (c *LogProbsCollector) Collect(data *CollectionData) (*TokenProbFeatures, error) {
	if len(data.LogProbs) == 0 {
		return nil, nil // 没有 logprobs 数据
	}

	features := &TokenProbFeatures{}

	// 计算基础统计
	features.AvgLogProb = c.calculateMean(data.LogProbs)
	features.LogProbVariance = c.calculateVariance(data.LogProbs, features.AvgLogProb)
	features.MinLogProb, features.MaxLogProb = c.findMinMax(data.LogProbs)

	// 计算熵特征
	if len(data.TopLogProbs) > 0 {
		entropies := c.calculateEntropies(data.TopLogProbs)
		features.AvgEntropy = c.calculateMean(entropies)
		features.EntropyVariance = c.calculateVariance(entropies, features.AvgEntropy)
		features.EntropyDistribution = c.bucketize(entropies, c.EntropyBuckets)

		// Top-k 分析
		features.AvgTopKProb = c.calculateAvgTopKProb(data.TopLogProbs)
		features.TopKConcentration = c.calculateTopKConcentration(data.TopLogProbs)
	}

	// 低概率 token 分析
	features.LowProbTokenRatio, features.LowProbPositions = c.analyzeLowProbTokens(data.LogProbs)

	return features, nil
}

// calculateMean 计算平均值
func (c *LogProbsCollector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateVariance 计算方差
func (c *LogProbsCollector) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values))
}

// findMinMax 找最小和最大值
func (c *LogProbsCollector) findMinMax(values []float64) (min, max float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max = values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// calculateEntropies 计算每个位置的熵
func (c *LogProbsCollector) calculateEntropies(topLogProbs [][]TopLogProb) []float64 {
	entropies := make([]float64, len(topLogProbs))

	for i, probs := range topLogProbs {
		if len(probs) == 0 {
			continue
		}

		// 将 log probs 转换为概率并计算熵
		// H = -sum(p * log(p))
		var entropy float64
		for _, p := range probs {
			prob := math.Exp(p.LogProb)
			if prob > 0 {
				entropy -= prob * p.LogProb
			}
		}
		entropies[i] = entropy
	}

	return entropies
}

// bucketize 将值分桶
func (c *LogProbsCollector) bucketize(values []float64, numBuckets int) []float64 {
	if len(values) == 0 || numBuckets <= 0 {
		return nil
	}

	// 找到范围
	min, max := c.findMinMax(values)
	if max == min {
		// 所有值相同
		buckets := make([]float64, numBuckets)
		buckets[numBuckets/2] = 1.0
		return buckets
	}

	buckets := make([]float64, numBuckets)
	bucketWidth := (max - min) / float64(numBuckets)

	for _, v := range values {
		idx := int((v - min) / bucketWidth)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		buckets[idx]++
	}

	// 归一化
	total := float64(len(values))
	for i := range buckets {
		buckets[i] /= total
	}

	return buckets
}

// analyzeLowProbTokens 分析低概率 token
func (c *LogProbsCollector) analyzeLowProbTokens(logProbs []float64) (ratio float64, positions []int) {
	if len(logProbs) == 0 {
		return 0, nil
	}

	var lowCount int
	var allPositions []int

	for i, lp := range logProbs {
		if lp < c.LowProbThreshold {
			lowCount++
			allPositions = append(allPositions, i)
		}
	}

	ratio = float64(lowCount) / float64(len(logProbs))

	// 采样位置
	if len(allPositions) > c.MaxLowProbPositions {
		// 均匀采样
		step := len(allPositions) / c.MaxLowProbPositions
		positions = make([]int, 0, c.MaxLowProbPositions)
		for i := 0; i < len(allPositions); i += step {
			positions = append(positions, allPositions[i])
			if len(positions) >= c.MaxLowProbPositions {
				break
			}
		}
	} else {
		positions = allPositions
	}

	return ratio, positions
}

// calculateAvgTopKProb 计算平均 top-k 概率
func (c *LogProbsCollector) calculateAvgTopKProb(topLogProbs [][]TopLogProb) float64 {
	if len(topLogProbs) == 0 {
		return 0
	}

	var sumTopProb float64
	var count int

	for _, probs := range topLogProbs {
		if len(probs) > 0 {
			// 取最高概率
			sumTopProb += math.Exp(probs[0].LogProb)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sumTopProb / float64(count)
}

// calculateTopKConcentration 计算 top-k 集中度
// 集中度 = top-1 概率 / sum(top-k 概率)
func (c *LogProbsCollector) calculateTopKConcentration(topLogProbs [][]TopLogProb) float64 {
	if len(topLogProbs) == 0 {
		return 0
	}

	var sumConcentration float64
	var count int

	for _, probs := range topLogProbs {
		if len(probs) == 0 {
			continue
		}

		// 计算所有概率之和
		var totalProb, topProb float64
		for i, p := range probs {
			prob := math.Exp(p.LogProb)
			totalProb += prob
			if i == 0 {
				topProb = prob
			}
		}

		if totalProb > 0 {
			sumConcentration += topProb / totalProb
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sumConcentration / float64(count)
}

// CalculatePerplexity 计算困惑度
func (c *LogProbsCollector) CalculatePerplexity(logProbs []float64) float64 {
	if len(logProbs) == 0 {
		return 0
	}

	// 困惑度 = exp(-1/N * sum(log(p)))
	sum := 0.0
	for _, lp := range logProbs {
		sum += lp
	}

	return math.Exp(-sum / float64(len(logProbs)))
}

// DetectProbabilityPattern 检测概率模式
// 返回概率序列的特征描述
func (c *LogProbsCollector) DetectProbabilityPattern(logProbs []float64) map[string]interface{} {
	if len(logProbs) == 0 {
		return nil
	}

	result := make(map[string]interface{})

	// 计算滑动窗口统计
	windowSize := 10
	if len(logProbs) < windowSize {
		windowSize = len(logProbs)
	}

	var windowMeans []float64
	for i := 0; i <= len(logProbs)-windowSize; i++ {
		window := logProbs[i : i+windowSize]
		windowMeans = append(windowMeans, c.calculateMean(window))
	}

	// 检测趋势
	if len(windowMeans) > 1 {
		firstHalf := c.calculateMean(windowMeans[:len(windowMeans)/2])
		secondHalf := c.calculateMean(windowMeans[len(windowMeans)/2:])

		if secondHalf > firstHalf+0.5 {
			result["trend"] = "increasing_confidence"
		} else if secondHalf < firstHalf-0.5 {
			result["trend"] = "decreasing_confidence"
		} else {
			result["trend"] = "stable"
		}
	}

	// 检测突变点
	spikes := c.detectSpikes(logProbs)
	result["spike_count"] = len(spikes)
	if len(spikes) > 0 && len(spikes) <= 5 {
		result["spike_positions"] = spikes
	}

	return result
}

// detectSpikes 检测突变点
func (c *LogProbsCollector) detectSpikes(logProbs []float64) []int {
	if len(logProbs) < 3 {
		return nil
	}

	mean := c.calculateMean(logProbs)
	variance := c.calculateVariance(logProbs, mean)
	threshold := mean - 2*math.Sqrt(variance)

	var spikes []int
	for i, lp := range logProbs {
		if lp < threshold {
			spikes = append(spikes, i)
		}
	}

	return spikes
}

// SortTopLogProbs 对 TopLogProb 排序（按概率降序）
func SortTopLogProbs(probs []TopLogProb) {
	sort.Slice(probs, func(i, j int) bool {
		return probs[i].LogProb > probs[j].LogProb
	})
}
