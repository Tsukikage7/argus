# Argus — UCloud 智能日志诊断与故障恢复

基于 AI Agent 的智能运维诊断与自愈平台。通过 ReAct 推理循环自动分析 Elasticsearch 中的微服务日志与分布式链路，定位故障根因并执行恢复操作。

> A-效能革新方向 · AI Agent 竞赛参赛项目

## 产品形态

| 端 | 说明 |
|---|------|
| **CLI** | `argus diagnose "order-service 504 超时"` |
| **Web UI** | 诊断面板，实时展示 AI 思考链 + 诊断结论，支持亮色/暗色主题 |
| **企业微信 Bot** | 告警推送 + 对话式诊断（详见下方接入说明） |

## 架构

```
CLI / Web UI / 企微 Bot
        │
   API Server (net/http, :9999)
        │
   Application (CQRS)
        │
   Domain (ReAct Agent)
   Think → Act → Observe
        │
   Infrastructure
   ├── LLM (DashScope / OpenAI 兼容)
   ├── Elasticsearch (日志 + 链路追踪)
   ├── Redis (任务状态)
   └── PostgreSQL (诊断历史)
```

**技术栈**: Go · servex · Elasticsearch 8 · Redis · PostgreSQL · DashScope (qwen-plus) · OpenTelemetry 日志格式

## 快速开始

### 前置依赖

