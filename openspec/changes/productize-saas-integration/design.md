## Context

Argus 当前是一个内部 AIOps 诊断工具，所有数据（Task、ReplaySession、ES 日志）在全局命名空间中运行，认证仅靠配置文件中的静态 API Key 列表。本次变更将 Argus 转型为面向外部客户的 SaaS 产品平台，核心挑战是在不重写整体架构的前提下，将「租户 Principal」变为一等公民，贯穿全链路。

### 现状关键缺口

| 层级 | 现状 | 目标 |
|------|------|------|
| 认证 | 静态 `[]string` API Key，无租户/角色上下文 | Principal 注入（tenant_id + key_id + role） |
| 领域模型 | Task/ReplaySession/TaskEvent 无 tenant_id | 全部核心结构体增加 TenantID |
| Redis | `argus:task:{id}` 全局 key | `argus:tenant:{tid}:task:{id}` |
| PostgreSQL | diagnosis_history 无 tenant_id 列 | 增加列 + 组合索引 |
| ES | `{prefix}_{k8s_namespace}-*` 全局索引 | `argus-{tenant_id}-logs-{date}` |
| SSE | 无认证，CORS `*` | stream_token 认证 + 租户级 CORS 白名单 |
| 前端 | 单体 web/index.html | Widget (Shadow DOM) + 管理控制台 (Vue 3 SPA) |

## Goals / Non-Goals

**Goals:**
- 全链路多租户隔离（认证 → 领域 → 存储 → API → SSE）
- 双级权限模型（AdminKey 管理租户 CRUD，TenantKey 业务 API/Widget）
- 嵌入式 Widget（`<script>` 加载，Shadow DOM 隔离，< 200KB gzip）
- 管理控制台（租户/Key CRUD、用量统计）
- 开放 API 标准（OpenAPI 3.0、分页、错误码、Rate Limit）

**Non-Goals:**
- 物理租户隔离（独立 schema/数据库）— MVP 仅逻辑隔离
- 租户自定义 LLM Provider/Model — 共享系统配置
- Widget CSS 主题自定义 — 固定样式
- RBAC 细粒度权限 — 仅 AdminKey/TenantKey 双级
- 多语言国际化 — MVP 不涉及

## Decisions

### D1: 认证架构 — 中间件 + Context Principal 注入

**选择**: 认证中间件解析 API Key → 注入 `Principal` 到 `context.Value` → Handler/Command/Query 通过 accessor 读取。

**替代方案**:
- Handler 手动解析传参 — 不推荐，容易遗漏导出/SSE/recover 等边角入口
- JWT token — 过重，MVP 不需要短期签发/验证分离

**理由**: 最符合当前 net/http 中间件结构，改造面可控，鉴权逻辑集中化。

### D2: 路由分离 — /admin/v1/* + /api/v1/*

**选择**: 双路由树，AdminKey 认证器仅挂载 `/admin/v1/*`，TenantKey 认证器仅挂载 `/api/v1/*`。

**替代方案**:
- 同一路由树内用角色分支 — 中间件分支多，业务 API 容易混入 admin 权限例外

**理由**: 双级权限模型下路由分离最清晰，认证器完全独立。

### D3: API Key 存储 — PG 主存储 + Redis 缓存

**选择**: `tenants` + `tenant_api_keys` PG 表为 source of truth，Redis 按 key_prefix 缓存热数据，配置文件仅保留 bootstrap AdminKey。

**Tenant ID 设计**: 内部 UUID 主键（`tenants.id UUID PRIMARY KEY DEFAULT gen_random_uuid()`）+ 不可变 slug 字段（`tenants.slug VARCHAR(32) UNIQUE NOT NULL`）。ES 索引名、Redis key 前缀、API Key 前缀均使用 slug。slug 创建后不可修改，避免存储迁移。

