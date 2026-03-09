package fingerprint

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
	"time"
)

func TestStatisticalCollector(t *testing.T) {
	collector := NewStatisticalCollector()

	data := &CollectionData{
		ModelID:          "gpt-4",
		ModelProvider:    "openai",
		OutputContent:    "Hello, World! This is a test response.",
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		StartTime:        time.Now().Add(-time.Second),
		EndTime:          time.Now(),
		FirstTokenAt:     time.Now().Add(-500 * time.Millisecond),
		ChunkSizes:       []int{5, 3, 8, 4, 10},
		ChunkLatencies:   []int64{50, 30, 40, 35, 45},
		FinishReason:     "stop",
	}

	features, err := collector.Collect(data)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if features.TotalTokens != 30 {
		t.Errorf("Expected TotalTokens=30, got %d", features.TotalTokens)
	}

	if features.ChunkCount != 5 {
		t.Errorf("Expected ChunkCount=5, got %d", features.ChunkCount)
	}

	if features.TokensPerSecond <= 0 {
		t.Error("TokensPerSecond should be > 0")
	}
}

func TestLogProbsCollector(t *testing.T) {
	collector := NewLogProbsCollector()

	data := &CollectionData{
		LogProbs: []float64{-0.5, -1.0, -0.3, -2.5, -0.8, -6.0, -0.4},
		TopLogProbs: [][]TopLogProb{
			{{Token: "hello", LogProb: -0.5}, {Token: "hi", LogProb: -1.2}},
			{{Token: "world", LogProb: -0.3}, {Token: "earth", LogProb: -1.5}},
		},
	}

	features, err := collector.Collect(data)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.AvgLogProb >= 0 {
		t.Error("AvgLogProb should be negative")
	}

	// 检查低概率 token 检测
	if features.LowProbTokenRatio == 0 {
		t.Error("Should detect low probability tokens")
	}
}

func TestSemanticCollector(t *testing.T) {
	collector := NewSemanticCollector()

	data := &CollectionData{
		OutputContent: `This is a test paragraph with multiple sentences.
It contains various words and phrases to test the semantic analysis.

Another paragraph here with more content. The system should detect
paragraphs, sentences, and calculate various metrics like vocabulary
diversity and text complexity.

- List item 1
- List item 2

Here is some code:
` + "```" + `
func hello() {}
` + "```",
	}

	features, err := collector.Collect(data)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.SentenceCount == 0 {
		t.Error("Should detect sentences")
	}

	if features.ParagraphCount == 0 {
		t.Error("Should detect paragraphs")
	}

	if !features.HasLists {
		t.Error("Should detect lists")
	}

	if !features.HasCodeBlocks {
		t.Error("Should detect code blocks")
	}

	if features.VocabularyDiversity <= 0 || features.VocabularyDiversity > 1 {
		t.Errorf("VocabularyDiversity should be in (0, 1], got %f", features.VocabularyDiversity)
	}
}

func TestDefaultCollector(t *testing.T) {
	collector := NewDefaultCollector()

	data := &CollectionData{
		ModelID:          "gpt-4",
		ModelProvider:    "openai",
		PromptContent:    "Tell me a story",
		OutputContent:    "Once upon a time, there was a brave knight. He fought many battles and won them all.",
		PromptTokens:     5,
		CompletionTokens: 20,
		TotalTokens:      25,
		StartTime:        time.Now().Add(-2 * time.Second),
		EndTime:          time.Now(),
		FirstTokenAt:     time.Now().Add(-time.Second),
		ChunkSizes:       []int{10, 15, 12, 8},
		ChunkLatencies:   []int64{100, 80, 90, 85},
		FinishReason:     "stop",
	}

	fp, err := collector.Collect(data)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if fp.ModelID != "gpt-4" {
		t.Errorf("Expected ModelID=gpt-4, got %s", fp.ModelID)
	}

	if fp.Statistical == nil {
		t.Error("Statistical features should not be nil")
	}

	if fp.Semantic == nil {
		t.Error("Semantic features should not be nil")
	}

	if fp.FingerprintHash == "" {
		t.Error("FingerprintHash should not be empty")
	}

	// 验证指纹完整性
	if !fp.Verify() {
		t.Error("Fingerprint verification failed")
	}
}

