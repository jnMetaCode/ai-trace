package fingerprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sort"
)

// VideoMetadata 视频元数据
type VideoMetadata struct {
	Duration  float64 `json:"duration"`   // 时长（秒）
	Width     int     `json:"width"`      // 宽度
	Height    int     `json:"height"`     // 高度
	FrameRate float64 `json:"frame_rate"` // 帧率
	Codec     string  `json:"codec"`      // 编解码器
	BitRate   int     `json:"bit_rate"`   // 比特率
}

// KeyFrame 关键帧
type KeyFrame struct {
	Timestamp float64 `json:"timestamp"` // 时间戳（秒）
	Hash      string  `json:"hash"`      // 感知哈希
	Index     int     `json:"index"`     // 帧索引
}

// VideoHasher 视频哈希器
type VideoHasher struct {
	imageHasher    ImageHasher
	maxKeyFrames   int     // 最大关键帧数
	frameInterval  float64 // 关键帧间隔（秒）
	similarityThreshold float64 // 相似度阈值
}

// NewVideoHasher 创建视频哈希器
func NewVideoHasher() VideoHasher {
	return VideoHasher{
		imageHasher:    NewImageHasher(),
		maxKeyFrames:   16,   // 最多提取 16 个关键帧
		frameInterval:  2.0,  // 每 2 秒提取一帧
		similarityThreshold: 0.85, // 相似度阈值
	}
}

// ComputeFingerprint 计算视频指纹
// 基于关键帧哈希的综合指纹
func (h VideoHasher) ComputeFingerprint(data []byte) (string, error) {
	// 提取关键帧哈希
	keyFrameHashes, err := h.ExtractKeyFrameHashes(data)
	if err != nil || len(keyFrameHashes) == 0 {
		// 如果无法提取关键帧，使用内容哈希作为后备
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:16]), nil
	}

	// 计算综合指纹
	return h.computeCombinedHash(keyFrameHashes), nil
}

// ExtractKeyFrameHashes 提取关键帧哈希
func (h VideoHasher) ExtractKeyFrameHashes(data []byte) ([]string, error) {
	// 尝试解析视频容器
	keyFrames, err := h.extractKeyFrames(data)
	if err != nil {
		return nil, err
	}

	hashes := make([]string, 0, len(keyFrames))
	for _, kf := range keyFrames {
		hashes = append(hashes, kf.Hash)
	}

	return hashes, nil
}

// extractKeyFrames 提取关键帧
func (h VideoHasher) extractKeyFrames(data []byte) ([]KeyFrame, error) {
	keyFrames := make([]KeyFrame, 0)

	// 尝试识别视频格式并提取帧
	format := h.detectFormat(data)

	switch format {
	case "mp4", "mov":
		return h.extractFromMP4(data)
	case "avi":
		return h.extractFromAVI(data)
	case "gif":
		return h.extractFromGIF(data)
	case "mjpeg":
		return h.extractFromMJPEG(data)
	default:
		// 尝试直接查找 JPEG 标记
		embedded := h.extractEmbeddedImages(data)
		keyFrames = append(keyFrames, embedded...)
	}

	return keyFrames, nil
}

// detectFormat 检测视频格式
func (h VideoHasher) detectFormat(data []byte) string {
	if len(data) < 12 {
		return "unknown"
	}

	// MP4/MOV: ftyp box
	if len(data) >= 8 && string(data[4:8]) == "ftyp" {
		return "mp4"
	}

	// AVI: RIFF....AVI
	if string(data[:4]) == "RIFF" && len(data) >= 12 && string(data[8:12]) == "AVI " {
		return "avi"
	}

	// GIF
	if string(data[:6]) == "GIF89a" || string(data[:6]) == "GIF87a" {
		return "gif"
	}

	// WebM/MKV: EBML header
	if data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		return "webm"
	}

	// MPEG-TS
	if data[0] == 0x47 {
		return "ts"
	}

	// 检查是否包含 JPEG 数据
	if h.containsJPEG(data) {
		return "mjpeg"
	}

	return "unknown"
}

