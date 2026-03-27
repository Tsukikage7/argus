# Tasks: fix-alerts-logs-bugs

## Bug 2: 告警触发诊断无响应

- [x] 1.1 修改 AlertsView.vue onDiagnose：`router.push({ name: 'chat', query: { input } })` 替换 `name: 'diagnose'`
- [x] 1.2 修改 router/index.ts diagnose 路由：改为函数式 redirect 透传 query `redirect: to => ({ path: '/chat', query: to.query })`
- [x] 1.3 修改 AgentChatView.vue：onMounted 检查 route.query.input，若存在则自动创建会话并发送首条消息，发送后 router.replace 清除 query

## Bug 1: 日志查询返回空结果

- [x] 2.1 在 scenarios.go 新增 udb-slow-query 场景：为 prj-udb/go-udb-http 生成 ERROR+WARN 级别日志（数据库慢查询 + 连接池告警）
- [x] 2.2 确认 argus-demo-key 在多租户模式下解析为 default 租户，验证 TenantIndex 返回 `argus_*` 能匹配 Mock 索引
- [x] 2.3 在 LogExplorerView.vue 添加空状态 UI：查询结果为空时显示引导提示（重置过滤器 + 生成 Mock 数据链接）

## 验证

- [ ] 3.1 端到端验证：just mock-generate → 前端查询 go-udb-http 故障日志 → 返回非空结果
- [ ] 3.2 端到端验证：告警中心点击触发诊断 → 跳转聊天页 → 自动发送诊断消息
