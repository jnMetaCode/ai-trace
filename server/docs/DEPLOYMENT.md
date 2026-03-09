# AI-Trace 上线部署方案

## 版本信息

- **版本**: v0.2.0
- **发布日期**: 2025-01
- **代号**: Genesis

---

## 一、部署架构

### 1.1 单节点部署（入门）

```
┌─────────────────────────────────────────────────┐
│                   用户应用                        │
│              (替换 OpenAI endpoint)              │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│              AI-Trace Server                     │
│                 :8006                            │
└──────┬───────────┬───────────┬──────────────────┘
       │           │           │
       ▼           ▼           ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│PostgreSQL│ │  Redis   │ │  MinIO   │
│  :5432   │ │  :6379   │ │  :9000   │
└──────────┘ └──────────┘ └──────────┘
```

### 1.2 高可用部署（生产）

```
                    ┌─────────────┐
                    │   Nginx     │
                    │ (负载均衡)   │
                    └──────┬──────┘
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │ AI-Trace-1 │  │ AI-Trace-2 │  │ AI-Trace-3 │
    │  (联邦节点) │  │  (联邦节点) │  │  (联邦节点) │
    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
    ┌─────────────────────┼─────────────────────┐
    │                     │                     │
    ▼                     ▼                     ▼
┌────────┐         ┌────────────┐         ┌────────┐
│PostgreSQL│       │   Redis    │         │ MinIO  │
│ (主从)  │        │  (哨兵)    │         │(集群)  │
└────────┘         └────────────┘         └────────┘
```

---

## 二、环境要求

### 2.1 硬件配置

| 环境 | CPU | 内存 | 磁盘 | 网络 |
|------|-----|------|------|------|
| 开发 | 2核 | 4GB | 50GB | 1Gbps |
| 测试 | 4核 | 8GB | 100GB | 1Gbps |
| 生产 | 8核+ | 16GB+ | 500GB+ SSD | 10Gbps |

### 2.2 软件依赖

| 组件 | 版本 | 用途 |
|------|------|------|
| Go | 1.21+ | 编译运行 |
| PostgreSQL | 15+ | 主数据库 |
| Redis | 7+ | 缓存/队列 |
| MinIO | RELEASE.2024+ | 对象存储 |
| Docker | 24+ | 容器化 |
| Nginx | 1.24+ | 反向代理 |

---

## 三、部署步骤

### 3.1 Docker Compose 部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/jnMetaCode/ai-trace.git
cd server

# 2. 配置环境变量
cp .env.example .env
vim .env

# 3. 启动服务
docker-compose up -d

# 4. 初始化数据库
docker exec -i ai-trace-postgres psql -U postgres -d ai_trace < deploy/init.sql

# 5. 验证服务
curl http://localhost:8006/health
```

### 3.2 二进制部署

```bash
# 1. 编译
go build -o ai-trace-server ./cmd/ai-trace-server

# 2. 准备配置
cp config.yaml /etc/ai-trace/config.yaml
vim /etc/ai-trace/config.yaml

# 3. 创建 systemd 服务
cat > /etc/systemd/system/ai-trace.service << 'EOF'
[Unit]
Description=AI-Trace Server
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=ai-trace
Group=ai-trace
WorkingDirectory=/opt/ai-trace
ExecStart=/opt/ai-trace/ai-trace-server
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# 4. 启动服务
systemctl daemon-reload
systemctl enable ai-trace
systemctl start ai-trace

# 5. 查看日志
journalctl -u ai-trace -f
```

### 3.3 Kubernetes 部署

```bash
# 1. 创建命名空间
kubectl create namespace ai-trace

# 2. 部署配置
kubectl apply -f deploy/k8s/

# 3. 检查状态
kubectl get pods -n ai-trace
kubectl get svc -n ai-trace

# 4. 获取访问地址
kubectl get ingress -n ai-trace
```

---

## 四、配置说明

### 4.1 生产环境配置模板

```yaml
# config.prod.yaml
server:
  port: 8006
  mode: release

database:
  host: ${DB_HOST}
  port: 5432
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  dbname: ai_trace
  sslmode: require

redis:
  host: ${REDIS_HOST}
  port: 6379
  password: ${REDIS_PASSWORD}
  db: 0

minio:
  endpoint: ${MINIO_ENDPOINT}
  access_key: ${MINIO_ACCESS_KEY}
  secret_key: ${MINIO_SECRET_KEY}
  bucket: ai-trace-prod
  use_ssl: true

gateway:
  openai:
    base_url: https://api.openai.com/v1
  timeout: 120
  max_retries: 3

auth:
  api_keys:
    - ${API_KEY_1}
    - ${API_KEY_2}

features:
  blockchain_anchor: false
  federated_nodes: true
  metrics: true
  reports: true

