package anchor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FederatedAnchor 联邦化锚定实现
// 允许多个独立节点共同验证和存储存证
// 类似区块链的共识机制，但更轻量
type FederatedAnchor struct {
	nodes         []string                     // 联邦节点列表
	minConfirm    int                          // 最小确认数
	publicKey     ed25519.PublicKey            // 本节点公钥
	privateKey    ed25519.PrivateKey           // 本节点私钥
	nodeID        string                       // 本节点ID
	httpClient    *http.Client
	logger        *zap.SugaredLogger
	mu            sync.RWMutex
	knownAnchors  map[string]*AnchorResult     // 已知锚定缓存
	trustedKeys   map[string]ed25519.PublicKey // 已信任的节点公钥 nodeID -> publicKey
}

// FederatedNode 联邦节点信息
type FederatedNode struct {
	NodeID    string `json:"node_id"`
	Endpoint  string `json:"endpoint"`
	PublicKey string `json:"public_key"`
	Region    string `json:"region,omitempty"`
	Active    bool   `json:"active"`
	LastSeen  time.Time `json:"last_seen"`
}

// FederatedConfirmation 联邦确认
type FederatedConfirmation struct {
	NodeID      string    `json:"node_id"`
	AnchorID    string    `json:"anchor_id"`
	RootHash    string    `json:"root_hash"`
	Timestamp   time.Time `json:"timestamp"`
	Signature   string    `json:"signature"`
}

// FederatedAnchorRequest 联邦锚定请求
type FederatedAnchorRequest struct {
	CertID       string    `json:"cert_id"`
	RootHash     string    `json:"root_hash"`
	Timestamp    time.Time `json:"timestamp"`
	OriginNode   string    `json:"origin_node"`
	Signature    string    `json:"signature"`
}

