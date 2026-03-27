# Spec: 日志查询空结果修复

## Requirement: Mock 数据覆盖告警中引用的服务

Mock 数据生成器 SHALL 为所有在 alerts handler 中引用的服务生成对应的故障日志数据。

### Scenario: go-udb-http 故障日志可查询
- GIVEN Mock 数据已生成（`just mock-generate`）
- WHEN 用户查询 `/api/v1/logs/faults?service=go-udb-http&level=ERROR,WARN&time_range=last 1h`
- THEN 返回 total > 0 且 logs 数组非空
- AND 每条日志包含 level=ERROR 或 level=WARN

### Scenario: 默认租户索引匹配
- GIVEN 使用 `argus-demo-key` 认证（解析为 default 租户）
- WHEN 查询故障日志
- THEN ES 查询使用 `{prefix}_*` 索引模式
- AND 能匹配到 Mock 写入的 `{prefix}_{namespace}-{date}` 索引

## Requirement: 日志空状态用户引导

LogExplorerView SHALL 在查询结果为空时显示友好的空状态提示。

### Scenario: 无日志数据时显示引导
- GIVEN 日志查询返回 total=0
- WHEN 页面渲染完成
- THEN 显示"未找到匹配的日志"提示
- AND 提供"重置过滤条件"按钮
- AND 提供"生成 Mock 数据"快捷链接

## PBT Properties

- **数据一致性**: alerts handler 中每个 service 在 Mock 数据中都有对应日志
- **索引匹配不变量**: `TenantIndex(tenantID)` 通配模式始终能匹配 Mock 写入的索引名
- **时间窗口覆盖**: Mock 生成的日志时间戳在 `now - 1h` 到 `now` 范围内
