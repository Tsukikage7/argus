# Design: UX Overhaul & Reliability Enhancement

## Multi-Model Analysis Summary

### Codex (Backend)
- #4 场景保存：ID 类型错误 + 状态机不闭环（draft 不可见）
- #8 SSE：后端 emit 时机正确，但无序列号/缓冲/重连机制，事件会丢失
- #10 任务列表：需合并 Redis active + PG history
- #9 架构：推荐 Detector/Notifier/Incident 三层抽象，优先应用侧轮询

### Gemini (Frontend)
- #3 结论格式化：marked + dompurify + @tailwindcss/typography
- #5 页面拆分：平级路由 /diagnose + /replay，共享 Pinia store
- #7 日志详情：flex 容器 min-width:0 + overflow-x-auto
- #1 动态拓扑：X6 Dagre 自动布局替代固定坐标

## Architecture Decisions

### AD-1: SSE 可靠流协议
- 为每个 task event 增加 `seq` 序列号
- SSEHub 维护最近 100 条事件的环形缓冲
- 支持 `Last-Event-ID` 重连补偿
- stream_token 改为可重用（绑定 taskID，TTL=任务生命周期）
- 前端断连后不清空已有 steps

### AD-2: 诊断结论双轨输出
- 保留 JSON 结构化字段（root_cause, confidence, affected_services, suggestion）
- 新增 `conclusion_markdown` 字段，由 Agent 在 system prompt 中要求 LLM 同时输出
- parseDiagnosis 优先解析 JSON，conclusion_markdown 作为展示层增强

### AD-3: 统一任务查询
- 新增 `TaskQueryHandler.ListAll()`：合并 Redis running + PG history
- 支持 status/source/type/time 过滤排序
- 前端独立 /tasks 页面，支持分页

### AD-4: 动态拓扑子图
- Phase 1：诊断完成后从 diagnosis.affected_services 构建子图
- Phase 2：Agent 推理过程中通过 SSE `topology_update` 事件实时更新
- X6 改用 Dagre 自动布局

### AD-5: 页面路由拆分
- `/diagnose` — 诊断模式（默认）
- `/replay` — 回放模式
- `/tasks` — 任务列表
- `/tasks/:id` — 任务详情
- 共享 Pinia store，URL 参数兼容

## Dependency Graph
```
#4 (场景保存) ──────────────────────────── 独立
#8 (SSE 可靠流) ──┬── #1 (动态拓扑)
                  └── #10 (任务列表) ──── #9 (K8s+飞书)
#3 (结论格式化) ──────────────────────────── 独立
#5 (页面拆分) ───── #10 (任务列表)
#7 (日志详情) ──────────────────────────── 独立
#2 (高级选项) ──────────────────────────── 独立
#6 (实例重复) ───── #1 (动态拓扑)
```