**Key 格式**: `arg_{tenant_slug}_{random32}`，前缀 `arg_{tenant_slug}_` 用于快速索引。Hash 算法选择 SHA-256（非 bcrypt，因为认证路径需要快速比对，SHA-256 + 固定 salt 即可满足需求）。只存 hash，明文仅返回一次。

**认证缓存 Redis 结构**:
- Key: `argus:auth:key:{key_prefix}` （如 `argus:auth:key:arg_acme_`）
- Value: JSON `{"key_hash":"...","tenant_id":"uuid","tenant_slug":"acme","role":"tenant","status":"active","expires_at":"..."}`
- TTL: 5min（认证时先查缓存，miss 时回源 PG 并写缓存）
- 失效: Key 轮换/吊销时主动 DEL 缓存

**替代方案**:
- 配置文件存储 — 不支持动态 CRUD/轮换/审计
- Redis 主存储 — 缺乏强一致元数据和审计能力
- bcrypt 哈希 — 每次认证 100ms+ 延迟不可接受，SHA-256 < 1μs

### D4: ES 索引策略 — 按 Tenant Slug 分索引

**选择**: 索引模式 `argus-{tenant_slug}-logs-{yyyy.MM.dd}`，使用不可变 slug（非 UUID）作为索引名组成部分，保证可读性和稳定性。文档内保留 `kubernetes_namespace` 字段作为租户内服务过滤维度。

**命名规范**: 代码中统一使用 `tenant_id`（内部 UUID）、`tenant_slug`（外部可读标识）与 `kubernetes_namespace`（K8s 命名空间），三者语义严格分离。禁止再用 `namespace` 同时表示两种概念。

**容错**: 新租户无索引时返回空结果。Go 代码实现：
```go
o.Search.WithIgnoreUnavailable(true),
o.Search.WithAllowNoIndices(true),
```

**allIndex() 处置**: 将 `allIndex()` 标记为 `// Deprecated: internal use only`，仅允许 mock generate 和系统运维场景调用。业务 API 路径一律使用 `TenantIndex(slug string) string` 方法。

### D5: SSE 认证 — Stream Token 方案

**选择**: REST 接口用 Bearer TenantKey 调用获取短期 `stream_token`（TTL=5min，一次性使用，绑定 task_id + tenant_id），SSE 连接用 `?stream_token=xxx` 认证。

**替代方案**:
- 直接 query 参数传 TenantKey — 暴露密钥到日志/代理/浏览器历史
- fetch + ReadableStream — 兼容性不如 EventSource

### D6: Widget 架构 — Vite Library Mode + Shadow DOM + defineCustomElement

**选择**: Widget 独立 Vite 构建目标（UMD/ES），采用 Vue 3.5+ `defineCustomElement` 方式创建自定义元素 `<argus-widget>`。Shadow DOM (Open Mode) 隔离 CSS，Tailwind 4 CSS 通过 SFC 的 `<style>` 块编译后自动注入 Shadow Root。

**加载方式**: `<script src="widget.js" data-api-key="xxx" data-base-url="https://api.example.com">`

**Vue 全量 Bundle**: Widget 将 Vue 3 打包进自身（不 externalize），确保宿主页面无需预装 Vue。预估 gzip 体积：Vue runtime (~45KB) + Widget 逻辑 (~30KB) + Tailwind CSS (~25KB) = ~100KB，远低于 200KB 限制。

**Widget 入口逻辑**:
1. `widget-main.ts` 作为入口，script 加载时立即执行
2. 读取当前 `<script>` 标签的 `data-api-key` 和 `data-base-url` 属性
3. 在 script 标签后创建 `<argus-widget>` 自定义元素并传入属性
4. Vue Custom Element 自动创建 Shadow DOM 并挂载组件

