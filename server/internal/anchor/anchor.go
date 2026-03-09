package anchor

import (
	"context"
	"errors"
	"time"
)

// AnchorType 锚定类型
type AnchorType string

const (
	AnchorTypeLocal      AnchorType = "local"      // 本地签名
	AnchorTypeWORM       AnchorType = "worm"       // WORM存储
	AnchorTypeEthereum   AnchorType = "ethereum"   // 以太坊
	AnchorTypePolygon    AnchorType = "polygon"    // Polygon
	AnchorTypeBSC        AnchorType = "bsc"        // BSC
	AnchorTypeArbitrum   AnchorType = "arbitrum"   // Arbitrum
	AnchorTypeFederated  AnchorType = "federated"  // 联邦化节点
)

// AnchorRequest 锚定请求
type AnchorRequest struct {
	CertID    string     `json:"cert_id"`
	RootHash  string     `json:"root_hash"`
	Timestamp time.Time  `json:"timestamp"`
	Metadata  string     `json:"metadata,omitempty"`
}

// AnchorResult 锚定结果
type AnchorResult struct {
	AnchorID        string     `json:"anchor_id"`
	AnchorType      AnchorType `json:"anchor_type"`
	TxHash          string     `json:"tx_hash,omitempty"`
	BlockNumber     uint64     `json:"block_number,omitempty"`
	BlockHash       string     `json:"block_hash,omitempty"`
	ContractAddress string     `json:"contract_address,omitempty"`
	ChainID         int64      `json:"chain_id,omitempty"`
	GasUsed         uint64     `json:"gas_used,omitempty"`
	Timestamp       time.Time  `json:"timestamp"`
	ProofURL        string     `json:"proof_url,omitempty"`

	// 联邦化节点信息
	FederatedNodes  []string   `json:"federated_nodes,omitempty"`
	Confirmations   int        `json:"confirmations,omitempty"`
}

// Anchorer 锚定接口
type Anchorer interface {
	// Anchor 执行锚定
	Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error)

	// Verify 验证锚定
	Verify(ctx context.Context, result *AnchorResult) (bool, error)

	// GetAnchorType 获取锚定类型
	GetAnchorType() AnchorType

	// IsAvailable 检查服务是否可用
	IsAvailable(ctx context.Context) bool
}

// Config 锚定配置
type Config struct {
	// 以太坊配置
	EthereumRPCURL    string `yaml:"ethereum_rpc_url"`
	EthereumPrivateKey string `yaml:"ethereum_private_key"`
	EthereumChainID   int64  `yaml:"ethereum_chain_id"`

	// Polygon配置
	PolygonRPCURL     string `yaml:"polygon_rpc_url"`
	PolygonPrivateKey string `yaml:"polygon_private_key"`
	PolygonChainID    int64  `yaml:"polygon_chain_id"`

	// 智能合约地址
	ContractAddress   string `yaml:"contract_address"`

	// 联邦化节点配置
	FederatedNodes    []string `yaml:"federated_nodes"`
	MinConfirmations  int      `yaml:"min_confirmations"`

	// 通用配置
	GasLimit          uint64 `yaml:"gas_limit"`
	MaxGasPrice       uint64 `yaml:"max_gas_price"`
	RetryAttempts     int    `yaml:"retry_attempts"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		EthereumChainID:  1, // Mainnet
		PolygonChainID:   137,
		GasLimit:         100000,
		MaxGasPrice:      100000000000, // 100 Gwei
		RetryAttempts:    3,
		RetryDelay:       5 * time.Second,
		MinConfirmations: 3,
	}
}

// 错误定义
var (
	ErrNotConfigured    = errors.New("anchor: not configured")
	ErrInsufficientGas  = errors.New("anchor: insufficient gas")
	ErrTxFailed         = errors.New("anchor: transaction failed")
	ErrTxTimeout        = errors.New("anchor: transaction timeout")
	ErrInvalidProof     = errors.New("anchor: invalid proof")
	ErrChainUnavailable = errors.New("anchor: blockchain unavailable")
	ErrNoConfirmations  = errors.New("anchor: insufficient confirmations from federated nodes")
)
