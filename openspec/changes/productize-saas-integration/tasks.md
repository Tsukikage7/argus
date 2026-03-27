## Phase 0: Bug 修复（前置）— 已完成

### BF1: 诊断/回放界面模式隔离

- [x] 0.1 `frontend/src/App.vue`：三栏容器 `<div class="grid">` 添加 `:key="store.mode"` 强制模式切换时重挂载所有子组件
- [x] 0.2 `frontend/src/App.vue`：`handleDiagnose()` 和 `handleReplay()` 入口处，**先调用** `taskSSE.disconnect()` / `replaySSE.disconnect()`，再调用 `store.reset()`，消除 SSE 异步写入竞态
- [x] 0.3 `frontend/src/composables/useSSE.ts`：确保 `disconnect()` 方法存在且会调用 `eventSource.close()` + 清空内部 ref
- [x] 0.4 `frontend/src/App.vue`：优化条件渲染——诊断模式仅显示 InferencePanel + ConclusionCard，回放模式仅显示 ReplayPanel + ImpactReportPanel。当回放关联了诊断任务时，在 ImpactReportPanel 下方以折叠方式展示关联的推理步骤
- [x] 0.5 `frontend/src/store/useTaskStore.ts`：`reset()` 方法增加 `replaySessionId = null` + `impactReport = null` 清理

### BF2: 诊断搜索策略优化

- [x] 0.6 `internal/infrastructure/tools/es_query.go`：默认 `time_range` 从 `"last 15m"` 改为 `"last 1h"`
- [x] 0.7 `internal/infrastructure/es/query.go`：`QueryByKeyword` 新增 fallback 逻辑——`match_phrase` 无结果时降级为 `match`（分词匹配），返回结果并附加 `[fuzzy_match]` 标记
- [x] 0.8 `internal/domain/agent/agent.go`：system prompt 追加搜索重试指导
- [x] 0.9 `frontend/src/components/Control/DiagnoseInput.vue`：输入框下方增加可折叠的"高级选项"面板
- [x] 0.10 `internal/interfaces/http/handler/diagnose.go`：解析请求 body 中的 `context.time_range` 和 `context.namespaces` 字段

### BF3: 场景自动沉淀

- [x] 0.11 编写 `005_up_create_captured_scenarios.sql`
- [x] 0.12 创建 `internal/domain/task/scenario.go`：`CapturedScenario` 领域模型 + `ScenarioRepository` 接口
- [x] 0.13 创建 `internal/infrastructure/persistence/scenario_pg.go`：ScenarioRepository PG 实现
- [x] 0.14 `internal/application/command/diagnose.go`：诊断完成后自动创建 draft 场景
- [x] 0.15 `internal/interfaces/http/handler/replay.go`：合并预置场景 + 沉淀场景返回
- [x] 0.16 新增 API：`POST /api/v1/scenarios`
- [x] 0.17 新增 API：`PATCH /api/v1/scenarios/{id}/publish`
- [x] 0.18 `frontend/src/components/Conclusion/ConclusionCard.vue`：保存为场景按钮 + Modal
- [x] 0.19 `frontend/src/components/Control/ReplayInput.vue`：场景分组展示

---

## S1: 前端视觉升级（最高优先级）

- [x] 1.1 全局视觉规范更新：页面背景增加两个模糊渐变圆（Mesh Gradient：左上 Indigo、右下 Purple），全局 `@keyframes shimmer` 扫描光效，卡片统一 `bg-base-100/80 backdrop-blur border-base-300/50 shadow-xl` 玻璃态
- [x] 1.2 `StatsBar.vue` 美化：带微型 Sparkline SVG 的仪表盘卡片 + 数字 count-up 动画 + hover 微浮起效果
- [x] 1.3 `InferencePanel.vue` + `StepCard.vue` 美化：步骤卡片"重力下落"入场动画 + 左侧绿色发光时间线连线 + 处理中步骤扫描光效 + Think/Act/Observe 三色标记（Indigo/Amber/Emerald）
- [x] 1.4 `TopologyPanel.vue` 美化：X6 节点升级为 HTML 渲染 Vue 组件（圆角卡片 + 状态色圆点）+ 异常节点脉冲红色光晕 + 边流动粒子动画
- [x] 1.5 `ConclusionCard.vue` 美化：玻璃态卡片 + 置信度动态标题色（>90% 金色、60-90% 蓝色、<60% 橙色）+ 置信度环形进度条（CSS conic-gradient）+ 恢复建议按钮强力视觉引导
- [x] 1.6 `TimelinePanel.vue` 美化：垂直时间轴设计（左侧时间戳 + 中间圆点连线 + 右侧事件描述）+ 圆点颜色随级别变化 + 最新事件淡入动画
- [x] 1.7 `HistoryPanel.vue` 美化：从列表改为卡片式设计，每张卡片显示状态标签 + 根因摘要 + 时间 + hover 高亮效果

