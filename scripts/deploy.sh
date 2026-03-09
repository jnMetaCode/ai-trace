#!/bin/bash

# AI-Trace 一键部署脚本
# 支持: Docker Compose / Kubernetes / 单机快速启动

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logo
print_logo() {
    echo -e "${BLUE}"
    echo "    _    ___   _____                    "
    echo "   / \  |_ _| |_   _| __ __ _  ___ ___  "
    echo "  / _ \  | |    | || '__/ _\` |/ __/ _ \ "
    echo " / ___ \ | |    | || | | (_| | (_|  __/ "
    echo "/_/   \_\___|   |_||_|  \__,_|\___\___| "
    echo ""
    echo -e "${NC}Enterprise AI Decision Audit Platform"
    echo "============================================"
    echo ""
}

# 检查依赖
check_dependencies() {
    echo -e "${YELLOW}Checking dependencies...${NC}"

    local missing=0

    # 检查 Docker
    if command -v docker &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Docker $(docker --version | awk '{print $3}' | tr -d ',')"
    else
        echo -e "  ${RED}✗${NC} Docker not found"
        missing=1
    fi

    # 检查 Docker Compose
    if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Docker Compose"
    else
        echo -e "  ${RED}✗${NC} Docker Compose not found"
        missing=1
    fi

    # 检查 kubectl (可选)
    if command -v kubectl &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} kubectl $(kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion": "[^"]*"' | head -1 | cut -d'"' -f4)"
    else
        echo -e "  ${YELLOW}○${NC} kubectl not found (optional, for K8s deployment)"
    fi

    echo ""

    if [ $missing -eq 1 ]; then
        echo -e "${RED}Please install missing dependencies first.${NC}"
        exit 1
    fi
}

# 生成配置文件
generate_config() {
    local env_file=".env"

    if [ -f "$env_file" ]; then
        echo -e "${YELLOW}Found existing .env file. Use it? [Y/n]${NC}"
        read -r use_existing
        if [[ "$use_existing" =~ ^[Nn] ]]; then
            mv "$env_file" "${env_file}.backup.$(date +%s)"
        else
            return
        fi
    fi

    echo -e "${BLUE}Generating configuration...${NC}"

    # 生成随机密钥
    local db_password=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)
    local redis_password=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)
    local minio_secret=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)
    local api_key=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
    local ed25519_seed=$(openssl rand -hex 32)

    cat > "$env_file" << EOF
# AI-Trace Configuration
# Generated at $(date)

# Server
SERVER_PORT=8080
SERVER_MODE=release
LOG_LEVEL=info

# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=ai_trace
POSTGRES_PASSWORD=${db_password}
POSTGRES_DB=ai_trace

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=${redis_password}

# MinIO (WORM Storage)
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=ai_trace_admin
MINIO_SECRET_KEY=${minio_secret}
MINIO_BUCKET=ai-trace-certs
MINIO_USE_SSL=false

# API Authentication
AI_TRACE_API_KEY=${api_key}

# Ed25519 Signing Key (32 bytes hex)
ED25519_SEED=${ed25519_seed}

# OpenAI (Optional - users can provide their own)
# OPENAI_API_KEY=sk-xxx
# OPENAI_BASE_URL=https://api.openai.com/v1

# Blockchain Anchoring (Optional)
# ETH_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY
# ETH_PRIVATE_KEY=0x...
# ANCHOR_CONTRACT=0x...
EOF

    echo -e "${GREEN}Configuration generated: ${env_file}${NC}"
    echo -e "${YELLOW}Your API Key: ${api_key}${NC}"
    echo ""
}

# Docker Compose 部署
deploy_docker_compose() {
    echo -e "${BLUE}Deploying with Docker Compose...${NC}"
    echo ""

    # 检查 docker-compose.yml
    if [ ! -f "docker-compose.yml" ]; then
        echo -e "${RED}docker-compose.yml not found!${NC}"
        exit 1
    fi

    # 加载环境变量
    if [ -f ".env" ]; then
        export $(cat .env | grep -v '^#' | xargs)
    fi

    # 创建网络
    docker network create ai-trace-network 2>/dev/null || true

    # 启动服务
    echo "Starting services..."
    docker compose up -d

    echo ""
    echo -e "${GREEN}Deployment complete!${NC}"
    echo ""
    echo "Services:"
    echo "  - API Server: http://localhost:8080"
    echo "  - Swagger UI: http://localhost:8080/swagger/index.html"
    echo "  - MinIO Console: http://localhost:9001 (admin/minioadmin)"
    echo ""
    echo "Quick test:"
    echo "  curl http://localhost:8080/health"
    echo ""
}