anchor:
  federated:
    enabled: true
    nodes:
      - https://node2.aitrace.cc
      - https://node3.aitrace.cc
    min_confirmations: 2
```

### 4.2 Nginx 反向代理配置

```nginx
# /etc/nginx/sites-available/ai-trace
upstream ai_trace {
    server 127.0.0.1:8006;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.aitrace.cc;

    ssl_certificate /etc/letsencrypt/live/aitrace.cc/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/aitrace.cc/privkey.pem;

    # SSL 配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # 安全头
    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;

    location / {
        proxy_pass http://ai_trace;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 120s;
        proxy_read_timeout 120s;
    }

    # API 文档
    location /swagger/ {
        proxy_pass http://ai_trace;
    }

    # 健康检查
    location /health {
        proxy_pass http://ai_trace;
        access_log off;
    }

    # 监控指标
    location /metrics {
        proxy_pass http://ai_trace;
        allow 10.0.0.0/8;
        deny all;
    }
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name api.aitrace.cc;
    return 301 https://$server_name$request_uri;
}
```

---

## 五、监控告警

### 5.1 Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'ai-trace'
    static_configs:
      - targets: ['ai-trace:8006']
    metrics_path: '/metrics'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

rule_files:
  - '/etc/prometheus/rules/*.yml'
```

### 5.2 告警规则

```yaml
# alerts.yml
groups:
  - name: ai-trace
    rules:
      - alert: HighErrorRate
        expr: rate(ai_trace_http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"

      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(ai_trace_http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"

      - alert: ServiceDown
        expr: up{job="ai-trace"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "AI-Trace service is down"
```

### 5.3 Grafana Dashboard

导入 `deploy/grafana/ai-trace-dashboard.json`

---

## 六、安全检查清单

### 6.1 部署前检查

- [ ] 修改所有默认密码
- [ ] 配置防火墙规则
- [ ] 启用 HTTPS/TLS
- [ ] 配置 API Key
- [ ] 设置速率限制
- [ ] 审计日志配置
- [ ] 备份策略确认

### 6.2 网络安全

```bash
# 防火墙规则示例
ufw allow 443/tcp        # HTTPS
ufw allow from 10.0.0.0/8 to any port 8006  # 内部访问
ufw deny 8006/tcp        # 禁止外部直接访问
ufw enable
```

### 6.3 密钥管理

```bash
# 生成强密码
openssl rand -base64 32

# 生成 API Key
openssl rand -hex 32
```

---

## 七、上线验收

### 7.1 功能验收

```bash
# 健康检查
curl -s https://api.aitrace.cc/health | jq .

# API 可用性
curl -s -H "X-API-Key: xxx" https://api.aitrace.cc/api/v1/certs/search | jq .

# 代理功能
curl -s -X POST https://api.aitrace.cc/api/v1/chat/completions \
  -H "X-API-Key: xxx" \
  -H "X-Upstream-API-Key: sk-xxx" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}'
```

### 7.2 性能验收

```bash
# 使用 wrk 进行压测
wrk -t12 -c400 -d30s -H "X-API-Key: xxx" https://api.aitrace.cc/health

# 预期指标
# - QPS > 1000
# - P99 延迟 < 100ms
# - 错误率 < 0.1%
```

### 7.3 安全验收

```bash
# SSL 评级检查
curl -s "https://api.ssllabs.com/api/v3/analyze?host=api.aitrace.cc"

# 安全头检查
curl -I https://api.aitrace.cc/health
```

---

## 八、回滚方案

### 8.1 快速回滚

```bash
# Docker 环境
docker-compose down
docker-compose -f docker-compose.backup.yml up -d

# K8s 环境
kubectl rollout undo deployment/ai-trace -n ai-trace
```

### 8.2 数据回滚

```bash
# 恢复数据库
pg_restore -U postgres -d ai_trace ai_trace_backup.dump

# 恢复 MinIO 数据
mc mirror backup/ai-trace minio/ai-trace
```

---

## 九、运维手册

### 9.1 常用命令

```bash
# 查看日志
docker logs -f ai-trace-server --tail 100

# 重启服务
docker-compose restart ai-trace

# 查看资源使用
docker stats ai-trace-server

# 数据库连接
docker exec -it ai-trace-postgres psql -U postgres -d ai_trace
```

### 9.2 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 连接超时 | 数据库未启动 | 检查 PostgreSQL 状态 |
| 401 错误 | API Key 无效 | 检查配置文件 |
| 503 错误 | 功能未启用 | 检查 features 配置 |
| 证书错误 | MinIO 连接失败 | 检查 MinIO 配置 |

---

## 十、联系支持

- **文档**: https://docs.aitrace.cc
- **Issues**: https://github.com/jnMetaCode/ai-trace/issues
- **Email**: support@aitrace.cc