**Widget 组件树**:
```
src/widget/
├── widget-main.ts               # 入口：注册自定义元素 + 自动挂载
├── ArgusWidget.ce.vue            # 根组件（.ce.vue 触发 Shadow DOM）
├── components/
│   ├── WidgetHeader.vue          # 标题栏 + 状态指示器
│   ├── DiagnoseInput.vue         # 玻璃态输入框 + 渐变发送按钮
│   ├── InferenceStream.vue       # 流式步骤展示 + 脉冲动画
│   │   └── MiniStepCard.vue      # 微型步骤卡片
│   └── ResultCard.vue            # 根因结论 + 置信度 + 折叠展开
├── composables/
│   └── useWidgetApi.ts           # API 通信 + stream_token + SSE
└── styles/
    └── widget.css                # Tailwind 4 入口（@import "tailwindcss"）
```

**UI 设计风格**: Glassmorphism（玻璃态），`backdrop-blur-md` + `bg-white/10` + `border-white/20`，三状态视图（输入态 / 推理态 / 结论态）流畅切换动画。

### D7: Key 轮换策略 — Grace Period 24h

**选择**: 轮换时旧 key 状态改为 `rotating`，grace period = 24h，期间新旧 key 均可认证。过期后旧 key 自动变为 `revoked`。

### D8: 租户删除策略 — 软删除 + 异步清理

**选择**: 标记 `deleted` → 立即拒绝新请求 → 异步清理 PG 行/Redis key/ES 索引 → 保留 tombstone 30 天。

### D9: Rate Limiting — Redis 滑动窗口（Lua 脚本）

**选择**: Redis Lua 脚本实现精确滑动窗口计数器（Sorted Set），默认 100 req/min。

**Lua 脚本逻辑**:
1. 使用 ZREMRANGEBYSCORE 移除窗口外的旧请求时间戳
2. 使用 ZCARD 获取当前窗口内的请求数
3. 若未超限，ZADD 当前时间戳
4. 设置 key TTL = window_seconds（自动清理无流量租户的 key）
5. 返回 `[当前计数, 限额, 窗口剩余秒数]`

**Redis Key 模式**: `argus:tenant:{slug}:ratelimit:{window_id}`
- `window_id` = 路由分组标识（如 `diagnose`、`replay`、`admin`）
- Sorted Set member = 请求时间戳（纳秒精度），score = 同值

**响应头注入**:
- `X-RateLimit-Limit: 100`
- `X-RateLimit-Remaining: 47`
- `X-RateLimit-Reset: 1711238400`（窗口重置 Unix 时间戳）
- 超限时: `429 Too Many Requests` + `Retry-After: 23`

**替代方案**:
- INCR + EXPIRE 固定窗口 — 窗口边界处允许 2 倍突发流量，不够精确
- 漏桶/令牌桶 — MVP 阶段过度设计

### D10: PG Migration — 版本化 SQL 文件

**选择**: `migrations/` 目录存放版本化 SQL migration 文件（`{序号}_{up|down}_{描述}.sql`），通过 `just migrate` 命令执行。每个 up 均有对应 down migration 支持回滚。

**迁移文件清单**:
- `001_up_create_tenants.sql` / `001_down_drop_tenants.sql`
- `002_up_create_tenant_api_keys.sql` / `002_down_drop_tenant_api_keys.sql`
- `003_up_add_tenant_id_to_history.sql` / `003_down_remove_tenant_id_from_history.sql`
- `004_up_create_stream_tokens.sql` / `004_down_drop_stream_tokens.sql`

**重要**: 当前 `history_pg.go` 中有启动时自动 `CREATE TABLE IF NOT EXISTS` 逻辑，需移除并迁移到 SQL 文件，避免与版本化 migration 冲突。

### D11: 管理控制台前端 — Vue 3 SPA + ApexCharts

**选择**: 管理控制台复用现有 frontend/ 项目，新增 `/admin/*` 路由分支。图表库选用 ApexCharts（~120KB gzip），与 Vue 3 和 Tailwind 适配良好。