// containsJPEG 检查是否包含 JPEG 数据
func (h VideoHasher) containsJPEG(data []byte) bool {
	jpegStart := []byte{0xFF, 0xD8, 0xFF}
	return bytes.Contains(data, jpegStart)
}

// extractFromMP4 从 MP4 提取关键帧
func (h VideoHasher) extractFromMP4(data []byte) ([]KeyFrame, error) {
	keyFrames := make([]KeyFrame, 0)

	// 简化的 MP4 解析：查找 mdat box 中的图像数据
	// 实际实现需要完整的 MP4 解析器

	// 查找 moov 和 mdat box
	offset := 0
	var mdatStart, mdatSize int

	for offset < len(data)-8 {
		boxSize := int(binary.BigEndian.Uint32(data[offset:]))
		boxType := string(data[offset+4 : offset+8])

		if boxSize == 0 {
			break
		}

		if boxType == "mdat" {
			mdatStart = offset + 8
			mdatSize = boxSize - 8
			break
		}

		offset += boxSize
	}

	if mdatStart > 0 && mdatSize > 0 {
		// 从 mdat 中提取嵌入的图像帧
		mdatData := data[mdatStart : mdatStart+mdatSize]
		embedded := h.extractEmbeddedImages(mdatData)
		keyFrames = append(keyFrames, embedded...)
	}

	// 限制关键帧数量
	if len(keyFrames) > h.maxKeyFrames {
		keyFrames = h.selectRepresentativeFrames(keyFrames)
	}

	return keyFrames, nil
}

// extractFromAVI 从 AVI 提取关键帧
func (h VideoHasher) extractFromAVI(data []byte) ([]KeyFrame, error) {
	keyFrames := make([]KeyFrame, 0)

	// 简化的 AVI 解析：查找 movi list 中的视频帧
	offset := 12 // 跳过 RIFF header

	for offset < len(data)-8 {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4:]))

		if chunkSize == 0 || offset+8+chunkSize > len(data) {
			break
		}

		// 查找视频帧 (00dc, 01dc, etc.)
		if len(chunkID) >= 4 && (chunkID[2:4] == "dc" || chunkID[2:4] == "db") {
			frameData := data[offset+8 : offset+8+chunkSize]

			// 尝试解码为图像
			if hash, err := h.hashFrameData(frameData); err == nil {
				keyFrames = append(keyFrames, KeyFrame{
					Index: len(keyFrames),
					Hash:  hash,
				})
			}
		}

		// RIFF chunks are padded to even boundaries
		offset += 8 + chunkSize
		if chunkSize%2 != 0 {
			offset++
		}

		if len(keyFrames) >= h.maxKeyFrames {
			break
		}
	}

	return keyFrames, nil
}

// extractFromGIF 从 GIF 提取关键帧
func (h VideoHasher) extractFromGIF(data []byte) ([]KeyFrame, error) {
	keyFrames := make([]KeyFrame, 0)

	// 解码 GIF
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// GIF 作为单帧处理
	hash, err := h.imageHasher.ComputePHash(data)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	keyFrames = append(keyFrames, KeyFrame{
		Index: 0,
		Hash:  hash,
		Timestamp: 0,
	})

	// 如果是动画 GIF，可以进一步解析帧
	// 这里简化处理，只取第一帧
	_ = bounds

	return keyFrames, nil
}

// extractFromMJPEG 从 MJPEG 提取关键帧
func (h VideoHasher) extractFromMJPEG(data []byte) ([]KeyFrame, error) {
	return h.extractEmbeddedImages(data), nil
}

// extractEmbeddedImages 提取嵌入的图像
func (h VideoHasher) extractEmbeddedImages(data []byte) []KeyFrame {
	keyFrames := make([]KeyFrame, 0)

	// JPEG 标记
	jpegStart := []byte{0xFF, 0xD8}
	jpegEnd := []byte{0xFF, 0xD9}

	offset := 0
	frameIndex := 0

	for offset < len(data)-4 && len(keyFrames) < h.maxKeyFrames {
		// 查找 JPEG 开始
		startIdx := bytes.Index(data[offset:], jpegStart)
		if startIdx == -1 {
			break
		}
		startIdx += offset

		// 查找 JPEG 结束
		endIdx := bytes.Index(data[startIdx+2:], jpegEnd)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx + 2 + 2 // 包含结束标记

		// 提取 JPEG 数据
		jpegData := data[startIdx:endIdx]

		// 计算哈希
		if hash, err := h.hashFrameData(jpegData); err == nil {
			keyFrames = append(keyFrames, KeyFrame{
				Index: frameIndex,
				Hash:  hash,
			})
			frameIndex++
		}

		offset = endIdx
	}

	return keyFrames
}

