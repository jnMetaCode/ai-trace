-- AI-Trace Server 数据库初始化脚本
-- 版本: 0.2
-- 使用方法: psql -U postgres -d ai_trace -f init.sql

-- ============================================
-- 创建数据库（如果不存在）
-- ============================================
-- CREATE DATABASE ai_trace;

-- ============================================
-- 扩展
-- ============================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 事件表 - 存储所有 AI 调用事件
-- ============================================
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    user_id VARCHAR(64),
    session_id VARCHAR(64),

    -- 事件类型和顺序
    event_type VARCHAR(32) NOT NULL,
    sequence INT NOT NULL DEFAULT 0,

    -- 哈希和链接
    event_hash VARCHAR(128) NOT NULL,
    prev_event_hash VARCHAR(128),
    parent_event_id VARCHAR(64),
    payload_hash VARCHAR(128),

    -- 数据
    payload JSONB,
    metadata JSONB,
    context JSONB,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT unique_trace_sequence UNIQUE (trace_id, sequence)
);

-- 事件表索引
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
CREATE INDEX IF NOT EXISTS idx_events_tenant_trace ON events(tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_events_session_id ON events(session_id) WHERE session_id IS NOT NULL;

-- ============================================
-- 证书表 - 存储存证证书
-- ============================================
CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    cert_id VARCHAR(64) UNIQUE NOT NULL,
    cert_version VARCHAR(16) DEFAULT '1.0',

    -- 关联
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',

    -- Merkle 树
    root_hash VARCHAR(128) NOT NULL,
    event_count INT NOT NULL DEFAULT 0,

    -- 存证级别: L1(本地), L2(WORM), L3(区块链)
    evidence_level VARCHAR(8) NOT NULL DEFAULT 'L1',

    -- 完整证书数据 (JSON)
    cert_data JSONB NOT NULL,

    -- 锚定信息
    anchor_type VARCHAR(32),
    anchor_id VARCHAR(128),
    anchor_tx_hash VARCHAR(128),
    anchor_block_number BIGINT,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- 状态
    status VARCHAR(16) DEFAULT 'active'
);

-- 证书表索引
CREATE INDEX IF NOT EXISTS idx_certs_trace_id ON certificates(trace_id);
CREATE INDEX IF NOT EXISTS idx_certs_tenant_id ON certificates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_certs_root_hash ON certificates(root_hash);
CREATE INDEX IF NOT EXISTS idx_certs_evidence_level ON certificates(evidence_level);
CREATE INDEX IF NOT EXISTS idx_certs_created_at ON certificates(created_at);
CREATE INDEX IF NOT EXISTS idx_certs_tenant_trace ON certificates(tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS idx_certs_anchor_type ON certificates(anchor_type) WHERE anchor_type IS NOT NULL;

-- ============================================
-- 联邦锚定表 - 存储联邦节点锚定记录
-- ============================================
CREATE TABLE IF NOT EXISTS federated_anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(64) UNIQUE NOT NULL,

    -- 关联
    cert_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,

    -- 发起节点
    origin_node VARCHAR(64) NOT NULL,
    origin_signature TEXT,

    -- 确认信息
    confirmations JSONB NOT NULL DEFAULT '[]',
    confirmation_count INT NOT NULL DEFAULT 0,
    min_confirmations INT NOT NULL DEFAULT 2,

    -- 状态
    status VARCHAR(16) DEFAULT 'pending', -- pending, confirmed, failed

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_cert FOREIGN KEY (cert_id) REFERENCES certificates(cert_id)
);

-- 联邦锚定表索引
CREATE INDEX IF NOT EXISTS idx_fed_anchors_cert_id ON federated_anchors(cert_id);
CREATE INDEX IF NOT EXISTS idx_fed_anchors_origin_node ON federated_anchors(origin_node);
CREATE INDEX IF NOT EXISTS idx_fed_anchors_status ON federated_anchors(status);

