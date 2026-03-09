package fingerprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/cmplx"
)

// AudioMetadata 音频元数据
type AudioMetadata struct {
	Duration   float64 `json:"duration"`    // 时长（秒）
	SampleRate int     `json:"sample_rate"` // 采样率
	Channels   int     `json:"channels"`    // 通道数
	BitDepth   int     `json:"bit_depth"`   // 位深度
	Format     string  `json:"format"`      // 格式
}

// AudioHasher 音频哈希器
type AudioHasher struct {
	frameSize    int     // FFT 帧大小
	hopSize      int     // 帧移动步长
	numBands     int     // 频带数量
	minFreq      float64 // 最低频率
	maxFreq      float64 // 最高频率
}

// NewAudioHasher 创建音频哈希器
func NewAudioHasher() AudioHasher {
	return AudioHasher{
		frameSize: 4096,
		hopSize:   2048,
		numBands:  33,   // Chromaprint 使用 33 个频带
		minFreq:   300,  // 最低频率 300Hz
		maxFreq:   2000, // 最高频率 2000Hz
	}
}

// ComputeFingerprint 计算音频指纹
// 基于 Chromaprint 算法的简化实现
func (h AudioHasher) ComputeFingerprint(data []byte) (string, error) {
	// 解析音频数据（简化：假设原始 PCM）
	samples, err := h.parsePCM(data)
	if err != nil {
		// 如果无法解析，使用内容哈希作为后备
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:16]), nil
	}

	// 计算 Chroma 特征
	chromaFeatures := h.computeChroma(samples, 44100) // 假设 44.1kHz

	// 量化为指纹
	fingerprint := h.quantizeChroma(chromaFeatures)

	return fingerprint, nil
}

// parsePCM 解析 PCM 音频数据
func (h AudioHasher) parsePCM(data []byte) ([]float64, error) {
	if len(data) < 44 {
		return nil, fmt.Errorf("data too short for PCM")
	}

	// 检查是否为 WAV 文件
	if string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return h.parseWAV(data)
	}

	// 尝试直接作为 16 位 PCM 解析
	samples := make([]float64, len(data)/2)
	reader := bytes.NewReader(data)
	for i := range samples {
		var sample int16
		if err := binary.Read(reader, binary.LittleEndian, &sample); err != nil {
			break
		}
		samples[i] = float64(sample) / 32768.0
	}

	return samples, nil
}

// parseWAV 解析 WAV 文件
func (h AudioHasher) parseWAV(data []byte) ([]float64, error) {
	if len(data) < 44 {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// 读取格式信息
	numChannels := int(binary.LittleEndian.Uint16(data[22:24]))
	bitsPerSample := int(binary.LittleEndian.Uint16(data[34:36]))

	// 找到数据块
	dataOffset := 44
	for i := 36; i < len(data)-8; i++ {
		if string(data[i:i+4]) == "data" {
			dataOffset = i + 8
			break
		}
	}

	// 读取采样数据
	audioData := data[dataOffset:]
	bytesPerSample := bitsPerSample / 8
	numSamples := len(audioData) / bytesPerSample / numChannels

	samples := make([]float64, numSamples)
	reader := bytes.NewReader(audioData)

	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < numChannels; ch++ {
			var sample float64
			switch bitsPerSample {
			case 8:
				var s uint8
				binary.Read(reader, binary.LittleEndian, &s)
				sample = (float64(s) - 128) / 128.0
			case 16:
				var s int16
				binary.Read(reader, binary.LittleEndian, &s)
				sample = float64(s) / 32768.0
			case 24:
				buf := make([]byte, 3)
				reader.Read(buf)
				s := int32(buf[0]) | int32(buf[1])<<8 | int32(buf[2])<<16
				if s&0x800000 != 0 {
					s |= -1 << 24
				}
				sample = float64(s) / 8388608.0
			case 32:
				var s int32
				binary.Read(reader, binary.LittleEndian, &s)
				sample = float64(s) / 2147483648.0
			}
			sum += sample
		}
		samples[i] = sum / float64(numChannels) // 混合为单声道
	}

	return samples, nil
}

// computeChroma 计算 Chroma 特征
func (h AudioHasher) computeChroma(samples []float64, sampleRate int) [][]float64 {
	numFrames := (len(samples) - h.frameSize) / h.hopSize
	if numFrames <= 0 {
		return nil
	}

	chroma := make([][]float64, numFrames)

	for i := 0; i < numFrames; i++ {
		start := i * h.hopSize
		frame := samples[start : start+h.frameSize]

		// 应用汉宁窗
		windowed := h.applyWindow(frame)

		// FFT
		spectrum := h.fft(windowed)

		// 计算 Chroma 向量
		chroma[i] = h.computeChromaVector(spectrum, sampleRate)
	}

	return chroma
}