// hashFrameData 计算帧数据的哈希
func (h VideoHasher) hashFrameData(frameData []byte) (string, error) {
	// 尝试作为图像解码
	_, _, err := image.Decode(bytes.NewReader(frameData))
	if err != nil {
		// 不是有效图像，使用内容哈希
		hash := sha256.Sum256(frameData)
		return hex.EncodeToString(hash[:8]), nil
	}

	// 计算感知哈希
	return h.imageHasher.ComputePHash(frameData)
}

// selectRepresentativeFrames 选择代表性帧
func (h VideoHasher) selectRepresentativeFrames(frames []KeyFrame) []KeyFrame {
	if len(frames) <= h.maxKeyFrames {
		return frames
	}

	// 均匀采样
	selected := make([]KeyFrame, 0, h.maxKeyFrames)
	step := float64(len(frames)) / float64(h.maxKeyFrames)

	for i := 0; i < h.maxKeyFrames; i++ {
		idx := int(float64(i) * step)
		if idx >= len(frames) {
			idx = len(frames) - 1
		}
		selected = append(selected, frames[idx])
	}

	return selected
}

// computeCombinedHash 计算组合哈希
func (h VideoHasher) computeCombinedHash(hashes []string) string {
	// 排序以保证一致性
	sorted := make([]string, len(hashes))
	copy(sorted, hashes)
	sort.Strings(sorted)

	// 拼接并哈希
	combined := ""
	for _, hash := range sorted {
		combined += hash
	}

	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16])
}

// CompareFingerprints 比较两个视频指纹
func (h VideoHasher) CompareFingerprints(fp1, fp2 *MultimodalFingerprint) (float64, error) {
	// 1. 首先比较综合指纹
	if fp1.PerceptualHash == fp2.PerceptualHash {
		return 1.0, nil
	}

	// 2. 如果有关键帧哈希，进行详细比较
	if len(fp1.KeyFrameHashes) > 0 && len(fp2.KeyFrameHashes) > 0 {
		return h.compareKeyFrameHashes(fp1.KeyFrameHashes, fp2.KeyFrameHashes)
	}

	// 3. 比较感知哈希
	return h.imageHasher.CompareHashes(fp1.PerceptualHash, fp2.PerceptualHash)
}

// compareKeyFrameHashes 比较关键帧哈希
func (h VideoHasher) compareKeyFrameHashes(hashes1, hashes2 []string) (float64, error) {
	if len(hashes1) == 0 || len(hashes2) == 0 {
		return 0, nil
	}

	// 计算最佳匹配得分
	totalScore := 0.0
	matchCount := 0

	// 对于每个 hash1，找到最佳匹配的 hash2
	for _, h1 := range hashes1 {
		bestScore := 0.0
		for _, h2 := range hashes2 {
			score, err := h.imageHasher.CompareHashes(h1, h2)
			if err != nil {
				continue
			}
			if score > bestScore {
				bestScore = score
			}
		}
		if bestScore > h.similarityThreshold {
			totalScore += bestScore
			matchCount++
		}
	}

	// 计算双向匹配
	for _, h2 := range hashes2 {
		bestScore := 0.0
		for _, h1 := range hashes1 {
			score, err := h.imageHasher.CompareHashes(h1, h2)
			if err != nil {
				continue
			}
			if score > bestScore {
				bestScore = score
			}
		}
		if bestScore > h.similarityThreshold {
			totalScore += bestScore
			matchCount++
		}
	}

	// 归一化
	maxMatches := len(hashes1) + len(hashes2)
	if maxMatches == 0 {
		return 0, nil
	}

	similarity := totalScore / float64(maxMatches)
	return similarity, nil
}

