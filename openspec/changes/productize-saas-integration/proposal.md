# Proposal: Argus 产品化 SaaS 集成平台

## 概述

将 Argus 从内部 AIOps 诊断工具转型为面向外部客户的 SaaS 产品平台，提供多租户隔离、租户级 API Key 管理、开放 RESTful API、嵌入式 JS Widget，参考 AskReply.ai (AskX) 的产品化集成模式。

## 用户决策

| 决策项 | 用户选择 |
|--------|----------|
| 目标用户 | 外部客户 SaaS 平台 |
| MVP 范围 | API 接入 + 嵌入式 Widget |
| 认证方式 | API Key 分发（管理员创建租户并生成 Key） |

---

## 约束集合

### 硬约束 (Hard Constraints)

#### HC-1: 认证层缺乏租户概念
- **现状**: `middleware/auth.go` 仅支持静态 `[]string` API Key，认证结果只有"是否有效"
- **缺失**: 无 TenantID、Role、KeyID、过期时间、禁用状态、轮换状态
- **风险**: `api_key` query 参数会暴露密钥到日志、代理和浏览器历史
- **约束**: 必须重建认证中间件，返回租户上下文，废弃 query 参数认证

#### HC-2: 领域模型无租户标识
- **现状**: `task.Task`、`task.ReplaySession`、`task.TaskEvent` 不含 `TenantID`
- **影响面**: 改动从领域层级联到命令、查询、HTTP、SSE、仓储、导出
- **约束**: 必须在领域模型核心结构体中增加 `TenantID` 字段

#### HC-3: 数据存储无租户隔离
- **Redis**: key 前缀 `argus:task:`、`argus:replay:` 无 tenant scope
- **PostgreSQL**: `diagnosis_history` 表无 `tenant_id` 列和索引
- **ES**: `allIndex()` 默认跨所有 namespace 搜索，无租户过滤
- **约束**: 三层存储均需引入租户维度隔离

#### HC-4: SSE 端点无认证
- **现状**: `/api/v1/stream/{id}` 和 `/api/v1/replay/{id}/stream` 无认证
- **风险**: 知道 ID 即可订阅事件，SaaS 下直接形成数据泄漏面
- **约束**: SSE 必须支持 API Key 认证（query 参数或初始握手）

#### HC-5: 无 CORS 策略
- **现状**: 仅 SSE 硬编码 `Access-Control-Allow-Origin: *`
- **约束**: SaaS 需要可配置的 CORS 白名单（按租户），而非全局 `*`

#### HC-6: 前端 API 调用硬编码相对路径
- **现状**: `useApi.ts` 和 `useSSE.ts` 使用 `/api/v1/...` 相对路径
- **约束**: Widget 跨域嵌入必须支持动态配置 API Base URL

### 软约束 (Soft Constraints)

#### SC-1: 配置驱动的单例架构
- 全局配置 `config.yaml` 包含 APIKeys、Provider、ES、Redis 等
- 依赖注入为全局单例模式
- **影响**: 租户级配置需要"系统配置 + 租户动态配置"双层模型

#### SC-2: 缺乏开放 API 标准
- 无 OpenAPI/Swagger 文档
- 无分页、错误码规范、幂等键、速率限制
- 无 API 版本化 schema

#### SC-3: Webhook 仅支持企微
- 当前仅企微单一 webhook
- 缺乏通用租户级 webhook 注册、签名、重试机制

#### SC-4: 无管理员后台
- `AdminKey` 存在于配置但未进入路由或管理流程
- 租户 CRUD、API Key 管理、用量统计无承载点

### 依赖关系 (Dependencies)

| 依赖链 | 说明 |
|--------|------|
| 认证 → 全链路 | 中间件返回 Tenant 后，所有 Handler/Command/Query/Repo 需接收 tenant context |
| 配置 → DI | 配置结构不改，DI 无法支持租户级实例或策略 |
| 领域 → 持久化 | `task.Task` 字段变更直接影响 Redis JSON、PG 表结构、SSE 事件载荷 |
| ES → 应用服务 | `es_query`、`trace_analyze`、`logwatch`、`replay` 依赖 ES Client 的跨 namespace 查询 |
| 前端 → Vite | Widget 需独立打包，与管理控制台分离构建 |
| servex → 分发 | `servex v1.0.0` 需确认持续可拉取、可审计、可复现 |

