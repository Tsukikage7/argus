# Tasks: APO-Inspired Overhaul

## 评审规则对齐
- 场景价值 35%：真实业务问题 → 微服务故障诊断、日志分析、链路追踪
- 效果验证 30%：效率提升量化 → 诊断耗时对比、人工步骤减少
- 能力沉淀 25%：可复用 Agent/Skills → ReAct Agent、Tool 抽象、场景沉淀
- 使用体验 10%：简单易用 → 一键诊断、自然语言输入、可视化展示

## Phase 1: 前端布局与导航基础

- [x] 1.1 创建 AppLayout.vue 侧边栏布局组件：可折叠侧边栏（64px/240px）+ 顶部栏 + 内容区
- [x] 1.2 创建 SideNav.vue 侧边导航组件：分组菜单（概览/可观测/诊断/管理），图标+文字，路由高亮
- [x] 1.3 创建 TopBar.vue 顶部栏组件：面包屑 + 全局时间范围选择器 + 主题切换
- [x] 1.4 创建 GlobalTimeRange.vue 全局时间范围选择器：预设（15m/1h/6h/24h/7d）+ 自定义范围
- [x] 1.5 创建 useTimeRange.ts composable：全局时间范围状态管理，跨页面同步
- [x] 1.6 更新 router/index.ts 路由配置：新增 /dashboard /topology /logs /traces /alerts /diagnose /replay /tasks /settings 路由
- [x] 1.7 将现有 DashboardView.vue 迁移为 DiagnoseView.vue，挂载到 /diagnose 路由
- [x] 1.8 创建 ReplayView.vue，从 DashboardView 提取回放模式逻辑，挂载到 /replay 路由

## Phase 2: Dashboard 总览页（场景价值 + 使用体验）

- [x] 2.1 创建 DashboardView.vue 总览页：四区域布局（统计卡片 + 服务健康 + 告警摘要 + 最近诊断）
- [x] 2.2 创建 StatCard.vue 统计卡片组件：数字 + 趋势箭头 + 迷你图（总服务数/活跃告警/诊断任务/平均响应时间）
- [x] 2.3 创建 ServiceHealthGrid.vue 服务健康网格：每个服务一张卡片，显示状态/错误率/延迟
- [x] 2.4 创建 AlertSummaryPanel.vue 告警摘要面板：按严重度分组的告警计数 + 最近告警列表
- [x] 2.5 创建 RecentDiagnosesPanel.vue 最近诊断面板：最近 5 条诊断任务摘要卡片
- [x] 2.6 创建 REDChartPanel.vue RED 指标图表：Request Rate / Error Rate / Duration 三线图（ApexCharts）
- [x] 2.7 后端：创建 GET /api/v1/dashboard/summary mock handler，返回总览页统计数据

## Phase 3: 服务拓扑增强（场景价值）

- [x] 3.1 创建 TopologyView.vue 独立拓扑页面：全屏拓扑图 + 右侧详情面板
- [x] 3.2 增强 GraphRenderer.vue：节点增加健康状态指示器（绿/黄/红圆点）+ 错误率标签
- [x] 3.3 实现拓扑节点点击下钻：点击节点 → 右侧面板显示服务详情（端点列表/最近日志/告警）
- [x] 3.4 实现拓扑到日志跳转：详情面板"查看日志"按钮 → 跳转 /logs?service={name}
- [x] 3.5 后端：增强 GET /api/v1/topology/graph API，返回节点健康度和错误数（基于 ES 聚合）

## Phase 4: 日志探索器（场景价值 + 效果验证）

- [x] 4.1 创建 LogExplorerView.vue 独立日志页面：左侧分面过滤 + 中间日志列表 + 右侧详情
- [x] 4.2 创建 LogFacetPanel.vue 分面过滤面板：namespace/service/level/pod 聚合筛选
- [x] 4.3 创建 LogTable.vue 日志列表组件：虚拟滚动、时间戳/级别/服务/消息列、行展开
- [x] 4.4 创建 LogContextDrawer.vue 日志上下文抽屉：按 request_uuid 展示前后文日志
- [x] 4.5 创建 LogSearchBar.vue 搜索栏：关键词搜索 + 级别过滤 + 时间范围（联动全局）
- [x] 4.6 后端：创建 GET /api/v1/logs/faults API（real），基于 ES 聚合故障日志
- [x] 4.7 后端：创建 GET /api/v1/logs/context API（real），按 request_uuid 返回上下文窗口
- [x] 4.8 后端：创建 GET /api/v1/logs/facets API（real），返回筛选项聚合

