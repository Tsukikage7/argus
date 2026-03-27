# Argus -- UCloud 智能日志诊断与故障恢复平台

## 项目愿景

Argus 是一个基于 AI Agent 的智能运维诊断与自愈平台（AIOps）。通过 ReAct（Reasoning + Acting）推理循环，自动分析 Elasticsearch 中的微服务日志与分布式链路追踪数据，定位故障根因并执行恢复操作。

项目定位：UCloud A-效能革新方向 AI Agent 竞赛参赛项目。

## 架构总览

- **语言**: Go 1.25+（纯 Go 单体仓库）
- **架构风格**: 洋葱架构（DDD 分层）+ CQRS（命令查询职责分离）
- **核心模式**: ReAct Agent（Think -> Act -> Observe 循环）+ LLM Function Calling
- **产品形态**: CLI + Web UI + 企业微信 Bot（三端统一后端）
- **依赖框架**: servex（自研微服务框架，本地 replace 引用）

### 分层架构

```
interfaces/       -- 对外适配层（HTTP Handler、CLI、配置解析）
application/      -- 应用层（CQRS: Command + Query Handler）
domain/           -- 领域层（Agent 核心、Task 模型、Tool 抽象）
infrastructure/   -- 基础设施层（LLM、ES、Redis、PG、Mock、企微）
```

### 外部依赖

| 组件 | 用途 | 版本 |
|------|------|------|
| Elasticsearch | 日志存储 + 链路追踪查询 | 8.13.0 |
| Redis | 任务状态缓存（TTL 24h） | 7 |
| PostgreSQL | 诊断历史持久化 | 17 |
| DashScope (qwen-plus) | LLM 推理（OpenAI 兼容接口） | - |
| servex | 自研框架（配置管理、日志、缓存、AI 客户端） | v1.0.0 (local replace) |

## 模块结构图

```mermaid
graph TD
    ROOT["Argus (根)"] --> CMD["cmd/"]
    CMD --> CLI["argus (CLI 入口)"]
    CMD --> SRV["server (API Server 入口)"]

    ROOT --> INTERNAL["internal/"]
    INTERNAL --> DOMAIN["domain/"]
    DOMAIN --> AGENT["agent (ReAct Agent 核心)"]
    DOMAIN --> TASK["task (任务与诊断模型)"]
    DOMAIN --> TOOL["tool (Tool 接口与注册)"]

    INTERNAL --> APP["application/"]
    APP --> APPCMD["command (诊断/恢复/告警/回放命令)"]
    APP --> APPQ["query (任务状态/历史查询)"]

    INTERNAL --> INFRA["infrastructure/"]
    INFRA --> LLM["llm (LLM 路由适配)"]
    INFRA --> ES["es (ES 客户端)"]
    INFRA --> TOOLS_IMPL["tools (4 个 Tool 实现)"]
    INFRA --> MOCK["mock (Mock 数据生成 + 回放引擎)"]
    INFRA --> PERSIST["persistence (Redis + PG 存储)"]
    INFRA --> WECHAT["wechat (企微 Bot)"]

    INTERNAL --> IFACE["interfaces/"]
    IFACE --> CFG["config (配置结构体)"]
    IFACE --> HTTP["http/ (Handler + Middleware)"]

    ROOT --> WEB["web/ (前端面板)"]

    click AGENT "./internal/domain/agent/CLAUDE.md" "查看 agent 模块文档"
    click MOCK "./internal/infrastructure/mock/CLAUDE.md" "查看 mock 模块文档"
```

## 模块索引

