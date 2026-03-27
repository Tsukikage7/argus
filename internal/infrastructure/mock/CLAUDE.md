[根目录](../../../CLAUDE.md) > [internal](../../) > [infrastructure](../) > **mock**

# mock -- Mock 数据生成与故障回放引擎

## 模块职责

1. 定义 6 个微服务的模拟拓扑（电商系统：gateway / user / order / payment / inventory / notification）
2. 提供 3 个预置故障场景，生成 OpenTelemetry 格式的日志和链路追踪数据
3. 将生成的数据批量写入 Elasticsearch
4. 提供故障回放引擎（ReplayEngine），支持按强度/倍率缩放数据，计算影响面

## 入口与启动

- **Generator.GenerateAll(ctx)** -- 生成所有场景数据（CLI `mock generate`）
- **ReplayEngine.RunFaultReplay(ctx, session)** -- 故障回放
- **ReplayEngine.RunTrafficReplay(ctx, session)** -- 流量回放
- **ReplayEngine.ComputeImpact(ctx, sessionID)** -- 基于 ES 聚合计算影响面

## 对外接口

### 故障场景

| 场景名 | 描述 | 根因 |
|--------|------|------|
| `payment-db-pool-exhausted` | order-service 504 超时 | payment-service 慢查询导致 DB 连接池耗尽 |
| `inventory-oom` | 库存扣减失败 | inventory-service 内存泄漏被 OOM kill |
| `gateway-disk-full` | 间歇性 502 | gateway 日志写满磁盘 |

### 服务拓扑

6 个服务，模拟 K8s 环境（namespace=production），每个服务有 pod name、node name、host IP。

### ES 索引结构

- `argus-logs-{service}-{date}` -- 按服务按天分索引
- `argus-traces-{date}` -- 链路追踪汇总索引

## 关键依赖与配置

- 依赖 `infrastructure/es.Client` -- 批量写入数据
- 依赖 `domain/task` -- ReplaySession、ImpactReport 等模型
- 回放数据会注入 `attributes.replay_session_id` 字段，用于影响面聚合

## 数据模型

- `Service` -- 微服务定义（name/port/version/instanceID/hostIP/namespace/podName/nodeName）
- `Scenario` -- 故障场景（name/description/GenerateLogs 函数）
- `ReplayEngine` -- 回放引擎（管理场景、执行回放、计算影响面）
- `ReplayResult` -- 数据注入结果（LogsWritten/TracesWritten）

## 测试与质量

> 当前无测试文件。

建议测试方向：
- 测试每个场景的 GenerateLogs 输出的日志数量和字段完整性
- 测试 scaleLogs / scaleTrafficLogs 的缩放正确性
- 测试 computeBlastRadius 的各种阈值边界
- 测试 ServiceByName 查找

## 常见问题 (FAQ)

**Q: 如何添加新的故障场景？**
A: 在 `scenarios.go` 中添加新的 Scenario 工厂函数，实现 GenerateLogs 生成日志和链路数据，然后加入 `AllScenarios()` 列表。

**Q: 回放数据如何与正常数据区分？**
A: 回放数据在 attributes 中携带 `replay_session_id` 字段，ComputeImpact 基于此字段做聚合查询。

## 相关文件清单

| 文件 | 职责 |
|------|------|
| `topology.go` | Service 结构体定义、6 个微服务拓扑、ServiceByName 查找 |
| `scenarios.go` | 3 个故障场景定义、makeLog/makeTrace 辅助函数 |
| `generator.go` | Generator -- 批量生成所有场景数据写入 ES |
| `replay.go` | ReplayEngine -- 故障/流量回放、影响面计算、数据缩放 |

## 变更记录 (Changelog)

| 时间 | 操作 | 说明 |
|------|------|------|
| 2026-03-18T00:09:25 | 初始生成 | 扫描生成模块文档 |
