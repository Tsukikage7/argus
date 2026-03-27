# Spec: 告警触发诊断路由修复

## Requirement: 告警详情触发诊断导航到聊天页

AlertsView 的 onDiagnose 函数 SHALL 使用 `router.push({ name: 'chat', query: { input } })` 直接导航到聊天页。

### Scenario: 用户从告警详情触发诊断
- GIVEN 用户在告警中心查看告警详情
- WHEN 用户点击"触发诊断"按钮
- THEN 页面导航到 `/chat?input=诊断告警: {message} (服务: {service})`
- AND AgentChatView 读取 query.input 并自动发送首条诊断消息
- AND 发送后 URL 中的 input 参数被清除（防止刷新重复触发）

### Scenario: 兼容旧 /diagnose 链接
- GIVEN 用户通过旧链接访问 `/diagnose?input=xxx`
- WHEN 路由匹配到 /diagnose
- THEN 重定向到 `/chat?input=xxx`（query 参数保留）

## PBT Properties

- **幂等性**: 多次点击"触发诊断"只创建一个聊天会话
- **参数完整性**: 告警 message 和 service 信息完整传递到聊天页
- **无副作用刷新**: 刷新聊天页不会重复发送诊断请求