## S2: 前端功能增强（BF4 + BF5）

### 刷新后状态恢复

- [x] 2.1 `frontend/package.json`：安装 `pinia-plugin-persistedstate`
- [x] 2.2 `frontend/src/main.ts`：Pinia 注册 `piniaPluginPersistedstate` 插件
- [x] 2.3 `frontend/src/store/useTaskStore.ts`：配置持久化字段 `persist: { paths: ['mode', 'history', 'currentTaskId', 'replaySessionId'] }`
- [x] 2.4 `frontend/src/App.vue`：`onMounted` 增加恢复逻辑——解析 URL query 参数（`?taskId=xxx` / `?sessionId=xxx` / `?mode=replay`），有 taskId 则 GET 恢复，有 sessionId 则恢复 replay，running 状态重建 SSE
- [x] 2.5 `frontend/src/App.vue`：`handleDiagnose`/`handleReplay` 获得 ID 后 `replaceState` 更新 URL；`reset()` 后清除 URL 参数

### 日志分类展示

- [x] 2.6 后端：`internal/infrastructure/es/query.go` 新增 `QueryLogSummary` 方法（ES composite aggregation 按 namespace × 级别聚合）+ 新增 `GET /api/v1/logs/summary` API + 所有查询方法增加 `limit` 参数 + 新增 `GET /api/v1/logs` 条件查询 API
- [x] 2.7 `frontend/src/components/Monitor/LogExplorer.vue`（新组件）：顶部时间范围 + 级别 toggle，左侧 namespace 树状列表（带 ERROR badge），右侧日志详情列表（级别色标 + 时间 + message 截断展开）
- [x] 2.8 `frontend/src/App.vue`：右侧增加 Tab 切换（事件时间线 / 日志浏览器 / 历史记录），日志浏览器 Tab 渲染 LogExplorer

## S3: Widget 前端（Vite Library Mode + Shadow DOM）

- [x] 3.1 创建 `frontend/vite.config.widget.ts`：Vite library mode，入口 `src/widget/widget-main.ts`，ES + UMD 双格式，Vue 全量 bundle，`vue({ customElement: true })`，Terser minify
- [x] 3.2 创建 `frontend/src/widget/widget-main.ts`：读取 script 标签 `data-api-key` + `data-base-url`，`defineCustomElement` 注册 `<argus-widget>`，自动插入元素
- [x] 3.3 创建 `frontend/src/widget/ArgusWidget.ce.vue`（根组件）：Props apiKey/baseUrl，三状态 idle→diagnosing→completed/failed，Glassmorphism 容器，最大 400×600px
- [x] 3.4 创建 Widget 子组件：`WidgetHeader.vue`（标题 + 脉冲状态指示器）、`DiagnoseInput.vue`（玻璃态输入框 + 渐变按钮）、`InferenceStream.vue` + `MiniStepCard.vue`（垂直时间线 + 三色步骤卡片 + 扫描光效）、`ResultCard.vue`（根因 + 置信度 + 折叠建议）
- [x] 3.5 创建 `frontend/src/widget/composables/useWidgetApi.ts`：diagnose + getStreamToken + connectSSE + 错误处理（复用现有后端 API，演示时直连同源）
- [x] 3.6 Tailwind 4 CSS 配置：Widget 专用入口，精简扫描 widget 目录
- [x] 3.7 `package.json` 添加 `build:widget` 脚本 + 构建验证（gzip < 200KB）

## S4: 管理控制台前端（Vue Router + ApexCharts + Mock 数据）