# Kubernetes 部署
deploy_kubernetes() {
    echo -e "${BLUE}Deploying to Kubernetes...${NC}"
    echo ""

    local namespace="${1:-ai-trace}"
    local k8s_dir="deploy/k8s"

    if [ ! -d "$k8s_dir" ]; then
        echo -e "${RED}Kubernetes manifests not found in ${k8s_dir}${NC}"
        exit 1
    fi

    # 创建命名空间
    kubectl create namespace "$namespace" 2>/dev/null || true

    # 创建 secrets
    if [ -f ".env" ]; then
        kubectl create secret generic ai-trace-secrets \
            --from-env-file=.env \
            -n "$namespace" \
            --dry-run=client -o yaml | kubectl apply -f -
    fi

    # 应用 manifests
    kubectl apply -f "$k8s_dir/" -n "$namespace"

    echo ""
    echo -e "${GREEN}Kubernetes deployment initiated!${NC}"
    echo ""
    echo "Check status:"
    echo "  kubectl get pods -n $namespace"
    echo "  kubectl get svc -n $namespace"
    echo ""
}

# 验证节点部署
deploy_verifier_node() {
    echo -e "${BLUE}Deploying Verifier Node...${NC}"
    echo ""

    # 验证节点只需要最小配置
    cat > docker-compose.verifier.yml << 'EOF'
version: '3.8'

services:
  ai-trace-verifier:
    image: ai-trace/server:latest
    container_name: ai-trace-verifier
    ports:
      - "8081:8080"
    environment:
      - SERVER_MODE=verifier
      - VERIFIER_ONLY=true
      - LOG_LEVEL=info
    command: ["./ai-trace-server", "--verifier-mode"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  default:
    name: ai-trace-verifier-network
EOF

    docker compose -f docker-compose.verifier.yml up -d

    echo ""
    echo -e "${GREEN}Verifier node deployed!${NC}"
    echo ""
    echo "Verifier API: http://localhost:8081"
    echo ""
    echo "Verify a certificate:"
    echo "  curl -X POST http://localhost:8081/api/v1/certs/verify \\"
    echo "    -H 'Content-Type: application/json' \\"
    echo "    -d '{\"cert_id\": \"cert_xxx\"}'"
    echo ""
}

# 停止服务
stop_services() {
    echo -e "${YELLOW}Stopping AI-Trace services...${NC}"
    docker compose down 2>/dev/null || true
    docker compose -f docker-compose.verifier.yml down 2>/dev/null || true
    echo -e "${GREEN}Services stopped.${NC}"
}

# 清理数据
cleanup() {
    echo -e "${RED}WARNING: This will delete all AI-Trace data!${NC}"
    echo -e "Are you sure? Type 'yes' to confirm: "
    read -r confirm

    if [ "$confirm" = "yes" ]; then
        stop_services
        docker volume rm ai-trace_postgres_data 2>/dev/null || true
        docker volume rm ai-trace_redis_data 2>/dev/null || true
        docker volume rm ai-trace_minio_data 2>/dev/null || true
        echo -e "${GREEN}Cleanup complete.${NC}"
    else
        echo "Cancelled."
    fi
}

# 显示状态
show_status() {
    echo -e "${BLUE}AI-Trace Service Status${NC}"
    echo ""

    docker compose ps 2>/dev/null || echo "Docker Compose services not running"

    echo ""
    echo "Health check:"
    curl -s http://localhost:8080/health 2>/dev/null && echo "" || echo "API server not responding"
}

# 显示帮助
show_help() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy         Deploy with Docker Compose (default)"
    echo "  k8s [ns]       Deploy to Kubernetes (optional namespace)"
    echo "  verifier       Deploy a verifier-only node"
    echo "  config         Generate configuration file"
    echo "  status         Show service status"
    echo "  stop           Stop all services"
    echo "  cleanup        Stop and remove all data"
    echo "  help           Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 deploy              # Full deployment with Docker Compose"
    echo "  $0 k8s production      # Deploy to K8s namespace 'production'"
    echo "  $0 verifier            # Deploy verification-only node"
    echo ""
}

# 主函数
main() {
    print_logo
    check_dependencies

    local command="${1:-deploy}"

    case "$command" in
        deploy)
            generate_config
            deploy_docker_compose
            ;;
        k8s|kubernetes)
            generate_config
            deploy_kubernetes "${2:-ai-trace}"
            ;;
        verifier)
            deploy_verifier_node
            ;;
        config)
            generate_config
            ;;
        status)
            show_status
            ;;
        stop)
            stop_services
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            show_help
            exit 1
            ;;
    esac
}

# 运行
main "$@"