- Go 1.25+
- Docker & Docker Compose
- [just](https://github.com/casey/just)
- [air](https://github.com/air-verse/air)（可选，开发热重载）

### 一键演示

```bash
just demo
```

自动完成：启动 ES/Redis/PG → 生成 mock 日志 → CLI 诊断 `"order-service 大量 504 超时"`

### 启动 Web 服务

```bash
just up
```

浏览器打开 http://localhost:9999，支持亮色/暗色主题切换。

### 分步操作

```bash
# 1. 启动基础设施
just infra-up

# 2. 生成 mock 日志
just mock-generate

# 3. 启动 API Server
just run-server

# 4. CLI 诊断
just diagnose "payment-service 连接池耗尽"
```

## Mock 服务拓扑

模拟 6 个微服务的电商系统，日志采用 OpenTelemetry 格式写入 ES：

```
                    ┌─────────────┐
         ┌────────→ │ user-service │
         │          └─────────────┘
┌────────┴───┐      ┌───────────────┐      ┌──────────────────┐
│  gateway   │─────→│ order-service │─────→│ payment-service  │
└────────┬───┘      └───────┬───────┘      └──────────────────┘
         │                  │
         │          ┌───────▼─────────┐    ┌───────────────────┐
         └────────→ │inventory-service│    │notification-service│
                    └─────────────────┘    └───────────────────┘
```

### 预置故障场景

| 场景 | 现象 | 根因 |
|------|------|------|
| payment-db-pool-exhausted | order-service 504 超时 | payment-service 慢查询导致数据库连接池耗尽 |
| inventory-oom | 库存扣减失败 | inventory-service 内存泄漏被 OOM kill |
| gateway-disk-full | 间歇性 502 | gateway 日志写满磁盘 |

### ES 索引结构

- `argus-logs-{service}-{date}` — 每个服务按天分索引
- `argus-traces-{date}` — 链路追踪汇总

## Agent 诊断流程

以 "order-service 504 超时" 为例，Agent 的 ReAct 推理过程：

```
Step 1  Think:  需要查 order-service 的错误日志
        Action: es_query_logs(service="order-service", severity="ERROR")
        Observe: "timeout calling payment-service, trace_id=4bf92f..."

Step 2  Think:  用 trace_id 追踪完整链路
        Action: trace_analyze(trace_id="4bf92f...")
        Observe: gateway(30s) → order(30s) → payment(TIMEOUT 30s)

Step 3  Think:  问题在 payment-service，查其错误日志
        Action: es_query_logs(service="payment-service", severity="ERROR")
        Observe: "pool exhausted, active=50, idle=0" × 200+

Step 4  Think:  连接池满，查慢查询
        Action: es_query_logs(service="payment-service", keyword="slow query")
        Observe: "SELECT ... duration=45s"

结论:   根因: payment-service MySQL 连接池耗尽（慢查询阻塞）
        置信度: 0.98
        建议: 重启服务 + 通知 DBA + 扩容连接池
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/diagnose` | 触发诊断（异步），返回 task_id |
| POST | `/api/v1/events` | 接收告警 Webhook（Prometheus / 自定义） |
| GET | `/api/v1/tasks/{id}` | 查询任务结果 |
| GET | `/api/v1/tasks` | 诊断历史列表 |
| GET | `/api/v1/stream/{id}` | SSE 实时推送诊断过程 |

认证：`Authorization: Bearer argus-demo-key`

```bash
# 触发诊断
curl -X POST http://localhost:9999/api/v1/diagnose \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer argus-demo-key" \
  -d '{"input": "order-service 大量 504 超时"}'

# 查询结果
curl http://localhost:9999/api/v1/tasks/<task_id> \
  -H "Authorization: Bearer argus-demo-key"
```

## Agent Tools

| Tool | 功能 |
|------|------|
| `es_query_logs` | 按服务名、日志级别、时间范围、关键词查询 ES 日志 |
| `trace_analyze` | 通过 trace_id 获取完整调用链路及各 span 耗时 |
| `exec_command` | 在目标机器执行运维命令（MVP 阶段模拟执行） |
| `send_notification` | 发送企微通知给 DBA / 运维 / 开发 |

## 企业微信 Bot 接入

### 方式一：群机器人 Webhook（推送诊断报告）

1. 企微群聊 → 群机器人 → 添加 → 复制 Webhook URL
2. 配置 `configs/config.yaml`：

```yaml
wechat:
  webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=你的key"
```

3. Agent 诊断完成后自动推送 Markdown 报告到群，由 `send_notification` Tool 触发。

### 方式二：企微自建应用（双向交互，@Bot 触发诊断）

1. 企微管理后台 → 应用管理 → 自建应用，获取 `corp_id` / `agent_id` / `secret`
2. 设置 API 接收消息 → 回调 URL：`https://你的域名/api/v1/wechat/callback`
3. 配置：

```yaml
wechat:
  corp_id: "ww1234567890"
  agent_id: 1000002
  secret: "应用Secret"
  token: "回调Token"
  encoding_aes_key: "回调EncodingAESKey"
```

4. 回调接口接收用户消息 → 触发诊断 → 完成后主动推送结果

## 目录结构

```
argus/
├── cmd/
│   ├── server/main.go            # API Server 入口
│   └── argus/main.go             # CLI 入口
├── internal/
│   ├── domain/
│   │   ├── agent/                # ReAct Agent 核心（推理循环 + 诊断解析）
│   │   ├── tool/                 # Tool 接口 + Registry
│   │   └── task/                 # 任务 & 诊断结果模型
│   ├── application/
│   │   ├── command/              # 诊断、恢复、告警事件命令
│   │   └── query/                # 任务状态、历史查询
│   ├── infrastructure/
│   │   ├── llm/                  # LLM 多 Provider 路由（DashScope / OpenAI）
│   │   ├── es/                   # Elasticsearch 客户端 + 日志/链路查询
│   │   ├── tools/                # 4 个 Tool 实现
│   │   ├── mock/                 # Mock 数据生成（3 故障场景 × 6 服务）
│   │   ├── persistence/          # Redis 任务状态 + PostgreSQL 诊断历史
│   │   └── wechat/               # 企微 Bot + Markdown 卡片
│   └── interfaces/
│       ├── config/               # 配置结构
│       └── http/                 # Handler（诊断/事件/任务/SSE）+ 认证中间件
├── web/index.html                # 诊断面板（亮色/暗色主题）
├── configs/config.yaml
├── docker-compose.yml            # ES + Redis + PostgreSQL
├── .air.toml                     # 热重载配置
└── justfile
```

## 开发

```bash
just dev          # air 热重载
just check        # 编译检查
just test         # 运行测试
just fmt          # 格式化
just lint         # lint 检查
just tidy         # 整理依赖
just infra-clean  # 清除数据重来
```

## 配置

编辑 `configs/config.yaml`：

| 配置项 | 说明 |
|--------|------|
| `providers` | LLM 提供商（DashScope / OpenAI 兼容接口） |
| `agent` | max_steps、置信度阈值、超时时间 |
| `elasticsearch` | ES 地址、索引前缀 |
| `redis` / `postgres` | 存储连接 |
| `wechat` | 企微 Webhook / 应用配置 |

## 分工建议（4 人）

| 角色 | 负责模块 | 关键产出 |
|------|---------|---------|
| P1 Agent 核心 | domain/agent + application/command + infrastructure/llm | ReAct 循环、Prompt 工程、function calling |
| P2 ES & Tools | infrastructure/es + infrastructure/tools + infrastructure/mock | ES 客户端、Tool 实现、Mock 数据 |
| P3 接入层 | cmd/ + interfaces/http + infrastructure/wechat | API Server、CLI、SSE、企微 Bot |
| P4 前端 & 演示 | web/ + docker-compose + 演示脚本 | 诊断面板、Demo 环境、PPT |
