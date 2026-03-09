package hash

import (
	"bytes"
	"strings"
	"testing"
)

func TestSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPfx  string
		wantLen  int
	}{
		{
			name:    "empty string",
			input:   "",
			wantPfx: "sha256:",
			wantLen: 7 + 64, // "sha256:" + 64 hex chars
		},
		{
			name:    "hello",
			input:   "hello",
			wantPfx: "sha256:",
			wantLen: 7 + 64,
		},
		{
			name:    "unicode",
			input:   "你好世界",
			wantPfx: "sha256:",
			wantLen: 7 + 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256(tt.input)
			if !strings.HasPrefix(got, tt.wantPfx) {
				t.Errorf("SHA256() prefix = %v, want %v", got[:7], tt.wantPfx)
			}
			if len(got) != tt.wantLen {
				t.Errorf("SHA256() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestSHA256Consistency(t *testing.T) {
	input := "test input for consistency"

	hash1 := SHA256(input)
	hash2 := SHA256(input)

	if hash1 != hash2 {
		t.Errorf("SHA256() not consistent: %v != %v", hash1, hash2)
	}
}

func TestSHA256Different(t *testing.T) {
	hash1 := SHA256("input1")
	hash2 := SHA256("input2")

	if hash1 == hash2 {
		t.Error("SHA256() should produce different hashes for different inputs")
	}
}

func TestSHA256Bytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantPfx string
		wantLen int
	}{
		{
			name:    "empty bytes",
			input:   []byte{},
			wantPfx: "sha256:",
			wantLen: 7 + 64,
		},
		{
			name:    "hello bytes",
			input:   []byte("hello"),
			wantPfx: "sha256:",
			wantLen: 7 + 64,
		},
		{
			name:    "binary data",
			input:   []byte{0x00, 0x01, 0x02, 0xff, 0xfe},
			wantPfx: "sha256:",
			wantLen: 7 + 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SHA256Bytes(tt.input)
			if !strings.HasPrefix(got, tt.wantPfx) {
				t.Errorf("SHA256Bytes() prefix = %v, want %v", got[:7], tt.wantPfx)
			}
			if len(got) != tt.wantLen {
				t.Errorf("SHA256Bytes() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestSHA256BytesMatchesSHA256(t *testing.T) {
	input := "test string"

	hashStr := SHA256(input)
	hashBytes := SHA256Bytes([]byte(input))

	if hashStr != hashBytes {
		t.Errorf("SHA256() and SHA256Bytes() differ: %v != %v", hashStr, hashBytes)
	}
}

func TestSHA256Reader(t *testing.T) {
	input := "hello world"
	reader := bytes.NewReader([]byte(input))

	hash, err := SHA256Reader(reader)
	if err != nil {
		t.Fatalf("SHA256Reader() error = %v", err)
	}

	expected := SHA256(input)
	if hash != expected {
		t.Errorf("SHA256Reader() = %v, want %v", hash, expected)
	}
}

func TestSHA256JSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "simple map",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "nested map",
			input:   map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			wantErr: false,
		},
		{
			name:    "slice",
			input:   []interface{}{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "struct",
			input:   struct{ Name string }{"test"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SHA256JSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SHA256JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.HasPrefix(got, "sha256:") {
				t.Errorf("SHA256JSON() = %v, should start with 'sha256:'", got)
			}
		})
	}
}

func TestSHA256JSONKeyOrder(t *testing.T) {
	// Same content, different key order should produce same hash
	map1 := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	map2 := map[string]interface{}{"c": 3, "b": 2, "a": 1}

	hash1, _ := SHA256JSON(map1)
	hash2, _ := SHA256JSON(map2)

	if hash1 != hash2 {
		t.Errorf("SHA256JSON() should normalize key order: %v != %v", hash1, hash2)
	}
}

func TestCombineHashes(t *testing.T) {
	tests := []struct {
		name   string
		hashes []string
	}{
		{"single hash", []string{"sha256:abc"}},
		{"two hashes", []string{"sha256:abc", "sha256:def"}},
		{"multiple hashes", []string{"sha256:a", "sha256:b", "sha256:c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CombineHashes(tt.hashes...)
			if !strings.HasPrefix(got, "sha256:") {
				t.Errorf("CombineHashes() = %v, should start with 'sha256:'", got)
			}
		})
	}
}

func TestCombineHashesConsistency(t *testing.T) {
	hashes := []string{"sha256:a", "sha256:b", "sha256:c"}

	hash1 := CombineHashes(hashes...)
	hash2 := CombineHashes(hashes...)

	if hash1 != hash2 {
		t.Errorf("CombineHashes() not consistent: %v != %v", hash1, hash2)
	}
}

func TestCombineHashesOrderMatters(t *testing.T) {
	hash1 := CombineHashes("sha256:a", "sha256:b")
	hash2 := CombineHashes("sha256:b", "sha256:a")

	if hash1 == hash2 {
		t.Error("CombineHashes() should produce different results for different order")
	}
}

func BenchmarkSHA256(b *testing.B) {
	input := "benchmark input string for sha256 hash function testing"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SHA256(input)
	}
}

func BenchmarkSHA256Bytes(b *testing.B) {
	input := []byte("benchmark input string for sha256 hash function testing")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SHA256Bytes(input)
	}
}

func BenchmarkSHA256JSON(b *testing.B) {
	input := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b", "c"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SHA256JSON(input)
	}
}