- [x] 4.1 安装依赖：`vue-router@4` + `vue3-apexcharts` + `apexcharts` + `shiki`
- [x] 4.2 Vue Router 配置：`/admin` → `/admin/tenants`，`/admin/login`，`/admin/tenants`，`/admin/tenants/:id`，`/admin/integration`，路由守卫
- [x] 4.3 创建 `frontend/src/composables/useAdminApi.ts`：管理 API 调用层，内置 mock 数据开关——当后端不可用时返回预设的租户/Key/用量 mock 数据（含 800-1500ms 模拟延迟）
- [x] 4.4 `AdminLayout.vue`：左侧固定侧边栏（Logo + 菜单 + 退出）+ 顶栏面包屑 + 右侧 router-view，深色主题 Indigo/Purple 渐变
- [x] 4.5 `AdminLogin.vue`：居中卡片 AdminKey 输入 + 登录按钮，localStorage 存储 Key
- [x] 4.6 `TenantList.vue`：DaisyUI table + 搜索 + 状态筛选 + 创建 Modal + 删除确认 Modal，数据来自 useAdminApi（mock 或真实）
- [x] 4.7 `TenantDetail.vue`：信息卡片 + Tab（API Keys / 用量统计），Key 列表含创建/轮换/吊销 Modal，用量统计含 ApexCharts 时间序列图 + 环形进度条 + count-up 概览卡片
- [x] 4.8 `IntegrationGuide.vue`：步骤引导卡片 + Shiki 代码高亮（Widget 嵌入 / curl / JS fetch / Python）+ 一键复制

## S5: 后端核心（精简多租户基座）

### 数据库 + 领域模型

- [x] 5.1 创建 `migrations/` 目录 + `just migrate` 任务 + 编写 `001_up_create_tenants.sql` + `002_up_create_tenant_api_keys.sql` + `003_up_add_tenant_id_to_history.sql`（合并原 1.1-1.4，跳过 down migration 和 stream_tokens 表）
- [x] 5.2 创建 `internal/domain/tenant/tenant.go`：Tenant + APIKey 实体 + NewAPIKey 工厂方法（`arg_{slug}_{rand32}`）+ Repository 接口 + Principal 值对象 + context accessor（合并原 2.1-2.3）
- [x] 5.3 实现 `internal/infrastructure/persistence/tenant_pg.go` + `apikey_pg.go`：TenantRepository + APIKeyRepository PG 实现（Create/GetBySlug/List/GetByPrefix，跳过 Rotate/SoftDelete 复杂逻辑）（合并原 4.1-4.2）

### 认证 + 路由

- [x] 5.4 实现 `internal/interfaces/http/middleware/tenant_auth.go`：从 Bearer 提取 key → 解析 prefix → 查 PG GetByPrefix → SHA-256 比对 → 注入 Principal（简化版：跳过 Redis 缓存和 last_used_at 更新）（合并原 3.1+3.4）
- [x] 5.5 实现 AdminAuthMiddleware：bootstrap AdminKey 列表匹配 + Principal 注入（原 3.2）
- [x] 5.6 `cmd/server/main.go` 路由重构：`/admin/v1/*` 挂 AdminAuth，`/api/v1/*` 挂 TenantAuth，`multi_tenant.enabled` 开关兼容旧模式（合并原 7.1+15.3+15.4）

### 管理 API（最小可演示）

- [x] 5.7 实现 `admin_tenant.go`：`POST /admin/v1/tenants` + `GET /admin/v1/tenants` + `GET /admin/v1/tenants/{id}`（跳过 DELETE）（原 7.2 简化）
- [x] 5.8 实现 `admin_apikey.go`：`POST /admin/v1/tenants/{id}/keys` + `GET /admin/v1/tenants/{id}/keys`（跳过 rotate/revoke）（原 7.3 简化）

### 配置扩展

- [x] 5.9 `config.yaml` + `config.go`：增加 `multi_tenant` 配置块（enabled + bootstrap_admin_keys）（合并原 15.1+15.3）

### 标准化响应

- [x] 5.10 创建 `internal/interfaces/http/response.go`：统一 `WriteError`/`WriteJSON` + 错误码常量，核心 Handler 替换 `http.Error`（合并原 12.1+12.2，跳过分页）

### CORS（最小可用）

- [x] 5.11 实现 `internal/interfaces/http/middleware/cors.go`：简化版——从配置读取静态 allowed_origins 列表，OPTIONS 预检处理，移除 SSE 硬编码 `*`（合并原 10.1+10.4，跳过动态租户 CORS）

## S6: OpenAPI 文档

- [ ] 6.1 编写 `openapi/argus-api-v1.yaml`：OpenAPI 3.0 spec 覆盖所有 `/api/v1/*` + `/admin/v1/*` 端点，含 SecuritySchemes + 请求/响应 schema + 错误 schema
- [ ] 6.2 实现 `GET /api/v1/openapi.json` 端点（go:embed 静态文件服务）