**布局方案**: 左侧固定侧边栏（菜单项：租户管理、用量分析、集成指南）+ 顶栏（面包屑 + AdminKey 状态）+ 右侧内容区。

**路由结构**:
```
/admin                → 重定向到 /admin/tenants
/admin/tenants        → 租户列表页
/admin/tenants/:id    → 租户详情页（含 API Key 管理 + 用量统计）
/admin/integration    → 集成指南页
```

**认证**: 前端 localStorage 存储 AdminKey，所有 `/admin/v1/*` API 请求携带 `Authorization: Bearer {AdminKey}`。登录页为简单的 Key 输入表单。

### D12: 兼容模式开关

**选择**: `config.yaml` 中 `multi_tenant.enabled` 控制运行模式：
- `false`（默认）: 使用旧的静态 API Key 认证（`app.api_keys` 列表），所有功能无租户维度，保持现有行为
- `true`: 启用多租户模式，静态 API Key 仅作为 bootstrap AdminKey，业务 API 必须使用 PG 中的 TenantKey

**DI 分支**: `cmd/server/main.go` 初始化时根据 `multi_tenant.enabled` 选择创建：
- `false` → 旧 `APIKeyAuth` 中间件 + 无 tenant repo
- `true` → `TenantAuthMiddleware` + `AdminAuthMiddleware` + tenant/api-key repo + stream token store + rate limiter

### D13: SSE 重构 — 统一 SSE Writer

**选择**: 当前 stream.go、replay.go、live.go 三处 SSE 实现有重复的 header 设置、flush 逻辑和 CORS 设置。提取统一的 `SSEWriter` 辅助结构体，集中处理：
- Content-Type / Cache-Control / Connection 响应头
- CORS 头（从 CORS 中间件统一处理，SSE handler 不再硬编码）
- Flusher 获取和错误处理
- Stream token 认证校验

## Cross-Cutting Findings

| 类型 | 发现 | 影响 | 处置 |
|------|------|------|------|
| 重复逻辑 | SSE 三处重复设置 header/flush/loop | 租户化改造要改 3 次 | D13 统一 SSE Writer |
| 命名冲突 | `namespace` 同时表示 K8s 命名空间和业务过滤维度 | 引入 tenant 后三义混淆 | D4 严格分离命名 |
| 自动建表 | `history_pg.go` 启动时 `CREATE TABLE IF NOT EXISTS` | 与版本化 migration 冲突 | 移除自动建表，迁移到 SQL 文件 |
| 全链路传递 | 不仅 HTTP Handler，command/query/tool/logwatch/mock/replay 都需要 tenant 传递 | 遗漏任何一环即跨租户泄漏 | 编译期检查（接口方法签名强制 tenantID 参数） |

## PBT Properties (Property-Based Testing)

| 属性 | 不变量 | 伪造策略 |
|------|--------|---------|
| 租户隔离 | ∀ key ∈ TenantA, ∀ resource ∈ TenantB: access(key, resource) = 404 | 创建 2 租户，A 的 key 访问 B 的 task/replay/history/ES，全部断言 404 |
| 认证幂等性 | ∀ key, ∀ n: auth(key) × n = same Principal | 同一 key 并发 100 次认证，断言返回相同 tenant_id + key_id |
| Key 轮换 round-trip | create → rotate → grace_period_both_ok → expire → old_revoked | 时间推进测试：轮换后新旧均可认证 → 24h 后旧 key 返回 401 |
| ES 索引边界 | write(A, doc) ∧ search(B) → results ∩ A_docs = ∅ | 双租户各写一条日志，搜索 B 验证不包含 A 的数据 |
| Rate limit 单调性 | count(t) ≤ count(t+Δ) within window; count > limit → 429 | 发送 limit+1 请求，验证第 limit+1 次返回 429 |
| Stream token 单次性 | use(token) → use(token) = 401 | 使用 token 建立 SSE → 断开 → 再次使用同一 token 断言 401 |
| Widget bundle 边界 | gzip(widget.js + widget.css) < 200KB | 构建后检查文件大小 |
| Slug 不可变性 | ∀ tenant: slug_at_create = slug_at_any_future_time | 创建租户后尝试修改 slug，断言被拒绝 |