### 风险 (Risks)

| 风险等级 | 风险 | 缓解策略 |
|----------|------|----------|
| **Critical** | 数据越权：任何有效 Key 可读取任意租户数据 | 全链路租户隔离 + 鉴权校验 |
| **Critical** | SSE 无认证泄漏 | SSE 端点增加 Key 认证 |
| **High** | ES 跨租户命中 | 租户专属索引前缀或强制 tenant filter |
| **High** | Widget 暴露公网，无限流 | Rate Limiter + 租户级配额 |
| **Medium** | 前端 Widget 包过大 | Tree-shaking + 独立构建目标 |
| **Medium** | CORS * 过于宽松 | 租户级域名白名单 |
| **Low** | 文档与 go.mod 依赖状态漂移 | 统一构建说明 |

---

## 成功判据 (Verifiable Success Criteria)

### SC-1: 租户隔离
- [ ] 管理员可创建租户并生成/轮换/禁用 API Key
- [ ] 认证后请求上下文稳定包含 `tenant_id` 和 `key_id`
- [ ] 跨租户读取同一 taskID/sessionID 返回 401/404

### SC-2: 数据隔离
- [ ] Redis key、PG 行、ES 索引至少有一层显式 tenant scope
- [ ] 历史列表、回放列表、搜索、导出仅返回当前租户数据
- [ ] ES 查询不再默认 `allIndex()` 扫全局数据

### SC-3: 开放 API
- [ ] 提供 OpenAPI 3.0 文档
- [ ] API 支持分页、标准错误码、速率限制
- [ ] Webhook 可按租户配置，支持签名和重试

### SC-4: 嵌入式 Widget
- [ ] `<script src="widget.js" data-api-key="xxx">` 即可在任意授权域加载诊断组件
- [ ] Widget 支持 CORS 跨域调用和 SSE 实时推送
- [ ] Widget JS 独立打包，首屏加载 < 200KB (gzip)

### SC-5: 管理控制台
- [ ] `/admin` 路径下可管理租户、API Key、查看用量统计
- [ ] 管理 API 与业务 API 使用不同认证策略（AdminKey vs TenantKey）

---

## MVP 范围定义

基于用户选择的 "API 接入 + 嵌入式 Widget" 范围：

### Phase 1: 多租户基础 (后端核心)
1. Tenant + API Key 领域模型（PG 持久化）
2. 租户感知认证中间件（替换静态 API Key）
3. Task/ReplaySession 增加 TenantID
4. Redis/PG/ES 存储层租户隔离
5. 管理 API（租户 CRUD、Key 生成/轮换/禁用）

### Phase 2: 开放 API (接入层)
1. CORS 中间件（租户级域名白名单）
2. Rate Limiter（租户级限流）
3. SSE 端点认证改造
4. OpenAPI 3.0 文档生成
5. Webhook 通用回调子系统

### Phase 3: 嵌入式 Widget (前端)
1. Widget 独立构建目标（Vite library mode）
2. `<script>` 标签加载 + `data-api-key` 初始化
3. 最小化诊断面板 UI（输入 + 步骤 + 结论）
4. 跨域 API/SSE 通信
5. CSS 隔离（Shadow DOM 或 scoped styles）

### Phase 4: 管理控制台 (前端)
1. 租户管理页面
2. API Key 管理页面
3. 集成文档 & Quick Start
4. 用量分析仪表盘

---

## 待确认问题

| # | 问题 | 影响范围 |
|---|------|----------|
| Q1 | 租户隔离采用逻辑隔离（共享 DB + tenant_id）还是物理隔离（独立 schema/索引）？ | Redis/PG/ES 全部存储设计 |
| Q2 | ES `namespace` 与 SaaS `tenant` 的关系？一租户多 namespace 还是 namespace = tenant？ | ES 索引策略 |
| Q3 | 租户是否允许自定义 LLM Provider/Model？ | 配置与 DI 架构 |
| Q4 | Widget Key 与 Admin Key 是否需要不同权限级别？ | API Key 模型设计 |
| Q5 | Widget 是否需要支持 CSS 主题自定义？ | Widget 打包策略 |
