package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// SHA256 计算字符串的SHA256哈希
func SHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}

// SHA256Bytes 计算字节的SHA256哈希
func SHA256Bytes(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}

// SHA256Reader 计算Reader的SHA256哈希
func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil))), nil
}

// SHA256JSON 计算JSON对象的SHA256哈希（规范化后）
func SHA256JSON(v interface{}) (string, error) {
	// 序列化为JSON
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	// 反序列化为map以进行规范化
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		// 如果不是map，直接计算哈希
		return SHA256Bytes(data), nil
	}

	// 规范化后重新序列化
	normalized, err := normalizeJSON(m)
	if err != nil {
		return "", err
	}

	return SHA256Bytes(normalized), nil
}

// normalizeJSON 规范化JSON（按key排序）
func normalizeJSON(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		// 获取所有key并排序
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// 按排序顺序构建结果
		result := make(map[string]interface{})
		for _, k := range keys {
			normalized, err := normalizeValue(val[k])
			if err != nil {
				return nil, err
			}
			result[k] = normalized
		}
		return json.Marshal(result)

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			normalized, err := normalizeValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = normalized
		}
		return json.Marshal(result)

	default:
		return json.Marshal(v)
	}
}

func normalizeValue(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := make(map[string]interface{})
		for _, k := range keys {
			normalized, err := normalizeValue(val[k])
			if err != nil {
				return nil, err
			}
			result[k] = normalized
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			normalized, err := normalizeValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = normalized
		}
		return result, nil

	default:
		return v, nil
	}
}

// CombineHashes 组合多个哈希值
func CombineHashes(hashes ...string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
	}
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil)))
}
