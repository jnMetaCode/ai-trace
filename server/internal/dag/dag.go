// Package dag 提供因果事件图（有向无环图）支持
// 用于表示和验证并行事件的因果关系
package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ai-trace/server/internal/event"
)

var (
	ErrCycleDetected       = errors.New("dag: cycle detected")
	ErrInvalidPredecessor  = errors.New("dag: invalid predecessor")
	ErrEventNotFound       = errors.New("dag: event not found")
	ErrDuplicateEvent      = errors.New("dag: duplicate event")
	ErrInvalidEventHash    = errors.New("dag: invalid event hash")
	ErrOrphanEvent         = errors.New("dag: orphan event detected")
)

// Node DAG 节点
type Node struct {
	Event       *event.Event
	Predecessors []*Node  // 前驱节点（入边）
	Successors   []*Node  // 后继节点（出边）
	Depth       int       // 节点深度（从根节点到此节点的最长路径）
	Verified    bool      // 哈希是否已验证
}

// DAG 有向无环图
type DAG struct {
	nodes    map[string]*Node // eventID -> Node
	roots    []*Node          // 根节点（无前驱）
	leaves   []*Node          // 叶子节点（无后继）
	traceID  string
	mux      sync.RWMutex
}

// NewDAG 创建新的 DAG
func NewDAG(traceID string) *DAG {
	return &DAG{
		nodes:   make(map[string]*Node),
		roots:   make([]*Node, 0),
		leaves:  make([]*Node, 0),
		traceID: traceID,
	}
}

// AddEvent 添加事件到 DAG
func (d *DAG) AddEvent(evt *event.Event) error {
	d.mux.Lock()
	defer d.mux.Unlock()

	// 检查重复
	if _, exists := d.nodes[evt.EventID]; exists {
		return ErrDuplicateEvent
	}

	// 创建节点
	node := &Node{
		Event:        evt,
		Predecessors: make([]*Node, 0),
		Successors:   make([]*Node, 0),
		Depth:        0,
	}

	// 查找前驱节点
	prevHashes := d.getPredecessorHashes(evt)
	if len(prevHashes) == 0 {
		// 这是根节点
		d.roots = append(d.roots, node)
	} else {
		// 连接前驱节点
		for _, prevHash := range prevHashes {
			predNode := d.findNodeByHash(prevHash)
			if predNode == nil {
				// 前驱不存在，可能是孤儿事件
				// 暂时允许，后续验证时检查
				continue
			}

			node.Predecessors = append(node.Predecessors, predNode)
			predNode.Successors = append(predNode.Successors, node)

			// 更新深度
			if predNode.Depth+1 > node.Depth {
				node.Depth = predNode.Depth + 1
			}
		}
	}

	// 添加到节点映射
	d.nodes[evt.EventID] = node

	// 更新叶子节点列表
	d.updateLeaves(node)

	// 检查是否形成环
	if d.hasCycleFrom(node, make(map[string]bool)) {
		// 移除节点
		delete(d.nodes, evt.EventID)
		return ErrCycleDetected
	}

	return nil
}

// getPredecessorHashes 获取前驱哈希列表
func (d *DAG) getPredecessorHashes(evt *event.Event) []string {
	if len(evt.PrevEventHashes) > 0 {
		return evt.PrevEventHashes
	}
	if evt.PrevEventHash != "" {
		return []string{evt.PrevEventHash}
	}
	return nil
}

// findNodeByHash 通过事件哈希查找节点
func (d *DAG) findNodeByHash(hash string) *Node {
	for _, node := range d.nodes {
		if node.Event.EventHash == hash {
			return node
		}
	}
	return nil
}

// updateLeaves 更新叶子节点列表
func (d *DAG) updateLeaves(newNode *Node) {
	// 从叶子中移除有了后继的节点
	newLeaves := make([]*Node, 0)
	for _, leaf := range d.leaves {
		if len(leaf.Successors) == 0 {
			newLeaves = append(newLeaves, leaf)
		}
	}

	// 如果新节点没有后继，它就是叶子
	if len(newNode.Successors) == 0 {
		newLeaves = append(newLeaves, newNode)
	}

	d.leaves = newLeaves
}