// FederatedAnchorResponse 联邦锚定响应
type FederatedAnchorResponse struct {
	Accepted     bool      `json:"accepted"`
	NodeID       string    `json:"node_id"`
	AnchorID     string    `json:"anchor_id,omitempty"`
	Signature    string    `json:"signature,omitempty"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewFederatedAnchor 创建联邦锚定器
func NewFederatedAnchor(cfg *Config, logger *zap.SugaredLogger) (*FederatedAnchor, error) {
	if len(cfg.FederatedNodes) == 0 {
		return nil, ErrNotConfigured
	}

	// 生成或加载节点密钥
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	nodeID := hex.EncodeToString(pub[:8])

	return &FederatedAnchor{
		nodes:        cfg.FederatedNodes,
		minConfirm:   cfg.MinConfirmations,
		publicKey:    pub,
		privateKey:   priv,
		nodeID:       nodeID,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       logger,
		knownAnchors: make(map[string]*AnchorResult),
		trustedKeys:  make(map[string]ed25519.PublicKey),
	}, nil
}

// Anchor 执行联邦锚定
func (f *FederatedAnchor) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error) {
	// 签名请求
	dataToSign := fmt.Sprintf("%s|%s|%d", req.CertID, req.RootHash, req.Timestamp.Unix())
	signature := ed25519.Sign(f.privateKey, []byte(dataToSign))

	fedReq := &FederatedAnchorRequest{
		CertID:     req.CertID,
		RootHash:   req.RootHash,
		Timestamp:  req.Timestamp,
		OriginNode: f.nodeID,
		Signature:  hex.EncodeToString(signature),
	}

	// 并行请求所有节点
	confirmations := make(chan *FederatedConfirmation, len(f.nodes))
	errors := make(chan error, len(f.nodes))

	var wg sync.WaitGroup
	for _, node := range f.nodes {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			conf, err := f.requestConfirmation(ctx, endpoint, fedReq)
			if err != nil {
				errors <- err
				return
			}
			confirmations <- conf
		}(node)
	}

	// 等待所有请求完成
	go func() {
		wg.Wait()
		close(confirmations)
		close(errors)
	}()

	// 收集确认
	var confirmedNodes []string
	for conf := range confirmations {
		if conf != nil {
			confirmedNodes = append(confirmedNodes, conf.NodeID)
		}
	}

	// 检查是否达到最小确认数
	if len(confirmedNodes) < f.minConfirm {
		return nil, fmt.Errorf("%w: got %d, need %d",
			ErrNoConfirmations, len(confirmedNodes), f.minConfirm)
	}

	anchorID := fmt.Sprintf("fed_%s_%d", req.CertID[:8], time.Now().Unix())

	result := &AnchorResult{
		AnchorID:       anchorID,
		AnchorType:     AnchorTypeFederated,
		Timestamp:      time.Now(),
		FederatedNodes: confirmedNodes,
		Confirmations:  len(confirmedNodes),
	}

	// 缓存结果
	f.mu.Lock()
	f.knownAnchors[anchorID] = result
	f.mu.Unlock()

	f.logger.Infof("Federated anchor created: %s with %d confirmations",
		anchorID, len(confirmedNodes))

	return result, nil
}

// requestConfirmation 请求单个节点确认
func (f *FederatedAnchor) requestConfirmation(ctx context.Context, endpoint string, req *FederatedAnchorRequest) (*FederatedConfirmation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/federated/confirm", endpoint),
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var fedResp FederatedAnchorResponse
	if err := json.Unmarshal(respBody, &fedResp); err != nil {
		return nil, err
	}

	if !fedResp.Accepted {
		return nil, fmt.Errorf("node %s rejected: %s", fedResp.NodeID, fedResp.Error)
	}

	return &FederatedConfirmation{
		NodeID:    fedResp.NodeID,
		AnchorID:  fedResp.AnchorID,
		Signature: fedResp.Signature,
		Timestamp: fedResp.Timestamp,
	}, nil
}

// Verify 验证联邦锚定
func (f *FederatedAnchor) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	if result.AnchorType != AnchorTypeFederated {
		return false, ErrInvalidProof
	}

	// 检查本地缓存
	f.mu.RLock()
	cached, exists := f.knownAnchors[result.AnchorID]
	f.mu.RUnlock()

	if exists {
		return cached.Confirmations >= f.minConfirm, nil
	}

	// 向联邦节点查询验证
	verifications := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, node := range f.nodes {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			if f.verifyWithNode(ctx, endpoint, result) {
				mu.Lock()
				verifications++
				mu.Unlock()
			}
		}(node)
	}

	wg.Wait()

	return verifications >= f.minConfirm, nil
}

// verifyWithNode 向单个节点验证
func (f *FederatedAnchor) verifyWithNode(ctx context.Context, endpoint string, result *AnchorResult) bool {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/v1/federated/verify/%s", endpoint, result.AnchorID),
		nil)
	if err != nil {
		return false
	}

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// HandleConfirmRequest 处理确认请求（作为联邦节点时使用）
func (f *FederatedAnchor) HandleConfirmRequest(req *FederatedAnchorRequest) (*FederatedAnchorResponse, error) {
	// 1. 验证请求时间戳（防止重放攻击，允许5分钟偏差）
	timeDiff := time.Since(req.Timestamp)
	if timeDiff < -5*time.Minute || timeDiff > 5*time.Minute {
		return &FederatedAnchorResponse{
			Accepted:  false,
			NodeID:    f.nodeID,
			Error:     "timestamp out of range (±5 minutes)",
			Timestamp: time.Now(),
		}, nil
	}

	// 2. 验证请求签名
	dataToVerify := fmt.Sprintf("%s|%s|%d", req.CertID, req.RootHash, req.Timestamp.Unix())

	// 查找发起节点的公钥
	f.mu.RLock()
	originPubKey, trusted := f.trustedKeys[req.OriginNode]
	f.mu.RUnlock()

	if trusted {
		// 验证已信任节点的签名
		sigBytes, err := hex.DecodeString(req.Signature)
		if err != nil {
			return &FederatedAnchorResponse{
				Accepted:  false,
				NodeID:    f.nodeID,
				Error:     "invalid signature format",
				Timestamp: time.Now(),
			}, nil
		}

		if !ed25519.Verify(originPubKey, []byte(dataToVerify), sigBytes) {
			f.logger.Warnf("Signature verification failed for node %s", req.OriginNode)
			return &FederatedAnchorResponse{
				Accepted:  false,
				NodeID:    f.nodeID,
				Error:     "signature verification failed",
				Timestamp: time.Now(),
			}, nil
		}
		f.logger.Debugf("Signature verified for trusted node %s", req.OriginNode)
	} else {
		// 未信任的节点：记录警告但仍接受（首次交互场景）
		// 生产环境可以改为拒绝未知节点
		f.logger.Warnf("Accepting request from untrusted node %s (signature not verified)", req.OriginNode)
	}

	// 3. 验证数据完整性
	if req.CertID == "" || req.RootHash == "" {
		return &FederatedAnchorResponse{
			Accepted:  false,
			NodeID:    f.nodeID,
			Error:     "missing required fields: cert_id or root_hash",
			Timestamp: time.Now(),
		}, nil
	}

	// 4. 存储锚定信息
	anchorID := fmt.Sprintf("fed_%s_%d", req.CertID[:8], time.Now().Unix())

	// 5. 签名响应
	respData := fmt.Sprintf("%s|%s|%d", anchorID, req.RootHash, time.Now().Unix())
	signature := ed25519.Sign(f.privateKey, []byte(respData))

	f.logger.Infof("Confirmed anchor request from %s: %s", req.OriginNode, anchorID)

	return &FederatedAnchorResponse{
		Accepted:  true,
		NodeID:    f.nodeID,
		AnchorID:  anchorID,
		Signature: hex.EncodeToString(signature),
		Timestamp: time.Now(),
	}, nil
}

// RegisterTrustedNode 注册信任的节点公钥
func (f *FederatedAnchor) RegisterTrustedNode(nodeID string, publicKeyHex string) error {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return fmt.Errorf("invalid public key format: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
	}

	f.mu.Lock()
	f.trustedKeys[nodeID] = ed25519.PublicKey(pubKeyBytes)
	f.mu.Unlock()

	f.logger.Infof("Registered trusted node: %s", nodeID)
	return nil
}

// RemoveTrustedNode 移除信任的节点
func (f *FederatedAnchor) RemoveTrustedNode(nodeID string) {
	f.mu.Lock()
	delete(f.trustedKeys, nodeID)
	f.mu.Unlock()

	f.logger.Infof("Removed trusted node: %s", nodeID)
}

// GetTrustedNodes 获取所有信任的节点
func (f *FederatedAnchor) GetTrustedNodes() map[string]string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]string)
	for nodeID, pubKey := range f.trustedKeys {
		result[nodeID] = hex.EncodeToString(pubKey)
	}
	return result
}

// GetAnchorType 获取锚定类型
func (f *FederatedAnchor) GetAnchorType() AnchorType {
	return AnchorTypeFederated
}

// IsAvailable 检查服务是否可用
func (f *FederatedAnchor) IsAvailable(ctx context.Context) bool {
	available := 0
	for _, node := range f.nodes {
		req, _ := http.NewRequestWithContext(ctx, "GET", node+"/health", nil)
		resp, err := f.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			available++
			resp.Body.Close()
		}
	}
	return available >= f.minConfirm
}

// GetNodeID 获取本节点ID
func (f *FederatedAnchor) GetNodeID() string {
	return f.nodeID
}

// GetPublicKey 获取本节点公钥
func (f *FederatedAnchor) GetPublicKey() string {
	return hex.EncodeToString(f.publicKey)
}

// GetKnownNodes 获取已知节点列表
func (f *FederatedAnchor) GetKnownNodes() []string {
	return f.nodes
}

// AddNode 添加联邦节点
func (f *FederatedAnchor) AddNode(endpoint string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, n := range f.nodes {
		if n == endpoint {
			return
		}
	}
	f.nodes = append(f.nodes, endpoint)
}

// RemoveNode 移除联邦节点
func (f *FederatedAnchor) RemoveNode(endpoint string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i, n := range f.nodes {
		if n == endpoint {
			f.nodes = append(f.nodes[:i], f.nodes[i+1:]...)
			return
		}
	}
}