-- ============================================
-- 联邦节点注册表 - 存储已知的联邦节点
-- ============================================
CREATE TABLE IF NOT EXISTS federated_nodes (
    id SERIAL PRIMARY KEY,
    node_id VARCHAR(64) UNIQUE NOT NULL,

    -- 节点信息
    endpoint VARCHAR(256) NOT NULL,
    public_key TEXT NOT NULL,

    -- 元数据
    name VARCHAR(128),
    description TEXT,
    region VARCHAR(64),
    organization VARCHAR(128),

    -- 信任状态
    is_trusted BOOLEAN DEFAULT false,
    trust_level INT DEFAULT 0, -- 0: 未知, 1: 基础, 2: 验证, 3: 完全信任

    -- 统计
    total_confirmations INT DEFAULT 0,
    successful_confirmations INT DEFAULT 0,
    last_seen_at TIMESTAMP WITH TIME ZONE,

    -- 状态
    status VARCHAR(16) DEFAULT 'active', -- active, inactive, banned

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 联邦节点索引
CREATE INDEX IF NOT EXISTS idx_fed_nodes_endpoint ON federated_nodes(endpoint);
CREATE INDEX IF NOT EXISTS idx_fed_nodes_is_trusted ON federated_nodes(is_trusted);
CREATE INDEX IF NOT EXISTS idx_fed_nodes_status ON federated_nodes(status);

-- ============================================
-- 区块链锚定表 - 存储区块链锚定记录（L3）
-- ============================================
CREATE TABLE IF NOT EXISTS blockchain_anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(64) UNIQUE NOT NULL,

    -- 关联
    cert_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,

    -- 区块链信息
    chain_type VARCHAR(32) NOT NULL, -- ethereum, polygon, bsc
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(128) NOT NULL,
    block_number BIGINT,
    block_hash VARCHAR(128),
    contract_address VARCHAR(64),

    -- 费用
    gas_used BIGINT,
    gas_price BIGINT,

    -- 确认
    confirmations INT DEFAULT 0,

    -- 状态
    status VARCHAR(16) DEFAULT 'pending', -- pending, confirmed, failed

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP WITH TIME ZONE,

    -- 验证 URL
    explorer_url VARCHAR(512),

    CONSTRAINT fk_cert_blockchain FOREIGN KEY (cert_id) REFERENCES certificates(cert_id)
);

-- 区块链锚定索引
CREATE INDEX IF NOT EXISTS idx_bc_anchors_cert_id ON blockchain_anchors(cert_id);
CREATE INDEX IF NOT EXISTS idx_bc_anchors_tx_hash ON blockchain_anchors(tx_hash);
CREATE INDEX IF NOT EXISTS idx_bc_anchors_chain_type ON blockchain_anchors(chain_type);

-- ============================================
-- API 密钥表 - 管理 API 访问密钥
-- ============================================
CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    key_id VARCHAR(64) UNIQUE NOT NULL,
    key_hash VARCHAR(128) NOT NULL, -- 存储密钥的哈希值
    key_prefix VARCHAR(16) NOT NULL, -- 密钥前缀用于识别

    -- 关联
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',

    -- 元数据
    name VARCHAR(128),
    description TEXT,

    -- 权限
    scopes JSONB DEFAULT '["read", "write"]',

    -- 限制
    rate_limit INT DEFAULT 1000, -- 每分钟请求数
    daily_limit INT DEFAULT 100000, -- 每日请求数

    -- 统计
    total_requests BIGINT DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- 状态
    status VARCHAR(16) DEFAULT 'active', -- active, revoked, expired

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE
);

-- API 密钥索引
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

-- ============================================
-- 审计日志表 - 记录重要操作
-- ============================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    log_id VARCHAR(64) UNIQUE NOT NULL DEFAULT uuid_generate_v4()::text,

    -- 操作信息
    action VARCHAR(64) NOT NULL, -- create_cert, verify_cert, anchor, etc.
    resource_type VARCHAR(32) NOT NULL, -- event, certificate, anchor
    resource_id VARCHAR(64),

    -- 执行者
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    user_id VARCHAR(64),
    api_key_id VARCHAR(64),

    -- 请求信息
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(64),

    -- 结果
    status VARCHAR(16) NOT NULL, -- success, failed, error
    error_message TEXT,

    -- 详情
    details JSONB,

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 审计日志索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);

-- ============================================
-- 指纹表 - 存储推理行为指纹
-- ============================================
CREATE TABLE IF NOT EXISTS fingerprints (
    id SERIAL PRIMARY KEY,
    fingerprint_id VARCHAR(64) UNIQUE NOT NULL,

    -- 关联
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
    cert_id VARCHAR(64),

    -- 指纹哈希（用于快速比较）
    fingerprint_hash VARCHAR(128) NOT NULL,

    -- 模型信息
    model_id VARCHAR(128) NOT NULL,
    model_provider VARCHAR(64),

    -- 统计特征 (Layer 1)
    statistical_features JSONB NOT NULL,
    -- token概率特征 (Layer 2) - 可选
    token_prob_features JSONB,
    -- 模型内部特征 (Layer 3) - 可选
    model_internal_features JSONB,
    -- 语义特征 (Layer 4)
    semantic_features JSONB NOT NULL,

    -- 完整指纹数据
    full_fingerprint JSONB NOT NULL,

    -- 状态
    status VARCHAR(16) DEFAULT 'active',

    -- 时间戳
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_fingerprint_cert FOREIGN KEY (cert_id) REFERENCES certificates(cert_id)
);