// hasCycleFrom 从指定节点检测环
func (d *DAG) hasCycleFrom(node *Node, visited map[string]bool) bool {
	if visited[node.Event.EventID] {
		return true
	}
	visited[node.Event.EventID] = true

	for _, succ := range node.Successors {
		if d.hasCycleFrom(succ, visited) {
			return true
		}
	}

	visited[node.Event.EventID] = false
	return false
}

// GetNode 获取节点
func (d *DAG) GetNode(eventID string) (*Node, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	node, exists := d.nodes[eventID]
	if !exists {
		return nil, ErrEventNotFound
	}
	return node, nil
}

// GetRoots 获取根节点
func (d *DAG) GetRoots() []*Node {
	d.mux.RLock()
	defer d.mux.RUnlock()
	return d.roots
}

// GetLeaves 获取叶子节点
func (d *DAG) GetLeaves() []*Node {
	d.mux.RLock()
	defer d.mux.RUnlock()
	return d.leaves
}

// Size 获取节点数量
func (d *DAG) Size() int {
	d.mux.RLock()
	defer d.mux.RUnlock()
	return len(d.nodes)
}

// TopologicalSort 拓扑排序
func (d *DAG) TopologicalSort() []*event.Event {
	d.mux.RLock()
	defer d.mux.RUnlock()

	result := make([]*event.Event, 0, len(d.nodes))
	visited := make(map[string]bool)
	var visit func(*Node)

	visit = func(node *Node) {
		if visited[node.Event.EventID] {
			return
		}
		visited[node.Event.EventID] = true

		// 先访问所有前驱
		for _, pred := range node.Predecessors {
			visit(pred)
		}

		result = append(result, node.Event)
	}

	// 从叶子节点开始反向遍历，或者从所有节点开始
	for _, node := range d.nodes {
		visit(node)
	}

	return result
}

// GetEventsByDepth 按深度获取事件
func (d *DAG) GetEventsByDepth() map[int][]*event.Event {
	d.mux.RLock()
	defer d.mux.RUnlock()

	result := make(map[int][]*event.Event)
	for _, node := range d.nodes {
		result[node.Depth] = append(result[node.Depth], node.Event)
	}
	return result
}

// GetParallelEvents 获取并行事件组
// 返回可以并行执行的事件组
func (d *DAG) GetParallelEvents() [][]*event.Event {
	d.mux.RLock()
	defer d.mux.RUnlock()

	depthEvents := d.GetEventsByDepth()

	// 按深度排序
	depths := make([]int, 0, len(depthEvents))
	for depth := range depthEvents {
		depths = append(depths, depth)
	}
	sort.Ints(depths)

	result := make([][]*event.Event, 0)
	for _, depth := range depths {
		events := depthEvents[depth]
		if len(events) > 1 {
			result = append(result, events)
		}
	}

	return result
}

