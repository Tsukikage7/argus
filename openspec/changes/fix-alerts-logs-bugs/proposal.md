# Fix: 告警中心触发诊断无响应 + 日志查询返回空结果

## 问题描述

用户报告两个关联 bug：
1. 告警中心点击"触发诊断"按钮无响应
2. 日志探索器查询故障日志返回空结果 `{"total":0,"logs":[]}`

## 根因分析

### Bug 1: 日志查询空结果（多因叠加）

| 严重度 | 根因 | 影响 |
|--------|------|------|
| Critical | Mock 写入索引 `argus_{ns}-{date}`，多租户查询用 `argus-{tenantID}-logs-*`，不匹配 | 非 default 租户查询稳定返回空 |
| High | Mock 场景无 `go-udb-http` 故障数据 | 该服务查询必定为空 |
| Medium | `kubernetes_labels_app.keyword` 无显式 ES mapping | 依赖动态映射，不可靠 |
| Medium | Mock 数据超 1h 被 `time_range=last 1h` 过滤 | 环境长时间运行后"数据消失" |
| Low | `IgnoreUnavailable + AllowNoIndices` 静默吞错 | 索引不匹配表现为空结果 |

### Bug 2: 告警触发诊断无响应

| 严重度 | 根因 | 影响 |
|--------|------|------|
| High | `router.push({ name: 'diagnose' })` 但路由无 name 属性 | 导航失败，后端无请求 |
| Medium | redirect 字符串不透传 query 参数 | 即使跳转成功参数也丢失 |
| Medium | AgentChatView 不消费 `route.query.input` | 跳转后不会预填诊断内容 |

## 修复范围

- 前端：路由修复 + AgentChatView query 消费 + 空状态 UI
- 后端：Mock 索引写入统一 + 补充 go-udb-http 场景数据
- 不涉及架构变更，纯 bug 修复

## 约束

- 不改变现有多租户索引策略，仅修复 Mock 写入与查询的一致性
- 保持向后兼容，旧索引数据通过 alias 或双读过渡
