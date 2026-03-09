package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// SemanticCollector Layer 4 语义特征采集器
type SemanticCollector struct{}

// NewSemanticCollector 创建语义特征采集器
func NewSemanticCollector() *SemanticCollector {
	return &SemanticCollector{}
}

// Collect 采集语义特征
func (c *SemanticCollector) Collect(data *CollectionData) (*SemanticFeatures, error) {
	if data.OutputContent == "" {
		return nil, nil
	}

	text := data.OutputContent
	features := &SemanticFeatures{}

	// 分词
	words := c.tokenize(text)
	sentences := c.splitSentences(text)
	paragraphs := c.splitParagraphs(text)

	// 文本复杂度
	features.TextComplexity = c.calculateFleschKincaid(words, sentences)
	features.ReadabilityScore = c.calculateReadabilityScore(text, words, sentences)

	// 词汇特征
	features.VocabularyDiversity = c.calculateTTR(words)
	features.UniqueWordRatio = c.calculateUniqueWordRatio(words)
	features.AvgWordLength = c.calculateAvgWordLength(words)

	// 句子特征
	features.SentenceCount = len(sentences)
	features.AvgSentenceLength = c.calculateAvgSentenceLength(words, sentences)
	features.SentenceLenVariance = c.calculateSentenceLenVariance(sentences)

	// 结构特征
	features.ParagraphCount = len(paragraphs)
	features.HasCodeBlocks = c.hasCodeBlocks(text)
	features.HasLists = c.hasLists(text)
	features.HasLinks = c.hasLinks(text)

	return features, nil
}

// tokenize 分词
func (c *SemanticCollector) tokenize(text string) []string {
	// 简单分词：按空格和标点分割
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-' {
			current.WriteRune(unicode.ToLower(r))
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// splitSentences 分句
func (c *SemanticCollector) splitSentences(text string) []string {
	// 按句号、问号、感叹号分割
	re := regexp.MustCompile(`[.!?]+[\s\n]+|[.!?]+$`)
	parts := re.Split(text, -1)

	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			sentences = append(sentences, p)
		}
	}

	return sentences
}

// splitParagraphs 分段
func (c *SemanticCollector) splitParagraphs(text string) []string {
	// 按双换行分割
	re := regexp.MustCompile(`\n\s*\n`)
	parts := re.Split(text, -1)

	var paragraphs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			paragraphs = append(paragraphs, p)
		}
	}

	return paragraphs
}