// GetMetadata 获取视频元数据
func (h VideoHasher) GetMetadata(data []byte) (*VideoMetadata, error) {
	meta := &VideoMetadata{}

	format := h.detectFormat(data)

	switch format {
	case "mp4", "mov":
		return h.getMP4Metadata(data)
	case "avi":
		return h.getAVIMetadata(data)
	case "gif":
		return h.getGIFMetadata(data)
	default:
		// 尝试基本解析
		meta.Codec = format
	}

	return meta, nil
}

// getMP4Metadata 获取 MP4 元数据
func (h VideoHasher) getMP4Metadata(data []byte) (*VideoMetadata, error) {
	meta := &VideoMetadata{
		Codec: "h264", // 假设
	}

	// 简化的 MP4 box 解析
	offset := 0
	for offset < len(data)-8 {
		boxSize := int(binary.BigEndian.Uint32(data[offset:]))
		boxType := string(data[offset+4 : offset+8])

		if boxSize == 0 {
			break
		}

		if boxSize > len(data)-offset {
			break
		}

		switch boxType {
		case "moov":
			// 解析 moov box 内部
			h.parseMP4MoovBox(data[offset+8:offset+boxSize], meta)
		case "ftyp":
			// 品牌类型
			if boxSize >= 12 {
				brand := string(data[offset+8 : offset+12])
				if brand == "qt  " {
					meta.Codec = "quicktime"
				}
			}
		}

		offset += boxSize
	}

	return meta, nil
}

// parseMP4MoovBox 解析 moov box
func (h VideoHasher) parseMP4MoovBox(data []byte, meta *VideoMetadata) {
	offset := 0
	for offset < len(data)-8 {
		boxSize := int(binary.BigEndian.Uint32(data[offset:]))
		boxType := string(data[offset+4 : offset+8])

		if boxSize == 0 || boxSize > len(data)-offset {
			break
		}

		switch boxType {
		case "mvhd":
			// Movie header box
			if boxSize >= 24 {
				version := data[offset+8]
				if version == 0 {
					timescale := binary.BigEndian.Uint32(data[offset+20:])
					duration := binary.BigEndian.Uint32(data[offset+24:])
					if timescale > 0 {
						meta.Duration = float64(duration) / float64(timescale)
					}
				}
			}
		case "trak":
			// Track box - 解析视频轨道
			h.parseMP4TrakBox(data[offset+8:offset+boxSize], meta)
		}

		offset += boxSize
	}
}

// parseMP4TrakBox 解析 trak box
func (h VideoHasher) parseMP4TrakBox(data []byte, meta *VideoMetadata) {
	offset := 0
	for offset < len(data)-8 {
		boxSize := int(binary.BigEndian.Uint32(data[offset:]))
		boxType := string(data[offset+4 : offset+8])

		if boxSize == 0 || boxSize > len(data)-offset {
			break
		}

		if boxType == "tkhd" {
			// Track header
			if boxSize >= 84 {
				version := data[offset+8]
				if version == 0 {
					width := binary.BigEndian.Uint32(data[offset+76:])
					height := binary.BigEndian.Uint32(data[offset+80:])
					meta.Width = int(width >> 16)
					meta.Height = int(height >> 16)
				}
			}
		}

		offset += boxSize
	}
}

// getAVIMetadata 获取 AVI 元数据
func (h VideoHasher) getAVIMetadata(data []byte) (*VideoMetadata, error) {
	meta := &VideoMetadata{
		Codec: "avi",
	}

	if len(data) < 56 {
		return meta, nil
	}

	// 查找 avih chunk
	offset := 12
	for offset < len(data)-8 {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4:]))

		if chunkID == "avih" && chunkSize >= 40 {
			// 微秒每帧
			microsPerFrame := binary.LittleEndian.Uint32(data[offset+8:])
			if microsPerFrame > 0 {
				meta.FrameRate = 1000000.0 / float64(microsPerFrame)
			}

			// 总帧数
			totalFrames := binary.LittleEndian.Uint32(data[offset+24:])

			// 宽度和高度
			meta.Width = int(binary.LittleEndian.Uint32(data[offset+40:]))
			meta.Height = int(binary.LittleEndian.Uint32(data[offset+44:]))

			// 计算时长
			if meta.FrameRate > 0 {
				meta.Duration = float64(totalFrames) / meta.FrameRate
			}

			break
		}

		if chunkID == "LIST" {
			offset += 12 // 跳过 LIST header
			continue
		}

		offset += 8 + chunkSize
		if chunkSize%2 != 0 {
			offset++
		}
	}

	return meta, nil
}