func TestInferenceFingerprintSerialization(t *testing.T) {
	fp := &InferenceFingerprint{
		ModelID:       "gpt-4",
		ModelProvider: "openai",
		GeneratedAt:   time.Now(),
		Statistical: &StatisticalFeatures{
			TotalTokens:     100,
			TokensPerSecond: 25.5,
			ChunkCount:      10,
		},
		Semantic: &SemanticFeatures{
			TextComplexity:      65.5,
			VocabularyDiversity: 0.75,
			SentenceCount:       5,
		},
	}
	fp.FingerprintHash = fp.ComputeFingerprintHash()

	// 序列化
	data, err := fp.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// 反序列化
	decoded, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if decoded.ModelID != fp.ModelID {
		t.Errorf("ModelID mismatch: %s != %s", decoded.ModelID, fp.ModelID)
	}

	if decoded.Statistical.TotalTokens != fp.Statistical.TotalTokens {
		t.Errorf("TotalTokens mismatch")
	}
}

func TestCompareFingerprints(t *testing.T) {
	fp1 := &InferenceFingerprint{
		Statistical: &StatisticalFeatures{
			TokensPerSecond: 25.0,
			FirstTokenMs:    100,
		},
		Semantic: &SemanticFeatures{
			VocabularyDiversity: 0.7,
			AvgSentenceLength:   15.0,
			TextComplexity:      60.0,
		},
	}

	fp2 := &InferenceFingerprint{
		Statistical: &StatisticalFeatures{
			TokensPerSecond: 26.0,
			FirstTokenMs:    105,
		},
		Semantic: &SemanticFeatures{
			VocabularyDiversity: 0.72,
			AvgSentenceLength:   14.0,
			TextComplexity:      62.0,
		},
	}

	similarity := CompareFingerprints(fp1, fp2)
	if similarity < 0.8 {
		t.Errorf("Similar fingerprints should have high similarity, got %f", similarity)
	}

	// 测试差异较大的指纹
	fp3 := &InferenceFingerprint{
		Statistical: &StatisticalFeatures{
			TokensPerSecond: 5.0,
			FirstTokenMs:    500,
		},
		Semantic: &SemanticFeatures{
			VocabularyDiversity: 0.3,
			AvgSentenceLength:   5.0,
			TextComplexity:      30.0,
		},
	}

	similarity2 := CompareFingerprints(fp1, fp3)
	if similarity2 > similarity {
		t.Error("Different fingerprints should have lower similarity")
	}
}

func TestBuildCollectionData(t *testing.T) {
	startTime := time.Now().Add(-time.Second)
	endTime := time.Now()
	firstTokenAt := time.Now().Add(-500 * time.Millisecond)

	data := BuildCollectionData(
		"gpt-4",
		"openai",
		"Hello",
		"World",
		10,
		20,
		startTime,
		endTime,
		firstTokenAt,
		[]int{5, 5},
		[]int64{50, 50},
		"stop",
	)

	if data.ModelID != "gpt-4" {
		t.Errorf("Expected ModelID=gpt-4, got %s", data.ModelID)
	}

	if data.TotalTokens != 30 {
		t.Errorf("Expected TotalTokens=30, got %d", data.TotalTokens)
	}
}