// applyWindow 应用汉宁窗
func (h AudioHasher) applyWindow(frame []float64) []float64 {
	windowed := make([]float64, len(frame))
	n := float64(len(frame))
	for i, v := range frame {
		// 汉宁窗
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/n))
		windowed[i] = v * window
	}
	return windowed
}

// fft 快速傅里叶变换
func (h AudioHasher) fft(x []float64) []complex128 {
	n := len(x)
	if n == 1 {
		return []complex128{complex(x[0], 0)}
	}

	// Cooley-Tukey FFT
	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	evenFFT := h.fft(even)
	oddFFT := h.fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * oddFFT[k]
		result[k] = evenFFT[k] + t
		result[k+n/2] = evenFFT[k] - t
	}

	return result
}

// computeChromaVector 计算 Chroma 向量
func (h AudioHasher) computeChromaVector(spectrum []complex128, sampleRate int) []float64 {
	chroma := make([]float64, 12) // 12 个半音

	n := len(spectrum) / 2 // 只使用正频率
	for i := 1; i < n; i++ {
		freq := float64(i) * float64(sampleRate) / float64(len(spectrum))
		if freq < h.minFreq || freq > h.maxFreq {
			continue
		}

		// 计算音高类别 (0-11)
		pitch := h.freqToPitch(freq)
		pitchClass := int(math.Mod(pitch, 12))
		if pitchClass < 0 {
			pitchClass += 12
		}

		// 累加功率
		power := cmplx.Abs(spectrum[i])
		chroma[pitchClass] += power * power
	}

	// 归一化
	var sum float64
	for _, v := range chroma {
		sum += v
	}
	if sum > 0 {
		for i := range chroma {
			chroma[i] /= sum
		}
	}

	return chroma
}

// freqToPitch 频率转音高
func (h AudioHasher) freqToPitch(freq float64) float64 {
	return 12 * math.Log2(freq/440) + 69
}

// quantizeChroma 量化 Chroma 特征为指纹
func (h AudioHasher) quantizeChroma(chroma [][]float64) string {
	if len(chroma) == 0 {
		return ""
	}

	// 计算帧间差异并量化
	fingerprint := make([]byte, 0)

	for i := 1; i < len(chroma); i++ {
		var bits byte
		for j := 0; j < 8 && j < 12; j++ {
			if chroma[i][j] > chroma[i-1][j] {
				bits |= 1 << j
			}
		}
		fingerprint = append(fingerprint, bits)
	}

	// 生成哈希
	hash := sha256.Sum256(fingerprint)
	return hex.EncodeToString(hash[:16])
}

// CompareFingerprints 比较两个音频指纹
func (h AudioHasher) CompareFingerprints(fp1, fp2 string) (float64, error) {
	if fp1 == fp2 {
		return 1.0, nil
	}

	// 解析指纹
	bytes1, err := hex.DecodeString(fp1)
	if err != nil {
		return 0, err
	}
	bytes2, err := hex.DecodeString(fp2)
	if err != nil {
		return 0, err
	}

	// 计算相似度（使用汉明距离）
	minLen := len(bytes1)
	if len(bytes2) < minLen {
		minLen = len(bytes2)
	}

	var hammingDist int
	for i := 0; i < minLen; i++ {
		xor := bytes1[i] ^ bytes2[i]
		for xor != 0 {
			hammingDist++
			xor &= xor - 1
		}
	}

	maxBits := minLen * 8
	similarity := 1.0 - float64(hammingDist)/float64(maxBits)
	return similarity, nil
}

// GetMetadata 获取音频元数据
func (h AudioHasher) GetMetadata(data []byte) (*AudioMetadata, error) {
	meta := &AudioMetadata{}

	if len(data) < 44 {
		return nil, fmt.Errorf("data too short")
	}

	// 检查 WAV
	if string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		meta.Format = "wav"
		meta.SampleRate = int(binary.LittleEndian.Uint32(data[24:28]))
		meta.Channels = int(binary.LittleEndian.Uint16(data[22:24]))
		meta.BitDepth = int(binary.LittleEndian.Uint16(data[34:36]))

		// 计算时长
		dataSize := binary.LittleEndian.Uint32(data[40:44])
		bytesPerSample := meta.BitDepth / 8 * meta.Channels
		if bytesPerSample > 0 {
			meta.Duration = float64(dataSize) / float64(meta.SampleRate) / float64(bytesPerSample)
		}
	}

	return meta, nil
}