// getGIFMetadata 获取 GIF 元数据
func (h VideoHasher) getGIFMetadata(data []byte) (*VideoMetadata, error) {
	meta := &VideoMetadata{
		Codec: "gif",
	}

	if len(data) < 10 {
		return meta, nil
	}

	// GIF 逻辑屏幕描述符
	meta.Width = int(binary.LittleEndian.Uint16(data[6:]))
	meta.Height = int(binary.LittleEndian.Uint16(data[8:]))

	// 计算帧数和时长（简化：假设每帧 100ms）
	frameCount := h.countGIFFrames(data)
	meta.Duration = float64(frameCount) * 0.1
	if meta.Duration > 0 {
		meta.FrameRate = float64(frameCount) / meta.Duration
	}

	return meta, nil
}

// countGIFFrames 计算 GIF 帧数
func (h VideoHasher) countGIFFrames(data []byte) int {
	count := 0
	offset := 13 // 跳过头部

	// 跳过全局颜色表
	if len(data) > 10 && data[10]&0x80 != 0 {
		colorTableSize := 1 << ((data[10] & 0x07) + 1)
		offset += colorTableSize * 3
	}

	for offset < len(data)-1 {
		switch data[offset] {
		case 0x21: // 扩展块
			if offset+2 < len(data) {
				offset += 2 // 跳过扩展类型
				for offset < len(data) && data[offset] != 0 {
					blockSize := int(data[offset])
					offset += 1 + blockSize
				}
				offset++ // 跳过块终止符
			}
		case 0x2C: // 图像描述符
			count++
			if offset+10 < len(data) {
				offset += 10
				// 跳过局部颜色表
				if data[offset-1]&0x80 != 0 {
					colorTableSize := 1 << ((data[offset-1] & 0x07) + 1)
					offset += colorTableSize * 3
				}
				// 跳过图像数据
				offset++ // LZW 最小代码大小
				for offset < len(data) && data[offset] != 0 {
					blockSize := int(data[offset])
					offset += 1 + blockSize
				}
				offset++ // 块终止符
			}
		case 0x3B: // 结束
			return count
		default:
			offset++
		}
	}

	return count
}

// SceneChangeDetector 场景变化检测器
type SceneChangeDetector struct {
	threshold float64
	imageHasher ImageHasher
}

// NewSceneChangeDetector 创建场景变化检测器
func NewSceneChangeDetector(threshold float64) *SceneChangeDetector {
	return &SceneChangeDetector{
		threshold:   threshold,
		imageHasher: NewImageHasher(),
	}
}

// DetectSceneChanges 检测场景变化
func (d *SceneChangeDetector) DetectSceneChanges(frames []KeyFrame) []int {
	if len(frames) < 2 {
		return nil
	}

	changes := make([]int, 0)

	for i := 1; i < len(frames); i++ {
		similarity, err := d.imageHasher.CompareHashes(frames[i-1].Hash, frames[i].Hash)
		if err != nil {
			continue
		}

		// 相似度低于阈值说明场景变化
		if similarity < d.threshold {
			changes = append(changes, i)
		}
	}

	return changes
}

// VideoFingerprint 视频指纹（扩展结构）
type VideoFingerprint struct {
	// 基本信息
	ContentHash    string   `json:"content_hash"`
	PerceptualHash string   `json:"perceptual_hash"`
	KeyFrameHashes []string `json:"key_frame_hashes"`

	// 时间特征
	Duration    float64 `json:"duration"`
	FrameRate   float64 `json:"frame_rate"`
	SceneCount  int     `json:"scene_count"`

	// 空间特征
	Width  int `json:"width"`
	Height int `json:"height"`

	// 场景变化点
	SceneChanges []float64 `json:"scene_changes,omitempty"`
}