func TestOllamaCollector(t *testing.T) {
	collector := NewOllamaCollector()

	data := &CollectionDataOllama{
		CollectionData: &CollectionData{
			ModelID:        "llama2",
			ModelProvider:  "ollama",
			ChunkLatencies: []int64{50, 45, 48, 52, 47, 49},
		},
		ModelInfo: &OllamaModelInfo{
			Name:   "llama2:7b-q4_0",
			Size:   4000000000,
			Digest: "sha256:abc123",
			Details: OllamaModelDetails{
				Family:            "llama",
				ParameterSize:     "7B",
				QuantizationLevel: "Q4_0",
			},
		},
		ContextTokens: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	features, err := collector.Collect(data)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if features.ModelWeightsHash != "sha256:abc123" {
		t.Errorf("Expected ModelWeightsHash=sha256:abc123, got %s", features.ModelWeightsHash)
	}

	if features.QuantizationType != "Q4_0" {
		t.Errorf("Expected QuantizationType=Q4_0, got %s", features.QuantizationType)
	}

	if features.ContextLength != 10 {
		t.Errorf("Expected ContextLength=10, got %d", features.ContextLength)
	}
}

func TestQuickFingerprint(t *testing.T) {
	collector := NewDefaultCollector()

	data := &CollectionData{
		ModelID:          "gpt-3.5-turbo",
		ModelProvider:    "openai",
		OutputContent:    "This is a quick test response.",
		CompletionTokens: 8,
		StartTime:        time.Now().Add(-100 * time.Millisecond),
		EndTime:          time.Now(),
		ChunkSizes:       []int{4, 4},
		FinishReason:     "stop",
	}

	fp, err := collector.QuickFingerprint(data)
	if err != nil {
		t.Fatalf("QuickFingerprint failed: %v", err)
	}

	// QuickFingerprint 只采集 Layer 1 和 Layer 4
	if fp.Statistical == nil {
		t.Error("Statistical features should not be nil")
	}

	if fp.Semantic == nil {
		t.Error("Semantic features should not be nil")
	}

	// TokenProbs 应该为 nil（没有提供 logprobs 数据）
	if fp.TokenProbs != nil {
		t.Error("TokenProbs should be nil for QuickFingerprint without logprobs")
	}
}

func TestEmbeddingHash(t *testing.T) {
	collector := NewSemanticCollector()

	text1 := "The quick brown fox jumps over the lazy dog."
	text2 := "The quick brown fox jumps over the lazy dog."
	text3 := "A completely different sentence with other words."

	hash1 := collector.ComputeEmbeddingHash(text1)
	hash2 := collector.ComputeEmbeddingHash(text2)
	hash3 := collector.ComputeEmbeddingHash(text3)

	// 相同文本应该产生相同哈希
	if hash1 != hash2 {
		t.Error("Same text should produce same embedding hash")
	}

	// 不同文本应该产生不同哈希
	if hash1 == hash3 {
		t.Error("Different text should produce different embedding hash")
	}
}

func TestCosineSimilarity(t *testing.T) {
	collector := NewSemanticCollector()

	text1 := "The cat sat on the mat"
	text2 := "The cat sat on the floor"
	text3 := "Programming in Go is fun"

	sim12 := collector.CalculateCosineSimilarity(text1, text2)
	sim13 := collector.CalculateCosineSimilarity(text1, text3)

	// 相似文本应该有更高的相似度
	if sim12 <= sim13 {
		t.Errorf("Similar texts should have higher similarity: sim12=%f, sim13=%f", sim12, sim13)
	}

	// 相同文本应该是 1.0
	sim11 := collector.CalculateCosineSimilarity(text1, text1)
	if sim11 < 0.99 {
		t.Errorf("Same text should have similarity ~1.0, got %f", sim11)
	}
}

func TestOutputPayloadWithFingerprint(t *testing.T) {
	// 测试 OutputPayload 中指纹字段的序列化
	fp := &InferenceFingerprint{
		ModelID:       "gpt-4",
		ModelProvider: "openai",
		Statistical: &StatisticalFeatures{
			TotalTokens: 100,
		},
		Semantic: &SemanticFeatures{
			TextComplexity: 65.0,
		},
	}
	fp.FingerprintHash = fp.ComputeFingerprintHash()

	fpBytes, _ := json.Marshal(fp)

	// 模拟 OutputPayload 结构
	type OutputPayload struct {
		OutputHash           string          `json:"output_hash"`
		InferenceFingerprint json.RawMessage `json:"inference_fingerprint,omitempty"`
		FingerprintHash      string          `json:"fingerprint_hash,omitempty"`
	}

	payload := OutputPayload{
		OutputHash:           "sha256:abc123",
		InferenceFingerprint: fpBytes,
		FingerprintHash:      fp.FingerprintHash,
	}

	// 序列化
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 反序列化并验证
	var decoded OutputPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.FingerprintHash != fp.FingerprintHash {
		t.Error("FingerprintHash mismatch")
	}

	// 验证嵌入的指纹
	var decodedFp InferenceFingerprint
	if err := json.Unmarshal(decoded.InferenceFingerprint, &decodedFp); err != nil {
		t.Fatalf("Unmarshal fingerprint failed: %v", err)
	}

	if decodedFp.ModelID != "gpt-4" {
		t.Errorf("Expected ModelID=gpt-4, got %s", decodedFp.ModelID)
	}
}

// ============= Multimodal Fingerprint Tests =============

// 创建测试用的 PNG 图像
func createTestPNG(width, height int, r, g, b uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// 创建测试用的 WAV 音频
func createTestWAV(samples int, frequency float64) []byte {
	sampleRate := 44100
	channels := 1
	bitsPerSample := 16

	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	dataSize := samples * channels * bitsPerSample / 8
	fileSize := uint32(36 + dataSize)
	binary.Write(buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(channels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	binary.Write(buf, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels * bitsPerSample / 8)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))

	// 生成正弦波采样
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		sample := int16(32767 * 0.5 * sinApprox(2*3.14159*frequency*t))
		binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}

func sinApprox(x float64) float64 {
	for x > 3.14159 {
		x -= 2 * 3.14159
	}
	for x < -3.14159 {
		x += 2 * 3.14159
	}
	return x - (x*x*x)/6 + (x*x*x*x*x)/120
}

// 创建测试用的 GIF（使用标准库生成有效GIF）
func createTestGIF(width, height int) []byte {
	// 使用标准库的 gif 包创建有效的 GIF
	img := image.NewPaletted(image.Rect(0, 0, width, height), color.Palette{
		color.Black,
		color.White,
	})
	// 填充白色
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetColorIndex(x, y, 1)
		}
	}

	var buf bytes.Buffer
	gif.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestImageHasher_ComputePHash(t *testing.T) {
	hasher := NewImageHasher()
	imgData := createTestPNG(100, 100, 255, 255, 255)

	hash, err := hasher.ComputePHash(imgData)
	if err != nil {
		t.Fatalf("ComputePHash failed: %v", err)
	}

	if len(hash) != 16 {
		t.Errorf("Expected hash length 16, got %d", len(hash))
	}
	t.Logf("pHash: %s", hash)
}

func TestImageHasher_ComputeAHash(t *testing.T) {
	hasher := NewImageHasher()
	imgData := createTestPNG(100, 100, 128, 128, 128)

	hash, err := hasher.ComputeAHash(imgData)
	if err != nil {
		t.Fatalf("ComputeAHash failed: %v", err)
	}

	if len(hash) != 16 {
		t.Errorf("Expected hash length 16, got %d", len(hash))
	}
	t.Logf("aHash: %s", hash)
}

func TestImageHasher_ComputeDHash(t *testing.T) {
	hasher := NewImageHasher()
	imgData := createTestPNG(100, 100, 255, 0, 0)

	hash, err := hasher.ComputeDHash(imgData)
	if err != nil {
		t.Fatalf("ComputeDHash failed: %v", err)
	}

	if len(hash) != 16 {
		t.Errorf("Expected hash length 16, got %d", len(hash))
	}
	t.Logf("dHash: %s", hash)
}

func TestImageHasher_SimilarImages(t *testing.T) {
	hasher := NewImageHasher()

	img1 := createTestPNG(100, 100, 255, 255, 255)
	img2 := createTestPNG(100, 100, 250, 250, 250)

	hash1, _ := hasher.ComputePHash(img1)
	hash2, _ := hasher.ComputePHash(img2)

	similarity, err := hasher.CompareHashes(hash1, hash2)
	if err != nil {
		t.Fatalf("CompareHashes failed: %v", err)
	}

	if similarity < 0.7 {
		t.Errorf("Expected high similarity, got %f", similarity)
	}
	t.Logf("Similarity between similar images: %f", similarity)
}

func TestImageHasher_GetDimensions(t *testing.T) {
	hasher := NewImageHasher()
	imgData := createTestPNG(200, 150, 255, 255, 255)

	width, height, err := hasher.GetDimensions(imgData)
	if err != nil {
		t.Fatalf("GetDimensions failed: %v", err)
	}

	if width != 200 || height != 150 {
		t.Errorf("Expected 200x150, got %dx%d", width, height)
	}
}

func TestAudioHasher_ComputeFingerprint(t *testing.T) {
	hasher := NewAudioHasher()
	audioData := createTestWAV(44100, 440)

	fp, err := hasher.ComputeFingerprint(audioData)
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}

	if fp == "" {
		t.Error("Expected non-empty fingerprint")
	}
	t.Logf("Audio fingerprint: %s", fp)
}

