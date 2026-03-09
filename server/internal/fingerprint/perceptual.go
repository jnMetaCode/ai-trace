package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported media format")
	ErrInvalidData       = errors.New("invalid media data")
	ErrProcessingFailed  = errors.New("media processing failed")
)

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
	MediaTypeText  MediaType = "text"
)

// PerceptualHash 感知哈希接口
type PerceptualHash interface {
	// Compute 计算感知哈希
	Compute(data []byte) (string, error)

	// Compare 比较两个哈希的相似度 (0-1, 1为完全相同)
	Compare(hash1, hash2 string) (float64, error)

	// GetMediaType 获取支持的媒体类型
	GetMediaType() MediaType
}

// MultimodalFingerprint 多模态指纹
type MultimodalFingerprint struct {
	MediaType       MediaType `json:"media_type"`
	PerceptualHash  string    `json:"perceptual_hash"`
	ContentHash     string    `json:"content_hash"`      // 传统 SHA256
	SizeBytes       int64     `json:"size_bytes"`
	Format          string    `json:"format"`
	Duration        float64   `json:"duration,omitempty"`         // 音视频时长（秒）
	Width           int       `json:"width,omitempty"`            // 图像/视频宽度
	Height          int       `json:"height,omitempty"`           // 图像/视频高度
	SampleRate      int       `json:"sample_rate,omitempty"`      // 音频采样率
	Channels        int       `json:"channels,omitempty"`         // 音频通道数
	FrameRate       float64   `json:"frame_rate,omitempty"`       // 视频帧率
	KeyFrameHashes  []string  `json:"key_frame_hashes,omitempty"` // 视频关键帧哈希
	FingerprintHash string    `json:"fingerprint_hash"`           // 综合指纹哈希
}

// ComputeContentHash 计算内容哈希
func ComputeContentHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ComputeFingerprintHash 计算综合指纹哈希
func (f *MultimodalFingerprint) ComputeFingerprintHash() string {
	combined := fmt.Sprintf("%s:%s:%s:%d",
		f.MediaType,
		f.PerceptualHash,
		f.ContentHash,
		f.SizeBytes,
	)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// Verify 验证指纹完整性
func (f *MultimodalFingerprint) Verify() bool {
	computed := f.ComputeFingerprintHash()
	return computed == f.FingerprintHash
}

// MultimodalProcessor 多模态处理器
type MultimodalProcessor struct {
	imageHasher ImageHasher
	audioHasher AudioHasher
	videoHasher VideoHasher
}

// NewMultimodalProcessor 创建多模态处理器
func NewMultimodalProcessor() *MultimodalProcessor {
	return &MultimodalProcessor{
		imageHasher: NewImageHasher(),
		audioHasher: NewAudioHasher(),
		videoHasher: NewVideoHasher(),
	}
}

// Process 处理媒体数据，生成指纹
func (p *MultimodalProcessor) Process(data []byte, mediaType MediaType, format string) (*MultimodalFingerprint, error) {
	fp := &MultimodalFingerprint{
		MediaType:   mediaType,
		ContentHash: ComputeContentHash(data),
		SizeBytes:   int64(len(data)),
		Format:      format,
	}

	var err error
	switch mediaType {
	case MediaTypeImage:
		fp, err = p.processImage(data, fp)
	case MediaTypeAudio:
		fp, err = p.processAudio(data, fp)
	case MediaTypeVideo:
		fp, err = p.processVideo(data, fp)
	case MediaTypeText:
		fp.PerceptualHash = fp.ContentHash // 文本使用内容哈希
	default:
		return nil, ErrUnsupportedFormat
	}

	if err != nil {
		return nil, err
	}

	fp.FingerprintHash = fp.ComputeFingerprintHash()
	return fp, nil
}

// processImage 处理图像
func (p *MultimodalProcessor) processImage(data []byte, fp *MultimodalFingerprint) (*MultimodalFingerprint, error) {
	// 计算感知哈希
	pHash, err := p.imageHasher.ComputePHash(data)
	if err != nil {
		return nil, err
	}
	fp.PerceptualHash = pHash

	// 获取图像尺寸
	width, height, err := p.imageHasher.GetDimensions(data)
	if err == nil {
		fp.Width = width
		fp.Height = height
	}

	return fp, nil
}

// processAudio 处理音频
func (p *MultimodalProcessor) processAudio(data []byte, fp *MultimodalFingerprint) (*MultimodalFingerprint, error) {
	// 计算音频指纹
	audioFp, err := p.audioHasher.ComputeFingerprint(data)
	if err != nil {
		return nil, err
	}
	fp.PerceptualHash = audioFp

	// 获取音频元数据
	meta, err := p.audioHasher.GetMetadata(data)
	if err == nil {
		fp.Duration = meta.Duration
		fp.SampleRate = meta.SampleRate
		fp.Channels = meta.Channels
	}

	return fp, nil
}

// processVideo 处理视频
func (p *MultimodalProcessor) processVideo(data []byte, fp *MultimodalFingerprint) (*MultimodalFingerprint, error) {
	// 提取关键帧并计算哈希
	keyFrameHashes, err := p.videoHasher.ExtractKeyFrameHashes(data)
	if err != nil {
		return nil, err
	}
	fp.KeyFrameHashes = keyFrameHashes

	// 计算综合视频指纹
	videoFp, err := p.videoHasher.ComputeFingerprint(data)
	if err != nil {
		return nil, err
	}
	fp.PerceptualHash = videoFp

	// 获取视频元数据
	meta, err := p.videoHasher.GetMetadata(data)
	if err == nil {
		fp.Duration = meta.Duration
		fp.Width = meta.Width
		fp.Height = meta.Height
		fp.FrameRate = meta.FrameRate
	}

	return fp, nil
}

// Compare 比较两个多模态指纹的相似度
func (p *MultimodalProcessor) Compare(fp1, fp2 *MultimodalFingerprint) (float64, error) {
	if fp1.MediaType != fp2.MediaType {
		return 0, fmt.Errorf("cannot compare different media types: %s vs %s", fp1.MediaType, fp2.MediaType)
	}

	switch fp1.MediaType {
	case MediaTypeImage:
		return p.imageHasher.CompareHashes(fp1.PerceptualHash, fp2.PerceptualHash)
	case MediaTypeAudio:
		return p.audioHasher.CompareFingerprints(fp1.PerceptualHash, fp2.PerceptualHash)
	case MediaTypeVideo:
		return p.videoHasher.CompareFingerprints(fp1, fp2)
	case MediaTypeText:
		if fp1.ContentHash == fp2.ContentHash {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, ErrUnsupportedFormat
	}
}