// calculateFleschKincaid 计算 Flesch-Kincaid 可读性指数
func (c *SemanticCollector) calculateFleschKincaid(words []string, sentences []string) float64 {
	if len(words) == 0 || len(sentences) == 0 {
		return 0
	}

	// 计算音节数（简化版）
	totalSyllables := 0
	for _, word := range words {
		totalSyllables += c.countSyllables(word)
	}

	wordsPerSentence := float64(len(words)) / float64(len(sentences))
	syllablesPerWord := float64(totalSyllables) / float64(len(words))

	// Flesch Reading Ease = 206.835 - 1.015 * (words/sentences) - 84.6 * (syllables/words)
	score := 206.835 - 1.015*wordsPerSentence - 84.6*syllablesPerWord

	// 限制范围 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// countSyllables 计算单词音节数（简化版英文）
func (c *SemanticCollector) countSyllables(word string) int {
	if len(word) == 0 {
		return 0
	}

	word = strings.ToLower(word)
	vowels := "aeiouy"
	count := 0
	prevVowel := false

	for i, r := range word {
		isVowel := strings.ContainsRune(vowels, r)
		if isVowel && !prevVowel {
			count++
		}
		prevVowel = isVowel

		// 处理结尾的 e
		if i == len(word)-1 && r == 'e' && count > 1 {
			count--
		}
	}

	if count == 0 {
		count = 1
	}

	return count
}

// calculateReadabilityScore 计算可读性评分
func (c *SemanticCollector) calculateReadabilityScore(text string, words []string, sentences []string) float64 {
	if len(words) == 0 {
		return 0
	}

	// 综合多个因素
	fleschScore := c.calculateFleschKincaid(words, sentences)

	// 归一化到 0-1
	return fleschScore / 100.0
}

// calculateTTR 计算 Type-Token Ratio（词汇多样性）
func (c *SemanticCollector) calculateTTR(words []string) float64 {
	if len(words) == 0 {
		return 0
	}

	uniqueWords := make(map[string]bool)
	for _, w := range words {
		uniqueWords[strings.ToLower(w)] = true
	}

	return float64(len(uniqueWords)) / float64(len(words))
}

// calculateUniqueWordRatio 计算唯一词比例
func (c *SemanticCollector) calculateUniqueWordRatio(words []string) float64 {
	return c.calculateTTR(words)
}

// calculateAvgWordLength 计算平均词长
func (c *SemanticCollector) calculateAvgWordLength(words []string) float64 {
	if len(words) == 0 {
		return 0
	}

	totalLen := 0
	for _, w := range words {
		totalLen += len(w)
	}

	return float64(totalLen) / float64(len(words))
}

// calculateAvgSentenceLength 计算平均句长（词数）
func (c *SemanticCollector) calculateAvgSentenceLength(words []string, sentences []string) float64 {
	if len(sentences) == 0 {
		return 0
	}

	return float64(len(words)) / float64(len(sentences))
}

// calculateSentenceLenVariance 计算句长方差
func (c *SemanticCollector) calculateSentenceLenVariance(sentences []string) float64 {
	if len(sentences) == 0 {
		return 0
	}

	// 计算每句的词数
	lengths := make([]float64, len(sentences))
	var sum float64

	for i, s := range sentences {
		words := c.tokenize(s)
		lengths[i] = float64(len(words))
		sum += lengths[i]
	}

	mean := sum / float64(len(sentences))

	// 计算方差
	var variance float64
	for _, l := range lengths {
		diff := l - mean
		variance += diff * diff
	}

	return variance / float64(len(sentences))
}

// hasCodeBlocks 检测是否包含代码块
func (c *SemanticCollector) hasCodeBlocks(text string) bool {
	// Markdown 代码块
	if strings.Contains(text, "```") {
		return true
	}
	// 缩进代码块
	if regexp.MustCompile(`(?m)^    \S`).MatchString(text) {
		return true
	}
	return false
}

// hasLists 检测是否包含列表
func (c *SemanticCollector) hasLists(text string) bool {
	// 有序列表
	if regexp.MustCompile(`(?m)^\s*\d+\.\s+`).MatchString(text) {
		return true
	}
	// 无序列表
	if regexp.MustCompile(`(?m)^\s*[-*+]\s+`).MatchString(text) {
		return true
	}
	return false
}

// hasLinks 检测是否包含链接
func (c *SemanticCollector) hasLinks(text string) bool {
	// URL
	if regexp.MustCompile(`https?://\S+`).MatchString(text) {
		return true
	}
	// Markdown 链接
	if regexp.MustCompile(`\[.+\]\(.+\)`).MatchString(text) {
		return true
	}
	return false
}

// ComputeEmbeddingHash 计算文本的伪嵌入哈希
// 注意：这是简化版，实际应使用真正的嵌入模型
func (c *SemanticCollector) ComputeEmbeddingHash(text string) string {
	// 提取特征向量（简化版）
	words := c.tokenize(text)

	// 基于词频的简单特征
	wordFreq := make(map[string]int)
	for _, w := range words {
		wordFreq[strings.ToLower(w)]++
	}

	// 选择高频词作为特征
	type wordCount struct {
		word  string
		count int
	}
	var wcs []wordCount
	for w, cnt := range wordFreq {
		wcs = append(wcs, wordCount{w, cnt})
	}

	// 稳定排序：先按频率降序，再按字母序升序（确保结果一致）
	for i := 0; i < len(wcs); i++ {
		for j := i + 1; j < len(wcs); j++ {
			// 先比较频率
			if wcs[j].count > wcs[i].count {
				wcs[i], wcs[j] = wcs[j], wcs[i]
			} else if wcs[j].count == wcs[i].count && wcs[j].word < wcs[i].word {
				// 频率相同时按字母序
				wcs[i], wcs[j] = wcs[j], wcs[i]
			}
		}
	}

	// 取前 N 个
	n := 50
	if len(wcs) < n {
		n = len(wcs)
	}

	// 构建特征字符串
	var features strings.Builder
	for i := 0; i < n; i++ {
		features.WriteString(wcs[i].word)
		features.WriteString(":")
		features.WriteString(string(rune('0' + wcs[i].count%10)))
		features.WriteString(";")
	}

	// 计算哈希
	hash := sha256.Sum256([]byte(features.String()))
	return hex.EncodeToString(hash[:16])
}

// CalculateCosineSimilarity 计算余弦相似度（用于比较两个文本）
func (c *SemanticCollector) CalculateCosineSimilarity(text1, text2 string) float64 {
	words1 := c.tokenize(text1)
	words2 := c.tokenize(text2)

	// 构建词频向量
	freq1 := make(map[string]float64)
	freq2 := make(map[string]float64)

	for _, w := range words1 {
		freq1[w]++
	}
	for _, w := range words2 {
		freq2[w]++
	}

	// 计算点积和范数
	var dotProduct, norm1, norm2 float64

	for w, f1 := range freq1 {
		if f2, ok := freq2[w]; ok {
			dotProduct += f1 * f2
		}
		norm1 += f1 * f1
	}

	for _, f2 := range freq2 {
		norm2 += f2 * f2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}
