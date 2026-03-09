//go:build blockchain
// +build blockchain

// Package contracts 提供智能合约的 Go 绑定
package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// RegistryABI AITraceRegistry 合约 ABI
const RegistryABI = `[
	{"inputs":[],"stateMutability":"nonpayable","type":"constructor"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"bytes32","name":"merkleRoot","type":"bytes32"},{"internalType":"bytes32","name":"fingerprintHash","type":"bytes32"},{"internalType":"bytes32","name":"inputHash","type":"bytes32"},{"internalType":"bytes32","name":"outputHash","type":"bytes32"},{"internalType":"string","name":"modelId","type":"string"},{"internalType":"string","name":"tenantId","type":"string"}],"name":"createAttestation","outputs":[],"stateMutability":"payable","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certHash","type":"bytes32"},{"internalType":"bytes32","name":"rootHash","type":"bytes32"},{"internalType":"uint256","name":"timestamp","type":"uint256"}],"name":"anchor","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"bytes32[]","name":"certIds","type":"bytes32[]"},{"internalType":"bytes32[]","name":"merkleRoots","type":"bytes32[]"},{"internalType":"bytes32[]","name":"fingerprintHashes","type":"bytes32[]"}],"name":"batchCreateAttestations","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"}],"name":"revokeAttestation","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"bytes32","name":"merkleRoot","type":"bytes32"}],"name":"verifyAttestation","outputs":[{"internalType":"bool","name":"valid","type":"bool"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"bytes32","name":"fingerprintHash","type":"bytes32"}],"name":"verifyFingerprint","outputs":[{"internalType":"bool","name":"valid","type":"bool"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"}],"name":"attestationExists","outputs":[{"internalType":"bool","name":"exists","type":"bool"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"bytes32","name":"certId","type":"bytes32"}],"name":"getAttestation","outputs":[{"components":[{"internalType":"bytes32","name":"certId","type":"bytes32"},{"internalType":"bytes32","name":"merkleRoot","type":"bytes32"},{"internalType":"bytes32","name":"fingerprintHash","type":"bytes32"},{"internalType":"bytes32","name":"inputHash","type":"bytes32"},{"internalType":"bytes32","name":"outputHash","type":"bytes32"},{"internalType":"address","name":"submitter","type":"address"},{"internalType":"uint256","name":"timestamp","type":"uint256"},{"internalType":"uint256","name":"blockNumber","type":"uint256"},{"internalType":"string","name":"modelId","type":"string"},{"internalType":"string","name":"tenantId","type":"string"},{"internalType":"bool","name":"isValid","type":"bool"}],"internalType":"struct AITraceRegistry.AttestationRecord","name":"record","type":"tuple"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"address","name":"submitter","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedSubmitter","outputs":[],"stateMutability":"nonpayable","type":"function"},
	{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"totalAttestations","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedSubmitters","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
]`

// AttestationRecord 存证记录结构
type AttestationRecord struct {
	CertId          [32]byte
	MerkleRoot      [32]byte
	FingerprintHash [32]byte
	InputHash       [32]byte
	OutputHash      [32]byte
	Submitter       common.Address
	Timestamp       *big.Int
	BlockNumber     *big.Int
	ModelId         string
	TenantId        string
	IsValid         bool
}

// RegistryContract AITraceRegistry 合约封装
type RegistryContract struct {
	address  common.Address
	client   *ethclient.Client
	abi      abi.ABI
	auth     *bind.TransactOpts
	chainID  *big.Int
}

