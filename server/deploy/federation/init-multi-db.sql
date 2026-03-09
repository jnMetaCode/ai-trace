-- AI-Trace 多节点数据库初始化脚本
-- 为每个联邦节点创建独立的数据库

-- 创建节点 1 数据库
CREATE DATABASE ai_trace_node1;

-- 创建节点 2 数据库
CREATE DATABASE ai_trace_node2;

-- 创建节点 3 数据库
CREATE DATABASE ai_trace_node3;

-- 连接到 node1 数据库并创建表
\c ai_trace_node1;

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    event_hash VARCHAR(128) NOT NULL,
    sequence INT NOT NULL,
    payload JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_trace_id (trace_id),
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_event_type (event_type)
);

CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    cert_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    evidence_level VARCHAR(8) NOT NULL,
    cert_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_cert_trace_id (trace_id),
    INDEX idx_cert_tenant_id (tenant_id),
    INDEX idx_root_hash (root_hash)
);

CREATE TABLE IF NOT EXISTS federated_anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(64) UNIQUE NOT NULL,
    cert_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    origin_node VARCHAR(64) NOT NULL,
    confirmations JSONB NOT NULL,
    confirmation_count INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_anchor_cert_id (cert_id)
);

-- 连接到 node2 数据库并创建表
\c ai_trace_node2;

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    event_hash VARCHAR(128) NOT NULL,
    sequence INT NOT NULL,
    payload JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    cert_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    evidence_level VARCHAR(8) NOT NULL,
    cert_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS federated_anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(64) UNIQUE NOT NULL,
    cert_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    origin_node VARCHAR(64) NOT NULL,
    confirmations JSONB NOT NULL,
    confirmation_count INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 连接到 node3 数据库并创建表
\c ai_trace_node3;

CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    event_hash VARCHAR(128) NOT NULL,
    sequence INT NOT NULL,
    payload JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificates (
    id SERIAL PRIMARY KEY,
    cert_id VARCHAR(64) UNIQUE NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    evidence_level VARCHAR(8) NOT NULL,
    cert_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS federated_anchors (
    id SERIAL PRIMARY KEY,
    anchor_id VARCHAR(64) UNIQUE NOT NULL,
    cert_id VARCHAR(64) NOT NULL,
    root_hash VARCHAR(128) NOT NULL,
    origin_node VARCHAR(64) NOT NULL,
    confirmations JSONB NOT NULL,
    confirmation_count INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 授权
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