-- 指纹表索引
CREATE INDEX IF NOT EXISTS idx_fingerprints_trace_id ON fingerprints(trace_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_tenant_id ON fingerprints(tenant_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_cert_id ON fingerprints(cert_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_fingerprint_hash ON fingerprints(fingerprint_hash);
CREATE INDEX IF NOT EXISTS idx_fingerprints_model_id ON fingerprints(model_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_generated_at ON fingerprints(generated_at);

-- ============================================
-- 加密内容表 - 存储加密内容引用
-- ============================================
CREATE TABLE IF NOT EXISTS encrypted_contents (
    id SERIAL PRIMARY KEY,
    content_id VARCHAR(64) UNIQUE NOT NULL,

    -- 关联
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',

    -- 内容类型: prompt, output
    content_type VARCHAR(16) NOT NULL,

    -- 存储引用 (minio://bucket/path)
    storage_ref VARCHAR(512) NOT NULL,

    -- 原始内容哈希（用于验证）
    content_hash VARCHAR(128) NOT NULL,

    -- 大小
    original_size BIGINT NOT NULL,
    encrypted_size BIGINT NOT NULL,

    -- 加密信息
    key_version INT NOT NULL DEFAULT 1,
    encryption_algorithm VARCHAR(32) DEFAULT 'AES-256-GCM',

    -- 时间戳
    encrypted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 加密内容索引
CREATE INDEX IF NOT EXISTS idx_encrypted_contents_trace_id ON encrypted_contents(trace_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_contents_tenant_id ON encrypted_contents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_contents_content_type ON encrypted_contents(content_type);
CREATE INDEX IF NOT EXISTS idx_encrypted_contents_storage_ref ON encrypted_contents(storage_ref);

-- ============================================
-- 解密审计表 - 记录解密操作
-- ============================================
CREATE TABLE IF NOT EXISTS decrypt_audit_logs (
    id SERIAL PRIMARY KEY,
    audit_id VARCHAR(64) UNIQUE NOT NULL DEFAULT uuid_generate_v4()::text,

    -- 解密的内容
    content_id VARCHAR(64) NOT NULL,
    encrypted_ref VARCHAR(512) NOT NULL,
    content_type VARCHAR(16) NOT NULL,

    -- 操作者
    tenant_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64),

    -- 请求信息
    client_ip VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(64),

    -- 结果
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,

    -- 时间戳
    decrypted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 解密审计索引
CREATE INDEX IF NOT EXISTS idx_decrypt_audit_content_id ON decrypt_audit_logs(content_id);
CREATE INDEX IF NOT EXISTS idx_decrypt_audit_tenant_id ON decrypt_audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_decrypt_audit_user_id ON decrypt_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_decrypt_audit_encrypted_ref ON decrypt_audit_logs(encrypted_ref);
CREATE INDEX IF NOT EXISTS idx_decrypt_audit_decrypted_at ON decrypt_audit_logs(decrypted_at);

-- ============================================
-- 报告表 - 存储生成的报告
-- ============================================
CREATE TABLE IF NOT EXISTS reports (
    id SERIAL PRIMARY KEY,
    report_id VARCHAR(64) UNIQUE NOT NULL,

    -- 类型
    report_type VARCHAR(32) NOT NULL, -- audit, compliance, summary
    report_format VARCHAR(16) NOT NULL DEFAULT 'json', -- json, html, pdf

    -- 关联
    tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',

    -- 内容
    title VARCHAR(256),
    summary JSONB,
    content_path VARCHAR(512), -- MinIO 路径
    content_size BIGINT,

    -- 过滤条件
    filters JSONB, -- {trace_ids, cert_ids, start_time, end_time}

    -- 状态
    status VARCHAR(16) DEFAULT 'completed', -- pending, generating, completed, failed

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- 报告索引
CREATE INDEX IF NOT EXISTS idx_reports_tenant_id ON reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_reports_report_type ON reports(report_type);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);

-- ============================================
-- 更新触发器
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要的表添加更新触发器
DROP TRIGGER IF EXISTS update_certificates_updated_at ON certificates;
CREATE TRIGGER update_certificates_updated_at
    BEFORE UPDATE ON certificates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_federated_nodes_updated_at ON federated_nodes;
CREATE TRIGGER update_federated_nodes_updated_at
    BEFORE UPDATE ON federated_nodes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 初始数据
-- ============================================

-- 插入默认 API 密钥（仅开发环境）
-- 生产环境应该通过管理界面创建
INSERT INTO api_keys (key_id, key_hash, key_prefix, tenant_id, name, scopes)
VALUES (
    'default-dev-key',
    'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', -- SHA256 of 'test-api-key-12345'
    'test-api-',
    'default',
    'Development API Key',
    '["read", "write", "admin"]'
) ON CONFLICT (key_id) DO NOTHING;

-- ============================================
-- 权限设置
-- ============================================
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ai_trace_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ai_trace_user;

-- ============================================
-- 完成
-- ============================================
SELECT 'AI-Trace database initialized successfully!' as message;
