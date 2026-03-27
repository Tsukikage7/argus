# APO-Inspired Overhaul — 参考 APO 改造 Argus

## Summary
以 Argus 现有代码为基础，融合 APO（AutoPilot Observability）项目的可观测性平台特性，将 Argus 从"AI 诊断工具"升级为"智能可观测性平台"。前端优先、非核心后端 mock、以竞赛演示为目标快速上线。

## Motivation
- Argus 核心优势（ReAct Agent 诊断、故障回放、SSE 实时推送）已经成熟
- 但产品形态偏"单一诊断工具"，缺少可观测性平台的全局视角
- APO 提供了成熟的可观测性平台参考：服务拓扑、日志探索、链路追踪、告警管理、指标监控
- 竞赛演示需要视觉冲击力和功能完整度

## Strategy
- **保留**：Vue 3 + DaisyUI + Pinia 前端技术栈、ReAct Agent 核心、ES 集成、回放引擎、多租户架构
- **融合**：APO 的页面结构（Dashboard/Topology/Logs/Traces/Alerts）、导航模式、数据关联
- **Mock**：RBAC、Prometheus 指标、火焰图、CPU 分析、告警规则引擎
- **不做**：ClickHouse 迁移、K8s 集成、DeepFlow 集成、完整 RBAC

## Requirements

### P0 — 高视觉冲击力（演示必备）
1. **Dashboard 总览页**：服务健康卡片 + RED 指标图表 + 最近告警摘要 + 诊断任务统计
2. **侧边栏导航**：从单页应用改为多页面侧边栏导航布局
3. **服务拓扑增强**：独立页面、节点健康状态指示、点击下钻到日志/链路
4. **日志探索器增强**：全功能搜索/过滤/分面导航/上下文查看
5. **链路追踪页面**：Trace 列表 + Trace 详情时间线 + Mock 火焰图
6. **告警仪表盘**：告警事件列表 + 严重度分布 + 告警关联诊断

### P1 — 演示加分项
7. **全局时间范围选择器**：跨页面同步时间窗口
8. **诊断控制台增强**：更好的布局、Markdown 渲染改进
9. **回放模拟增强**：底部 HUD 控制条、速度控制
10. **设置/配置页面**：集成配置、系统信息展示

### P2 — 锦上添花
11. **国际化**：中英文切换
12. **Mock RBAC UI**：角色/权限管理界面（纯前端 mock）
13. **Mock 指标图表**：RED Charts 模拟数据

## Scope
- 前端：Vue 3（扩展现有 frontend/ 项目）
- 后端：Go（新增 API 端点 + Mock Handler）
- 不含：ClickHouse、Prometheus 真实集成、K8s 部署、完整 RBAC 后端

## Constraints
- 前端技术栈：Vue 3 + DaisyUI + Pinia + Vite（不迁移到 React）
- 后端框架：保持 servex + stdlib http（不引入 Gin）
- 数据存储：保持 ES + Redis + PG（不引入 ClickHouse）
- 演示优先：视觉效果 > 数据真实性 > 功能深度
