# AI-Trace Makefile
# 版本: 0.1

.PHONY: all build run test clean docker-build docker-up docker-down help

# 默认目标
all: build

# 变量
GO := go
SERVER_DIR := server
CONSOLE_DIR := console
BINARY_NAME := ai-trace-server
DOCKER_COMPOSE := docker-compose

# ==================== 构建 ====================

# 构建服务端
build:
	@echo "Building server..."
	cd $(SERVER_DIR) && $(GO) build -o ../bin/$(BINARY_NAME) ./cmd/ai-trace-server

# 构建并安装依赖
build-deps:
	@echo "Installing dependencies..."
	cd $(SERVER_DIR) && $(GO) mod download
	cd $(SERVER_DIR) && $(GO) mod tidy

# 构建前端
build-console:
	@echo "Building console..."
	cd $(CONSOLE_DIR) && npm install && npm run build

# ==================== 运行 ====================

# 运行服务端（开发模式）
run:
	@echo "Running server..."
	cd $(SERVER_DIR) && $(GO) run ./cmd/ai-trace-server

# 运行带热重载
run-dev:
	@echo "Running server with hot reload..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	cd $(SERVER_DIR) && air

# 运行前端开发服务器
run-console:
	@echo "Running console..."
	cd $(CONSOLE_DIR) && npm run dev

# ==================== 测试 ====================

# 运行测试
test:
	@echo "Running tests..."
	cd $(SERVER_DIR) && $(GO) test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "Running tests with coverage..."
	cd $(SERVER_DIR) && $(GO) test -coverprofile=coverage.out ./...
	cd $(SERVER_DIR) && $(GO) tool cover -html=coverage.out -o coverage.html

# 运行基准测试
bench:
	@echo "Running benchmarks..."
	cd $(SERVER_DIR) && $(GO) test -bench=. ./...

# ==================== Docker ====================

# 构建Docker镜像
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

# 启动所有服务
docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d

# 启动并查看日志
docker-up-logs:
	@echo "Starting services with logs..."
	$(DOCKER_COMPOSE) up

# 停止所有服务
docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

# 停止并删除数据
docker-clean:
	@echo "Cleaning up..."
	$(DOCKER_COMPOSE) down -v

# 查看日志
docker-logs:
	$(DOCKER_COMPOSE) logs -f

# 查看服务状态
docker-ps:
	$(DOCKER_COMPOSE) ps

# ==================== 数据库 ====================

# 初始化数据库
db-init:
	@echo "Initializing database..."
	docker exec -i ai-trace-postgres psql -U postgres -d ai_trace < scripts/init.sql

# 连接数据库
db-shell:
	docker exec -it ai-trace-postgres psql -U postgres -d ai_trace

# ==================== 代码质量 ====================

# 格式化代码
fmt:
	@echo "Formatting code..."
	cd $(SERVER_DIR) && $(GO) fmt ./...

# 代码检查
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cd $(SERVER_DIR) && golangci-lint run

# 静态分析
vet:
	@echo "Running go vet..."
	cd $(SERVER_DIR) && $(GO) vet ./...

# ==================== 清理 ====================

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf $(SERVER_DIR)/coverage.out
	rm -rf $(SERVER_DIR)/coverage.html
	rm -rf $(CONSOLE_DIR)/dist
	rm -rf $(CONSOLE_DIR)/node_modules

# ==================== 工具 ====================

# 生成API文档
docs:
	@echo "Generating API docs..."
	@which swag > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest
	cd $(SERVER_DIR) && swag init -g cmd/ai-trace-server/main.go

# 安装开发工具
tools:
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# ==================== 帮助 ====================

help:
	@echo "AI-Trace Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build the server binary"
	@echo "  make run            - Run the server"
	@echo "  make run-dev        - Run with hot reload"
	@echo "  make test           - Run tests"
	@echo "  make docker-up      - Start all services"
	@echo "  make docker-down    - Stop all services"
	@echo "  make docker-logs    - View service logs"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make help           - Show this help"
