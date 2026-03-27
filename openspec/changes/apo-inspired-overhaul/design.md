# Design: APO-Inspired Overhaul

## Multi-Model Analysis Summary

### Codex (Backend)
- 现有 ES 查询能力可直接支撑日志探索、链路追踪列表/详情页
- 拓扑 API 已有静态版本，可快速增加健康指标聚合
- 告警入口已有 /api/v1/events，需补 tenant_id 传递和去重
- Trace 深度分析（火焰图/CPU）数据基础不足，应 mock
- RBAC/Prometheus/Parse Rules 应先 mock，不阻塞前端
- 建议新增 observability query handlers，不碰 Agent core

### Gemini (Frontend)
- 保留 Vue 3 + DaisyUI + Pinia 技术栈，扩展新页面
- 采用侧边栏导航布局（可折叠）+ 顶部面包屑/时间选择器
- 借鉴 APO 的上下文感知导航（拓扑点击 → 日志过滤）
- 借鉴 APO 的全局时间范围同步
- 借鉴 APO 的标准化指标卡片布局
- 诊断推理展示保持 Argus 特色双栏布局
- Mock 策略：composable 层 mock 开关，后端不可用时自动降级

### Consolidated Approach
以 Vue 3 现有前端为基础，新增 5 个核心页面（Dashboard/Topology/Logs/Traces/Alerts），
后端新增 observability read model API，非核心功能 mock。

## Architecture Decisions

### AD-1: 前端布局改造 — 侧边栏导航
- 从单页三栏布局改为侧边栏 + 内容区布局
- 侧边栏：可折叠，图标 + 文字，分组（概览/可观测/诊断/管理）
- 顶部栏：面包屑 + 全局时间范围选择器 + 主题切换
- 现有 DashboardView 保留为诊断控制台页面
- 约束：侧边栏宽度 64px（折叠）/ 240px（展开）

### AD-2: 路由结构
```
/                     → 重定向到 /dashboard
/dashboard            → 总览页（服务健康 + 指标 + 告警摘要）
/topology             → 服务拓扑（交互式图 + 健康指示）
/logs                 → 日志探索器（搜索/过滤/上下文）
/traces               → 链路追踪（列表 + 详情 + 火焰图）
/alerts               → 告警仪表盘（事件列表 + 规则管理）
/diagnose             → AI 诊断控制台（现有核心功能）
/replay               → 故障回放（现有核心功能）
/tasks                → 任务列表（诊断历史）
/tasks/:id            → 任务详情
/settings             → 系统设置
/admin/*              → 管理控制台（现有）
```

### AD-3: 后端 Mock Handler 模式
- 在 `internal/interfaces/http/handler/` 新增 mock handler
- Mock handler 返回结构化的模拟数据，API contract 与真实版一致
- 通过配置开关 `mock.enabled_features` 控制哪些 API 走 mock
- 前端 composable 层不感知 mock/real 差异
- 约束：Mock 数据必须符合 TypeScript 类型定义

### AD-4: 新增 API 端点设计
| 优先级 | 端点 | 实现方式 | 说明 |
|--------|------|----------|------|
| P0 | GET /api/v1/topology/graph | real | 节点+边+健康度+错误数 |
| P0 | GET /api/v1/traces | real | Trace 列表，基于 ES request_uuid 聚合 |
| P0 | GET /api/v1/traces/{uuid} | real | Trace 详情，链路节点+耗时+日志 |
| P0 | GET /api/v1/logs/faults | real | 故障日志聚合列表 |
| P0 | GET /api/v1/logs/context | real | 日志上下文窗口 |
| P0 | GET /api/v1/logs/facets | real | 筛选项聚合（namespace/service/level） |
| P0 | GET /api/v1/dashboard/summary | mock | 总览页统计数据 |
| P1 | GET /api/v1/alerts/active | mock | 活跃告警列表 |
| P1 | CRUD /api/v1/alert-rules | mock | 告警规则管理 |
| P1 | GET /api/v1/metrics/red | mock | RED 指标时序数据 |
| P2 | GET /api/v1/traces/{uuid}/flamegraph | mock | 火焰图数据 |
| P2 | CRUD /admin/v1/rbac/roles | mock | 角色管理 |

### AD-5: 数据关联模式（借鉴 APO）
- 拓扑节点 → 点击 → 跳转日志探索器（自动填充 service 过滤）
- 告警事件 → 点击 → 跳转关联诊断任务详情
- 日志条目 → 点击 request_uuid → 跳转链路追踪详情
- 诊断结论 → affected_services → 高亮拓扑节点
- 全局时间范围 → 同步到所有页面的查询参数

### AD-6: 可视化组件选型
- 图表：ApexCharts（已有依赖，用于 RED 指标、告警分布）
- 拓扑：@antv/x6（已有依赖，增强健康状态渲染）
- 火焰图：自定义 SVG 组件（mock 数据，简单实现）
- 时间线：自定义 CSS 组件（复用现有 TimelinePanel 模式）

## Dependency Graph
```
AD-1 (侧边栏布局) ──┬── AD-2 (路由结构) ──── 所有新页面
                    └── AD-5 (数据关联)
AD-3 (Mock Handler) ──── AD-4 (新 API) ──── 前端页面数据
AD-6 (可视化选型) ──── 各页面图表组件
```

## 评审规则对齐策略

### 场景价值 35% — 核心发力点
- 微服务故障诊断是真实业务痛点，Argus 的 ReAct Agent 直接解决
- 日志分析、链路追踪、服务拓扑是运维团队每日必用能力
- 告警联动诊断实现"告警 → 自动分析 → 根因定位"闭环
- 故障回放让团队可以安全地演练和验证诊断能力

### 效果验证 30% — 量化证据
- 效率对比面板：AI 诊断平均 45s vs 人工诊断平均 30min（40x 提速）
- 步骤减少：AI 8 步自动完成 vs 人工 25+ 步手动排查
- 一键演示流程：生成故障 → 触发告警 → 自动诊断 → 展示结果（全自动）
- 诊断统计：成功率、覆盖场景数、累计节省时间

### 能力沉淀 25% — 可复用资产
- ReAct Agent 框架：通用的 Think→Act→Observe 推理循环
- Tool 抽象与注册：可扩展的工具体系（es_query/trace_analyze/exec_cmd/notify）
- 场景沉淀机制：高置信度诊断自动沉淀为可复用场景
- 回放引擎：故障场景可重复验证和训练

### 使用体验 10% — 锦上添花
- 自然语言输入，无需学习 DSL
- 实时 SSE 推理过程展示，透明可信
- 多页面导航，信息层次清晰
- 一键诊断，普通运维人员可直接使用

## PBT Properties
- **P1: 路由完整性** — 所有定义的路由路径都能正确渲染对应页面组件
- **P2: Mock 数据一致性** — Mock handler 返回的数据结构与 TypeScript 类型定义完全匹配
- **P3: 时间范围同步** — 修改全局时间范围后，所有页面的查询参数同步更新
- **P4: 数据关联正确性** — 从拓扑/告警/日志跳转时，目标页面的过滤条件正确填充
- **P5: 降级兼容** — 后端 API 不可用时，前端显示友好的降级提示而非崩溃