## Phase 5: 链路追踪页面（场景价值 + 效果验证）

- [x] 5.1 创建 TracesView.vue 链路追踪页面：Trace 列表 + 过滤条件
- [x] 5.2 创建 TraceTable.vue Trace 列表组件：request_uuid/服务/状态码/耗时/时间戳
- [x] 5.3 创建 TraceDetailView.vue Trace 详情页：链路时间线 + 各节点耗时 + 关联日志
- [x] 5.4 创建 TraceTimeline.vue 链路时间线组件：水平瀑布图展示各服务调用耗时
- [x] 5.5 创建 MockFlameGraph.vue 火焰图组件：SVG 渲染 mock 火焰图数据
- [x] 5.6 后端：创建 GET /api/v1/traces API（real），基于 ES request_uuid 聚合 Trace 列表
- [x] 5.7 后端：创建 GET /api/v1/traces/{uuid} API（real），返回链路详情
- [x] 5.8 后端：创建 GET /api/v1/traces/{uuid}/flamegraph API（mock），返回模拟火焰图数据

## Phase 6: 告警仪表盘（场景价值 + 能力沉淀）

- [x] 6.1 创建 AlertsView.vue 告警页面：告警事件列表 + 严重度分布图 + 过滤
- [x] 6.2 创建 AlertEventTable.vue 告警事件列表：时间/严重度/服务/描述/状态/关联诊断
- [x] 6.3 创建 AlertSeverityChart.vue 严重度分布图：饼图或柱状图（ApexCharts）
- [x] 6.4 创建 AlertDetailDrawer.vue 告警详情抽屉：告警信息 + 关联诊断任务 + 跳转按钮
- [x] 6.5 实现告警到诊断关联：点击告警 → 查看/触发关联诊断任务
- [x] 6.6 后端：创建 GET /api/v1/alerts/active mock handler，返回模拟告警列表
- [x] 6.7 后端：增强 /api/v1/events，补充 tenant_id 传递和告警去重

## Phase 7: 效果验证与演示支撑（效果验证 30%）

- [x] 7.1 创建 DemoScenarioRunner：一键执行完整演示流程（生成故障 → 触发告警 → 自动诊断 → 展示结果）
- [x] 7.2 创建效率对比面板：展示 AI 诊断 vs 人工诊断的耗时/步骤对比数据
- [x] 7.3 增强诊断结论展示：增加"节省时间"和"减少步骤"的量化指标
- [x] 7.4 创建诊断统计页面：历史诊断成功率、平均耗时、覆盖场景数等统计图表
- [x] 7.5 后端：创建 GET /api/v1/stats/efficiency mock handler，返回效率对比数据

## Phase 8: 能力沉淀展示（能力沉淀 25%）

- [x] 8.1 创建 AgentCapabilityView.vue 能力展示页：展示 ReAct Agent 架构、Tool 列表、场景库
- [x] 8.2 创建 ToolRegistryPanel.vue Tool 注册表展示：列出所有可用 Tool 及其能力描述
- [x] 8.3 创建 ScenarioLibraryPanel.vue 场景库面板：已沉淀的故障场景列表 + 场景详情
- [x] 8.4 增强场景保存功能：诊断完成后一键沉淀为可复用场景，支持场景编辑和标签
- [x] 8.5 创建 AgentArchDiagram.vue 架构图组件：展示 ReAct 循环、Tool 调用、LLM 交互的动态架构图

## Phase 9: 诊断控制台增强（使用体验 10%）

- [x] 9.1 增强 DiagnoseInput.vue：增加自然语言输入提示、历史输入记忆
- [x] 9.2 增强 StepCard.vue：Think 内容 Markdown 渲染 + 代码高亮（Shiki）
- [x] 9.3 增强 ConclusionCard.vue：结构化卡片布局 + Markdown 渲染 + 一键复制
- [x] 9.4 创建 TaskListView.vue 任务列表页：分页/筛选/排序，支持状态过滤
- [x] 9.5 创建 TaskDetailView.vue 任务详情页：完整诊断过程回放 + 结论展示

## Phase 10: 设置与配置页面

- [x] 10.1 创建 SettingsView.vue 设置页面：系统信息 + 集成配置 + Agent 配置展示
- [x] 10.2 创建 IntegrationPanel.vue 集成配置面板：ES/Redis/PG 连接状态 + LLM Provider 信息
- [x] 10.3 创建 AgentConfigPanel.vue Agent 配置面板：max_steps/阈值/超时等参数展示