| 模块路径 | 职责 | 语言 | 入口文件 | 测试 |
|----------|------|------|----------|------|
| `cmd/server` | API Server 启动入口，完成依赖注入与路由注册 | Go | `main.go` | 无 |
| `cmd/argus` | CLI 入口（diagnose / mock / scenarios / replay） | Go | `main.go` | 无 |
| `internal/domain/agent` | ReAct Agent 核心推理循环 + 诊断结果解析 + 恢复验证 | Go | `agent.go` | 无 |
| `internal/domain/task` | 诊断任务、回放会话、影响面报告等领域模型 | Go | `task.go` | 无 |
| `internal/domain/tool` | Tool 接口定义 + Registry + OpenAI Function Calling 适配 | Go | `tool.go` | 无 |
| `internal/application/command` | CQRS 命令处理：诊断、告警、恢复、回放 | Go | `diagnose.go` | 无 |
| `internal/application/query` | CQRS 查询处理：任务状态、诊断历史 | Go | `task.go` | 无 |
| `internal/infrastructure/llm` | servex/ai 适配层，多 Provider 路由 | Go | `router.go` | 无 |
| `internal/infrastructure/es` | ES 8 客户端封装：日志查询、链路追踪、批量写入、回放聚合 | Go | `client.go` | 无 |
| `internal/infrastructure/tools` | 4 个 Agent Tool 实现（es_query_logs / trace_analyze / exec_command / send_notification） | Go | `es_query.go` | 无 |
| `internal/infrastructure/mock` | Mock 数据生成器 + 3 个故障场景 + 6 服务拓扑 + 回放引擎 | Go | `generator.go` | 无 |
| `internal/infrastructure/persistence` | Redis（任务/回放状态）+ PostgreSQL（诊断历史） | Go | `task_redis.go` | 无 |
| `internal/infrastructure/wechat` | 企微 Bot Webhook + Markdown 卡片格式化 | Go | `bot.go` | 无 |
| `internal/interfaces/config` | 配置结构体定义（App/Provider/Agent/ES/Redis/PG/WeChat/Mock/Replay） | Go | `config.go` | 无 |
| `internal/interfaces/http` | HTTP Handler（诊断/事件/任务/SSE/回放）+ APIKey 中间件 | Go | `handler/diagnose.go` | 无 |
| `web` | 单文件前端诊断面板（DaisyUI + TailwindCSS + SSE + Markdown 渲染） | HTML/JS | `index.html` | 无 |

## 运行与开发

### 前置依赖

