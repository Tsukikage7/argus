# Tasks: UX Overhaul & Reliability Enhancement

## Phase 1: P0 核心链路修复

- [ ] 1.1 修复场景保存 ID 生成：replay.go L262 `fmt.Sprintf("%d", time.Now().UnixNano())` 改为 `uuid.New().String()`
- [ ] 1.2 场景保存后自动发布：创建时 status 改为 published，或在保存成功后自动调用 publish
- [ ] 1.3 前端场景保存成功提示：ConclusionCard.vue 保存后显示 Toast + 刷新场景列表
- [ ] 1.4 SSE 事件增加序列号：Agent emit 时附加递增 seq，SSE 输出 `id: {seq}`
- [ ] 1.5 SSEHub 增加环形缓冲：维护每个 taskID 最近 100 条事件
- [ ] 1.6 SSE 支持 Last-Event-ID 重连：stream handler 解析 header 并从缓冲补发
- [ ] 1.7 stream_token 改为可重用：绑定 taskID + TTL=任务生命周期，支持 EventSource 自动重连
- [ ] 1.8 前端 SSE 断连不清空状态：useSSE.ts disconnect 时保留已有 steps/diagnosis
- [ ] 1.9 统一任务查询 API：新增 TaskQueryHandler.ListAll() 合并 Redis running + PG history
- [ ] 1.10 统一任务查询支持过滤排序：status/source/type/time 参数

## Phase 2: P1 体验增强

- [ ] 2.1 安装 marked + dompurify + @tailwindcss/typography
- [ ] 2.2 ConclusionCard.vue 增加 Markdown 渲染：root_cause/summary/suggestion 字段用 prose 类渲染
- [ ] 2.3 Agent system prompt 增加 conclusion_markdown 字段要求
- [ ] 2.4 parseDiagnosis 支持 conclusion_markdown 字段解析
- [ ] 2.5 StepCard.vue Think 内容 Markdown 渲染：推理过程也用 prose 渲染
- [ ] 2.6 日志详情布局修复：LogExplorer.vue 详情区域改为 flex-col + min-width:0 + overflow-x-auto
- [ ] 2.7 日志详情改为 Drawer/Modal：点击日志行弹出侧边抽屉展示完整详情
- [ ] 2.8 路由拆分：新增 /diagnose、/replay、/tasks 路由
- [ ] 2.9 DiagnoseView.vue：从 DashboardView 提取诊断模式逻辑
- [ ] 2.10 ReplayView.vue：从 DashboardView 提取回放模式逻辑
- [ ] 2.11 TaskListView.vue：独立任务管理页面，支持分页/筛选/排序
- [ ] 2.12 TaskDetailView.vue：任务详情页，展示完整诊断/回放结果
- [ ] 2.13 拓扑动态子图 Phase 1：诊断完成后从 affected_services 构建子图
- [ ] 2.14 useTopology.ts 改用 Dagre 自动布局替代固定坐标

## Phase 3: P2 功能补完

- [ ] 3.1 高级选项 tooltip：DiagnoseInput.vue 为时间范围和 namespace 添加说明文字
- [ ] 3.2 拓扑节点 label 去重：useTopology.ts 节点渲染时检查 namespace vs labelsApp 重复
- [ ] 3.3 拓扑动态子图 Phase 2：Agent 推理过程中通过 SSE topology_update 事件实时更新
- [ ] 3.4 飞书 Notifier 接口设计：定义 Notifier 抽象接口 + FeishuNotifier 实现
- [ ] 3.5 Detector 接口设计：定义 Detector 抽象接口 + ESPollerDetector 实现（复用 LogWatchService）
- [ ] 3.6 Incident 模型设计：去重/冷却/聚合/升级/落任务
- [ ] 3.7 自动监控配置页面：admin 下新增 Webhook 管理 + 告警规则 UI