## Risks / Trade-offs

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| 全链路数据越权（10+ 端点裸 ID 读取） | Critical | 全链路 tenant_id 强制传递，禁止无 tenant 查询入口 |
| SSE 无认证泄漏 | Critical | stream_token 方案（D5） |
| ES allIndex() 跨租户查询 | High | 删除/内部化 allIndex()，强制 tenantID 参数 |
| mock/replay 系统无租户概念 | High | ReplaySession 固化 tenant_id，写入文档同步写 tenant_id |
| namespace 语义冲突（k8s vs tenant） | Medium | 统一命名规范（D4） |
| API Key 轮换期间并发失败 | Medium | 24h grace period（D7） |
| Redis key 迁移兼容 | Medium | 读路径兼容旧 key（先查新格式，miss 时查旧格式），写路径使用新格式 |
| Widget 首屏体积超标 | Low | Vue 全量 bundle + Tailwind 精简，预估 ~100KB gzip |
| PG 自动建表与 migration 冲突 | Medium | 移除 history_pg.go 自动建表，迁移到 SQL 文件 |

## Bug Fix Decisions (前置修复)

以下 5 个 bug 需在 SaaS 改造前修复，避免带病上线。

### BF1: 诊断/回放界面混乱 — 模式隔离 + SSE 竞态修复

**根因**: `useTaskStore` 单例共享 `steps/diagnosis/timeline`，切换模式时 `reset()` 与 SSE 异步写入存在竞态，组件缺少 `:key` 绑定不触发重挂载。

**修复**:
1. `App.vue` 三栏容器添加 `:key="store.mode"` 强制重挂载
2. `reset()` 方法中**先断开 SSE 连接**（`taskSSE.disconnect()` + `replaySSE.disconnect()`），再清空状态
3. 模式切换时增加 `nextTick` 防护，确保组件销毁完成后再渲染新模式
4. 条件渲染优化：诊断模式显示 InferencePanel + ConclusionCard，回放模式显示 ReplayPanel + ImpactReportPanel + 关联诊断步骤（分区展示）

### BF2: 诊断搜索找不到日志 — 扩大搜索策略 + 模糊匹配

**根因（后端）**:
1. `es_query_logs` 默认 `time_range="last 15m"` 太短，用户描述的故障可能发生在数小时前
2. `match_phrase` 是严格短语匹配，用户描述与实际日志用词不一致时匹配不上
3. Agent system prompt 未指导 LLM 在首次搜索无结果时扩大搜索范围

**修复**:
1. `es_query_logs` 默认 `time_range` 改为 `"last 1h"`
2. 新增 `match` 查询模式（非 phrase），对 keyword 搜索降低精确要求，优先 `match_phrase`，fallback 到 `match`
3. Agent system prompt 追加指导："如果首次搜索结果为空，使用更大的 time_range（last 6h 或 last 24h）重试；如果 keyword 搜索无结果，尝试拆解关键词或使用相关同义词"
4. 前端 DiagnoseInput 增加可折叠的高级选项面板（时间范围选择器 + namespace 多选）

### BF3: 场景自动沉淀 — 诊断结论 → CapturedScenario

**根因**: `AllScenarios()` 硬编码 3 个工厂函数，无持久化存储，无动态加载能力。

**修复**:
1. PG 新增 `captured_scenarios` 表：
   ```sql
   CREATE TABLE captured_scenarios (
     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     name VARCHAR(128) NOT NULL,
     description TEXT NOT NULL,
     source_task_id VARCHAR(64),
     root_cause TEXT,
     confidence DECIMAL(3,2),
     log_patterns JSONB NOT NULL DEFAULT '[]',
     affected_namespaces TEXT[] DEFAULT '{}',
     status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
     created_at TIMESTAMPTZ NOT NULL DEFAULT now()
   );
   ```
