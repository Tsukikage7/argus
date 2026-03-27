# Design: 告警诊断 + 日志查询 Bug 修复

## 修复策略

### D1: 告警触发诊断 — 直接导航到 chat

**选择**: `AlertsView.onDiagnose()` 直接 `router.push({ name: 'chat', query: { input } })`，不再依赖中间 redirect 路由。

**理由**: redirect 路由无 name、不透传 query，修复成本高于直接导航。保留 `/diagnose` 路由做兼容跳转（函数式 redirect 透传 query）。

### D2: AgentChatView 消费 query.input

**选择**: `onMounted` 时检查 `route.query.input`，若存在则预填输入框并自动发送首条消息，发送后 `router.replace` 清除 query 防止刷新重复触发。

### D3: Mock 索引写入统一

**选择**: Mock generator/live/replay 写入时，默认租户使用 `{prefix}_{namespace}-{date}` 格式（与 `TenantIndex("default")` 返回的 `{prefix}_*` 通配匹配）。不改变现有索引命名，因为默认租户的读写已经一致。

**真正问题**: 前端请求经过 Vite proxy 到后端，认证中间件解析 API Key 得到的 tenantID 可能不是 "default"。需确认 `argus-demo-key` 在多租户模式下解析为哪个 tenant。

### D4: 补充 go-udb-http 故障场景

**选择**: 在 `scenarios.go` 新增 `udb-slow-query` 场景，为 `prj-udb` namespace 生成 ERROR/WARN 级别日志，确保告警中 `go-udb-http` 相关告警有对应的日志数据。

### D5: 日志空状态 UI

**选择**: LogExplorerView 在查询结果为空时显示 EmptyState 组件，提示"未找到日志"并提供"重置过滤器"和"生成 Mock 数据"快捷操作。

## 不做的事

- 不重构 ES 索引策略（属于 productize-saas-integration 变更范围）
- 不添加 ES index template（当前 Mock 环境动态映射足够）
- 不修改多租户认证链路
