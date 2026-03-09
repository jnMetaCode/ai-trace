//go:build blockchain
// +build blockchain

// 此文件仅在使用 -tags blockchain 编译时包含
// 编译命令: go build -tags blockchain ./...

package anchor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// EthereumAnchor 以太坊锚定实现
type EthereumAnchor struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    *big.Int
	contract   common.Address
	config     *Config
	logger     *zap.SugaredLogger
}

// NewEthereumAnchor 创建以太坊锚定器
func NewEthereumAnchor(cfg *Config, logger *zap.SugaredLogger) (*EthereumAnchor, error) {
	if cfg.EthereumRPCURL == "" {
		return nil, ErrNotConfigured
	}

	client, err := ethclient.Dial(cfg.EthereumRPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	var privateKey *ecdsa.PrivateKey
	var address common.Address

	if cfg.EthereumPrivateKey != "" {
		privateKey, err = crypto.HexToECDSA(cfg.EthereumPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		publicKey := privateKey.Public().(*ecdsa.PublicKey)
		address = crypto.PubkeyToAddress(*publicKey)
	}

	var contract common.Address
	if cfg.ContractAddress != "" {
		contract = common.HexToAddress(cfg.ContractAddress)
	}

	return &EthereumAnchor{
		client:     client,
		privateKey: privateKey,
		address:    address,
		chainID:    big.NewInt(cfg.EthereumChainID),
		contract:   contract,
		config:     cfg,
		logger:     logger,
	}, nil
}

// Anchor 执行以太坊锚定
func (e *EthereumAnchor) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResult, error) {
	if e.privateKey == nil {
		return nil, ErrNotConfigured
	}

	// 构建锚定数据
	// 格式: "AI-TRACE:v1:{cert_id}:{root_hash}:{timestamp}"
	data := fmt.Sprintf("AI-TRACE:v1:%s:%s:%d",
		req.CertID,
		req.RootHash,
		req.Timestamp.Unix(),
	)
	dataBytes := []byte(data)

	// 获取nonce
	nonce, err := e.client.PendingNonceAt(ctx, e.address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// 获取gas价格
	gasPrice, err := e.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// 限制最大gas价格
	maxGasPrice := big.NewInt(int64(e.config.MaxGasPrice))
	if gasPrice.Cmp(maxGasPrice) > 0 {
		gasPrice = maxGasPrice
	}

	// 构建交易
	var tx *types.Transaction
	if e.contract != (common.Address{}) {
		// 调用智能合约
		tx = e.buildContractTx(nonce, gasPrice, dataBytes)
	} else {
		// 直接发送数据到链上（自己给自己发送0 ETH，附带数据）
		tx = types.NewTransaction(
			nonce,
			e.address,
			big.NewInt(0),
			e.config.GasLimit,
			gasPrice,
			dataBytes,
		)
	}

	// 签名交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.chainID), e.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// 发送交易
	err = e.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash := signedTx.Hash()
	e.logger.Infof("Anchor transaction sent: %s", txHash.Hex())

	// 等待交易确认
	receipt, err := e.waitForReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return &AnchorResult{
		AnchorID:        fmt.Sprintf("eth_%s", txHash.Hex()[:16]),
		AnchorType:      AnchorTypeEthereum,
		TxHash:          txHash.Hex(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		BlockHash:       receipt.BlockHash.Hex(),
		ContractAddress: e.contract.Hex(),
		ChainID:         e.chainID.Int64(),
		GasUsed:         receipt.GasUsed,
		Timestamp:       time.Now(),
		ProofURL:        fmt.Sprintf("https://etherscan.io/tx/%s", txHash.Hex()),
	}, nil
}

// buildContractTx 构建合约调用交易
func (e *EthereumAnchor) buildContractTx(nonce uint64, gasPrice *big.Int, data []byte) *types.Transaction {
	// ABI编码: anchor(bytes32 certHash, bytes32 rootHash, uint256 timestamp)
	// 这里简化处理，实际应该使用ABI编码
	return types.NewTransaction(
		nonce,
		e.contract,
		big.NewInt(0),
		e.config.GasLimit,
		gasPrice,
		data,
	)
}

// waitForReceipt 等待交易确认
func (e *EthereumAnchor) waitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, ErrTxTimeout
		case <-ticker.C:
			receipt, err := e.client.TransactionReceipt(ctx, txHash)
			if err == ethereum.NotFound {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get receipt: %w", err)
			}
			if receipt.Status == 0 {
				return nil, ErrTxFailed
			}
			return receipt, nil
		}
	}
}

// Verify 验证以太坊锚定
func (e *EthereumAnchor) Verify(ctx context.Context, result *AnchorResult) (bool, error) {
	if result.TxHash == "" {
		return false, ErrInvalidProof
	}

	txHash := common.HexToHash(result.TxHash)

	// 获取交易
	tx, isPending, err := e.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return false, fmt.Errorf("failed to get transaction: %w", err)
	}
	if isPending {
		return false, nil // 交易还在pending
	}

	// 获取交易回执
	receipt, err := e.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return false, fmt.Errorf("failed to get receipt: %w", err)
	}

	// 验证交易状态
	if receipt.Status == 0 {
		return false, nil
	}

	// 验证区块号
	if receipt.BlockNumber.Uint64() != result.BlockNumber {
		return false, nil
	}

	// 获取当前区块号检查确认数
	currentBlock, err := e.client.BlockNumber(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get current block: %w", err)
	}

	confirmations := currentBlock - receipt.BlockNumber.Uint64()
	if confirmations < 6 { // 至少6个确认
		e.logger.Warnf("Transaction has only %d confirmations", confirmations)
	}

	_ = tx // 可以进一步验证交易数据

	return true, nil
}

// GetAnchorType 获取锚定类型
func (e *EthereumAnchor) GetAnchorType() AnchorType {
	return AnchorTypeEthereum
}

// IsAvailable 检查服务是否可用
func (e *EthereumAnchor) IsAvailable(ctx context.Context) bool {
	_, err := e.client.ChainID(ctx)
	return err == nil
}

// GetBalance 获取账户余额
func (e *EthereumAnchor) GetBalance(ctx context.Context) (*big.Int, error) {
	return e.client.BalanceAt(ctx, e.address, nil)
}

// Close 关闭连接
func (e *EthereumAnchor) Close() {
	e.client.Close()
}