func TestAudioHasher_GetMetadata(t *testing.T) {
	hasher := NewAudioHasher()
	audioData := createTestWAV(44100, 440)

	meta, err := hasher.GetMetadata(audioData)
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if meta.Format != "wav" {
		t.Errorf("Expected format 'wav', got '%s'", meta.Format)
	}

	if meta.SampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", meta.SampleRate)
	}

	if meta.Channels != 1 {
		t.Errorf("Expected 1 channel, got %d", meta.Channels)
	}
	t.Logf("Audio metadata: %+v", meta)
}

func TestVideoHasher_DetectFormat(t *testing.T) {
	hasher := NewVideoHasher()

	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"GIF", []byte("GIF89a" + string(make([]byte, 10))), "gif"},
		{"AVI", []byte("RIFF" + string(make([]byte, 4)) + "AVI " + string(make([]byte, 4))), "avi"},
		{"Unknown", []byte("unknown data"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := hasher.detectFormat(tt.data)
			if format != tt.expected {
				t.Errorf("Expected format '%s', got '%s'", tt.expected, format)
			}
		})
	}
}

func TestVideoHasher_ComputeFingerprint_GIF(t *testing.T) {
	hasher := NewVideoHasher()
	gifData := createTestGIF(100, 100)

	fp, err := hasher.ComputeFingerprint(gifData)
	if err != nil {
		t.Fatalf("ComputeFingerprint failed: %v", err)
	}

	if fp == "" {
		t.Error("Expected non-empty fingerprint")
	}
	t.Logf("GIF fingerprint: %s", fp)
}

