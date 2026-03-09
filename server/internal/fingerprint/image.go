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
	"math"
)

// ImageHasher 图像哈希器
type ImageHasher struct {
	hashSize int // 哈希尺寸 (8 = 64位哈希)
}

// NewImageHasher 创建图像哈希器
func NewImageHasher() ImageHasher {
	return ImageHasher{
		hashSize: 8, // 8x8 = 64位哈希
	}
}

// ComputePHash 计算感知哈希 (pHash)
// 基于 DCT（离散余弦变换）的感知哈希
func (h ImageHasher) ComputePHash(data []byte) (string, error) {
	// 解码图像
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 1. 缩小尺寸 (32x32 用于 DCT)
	resized := h.resize(img, 32, 32)

	// 2. 转换为灰度
	gray := h.toGrayscale(resized)

	// 3. 计算 DCT
	dct := h.computeDCT(gray)

	// 4. 取左上角 8x8 (低频部分)
	lowFreq := make([]float64, h.hashSize*h.hashSize)
	for y := 0; y < h.hashSize; y++ {
		for x := 0; x < h.hashSize; x++ {
			lowFreq[y*h.hashSize+x] = dct[y*32+x]
		}
	}

	// 5. 计算平均值 (排除第一个系数，即 DC 分量)
	var sum float64
	for i := 1; i < len(lowFreq); i++ {
		sum += lowFreq[i]
	}
	avg := sum / float64(len(lowFreq)-1)

	// 6. 生成哈希：大于平均值为 1，否则为 0
	var hash uint64
	for i := 0; i < 64; i++ {
		if lowFreq[i] > avg {
			hash |= 1 << (63 - i)
		}
	}

	return fmt.Sprintf("%016x", hash), nil
}

// ComputeAHash 计算平均哈希 (aHash)
// 更简单但精度较低的感知哈希
func (h ImageHasher) ComputeAHash(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 1. 缩小为 8x8
	resized := h.resize(img, h.hashSize, h.hashSize)

	// 2. 转换为灰度
	gray := h.toGrayscale(resized)

	// 3. 计算平均值
	var sum float64
	for _, v := range gray {
		sum += v
	}
	avg := sum / float64(len(gray))

	// 4. 生成哈希
	var hash uint64
	for i, v := range gray {
		if v > avg {
			hash |= 1 << (63 - i)
		}
	}

	return fmt.Sprintf("%016x", hash), nil
}

// ComputeDHash 计算差异哈希 (dHash)
// 基于相邻像素差异的感知哈希
func (h ImageHasher) ComputeDHash(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// 1. 缩小为 9x8 (宽度多一个像素用于比较)
	resized := h.resize(img, h.hashSize+1, h.hashSize)

	// 2. 转换为灰度
	gray := h.toGrayscale(resized)

	// 3. 计算差异哈希：比较相邻像素
	var hash uint64
	bit := 0
	for y := 0; y < h.hashSize; y++ {
		for x := 0; x < h.hashSize; x++ {
			left := gray[y*(h.hashSize+1)+x]
			right := gray[y*(h.hashSize+1)+x+1]
			if left > right {
				hash |= 1 << (63 - bit)
			}
			bit++
		}
	}

	return fmt.Sprintf("%016x", hash), nil
}

// CompareHashes 比较两个哈希的相似度
func (h ImageHasher) CompareHashes(hash1, hash2 string) (float64, error) {
	// 解析哈希
	h1, err := h.parseHash(hash1)
	if err != nil {
		return 0, err
	}
	h2, err := h.parseHash(hash2)
	if err != nil {
		return 0, err
	}

	// 计算汉明距离
	xor := h1 ^ h2
	distance := h.popCount(xor)

	// 转换为相似度 (0-1)
	similarity := 1.0 - float64(distance)/64.0
	return similarity, nil
}

// GetDimensions 获取图像尺寸
func (h ImageHasher) GetDimensions(data []byte) (width, height int, err error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}

// resize 缩放图像
func (h ImageHasher) resize(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	result := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * srcWidth / width
			srcY := y * srcHeight / height
			result.Set(x, y, img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y))
		}
	}

	return result
}

// toGrayscale 转换为灰度数组
func (h ImageHasher) toGrayscale(img image.Image) []float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	gray := make([]float64, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			// 标准灰度转换
			lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			gray[y*width+x] = lum / 65535.0 * 255.0
		}
	}

	return gray
}

// computeDCT 计算二维 DCT
func (h ImageHasher) computeDCT(gray []float64) []float64 {
	size := int(math.Sqrt(float64(len(gray))))
	result := make([]float64, len(gray))

	// 简化的 DCT-II 实现
	for v := 0; v < size; v++ {
		for u := 0; u < size; u++ {
			var sum float64
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					sum += gray[y*size+x] *
						math.Cos((2*float64(x)+1)*float64(u)*math.Pi/(2*float64(size))) *
						math.Cos((2*float64(y)+1)*float64(v)*math.Pi/(2*float64(size)))
				}
			}

			cu := 1.0
			cv := 1.0
			if u == 0 {
				cu = 1.0 / math.Sqrt(2)
			}
			if v == 0 {
				cv = 1.0 / math.Sqrt(2)
			}

			result[v*size+u] = 0.25 * cu * cv * sum
		}
	}

	return result
}

// parseHash 解析哈希字符串
func (h ImageHasher) parseHash(hash string) (uint64, error) {
	var result uint64
	_, err := fmt.Sscanf(hash, "%016x", &result)
	if err != nil {
		return 0, fmt.Errorf("invalid hash format: %w", err)
	}
	return result, nil
}

// popCount 计算二进制中 1 的个数
func (h ImageHasher) popCount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// ComputeColorHistogram 计算颜色直方图哈希
func (h ImageHasher) ComputeColorHistogram(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()

	// 简化的颜色直方图 (每个通道 16 个桶)
	histogram := make([]uint32, 16*3)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// 量化到 16 级
			histogram[int(r>>12)]++
			histogram[16+int(g>>12)]++
			histogram[32+int(b>>12)]++
		}
	}

	// 生成哈希
	buf := new(bytes.Buffer)
	for _, v := range histogram {
		binary.Write(buf, binary.LittleEndian, v)
	}

	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:16]), nil
}