// ComputeMergeHash 计算合并节点的哈希
// 当多个并行事件汇聚到一个节点时，计算合并哈希
func (d *DAG) ComputeMergeHash(predecessorHashes []string) string {
	// 排序以保证一致性
	sorted := make([]string, len(predecessorHashes))
	copy(sorted, predecessorHashes)
	sort.Strings(sorted)

	// 拼接并哈希
	var combined string
	for _, h := range sorted {
		combined += h
	}

	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// Verify 验证整个 DAG 的完整性
func (d *DAG) Verify() (*VerificationResult, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	result := &VerificationResult{
		TraceID:      d.traceID,
		TotalNodes:   len(d.nodes),
		VerifiedAt:   time.Now(),
		InvalidNodes: make([]string, 0),
		OrphanNodes:  make([]string, 0),
	}

	validator := NewValidator()

	for _, node := range d.nodes {
		// 验证事件哈希
		if !validator.VerifyEventHash(node.Event) {
			result.InvalidNodes = append(result.InvalidNodes, node.Event.EventID)
		} else {
			node.Verified = true
		}

		// 验证前驱连接
		prevHashes := d.getPredecessorHashes(node.Event)
		for _, prevHash := range prevHashes {
			if d.findNodeByHash(prevHash) == nil {
				result.OrphanNodes = append(result.OrphanNodes, node.Event.EventID)
				break
			}
		}
	}

	result.Valid = len(result.InvalidNodes) == 0 && len(result.OrphanNodes) == 0
	return result, nil
}

// VerificationResult 验证结果
type VerificationResult struct {
	TraceID      string    `json:"trace_id"`
	Valid        bool      `json:"valid"`
	TotalNodes   int       `json:"total_nodes"`
	InvalidNodes []string  `json:"invalid_nodes,omitempty"`
	OrphanNodes  []string  `json:"orphan_nodes,omitempty"`
	VerifiedAt   time.Time `json:"verified_at"`
}

// GetPath 获取两个节点之间的路径
func (d *DAG) GetPath(fromEventID, toEventID string) ([]*event.Event, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	fromNode, exists := d.nodes[fromEventID]
	if !exists {
		return nil, fmt.Errorf("from event not found: %s", fromEventID)
	}

	_, exists = d.nodes[toEventID]
	if !exists {
		return nil, fmt.Errorf("to event not found: %s", toEventID)
	}

	// BFS 查找路径
	visited := make(map[string]bool)
	parent := make(map[string]*Node)
	queue := []*Node{fromNode}
	visited[fromEventID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.Event.EventID == toEventID {
			// 重建路径
			path := make([]*event.Event, 0)
			for n := current; n != nil; n = parent[n.Event.EventID] {
				path = append([]*event.Event{n.Event}, path...)
			}
			return path, nil
		}

		for _, succ := range current.Successors {
			if !visited[succ.Event.EventID] {
				visited[succ.Event.EventID] = true
				parent[succ.Event.EventID] = current
				queue = append(queue, succ)
			}
		}
	}

	return nil, fmt.Errorf("no path found from %s to %s", fromEventID, toEventID)
}

// GetAncestors 获取节点的所有祖先
func (d *DAG) GetAncestors(eventID string) ([]*event.Event, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	node, exists := d.nodes[eventID]
	if !exists {
		return nil, ErrEventNotFound
	}

	ancestors := make([]*event.Event, 0)
	visited := make(map[string]bool)

	var collect func(*Node)
	collect = func(n *Node) {
		for _, pred := range n.Predecessors {
			if !visited[pred.Event.EventID] {
				visited[pred.Event.EventID] = true
				ancestors = append(ancestors, pred.Event)
				collect(pred)
			}
		}
	}

	collect(node)
	return ancestors, nil
}

// GetDescendants 获取节点的所有后代
func (d *DAG) GetDescendants(eventID string) ([]*event.Event, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()

	node, exists := d.nodes[eventID]
	if !exists {
		return nil, ErrEventNotFound
	}

	descendants := make([]*event.Event, 0)
	visited := make(map[string]bool)

	var collect func(*Node)
	collect = func(n *Node) {
		for _, succ := range n.Successors {
			if !visited[succ.Event.EventID] {
				visited[succ.Event.EventID] = true
				descendants = append(descendants, succ.Event)
				collect(succ)
			}
		}
	}

	collect(node)
	return descendants, nil
}

// ToJSON 导出为 JSON 格式（用于可视化）
func (d *DAG) ToJSON() *DAGExport {
	d.mux.RLock()
	defer d.mux.RUnlock()

	export := &DAGExport{
		TraceID: d.traceID,
		Nodes:   make([]NodeExport, 0, len(d.nodes)),
		Edges:   make([]EdgeExport, 0),
	}

	for _, node := range d.nodes {
		nodeExport := NodeExport{
			EventID:   node.Event.EventID,
			EventType: string(node.Event.EventType),
			EventHash: node.Event.EventHash,
			Timestamp: node.Event.Timestamp,
			Depth:     node.Depth,
		}
		export.Nodes = append(export.Nodes, nodeExport)

		// 添加边
		for _, succ := range node.Successors {
			export.Edges = append(export.Edges, EdgeExport{
				From: node.Event.EventID,
				To:   succ.Event.EventID,
			})
		}
	}

	return export
}

// DAGExport DAG 导出格式
type DAGExport struct {
	TraceID string       `json:"trace_id"`
	Nodes   []NodeExport `json:"nodes"`
	Edges   []EdgeExport `json:"edges"`
}

// NodeExport 节点导出格式
type NodeExport struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	EventHash string    `json:"event_hash"`
	Timestamp time.Time `json:"timestamp"`
	Depth     int       `json:"depth"`
}

// EdgeExport 边导出格式
type EdgeExport struct {
	From string `json:"from"`
	To   string `json:"to"`
}