- Go 1.25+
- Docker & Docker Compose
- [just](https://github.com/casey/just)（任务运行器）
- [air](https://github.com/air-verse/air)（开发热重载，可选）
- servex 框架需本地存在于 `/Users/tsukikage/work/servex`（go.mod replace 指令）

### 常用命令（justfile）

```bash
just infra-up        # 启动 ES + Redis + PG
just mock-generate   # 生成 mock 日志到 ES
just run-server      # 启动 API Server (:9999)
just up              # 一键启动（基础设施 + mock + server）
just demo            # 一键演示（基础设施 + mock + CLI 诊断）
just dev             # air 热重载开发
just build           # 构建 server + CLI
just test            # 运行测试
just fmt             # gofmt + goimports
just lint            # golangci-lint
just check           # 编译检查
just tidy            # go mod tidy
just infra-clean     # 停止并清除数据卷
```

### 配置

复制 `configs/config.example.yaml` 为 `configs/config.yaml`，核心配置项：

| 配置块 | 说明 |
|--------|------|
| `app` | 服务名称、监听地址 (`:9999`)、API Keys |
| `providers` | LLM Provider（DashScope，OpenAI 兼容），支持多模型路由 |
| `agent` | max_steps=15、置信度阈值（auto_recover=0.8, confirm=0.5）、超时 5m |
| `elasticsearch` | 地址 + 索引前缀 `argus` |
| `redis` | 地址 (`localhost:6379`) |
| `postgres` | DSN (`postgres://argus:argus@localhost:5432/argus`) |
| `wechat` | 企微 corp_id / agent_id / secret |
| `mock` | 6 个模拟微服务名称 |
| `replay` | 回放功能开关、默认故障强度/流量倍率、最大时长 |

### API 端点

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/api/v1/diagnose` | 触发异步诊断，返回 task_id | Bearer |
| POST | `/api/v1/events` | 接收告警 Webhook | Bearer |
| GET | `/api/v1/tasks/{id}` | 查询任务结果 | Bearer |
| GET | `/api/v1/tasks` | 诊断历史列表 | Bearer |
| GET | `/api/v1/stream/{id}` | SSE 实时推送诊断过程 | 无 |
| GET | `/api/v1/scenarios` | 列出可用故障场景 | Bearer |
| POST | `/api/v1/replay` | 创建回放会话 | Bearer |
| GET | `/api/v1/replay/{id}` | 查询回放会话 | Bearer |
| GET | `/api/v1/replay/{id}/stream` | 回放 SSE 流 | 无 |
| GET | `/health` | 健康检查 | 无 |

认证方式：`Authorization: Bearer argus-demo-key`（或 `?api_key=argus-demo-key`）

### Docker Compose 基础设施

- Elasticsearch 8.13.0（单节点，无安全认证）
- Redis 7 Alpine
- PostgreSQL 17 Alpine（用户/密码/库: argus）

## 测试策略

> 当前项目尚无任何测试文件（`*_test.go` 为 0）。这是最主要的质量缺口。

建议优先补充：
1. `internal/domain/agent` -- Agent 推理循环单元测试（mock LLM + mock Tools）
2. `internal/domain/tool` -- Registry 注册/查找测试
3. `internal/infrastructure/es` -- 查询构建器测试（`buildLogQuery` 等纯函数）
4. `internal/application/command` -- DiagnoseHandler 集成测试
5. `internal/infrastructure/mock/scenarios.go` -- 场景数据生成正确性

## 编码规范

- Go 标准项目布局：`cmd/` + `internal/`
- 注释语言：中文
- 错误处理：`fmt.Errorf("context: %w", err)` 包装链
- 依赖注入：手动构造函数注入（无 DI 框架）
- HTTP 路由：Go 1.22+ `http.NewServeMux` 原生路由模式（`GET /api/v1/tasks/{id}`）
- 中间件模式：`func(http.Handler) http.Handler`
- Tool 实现需满足 `tool.Tool` 接口，并用 `var _ tool.Tool = (*)` 编译期检查
- air 热重载配置排除 `_test.go` 文件和 `tmp/bin/web/vendor` 目录

## AI 使用指引

### 理解 ReAct Agent 循环

核心循环在 `internal/domain/agent/agent.go` 的 `Run()` 方法中：
1. 构建初始消息（system prompt + user input）
2. 循环调用 LLM（最多 MaxSteps 步）
3. 如果 LLM 返回 tool_calls -> 执行工具 -> 将结果作为 tool message 追加
4. 如果 LLM 返回纯文本 -> 解析 JSON 诊断结论 -> 结束
5. 每步通过 EventHandler 回调推送 SSE 事件

### 添加新 Tool 的步骤

1. 在 `internal/infrastructure/tools/` 创建新文件
2. 实现 `tool.Tool` 接口（Name/Description/Parameters/Execute）
3. Parameters 返回 JSON Schema（用于 LLM function calling）
4. 在 `cmd/server/main.go` 和 `cmd/argus/main.go` 中注册到 `toolRegistry`

### 添加新故障场景的步骤

1. 在 `internal/infrastructure/mock/scenarios.go` 中添加新的 `Scenario` 工厂函数
2. 使用 `makeLog()` 和 `makeTrace()` 辅助函数生成日志和链路数据
3. 将新场景加入 `AllScenarios()` 返回列表
4. 场景会自动在 mock generate、replay、scenarios 命令中可用

### 关键依赖说明

- `servex` 是本地 replace 依赖，提供：配置管理（Viper 封装）、日志（Zap）、缓存（Redis）、AI 客户端（OpenAI 兼容）
- LLM 适配在 `internal/infrastructure/llm/router.go`，将 `servex/ai` 接口适配为 `agent.LLMClient`

## 变更记录 (Changelog)

| 时间 | 操作 | 说明 |
|------|------|------|
| 2026-03-18T00:09:25 | 初始生成 | 全仓扫描，生成根级 + 模块级 CLAUDE.md、.claude/index.json |
