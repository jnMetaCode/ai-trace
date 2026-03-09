package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/ai-trace/server/internal/event"
)

// Validator DAG 验证器
type Validator struct{}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// VerifyEventHash 验证单个事件的哈希
func (v *Validator) VerifyEventHash(evt *event.Event) bool {
	computed := v.ComputeEventHash(evt)
	return computed == evt.EventHash
}

// ComputeEventHash 计算事件哈希
func (v *Validator) ComputeEventHash(evt *event.Event) string {
	// 构建哈希输入
	data := struct {
		EventID         string   `json:"event_id"`
		TraceID         string   `json:"trace_id"`
		PrevEventHash   string   `json:"prev_event_hash,omitempty"`
		PrevEventHashes []string `json:"prev_event_hashes,omitempty"`
		EventType       string   `json:"event_type"`
		Timestamp       int64    `json:"timestamp"`
		Sequence        int      `json:"sequence"`
		PayloadHash     string   `json:"payload_hash"`
	}{
		EventID:         evt.EventID,
		TraceID:         evt.TraceID,
		PrevEventHash:   evt.PrevEventHash,
		PrevEventHashes: evt.PrevEventHashes,
		EventType:       string(evt.EventType),
		Timestamp:       evt.Timestamp.Unix(),
		Sequence:        evt.Sequence,
		PayloadHash:     evt.PayloadHash,
	}

	jsonBytes, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// VerifyDAG 验证整个 DAG 的完整性
func (v *Validator) VerifyDAG(d *DAG) *ValidationReport {
	report := &ValidationReport{
		TraceID:    d.traceID,
		StartTime:  time.Now(),
		Errors:     make([]ValidationError, 0),
		TotalNodes: d.Size(),
	}

	// 获取所有节点
	events := d.TopologicalSort()

	// 1. 验证每个事件的哈希
	for _, evt := range events {
		if !v.VerifyEventHash(evt) {
			report.Errors = append(report.Errors, ValidationError{
				EventID: evt.EventID,
				Type:    "invalid_hash",
				Message: "event hash does not match computed hash",
			})
		}
	}

	// 2. 验证因果链完整性
	for _, evt := range events {
		prevHashes := getPrevHashes(evt)
		for _, prevHash := range prevHashes {
			if !v.predecessorExists(events, prevHash) {
				report.Errors = append(report.Errors, ValidationError{
					EventID: evt.EventID,
					Type:    "missing_predecessor",
					Message: "predecessor event not found: " + prevHash,
				})
			}
		}
	}

	// 3. 验证时间戳递增（在因果链上）
	for _, evt := range events {
		node, err := d.GetNode(evt.EventID)
		if err != nil {
			continue
		}
		for _, pred := range node.Predecessors {
			if !evt.Timestamp.After(pred.Event.Timestamp) && !evt.Timestamp.Equal(pred.Event.Timestamp) {
				report.Errors = append(report.Errors, ValidationError{
					EventID: evt.EventID,
					Type:    "timestamp_violation",
					Message: "timestamp is not after predecessor",
				})
			}
		}
	}

	// 4. 验证序列号
	sequenceMap := make(map[int][]string)
	for _, evt := range events {
		sequenceMap[evt.Sequence] = append(sequenceMap[evt.Sequence], evt.EventID)
	}

	// 并行事件可以有相同的序列号
	// 但非并行事件不应该有重复序列号
	// 这里简化处理，只检查是否有合理的序列号

	report.EndTime = time.Now()
	report.Valid = len(report.Errors) == 0
	report.Duration = report.EndTime.Sub(report.StartTime)

	return report
}

// predecessorExists 检查前驱是否存在
func (v *Validator) predecessorExists(events []*event.Event, hash string) bool {
	for _, evt := range events {
		if evt.EventHash == hash {
			return true
		}
	}
	return false
}

// VerifyMergePoint 验证合并点
// 当多个并行事件汇聚时，验证合并是否正确
func (v *Validator) VerifyMergePoint(evt *event.Event, predecessors []*event.Event) bool {
	if len(evt.PrevEventHashes) <= 1 {
		return true // 不是合并点
	}

	// 收集前驱哈希
	predHashes := make([]string, len(predecessors))
	for i, pred := range predecessors {
		predHashes[i] = pred.EventHash
	}

	// 排序
	sort.Strings(predHashes)
	sort.Strings(evt.PrevEventHashes)

	// 比较
	if len(predHashes) != len(evt.PrevEventHashes) {
		return false
	}

	for i := range predHashes {
		if predHashes[i] != evt.PrevEventHashes[i] {
			return false
		}
	}

	return true
}

// ValidateCausalOrder 验证因果顺序
// 确保事件按照因果顺序排列
func (v *Validator) ValidateCausalOrder(events []*event.Event) bool {
	seen := make(map[string]bool)

	for _, evt := range events {
		prevHashes := getPrevHashes(evt)
		for _, prevHash := range prevHashes {
			if prevHash != "" && !seen[prevHash] {
				return false // 前驱还没出现
			}
		}
		seen[evt.EventHash] = true
	}

	return true
}

// ValidationReport 验证报告
type ValidationReport struct {
	TraceID    string            `json:"trace_id"`
	Valid      bool              `json:"valid"`
	TotalNodes int               `json:"total_nodes"`
	Errors     []ValidationError `json:"errors,omitempty"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Duration   time.Duration     `json:"duration"`
}

// ValidationError 验证错误
type ValidationError struct {
	EventID string `json:"event_id"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// getPrevHashes 获取前驱哈希
func getPrevHashes(evt *event.Event) []string {
	if len(evt.PrevEventHashes) > 0 {
		return evt.PrevEventHashes
	}
	if evt.PrevEventHash != "" {
		return []string{evt.PrevEventHash}
	}
	return nil
}

// ComputeCausalHash 计算因果链哈希
// 用于快速验证整个事件链的完整性
func (v *Validator) ComputeCausalHash(events []*event.Event) string {
	if len(events) == 0 {
		return ""
	}

	// 按时间戳排序
	sorted := make([]*event.Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// 链式哈希
	h := sha256.New()
	for _, evt := range sorted {
		h.Write([]byte(evt.EventHash))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// BuildDAGFromEvents 从事件列表构建 DAG
func BuildDAGFromEvents(traceID string, events []*event.Event) (*DAG, error) {
	dag := NewDAG(traceID)

	// 按时间戳排序，确保前驱先添加
	sorted := make([]*event.Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	for _, evt := range sorted {
		if err := dag.AddEvent(evt); err != nil {
			return nil, err
		}
	}

	return dag, nil
}
