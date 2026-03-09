-- AI-Trace 数据库初始化脚本
-- 版本: 0.1
-- 日期: 2026-01-09

-- 创建数据库（如果不存在）
-- CREATE DATABASE ai_trace;

-- 事件表
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(50) UNIQUE NOT NULL,
    trace_id VARCHAR(50) NOT NULL,
    parent_event_id VARCHAR(50),
    prev_event_hash VARCHAR(100),

    -- 事件类型
    event_type VARCHAR(20) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    sequence INTEGER NOT NULL,

    -- 租户与用户
    tenant_id VARCHAR(50) NOT NULL,
    user_id VARCHAR(50),
    session_id VARCHAR(50),

    -- 上下文
    context JSONB,

    -- 载荷
    payload JSONB NOT NULL,
    payload_hash VARCHAR(100) NOT NULL,
    event_hash VARCHAR(100) NOT NULL,

    -- 元信息
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    -- 索引
    CONSTRAINT events_event_id_key UNIQUE (event_id)
);

-- 事件索引
CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant_id ON events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_events_tenant_trace ON events(tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_events_event_hash ON events(event_hash);

-- 存证表
CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    cert_id VARCHAR(50) UNIQUE NOT NULL,
    trace_id VARCHAR(50) NOT NULL,
    tenant_id VARCHAR(50) NOT NULL,

    -- 存证信息
    root_hash VARCHAR(100) NOT NULL,
    event_count INTEGER NOT NULL,
    evidence_level VARCHAR(10) NOT NULL DEFAULT 'L1',

    -- 完整存证数据
    cert_data JSONB NOT NULL,

    -- 时间
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT certificates_cert_id_key UNIQUE (cert_id)
);

-- 存证索引
CREATE INDEX IF NOT EXISTS idx_certs_trace_id ON certificates(trace_id);
CREATE INDEX IF NOT EXISTS idx_certs_tenant_id ON certificates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_certs_root_hash ON certificates(root_hash);
CREATE INDEX IF NOT EXISTS idx_certs_created_at ON certificates(created_at);

-- 锚定记录表
CREATE TABLE IF NOT EXISTS anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(50) UNIQUE NOT NULL,
    cert_id VARCHAR(50) NOT NULL REFERENCES certificates(cert_id),

    -- 锚定信息
    anchor_type VARCHAR(20) NOT NULL,
    storage_provider VARCHAR(50),
    object_key VARCHAR(255),

    -- 区块链信息（如果适用）
    chain_id VARCHAR(50),
    tx_hash VARCHAR(100),
    block_height BIGINT,
    contract_address VARCHAR(100),

    -- 时间
    anchor_timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT anchors_anchor_id_key UNIQUE (anchor_id)
);

-- 锚定索引
CREATE INDEX IF NOT EXISTS idx_anchors_cert_id ON anchors(cert_id);
CREATE INDEX IF NOT EXISTS idx_anchors_tx_hash ON anchors(tx_hash);

-- API Keys表（简单实现）
CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    key_id VARCHAR(50) UNIQUE NOT NULL,
    api_key VARCHAR(100) UNIQUE NOT NULL,
    tenant_id VARCHAR(50) NOT NULL,
    name VARCHAR(100),
    permissions JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_api_key ON api_keys(api_key);

-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    settings JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认租户
INSERT INTO tenants (tenant_id, name)
VALUES ('default', 'Default Tenant')
ON CONFLICT (tenant_id) DO NOTHING;

-- 插入测试API Key
INSERT INTO api_keys (key_id, api_key, tenant_id, name)
VALUES ('key_default', 'test-api-key-12345', 'default', 'Test API Key')
ON CONFLICT (key_id) DO NOTHING;

-- 视图：事件统计
CREATE OR REPLACE VIEW event_stats AS
SELECT
    tenant_id,
    event_type,
    DATE(timestamp) as event_date,
    COUNT(*) as event_count
FROM events
GROUP BY tenant_id, event_type, DATE(timestamp);

-- 视图：存证统计
CREATE OR REPLACE VIEW cert_stats AS
SELECT
    tenant_id,
    evidence_level,
    DATE(created_at) as cert_date,
    COUNT(*) as cert_count,
    SUM(event_count) as total_events
FROM certificates
GROUP BY tenant_id, evidence_level, DATE(created_at);

-- 函数：清理过期数据（可选）
CREATE OR REPLACE FUNCTION cleanup_old_events(retention_days INTEGER)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM events
    WHERE created_at < CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL
    AND trace_id NOT IN (SELECT trace_id FROM certificates);

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE events IS 'AI调用事件记录表';
COMMENT ON TABLE certificates IS 'Merkle树存证表';
COMMENT ON TABLE anchors IS '防篡改锚定记录表';
COMMENT ON TABLE api_keys IS 'API访问密钥表';
COMMENT ON TABLE tenants IS '租户信息表';