2. `DiagnoseHandler` 诊断完成后，若 `confidence >= 0.7`，自动创建 draft 场景
3. `log_patterns` 保存诊断过程中发现的关键日志特征（error message 模板 + namespace + keyword）
4. `GET /api/v1/scenarios` 返回预置场景 + 已 published 的沉淀场景
5. 前端 ConclusionCard 添加"保存为场景"按钮，ReplayInput 分 Preset/Captured 两类展示
6. 回放引擎支持从 CapturedScenario 的 log_patterns 重新生成类似模式的日志

### BF4: 刷新丢失状态 — URL 参数 + API 恢复 + 部分持久化

**根因**: Pinia 纯内存，URL 不携带任务 ID，刷新后无法恢复。

**修复**:
1. URL hash 方案：`/#/diagnose?taskId=xxx` 或 `/#/replay?sessionId=xxx`
2. `App.vue` 的 `onMounted` 检查 URL 参数：
   - 有 `taskId` → `GET /api/v1/tasks/{id}` → 恢复 steps + diagnosis + status
   - 有 `sessionId` → `GET /api/v1/replay/{id}` → 恢复 replay session + impact_report
3. `handleDiagnose` / `handleReplay` 成功获取 ID 后更新 URL 参数
4. `pinia-plugin-persistedstate` 持久化到 localStorage：`mode`、`history`（不持久化 steps/diagnosis，从 API 恢复更可靠）
5. 恢复时如果任务仍在 running，重新建立 SSE 连接

### BF5: 日志分类展示 — ES 聚合 API + 前端分组视图

**根因**:
1. 查询 `size=50` 硬编码，大量日志被截断
2. 缺少按维度聚合的查询 API
3. 前端只有平铺列表，无分类/过滤能力

**修复**:
1. ES 新增聚合查询方法 `QueryLogSummary(ctx, timeRange) → map[namespace]map[level]count`
2. 新增 API 端点 `GET /api/v1/logs/summary?time_range=last_1h` → 按 namespace × level 统计
3. ES 查询 `size` 参数化：默认 50，支持前端传入 `limit`（最大 500）
4. 前端新增 `LogExplorer` 组件：
   - 顶部：时间范围选择 + 级别多选筛选器（ERROR/WARN/INFO toggle）
   - 左侧：namespace 树状列表（DaisyUI Collapse），点击展开该 namespace 的日志
   - 右侧：日志详情列表（级别色标 + 时间 + message 截断 + 展开详情）
5. 日志查看面板作为 App.vue 新增的第四栏或抽屉面板

## Migration Plan

### 部署顺序

0. **Phase 0 Bug 修复**: 先完成 5 个 bug 修复（BF1-BF5），确保现有功能稳定
1. **Phase 1 数据库迁移**: 执行 PG migration（新增 tenants/tenant_api_keys/captured_scenarios 表，diagnosis_history 增加 tenant_id 列）
2. **Phase 1 后端**: 部署认证中间件 + 租户模型 + 存储层改造，旧 API Key 继续作为 bootstrap AdminKey
3. **Phase 2 后端**: CORS + Rate Limiter + SSE 认证 + OpenAPI
4. **Phase 3 前端**: Widget 独立构建 + 部署
5. **Phase 4 前端**: 管理控制台上线

### 回滚策略

- PG migration 均提供 down migration
- 认证中间件保留开关：`config.yaml` 中 `multi_tenant.enabled: false` 时退回静态 API Key 模式
- Redis 新旧 key 格式读路径兼容，回滚不丢数据

## Open Questions

> 当前无未解决问题。所有决策点已在规划阶段确认。
