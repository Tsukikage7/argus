# Argus - 智能日志诊断与故障恢复

set dotenv-load := false

default:
    @just --list

# ─── 基础设施 ──────────────────────────────────────────────

# 启动基础设施（ES + Redis + PostgreSQL）
infra-up:
    docker compose up -d
    @echo "Waiting for Elasticsearch..."
    @for i in $(seq 1 30); do \
        curl -sf http://localhost:9200/_cluster/health > /dev/null 2>&1 && echo "ES ready!" && break; \
        sleep 2; \
    done

# 停止基础设施
infra-down:
    docker compose down

# 停止并清除数据卷
infra-clean:
    docker compose down -v

# 查看基础设施状态
infra-status:
    docker compose ps

# ─── 数据库迁移 ──────────────────────────────────────────

# 执行 PG 迁移（按序号顺序执行 migrations/*.sql）
migrate:
    @echo "Running migrations..."
    @for f in $(ls migrations/*_up_*.sql | sort); do \
        echo "  → $f"; \
        psql "postgres://argus:argus@localhost:5432/argus?sslmode=disable" -f "$f" 2>&1 || true; \
    done
    @echo "Migrations complete."

# ─── 构建 ──────────────────────────────────────────────────

# 构建 server
build-server:
    go build -o bin/argus-server ./cmd/server

# 构建 CLI
build-cli:
    go build -o bin/argus ./cmd/argus

# 构建全部
build: build-server build-cli

# ─── 运行 ──────────────────────────────────────────────────

# 启动 API Server
run-server:
    go run ./cmd/server

# 生成 mock 日志数据到 ES
mock-generate:
    go run ./cmd/argus mock generate

# CLI 诊断（用法: just diagnose "order-service 大量 504"）
diagnose msg:
    go run ./cmd/argus diagnose "{{msg}}"

# ─── 一键启动 ─────────────────────────────────────────────

# 一键启动：基础设施 + mock 数据 + API Server
up: infra-up
    @echo "Generating mock data..."
    go run ./cmd/argus mock generate
    @echo ""
    @echo "Starting API Server on :8080 ..."
    go run ./cmd/server

# 一键演示：基础设施 + mock 数据 + CLI 诊断
demo: infra-up
    @echo "Generating mock data..."
    go run ./cmd/argus mock generate
    @echo ""
    @echo "Running diagnosis..."
    go run ./cmd/argus diagnose "order-service 大量 504 超时"

# ─── 开发 ──────────────────────────────────────────────────

# 开发模式：air 热重载 Server
dev:
    air

# 运行测试
test:
    go test ./...

# 格式化代码
fmt:
    gofmt -w .
    goimports -w .

# lint 检查
lint:
    golangci-lint run ./...

# 检查编译
check:
    go build ./...

# 整理依赖
tidy:
    go mod tidy
