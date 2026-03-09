// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title AITraceRegistry
 * @notice AI 生成内容全链路存证注册合约
 * @dev 存储 AI 推理过程的存证记录，包括 Merkle Root 和指纹哈希
 */
contract AITraceRegistry {
    // ============ 结构体 ============

    /// @notice 存证记录
    struct AttestationRecord {
        bytes32 certId;           // 存证证书 ID
        bytes32 merkleRoot;       // Merkle Tree 根哈希
        bytes32 fingerprintHash;  // 推理行为指纹哈希
        bytes32 inputHash;        // 输入内容哈希（加密后）
        bytes32 outputHash;       // 输出内容哈希（加密后）
        address submitter;        // 提交者地址
        uint256 timestamp;        // 区块时间戳
        uint256 blockNumber;      // 区块号
        string modelId;           // 模型标识
        string tenantId;          // 租户标识
        bool isValid;             // 是否有效
    }

    /// @notice 租户信息
    struct Tenant {
        address owner;            // 租户所有者
        bool isActive;            // 是否激活
        uint256 attestationCount; // 存证数量
        uint256 registeredAt;     // 注册时间
    }

    // ============ 状态变量 ============

    /// @notice 合约所有者
    address public owner;

    /// @notice 存证记录映射 (certId => AttestationRecord)
    mapping(bytes32 => AttestationRecord) public attestations;

    /// @notice 租户映射 (tenantId hash => Tenant)
    mapping(bytes32 => Tenant) public tenants;

    /// @notice 授权提交者映射 (address => isAuthorized)
    mapping(address => bool) public authorizedSubmitters;

    /// @notice 存证总数
    uint256 public totalAttestations;

    /// @notice 费用（可选，用于防止滥用）
    uint256 public attestationFee;

    // ============ 事件 ============

    /// @notice 存证记录创建事件
    event AttestationCreated(
        bytes32 indexed certId,
        bytes32 indexed merkleRoot,
        bytes32 fingerprintHash,
        address indexed submitter,
        string modelId,
        string tenantId,
        uint256 timestamp
    );

    /// @notice 存证撤销事件
    event AttestationRevoked(
        bytes32 indexed certId,
        address indexed revoker,
        uint256 timestamp
    );

    /// @notice 租户注册事件
    event TenantRegistered(
        bytes32 indexed tenantIdHash,
        address indexed owner,
        uint256 timestamp
    );

    /// @notice 提交者授权变更事件
    event SubmitterAuthorizationChanged(
        address indexed submitter,
        bool authorized
    );

    // ============ 修饰符 ============

    modifier onlyOwner() {
        require(msg.sender == owner, "AITraceRegistry: caller is not owner");
        _;
    }

    modifier onlyAuthorized() {
        require(
            authorizedSubmitters[msg.sender] || msg.sender == owner,
            "AITraceRegistry: caller is not authorized"
        );
        _;
    }

    // ============ 构造函数 ============

    constructor() {
        owner = msg.sender;
        authorizedSubmitters[msg.sender] = true;
    }

    // ============ 核心功能 ============

    /**
     * @notice 创建存证记录
     * @param certId 存证证书 ID
     * @param merkleRoot Merkle Tree 根哈希
     * @param fingerprintHash 推理行为指纹哈希
     * @param inputHash 输入内容哈希
     * @param outputHash 输出内容哈希
     * @param modelId 模型标识
     * @param tenantId 租户标识
     */
    function createAttestation(
        bytes32 certId,
        bytes32 merkleRoot,
        bytes32 fingerprintHash,
        bytes32 inputHash,
        bytes32 outputHash,
        string calldata modelId,
        string calldata tenantId
    ) external payable onlyAuthorized {
        require(certId != bytes32(0), "AITraceRegistry: invalid certId");
        require(merkleRoot != bytes32(0), "AITraceRegistry: invalid merkleRoot");
        require(!attestations[certId].isValid, "AITraceRegistry: attestation exists");

        if (attestationFee > 0) {
            require(msg.value >= attestationFee, "AITraceRegistry: insufficient fee");
        }

        AttestationRecord storage record = attestations[certId];
        record.certId = certId;
        record.merkleRoot = merkleRoot;
        record.fingerprintHash = fingerprintHash;
        record.inputHash = inputHash;
        record.outputHash = outputHash;
        record.submitter = msg.sender;
        record.timestamp = block.timestamp;
        record.blockNumber = block.number;
        record.modelId = modelId;
        record.tenantId = tenantId;
        record.isValid = true;

        // 更新租户统计
        bytes32 tenantIdHash = keccak256(abi.encodePacked(tenantId));
        if (tenants[tenantIdHash].isActive) {
            tenants[tenantIdHash].attestationCount++;
        }

        totalAttestations++;

        emit AttestationCreated(
            certId,
            merkleRoot,
            fingerprintHash,
            msg.sender,
            modelId,
            tenantId,
            block.timestamp
        );
    }

    /**
     * @notice 简化版存证（兼容旧接口）
     * @param certHash 证书哈希
     * @param rootHash Merkle 根哈希
     * @param timestamp 时间戳
     */
    function anchor(
        bytes32 certHash,
        bytes32 rootHash,
        uint256 timestamp
    ) external onlyAuthorized {
        require(certHash != bytes32(0), "AITraceRegistry: invalid certHash");
        require(!attestations[certHash].isValid, "AITraceRegistry: attestation exists");

        AttestationRecord storage record = attestations[certHash];
        record.certId = certHash;
        record.merkleRoot = rootHash;
        record.submitter = msg.sender;
        record.timestamp = timestamp;
        record.blockNumber = block.number;
        record.isValid = true;

        totalAttestations++;

        emit AttestationCreated(
            certHash,
            rootHash,
            bytes32(0),
            msg.sender,
            "",
            "",
            timestamp
        );
    }

    /**
     * @notice 批量创建存证
     * @param certIds 证书 ID 数组
     * @param merkleRoots Merkle 根哈希数组
     * @param fingerprintHashes 指纹哈希数组
     */
    function batchCreateAttestations(
        bytes32[] calldata certIds,
        bytes32[] calldata merkleRoots,
        bytes32[] calldata fingerprintHashes
    ) external onlyAuthorized {
        require(
            certIds.length == merkleRoots.length &&
            certIds.length == fingerprintHashes.length,
            "AITraceRegistry: array length mismatch"
        );
        require(certIds.length <= 100, "AITraceRegistry: batch too large");

        for (uint256 i = 0; i < certIds.length; i++) {
            if (attestations[certIds[i]].isValid) continue;

            AttestationRecord storage record = attestations[certIds[i]];
            record.certId = certIds[i];
            record.merkleRoot = merkleRoots[i];
            record.fingerprintHash = fingerprintHashes[i];
            record.submitter = msg.sender;
            record.timestamp = block.timestamp;
            record.blockNumber = block.number;
            record.isValid = true;

            totalAttestations++;

            emit AttestationCreated(
                certIds[i],
                merkleRoots[i],
                fingerprintHashes[i],
                msg.sender,
                "",
                "",
                block.timestamp
            );
        }
    }

    /**
     * @notice 撤销存证（仅限提交者或所有者）
     * @param certId 证书 ID
     */
    function revokeAttestation(bytes32 certId) external {
        AttestationRecord storage record = attestations[certId];
        require(record.isValid, "AITraceRegistry: attestation not found");
        require(
            record.submitter == msg.sender || msg.sender == owner,
            "AITraceRegistry: not authorized to revoke"
        );

        record.isValid = false;

        emit AttestationRevoked(certId, msg.sender, block.timestamp);
    }

    // ============ 查询功能 ============

    /**
     * @notice 验证存证是否有效
     * @param certId 证书 ID
     * @param merkleRoot 预期的 Merkle 根哈希
     * @return valid 是否有效
     */
    function verifyAttestation(
        bytes32 certId,
        bytes32 merkleRoot
    ) external view returns (bool valid) {
        AttestationRecord storage record = attestations[certId];
        return record.isValid && record.merkleRoot == merkleRoot;
    }

    /**
     * @notice 获取存证详情
     * @param certId 证书 ID
     * @return record 存证记录
     */
    function getAttestation(bytes32 certId)
        external
        view
        returns (AttestationRecord memory record)
    {
        return attestations[certId];
    }

    /**
     * @notice 验证指纹哈希
     * @param certId 证书 ID
     * @param fingerprintHash 预期的指纹哈希
     * @return valid 是否匹配
     */
    function verifyFingerprint(
        bytes32 certId,
        bytes32 fingerprintHash
    ) external view returns (bool valid) {
        AttestationRecord storage record = attestations[certId];
        return record.isValid && record.fingerprintHash == fingerprintHash;
    }

    /**
     * @notice 检查存证是否存在
     * @param certId 证书 ID
     * @return exists 是否存在
     */
    function attestationExists(bytes32 certId) external view returns (bool exists) {
        return attestations[certId].isValid;
    }

    // ============ 租户管理 ============

    /**
     * @notice 注册租户
     * @param tenantId 租户标识
     */
    function registerTenant(string calldata tenantId) external {
        bytes32 tenantIdHash = keccak256(abi.encodePacked(tenantId));
        require(!tenants[tenantIdHash].isActive, "AITraceRegistry: tenant exists");

        tenants[tenantIdHash] = Tenant({
            owner: msg.sender,
            isActive: true,
            attestationCount: 0,
            registeredAt: block.timestamp
        });

        emit TenantRegistered(tenantIdHash, msg.sender, block.timestamp);
    }

    /**
     * @notice 获取租户信息
     * @param tenantId 租户标识
     * @return tenant 租户信息
     */
    function getTenant(string calldata tenantId)
        external
        view
        returns (Tenant memory tenant)
    {
        bytes32 tenantIdHash = keccak256(abi.encodePacked(tenantId));
        return tenants[tenantIdHash];
    }

    // ============ 管理功能 ============

    /**
     * @notice 设置授权提交者
     * @param submitter 提交者地址
     * @param authorized 是否授权
     */
    function setAuthorizedSubmitter(
        address submitter,
        bool authorized
    ) external onlyOwner {
        authorizedSubmitters[submitter] = authorized;
        emit SubmitterAuthorizationChanged(submitter, authorized);
    }

    /**
     * @notice 设置存证费用
     * @param fee 新费用（wei）
     */
    function setAttestationFee(uint256 fee) external onlyOwner {
        attestationFee = fee;
    }

    /**
     * @notice 转移所有权
     * @param newOwner 新所有者地址
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "AITraceRegistry: invalid address");
        owner = newOwner;
    }

    /**
     * @notice 提取合约余额
     */
    function withdraw() external onlyOwner {
        payable(owner).transfer(address(this).balance);
    }
}