func TestVideoHasher_GetMetadata_GIF(t *testing.T) {
	hasher := NewVideoHasher()
	gifData := createTestGIF(320, 240)

	meta, err := hasher.GetMetadata(gifData)
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if meta.Width != 320 {
		t.Errorf("Expected width 320, got %d", meta.Width)
	}

	if meta.Height != 240 {
		t.Errorf("Expected height 240, got %d", meta.Height)
	}
	t.Logf("GIF metadata: %+v", meta)
}

func TestMultimodalProcessor_ProcessImage(t *testing.T) {
	processor := NewMultimodalProcessor()
	imgData := createTestPNG(100, 100, 255, 255, 255)

	fp, err := processor.Process(imgData, MediaTypeImage, "png")
	if err != nil {
		t.Fatalf("Process image failed: %v", err)
	}

	if fp.MediaType != MediaTypeImage {
		t.Errorf("Expected media type 'image', got '%s'", fp.MediaType)
	}

	if fp.PerceptualHash == "" {
		t.Error("Expected non-empty perceptual hash")
	}

	if !fp.Verify() {
		t.Error("Fingerprint verification failed")
	}
	t.Logf("Image fingerprint: %+v", fp)
}

func TestMultimodalProcessor_ProcessAudio(t *testing.T) {
	processor := NewMultimodalProcessor()
	audioData := createTestWAV(44100, 440)

	fp, err := processor.Process(audioData, MediaTypeAudio, "wav")
	if err != nil {
		t.Fatalf("Process audio failed: %v", err)
	}

	if fp.MediaType != MediaTypeAudio {
		t.Errorf("Expected media type 'audio', got '%s'", fp.MediaType)
	}

	if fp.PerceptualHash == "" {
		t.Error("Expected non-empty perceptual hash")
	}

	if !fp.Verify() {
		t.Error("Fingerprint verification failed")
	}
	t.Logf("Audio fingerprint: %+v", fp)
}