// NewRegistryContract 创建 Registry 合约实例
func NewRegistryContract(
	client *ethclient.Client,
	contractAddr common.Address,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
) (*RegistryContract, error) {
	parsedABI, err := abi.JSON(strings.NewReader(RegistryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	var auth *bind.TransactOpts
	if privateKey != nil {
		auth, err = bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor: %w", err)
		}
	}

	return &RegistryContract{
		address: contractAddr,
		client:  client,
		abi:     parsedABI,
		auth:    auth,
		chainID: chainID,
	}, nil
}

// CreateAttestation 创建完整存证记录
func (r *RegistryContract) CreateAttestation(
	ctx context.Context,
	certId [32]byte,
	merkleRoot [32]byte,
	fingerprintHash [32]byte,
	inputHash [32]byte,
	outputHash [32]byte,
	modelId string,
	tenantId string,
) (*types.Transaction, error) {
	if r.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := r.abi.Pack(
		"createAttestation",
		certId,
		merkleRoot,
		fingerprintHash,
		inputHash,
		outputHash,
		modelId,
		tenantId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return r.sendTransaction(ctx, data, big.NewInt(0))
}

// Anchor 简化版锚定（兼容旧接口）
func (r *RegistryContract) Anchor(
	ctx context.Context,
	certHash [32]byte,
	rootHash [32]byte,
	timestamp *big.Int,
) (*types.Transaction, error) {
	if r.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := r.abi.Pack("anchor", certHash, rootHash, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return r.sendTransaction(ctx, data, big.NewInt(0))
}

// BatchCreateAttestations 批量创建存证
func (r *RegistryContract) BatchCreateAttestations(
	ctx context.Context,
	certIds [][32]byte,
	merkleRoots [][32]byte,
	fingerprintHashes [][32]byte,
) (*types.Transaction, error) {
	if r.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := r.abi.Pack("batchCreateAttestations", certIds, merkleRoots, fingerprintHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return r.sendTransaction(ctx, data, big.NewInt(0))
}

// RevokeAttestation 撤销存证
func (r *RegistryContract) RevokeAttestation(
	ctx context.Context,
	certId [32]byte,
) (*types.Transaction, error) {
	if r.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := r.abi.Pack("revokeAttestation", certId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return r.sendTransaction(ctx, data, big.NewInt(0))
}

// VerifyAttestation 验证存证
func (r *RegistryContract) VerifyAttestation(
	ctx context.Context,
	certId [32]byte,
	merkleRoot [32]byte,
) (bool, error) {
	data, err := r.abi.Pack("verifyAttestation", certId, merkleRoot)
	if err != nil {
		return false, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return false, err
	}

	var valid bool
	err = r.abi.UnpackIntoInterface(&valid, "verifyAttestation", result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack result: %w", err)
	}

	return valid, nil
}

// VerifyFingerprint 验证指纹
func (r *RegistryContract) VerifyFingerprint(
	ctx context.Context,
	certId [32]byte,
	fingerprintHash [32]byte,
) (bool, error) {
	data, err := r.abi.Pack("verifyFingerprint", certId, fingerprintHash)
	if err != nil {
		return false, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return false, err
	}

	var valid bool
	err = r.abi.UnpackIntoInterface(&valid, "verifyFingerprint", result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack result: %w", err)
	}

	return valid, nil
}

// AttestationExists 检查存证是否存在
func (r *RegistryContract) AttestationExists(
	ctx context.Context,
	certId [32]byte,
) (bool, error) {
	data, err := r.abi.Pack("attestationExists", certId)
	if err != nil {
		return false, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return false, err
	}

	var exists bool
	err = r.abi.UnpackIntoInterface(&exists, "attestationExists", result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack result: %w", err)
	}

	return exists, nil
}

// GetAttestation 获取存证记录
func (r *RegistryContract) GetAttestation(
	ctx context.Context,
	certId [32]byte,
) (*AttestationRecord, error) {
	data, err := r.abi.Pack("getAttestation", certId)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return nil, err
	}

	// 解包结果
	var record AttestationRecord
	outputs, err := r.abi.Unpack("getAttestation", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) > 0 {
		// 处理结构体解包
		if recordData, ok := outputs[0].(struct {
			CertId          [32]byte       `json:"certId"`
			MerkleRoot      [32]byte       `json:"merkleRoot"`
			FingerprintHash [32]byte       `json:"fingerprintHash"`
			InputHash       [32]byte       `json:"inputHash"`
			OutputHash      [32]byte       `json:"outputHash"`
			Submitter       common.Address `json:"submitter"`
			Timestamp       *big.Int       `json:"timestamp"`
			BlockNumber     *big.Int       `json:"blockNumber"`
			ModelId         string         `json:"modelId"`
			TenantId        string         `json:"tenantId"`
			IsValid         bool           `json:"isValid"`
		}); ok {
			record = AttestationRecord{
				CertId:          recordData.CertId,
				MerkleRoot:      recordData.MerkleRoot,
				FingerprintHash: recordData.FingerprintHash,
				InputHash:       recordData.InputHash,
				OutputHash:      recordData.OutputHash,
				Submitter:       recordData.Submitter,
				Timestamp:       recordData.Timestamp,
				BlockNumber:     recordData.BlockNumber,
				ModelId:         recordData.ModelId,
				TenantId:        recordData.TenantId,
				IsValid:         recordData.IsValid,
			}
		}
	}

	return &record, nil
}

// GetOwner 获取合约所有者
func (r *RegistryContract) GetOwner(ctx context.Context) (common.Address, error) {
	data, err := r.abi.Pack("owner")
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return common.Address{}, err
	}

	var owner common.Address
	err = r.abi.UnpackIntoInterface(&owner, "owner", result)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to unpack result: %w", err)
	}

	return owner, nil
}

// GetTotalAttestations 获取总存证数量
func (r *RegistryContract) GetTotalAttestations(ctx context.Context) (*big.Int, error) {
	data, err := r.abi.Pack("totalAttestations")
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return nil, err
	}

	var total *big.Int
	err = r.abi.UnpackIntoInterface(&total, "totalAttestations", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	return total, nil
}

// IsAuthorizedSubmitter 检查是否为授权提交者
func (r *RegistryContract) IsAuthorizedSubmitter(
	ctx context.Context,
	addr common.Address,
) (bool, error) {
	data, err := r.abi.Pack("authorizedSubmitters", addr)
	if err != nil {
		return false, fmt.Errorf("failed to pack data: %w", err)
	}

	result, err := r.call(ctx, data)
	if err != nil {
		return false, err
	}

	var authorized bool
	err = r.abi.UnpackIntoInterface(&authorized, "authorizedSubmitters", result)
	if err != nil {
		return false, fmt.Errorf("failed to unpack result: %w", err)
	}

	return authorized, nil
}

// SetAuthorizedSubmitter 设置授权提交者
func (r *RegistryContract) SetAuthorizedSubmitter(
	ctx context.Context,
	submitter common.Address,
	authorized bool,
) (*types.Transaction, error) {
	if r.auth == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	data, err := r.abi.Pack("setAuthorizedSubmitter", submitter, authorized)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}

	return r.sendTransaction(ctx, data, big.NewInt(0))
}

// sendTransaction 发送交易
func (r *RegistryContract) sendTransaction(
	ctx context.Context,
	data []byte,
	value *big.Int,
) (*types.Transaction, error) {
	nonce, err := r.client.PendingNonceAt(ctx, r.auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := r.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// 估算 gas
	msg := ethereum.CallMsg{
		From:  r.auth.From,
		To:    &r.address,
		Value: value,
		Data:  data,
	}
	gasLimit, err := r.client.EstimateGas(ctx, msg)
	if err != nil {
		// 如果估算失败，使用默认值
		gasLimit = 200000
	}

	tx := types.NewTransaction(
		nonce,
		r.address,
		value,
		gasLimit,
		gasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(r.chainID), r.auth.Signer.(interface{ PrivateKey() *ecdsa.PrivateKey }).PrivateKey())
	if err != nil {
		// 尝试使用 auth 直接签名
		signedTx, err = r.auth.Signer(r.auth.From, tx)
		if err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}
	}

	err = r.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx, nil
}

// call 执行只读调用
func (r *RegistryContract) call(ctx context.Context, data []byte) ([]byte, error) {
	msg := ethereum.CallMsg{
		To:   &r.address,
		Data: data,
	}

	result, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return result, nil
}

// HashToCertId 将字符串哈希转换为 certId
func HashToCertId(hash string) [32]byte {
	var certId [32]byte
	hashBytes := common.HexToHash(hash)
	copy(certId[:], hashBytes[:])
	return certId
}

// CertIdToHash 将 certId 转换为字符串哈希
func CertIdToHash(certId [32]byte) string {
	return common.BytesToHash(certId[:]).Hex()
}

// StringToBytes32 将字符串转换为 bytes32
func StringToBytes32(s string) [32]byte {
	var b [32]byte
	hash := crypto.Keccak256Hash([]byte(s))
	copy(b[:], hash[:])
	return b
}
