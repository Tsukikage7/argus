# UX Overhaul & Reliability Enhancement

## Summary
前端体验全面升级 + 后端可靠性增强，涵盖 10 项改进需求。

## Motivation
用户在实际使用中发现多个体验和功能问题，影响产品演示效果和核心链路可用性。

## Requirements

### P0 — 核心链路修复
1. **#4 场景保存 bug**：ID 生成用 UnixNano 而非 UUID，PG 插入失败；draft 状态场景不在列表显示
2. **#8 SSE 流式可靠性**：事件无序列号、无缓冲、断连丢失、token 不可重连
3. **#10 任务列表模式**：GET /api/v1/tasks 只返回 PG 历史，不含 Redis 中运行中任务

### P1 — 体验增强
4. **#3 诊断结论格式化**：纯文本 → Markdown 渲染 + 结构化卡片
5. **#1 调用链路动态化**：写死 6 节点 → 按 affected_services 构建子图
6. **#7 日志详情太窄**：flex 容器宽度不足导致竖排
7. **#5 诊断/回放页面拆分**：单路由 → /diagnose + /replay 分离

### P2 — 功能补完
8. **#2 高级选项说明**：加 tooltip/helper text
9. **#6 实例名称重复**：拓扑节点 label 去重
10. **#9 K8s + 飞书自动化**：架构级改进，引入 Detector/Notifier/Incident 三层抽象

## Scope
- 后端：Go（agent、handler、mock、persistence、SSE）
- 前端：Vue3（composables、components、router、store）
- 不含 K8s 部署配置（#9 仅设计接口层）