func TestMultimodalProcessor_ProcessVideo(t *testing.T) {
	processor := NewMultimodalProcessor()
	videoData := createTestGIF(100, 100)

	fp, err := processor.Process(videoData, MediaTypeVideo, "gif")
	if err != nil {
		t.Fatalf("Process video failed: %v", err)
	}

	if fp.MediaType != MediaTypeVideo {
		t.Errorf("Expected media type 'video', got '%s'", fp.MediaType)
	}

	if fp.PerceptualHash == "" {
		t.Error("Expected non-empty perceptual hash")
	}

	if !fp.Verify() {
		t.Error("Fingerprint verification failed")
	}
	t.Logf("Video fingerprint: %+v", fp)
}

func TestMultimodalProcessor_ProcessText(t *testing.T) {
	processor := NewMultimodalProcessor()
	textData := []byte("This is a test text for fingerprinting.")

	fp, err := processor.Process(textData, MediaTypeText, "txt")
	if err != nil {
		t.Fatalf("Process text failed: %v", err)
	}

	if fp.MediaType != MediaTypeText {
		t.Errorf("Expected media type 'text', got '%s'", fp.MediaType)
	}

	if fp.PerceptualHash != fp.ContentHash {
		t.Error("Expected perceptual hash to equal content hash for text")
	}

	if !fp.Verify() {
		t.Error("Fingerprint verification failed")
	}
	t.Logf("Text fingerprint: %+v", fp)
}

func TestMultimodalProcessor_Compare(t *testing.T) {
	processor := NewMultimodalProcessor()

	img1 := createTestPNG(100, 100, 255, 255, 255)
	img2 := createTestPNG(100, 100, 250, 250, 250)

	fp1, _ := processor.Process(img1, MediaTypeImage, "png")
	fp2, _ := processor.Process(img2, MediaTypeImage, "png")

	similarity, err := processor.Compare(fp1, fp2)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if similarity < 0 || similarity > 1 {
		t.Errorf("Similarity should be between 0 and 1, got %f", similarity)
	}
	t.Logf("Similarity: %f", similarity)
}

func TestMultimodalProcessor_CompareTypeMismatch(t *testing.T) {
	processor := NewMultimodalProcessor()

	img := createTestPNG(100, 100, 255, 255, 255)
	audio := createTestWAV(44100, 440)

	fp1, _ := processor.Process(img, MediaTypeImage, "png")
	fp2, _ := processor.Process(audio, MediaTypeAudio, "wav")

	_, err := processor.Compare(fp1, fp2)
	if err == nil {
		t.Error("Expected error when comparing different media types")
	}
}

func TestComputeContentHash(t *testing.T) {
	data := []byte("test data for hashing")

	hash := ComputeContentHash(data)

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	hash2 := ComputeContentHash(data)
	if hash != hash2 {
		t.Error("Same data should produce same hash")
	}

	hash3 := ComputeContentHash([]byte("different data"))
	if hash == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func BenchmarkImageHasher_ComputePHash(b *testing.B) {
	hasher := NewImageHasher()
	imgData := createTestPNG(256, 256, 255, 255, 255)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.ComputePHash(imgData)
	}
}

func BenchmarkAudioHasher_ComputeFingerprint(b *testing.B) {
	hasher := NewAudioHasher()
	audioData := createTestWAV(44100, 440)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.ComputeFingerprint(audioData)
	}
}

func BenchmarkMultimodalProcessor_Process(b *testing.B) {
	processor := NewMultimodalProcessor()
	imgData := createTestPNG(100, 100, 255, 255, 255)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(imgData, MediaTypeImage, "png")
	}
}