// ComputeDetailedFingerprint 计算详细的视频指纹
func (h VideoHasher) ComputeDetailedFingerprint(data []byte) (*VideoFingerprint, error) {
	fp := &VideoFingerprint{
		ContentHash: ComputeContentHash(data),
	}

	// 获取元数据
	meta, err := h.GetMetadata(data)
	if err == nil {
		fp.Duration = meta.Duration
		fp.FrameRate = meta.FrameRate
		fp.Width = meta.Width
		fp.Height = meta.Height
	}

	// 提取关键帧
	keyFrames, err := h.extractKeyFrames(data)
	if err != nil {
		// 使用内容哈希作为后备
		hash := sha256.Sum256(data)
		fp.PerceptualHash = hex.EncodeToString(hash[:16])
		return fp, nil
	}

	// 提取关键帧哈希
	fp.KeyFrameHashes = make([]string, len(keyFrames))
	for i, kf := range keyFrames {
		fp.KeyFrameHashes[i] = kf.Hash
	}

	// 计算综合指纹
	fp.PerceptualHash = h.computeCombinedHash(fp.KeyFrameHashes)

	// 检测场景变化
	detector := NewSceneChangeDetector(0.7)
	sceneChangeIndices := detector.DetectSceneChanges(keyFrames)
	fp.SceneCount = len(sceneChangeIndices) + 1

	// 转换为时间点
	if fp.Duration > 0 && len(keyFrames) > 0 {
		fp.SceneChanges = make([]float64, len(sceneChangeIndices))
		for i, idx := range sceneChangeIndices {
			fp.SceneChanges[i] = float64(idx) / float64(len(keyFrames)) * fp.Duration
		}
	}

	return fp, nil
}

// CompareDetailedFingerprints 比较详细的视频指纹
func (h VideoHasher) CompareDetailedFingerprints(fp1, fp2 *VideoFingerprint) (float64, error) {
	// 多维度相似度计算
	scores := make([]float64, 0)
	weights := make([]float64, 0)

	// 1. 感知哈希相似度 (权重最高)
	if fp1.PerceptualHash != "" && fp2.PerceptualHash != "" {
		hashSim, err := h.imageHasher.CompareHashes(fp1.PerceptualHash, fp2.PerceptualHash)
		if err == nil {
			scores = append(scores, hashSim)
			weights = append(weights, 0.5)
		}
	}

	// 2. 关键帧匹配度
	if len(fp1.KeyFrameHashes) > 0 && len(fp2.KeyFrameHashes) > 0 {
		kfSim, err := h.compareKeyFrameHashes(fp1.KeyFrameHashes, fp2.KeyFrameHashes)
		if err == nil {
			scores = append(scores, kfSim)
			weights = append(weights, 0.3)
		}
	}

	// 3. 时长相似度
	if fp1.Duration > 0 && fp2.Duration > 0 {
		durationDiff := abs(fp1.Duration-fp2.Duration) / max(fp1.Duration, fp2.Duration)
		durationSim := 1.0 - min(durationDiff, 1.0)
		scores = append(scores, durationSim)
		weights = append(weights, 0.1)
	}

	// 4. 分辨率相似度
	if fp1.Width > 0 && fp1.Height > 0 && fp2.Width > 0 && fp2.Height > 0 {
		area1 := float64(fp1.Width * fp1.Height)
		area2 := float64(fp2.Width * fp2.Height)
		areaDiff := abs(area1-area2) / max(area1, area2)
		areaSim := 1.0 - min(areaDiff, 1.0)
		scores = append(scores, areaSim)
		weights = append(weights, 0.1)
	}

	// 计算加权平均
	if len(scores) == 0 {
		return 0, fmt.Errorf("no comparable features")
	}

	var totalWeight, weightedSum float64
	for i := range scores {
		weightedSum += scores[i] * weights[i]
		totalWeight += weights[i]
	}

	return weightedSum / totalWeight, nil
}

// 辅助函数
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
