#!/bin/bash

# AI-Trace 联邦网络启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 显示帮助
show_help() {
    cat << EOF
AI-Trace 联邦网络管理脚本

用法: $0 <命令> [选项]

命令:
  start       启动联邦网络（3 节点）
  stop        停止联邦网络
  status      查看联邦网络状态
  logs        查看节点日志
  test        测试联邦共识
  clean       清理所有数据

选项:
  --monitoring    同时启动 Prometheus + Grafana 监控
  --node <n>      指定节点 (1, 2, 3)

示例:
  $0 start                    # 启动 3 节点联邦网络
  $0 start --monitoring       # 启动并包含监控
  $0 logs --node 1            # 查看节点 1 日志
  $0 test                     # 测试联邦共识
EOF
}

# 构建镜像
build_image() {
    log_info "构建 AI-Trace Server 镜像..."
    cd ../..
    docker build -t ai-trace-server:latest .
    cd "$SCRIPT_DIR"
    log_success "镜像构建完成"
}

# 启动联邦网络
start_federation() {
    local with_monitoring=$1

    log_info "启动 AI-Trace 联邦网络..."

    # 检查是否需要构建镜像
    if ! docker images | grep -q "ai-trace-server"; then
        build_image
    fi

    # 启动服务
    if [ "$with_monitoring" = "true" ]; then
        docker-compose -f docker-compose.multi-node.yml --profile monitoring up -d
        log_info "监控地址: http://localhost:9090 (Prometheus), http://localhost:3000 (Grafana)"
    else
        docker-compose -f docker-compose.multi-node.yml up -d
    fi

    # 等待服务就绪
    log_info "等待服务启动..."
    sleep 10

    # 检查状态
    check_status

    log_success "联邦网络启动完成"
    echo ""
    echo "节点地址:"
    echo "  Node 1: http://localhost:8006"
    echo "  Node 2: http://localhost:8007"
    echo "  Node 3: http://localhost:8008"
    echo ""
    echo "API Key:"
    echo "  Node 1: node1-api-key-12345"
    echo "  Node 2: node2-api-key-12345"
    echo "  Node 3: node3-api-key-12345"
}

# 停止联邦网络
stop_federation() {
    log_info "停止联邦网络..."
    docker-compose -f docker-compose.multi-node.yml --profile monitoring down
    log_success "联邦网络已停止"
}

# 检查状态
check_status() {
    log_info "检查节点状态..."
    echo ""

    for i in 1 2 3; do
        port=$((8005 + i))  # 8006, 8007, 8008
        if curl -s "http://localhost:$port/health" > /dev/null 2>&1; then
            node_info=$(curl -s "http://localhost:$port/api/v1/federated/node/info" 2>/dev/null || echo "{}")
            node_id=$(echo "$node_info" | grep -o '"node_id":"[^"]*"' | cut -d'"' -f4)
            echo -e "  Node $i (port $port): ${GREEN}运行中${NC} [ID: ${node_id:-unknown}]"
        else
            echo -e "  Node $i (port $port): ${RED}未运行${NC}"
        fi
    done
    echo ""
}

# 查看日志
view_logs() {
    local node=$1
    if [ -n "$node" ]; then
        docker logs -f "ai-trace-node$node"
    else
        docker-compose -f docker-compose.multi-node.yml logs -f node1 node2 node3
    fi
}

# 测试联邦共识
test_federation() {
    log_info "测试联邦共识..."

    # 1. 创建测试事件
    log_info "1. 创建测试事件..."
    trace_id="test_trace_$(date +%s)"

    curl -s -X POST "http://localhost:8006/api/v1/events/ingest" \
        -H "X-API-Key: node1-api-key-12345" \
        -H "Content-Type: application/json" \
        -d "{
            \"trace_id\": \"$trace_id\",
            \"events\": [
                {
                    \"event_type\": \"llm_request\",
                    \"payload\": {\"prompt\": \"Test prompt for federation\"}
                },
                {
                    \"event_type\": \"llm_response\",
                    \"payload\": {\"response\": \"Test response\"}
                }
            ]
        }" > /dev/null

    # 2. 生成存证
    log_info "2. 生成存证（触发联邦共识）..."
    result=$(curl -s -X POST "http://localhost:8006/api/v1/certs/commit" \
        -H "X-API-Key: node1-api-key-12345" \
        -H "Content-Type: application/json" \
        -d "{
            \"trace_id\": \"$trace_id\",
            \"evidence_level\": \"L2\"
        }")

    cert_id=$(echo "$result" | grep -o '"cert_id":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$cert_id" ]; then
        log_success "存证生成成功: $cert_id"

        # 3. 验证存证
        log_info "3. 验证存证..."
        verify_result=$(curl -s -X POST "http://localhost:8006/api/v1/certs/verify" \
            -H "X-API-Key: node1-api-key-12345" \
            -H "Content-Type: application/json" \
            -d "{\"cert_id\": \"$cert_id\"}")

        valid=$(echo "$verify_result" | grep -o '"valid":[^,]*' | cut -d':' -f2)

        if [ "$valid" = "true" ]; then
            log_success "存证验证通过"
        else
            log_warn "存证验证未通过"
        fi

        # 4. 检查联邦节点信息
        log_info "4. 检查联邦节点..."
        for port in 8006 8007 8008; do
            nodes=$(curl -s "http://localhost:$port/api/v1/federated/nodes" \
                -H "X-API-Key: node1-api-key-12345" 2>/dev/null || echo "{}")
            count=$(echo "$nodes" | grep -o '"count":[0-9]*' | cut -d':' -f2)
            echo "    Port $port: 已知 ${count:-0} 个联邦节点"
        done

    else
        log_error "存证生成失败"
        echo "$result"
    fi

    echo ""
    log_success "联邦测试完成"
}

# 清理数据
clean_data() {
    log_warn "即将删除所有数据..."
    read -p "确认删除? (y/N) " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        docker-compose -f docker-compose.multi-node.yml --profile monitoring down -v
        log_success "数据已清理"
    else
        log_info "取消操作"
    fi
}

# 主逻辑
main() {
    case "$1" in
        start)
            shift
            monitoring="false"
            while [ $# -gt 0 ]; do
                case "$1" in
                    --monitoring) monitoring="true" ;;
                esac
                shift
            done
            start_federation "$monitoring"
            ;;
        stop)
            stop_federation
            ;;
        status)
            check_status
            ;;
        logs)
            shift
            node=""
            while [ $# -gt 0 ]; do
                case "$1" in
                    --node) shift; node="$1" ;;
                esac
                shift
            done
            view_logs "$node"
            ;;
        test)
            test_federation
            ;;
        clean)
            clean_data
            ;;
        build)
            build_image
            ;;
        *)
            show_help
            ;;
    esac
}

main "$@"
