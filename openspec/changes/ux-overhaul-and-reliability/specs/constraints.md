# Specs: UX Overhaul & Reliability Enhancement

## CONSTRAINT-1: 场景保存 ID 必须为 UUID
- CapturedScenario.ID 必须使用 `uuid.New().String()` 生成
- 数据库 captured_scenarios.id 列类型为 UUID
- INVARIANT: `uuid.Parse(scenario.ID)` 永远成功
- FALSIFICATION: 传入非 UUID 字符串，PG 插入必须失败

## CONSTRAINT-2: SSE 事件有序且可重连
- 每个 task event 必须携带递增 seq（从 1 开始）
- SSEHub 必须为每个 taskID 维护最近 N 条事件缓冲（N ≥ 50）
- 客户端发送 Last-Event-ID 时，服务端必须从 seq+1 开始补发
- INVARIANT: 对于同一 taskID，客户端收到的 seq 严格递增
- FALSIFICATION: 断连 5s 后重连，验证补发事件的 seq 连续性

## CONSTRAINT-3: stream_token 可重用
- stream_token 绑定 (tenantID, taskID)，TTL = 任务完成后 5 分钟
- 同一 token 可多次用于建立 SSE 连接
- INVARIANT: 任务运行期间 token 始终有效
- FALSIFICATION: 任务完成 6 分钟后使用 token，必须返回 401

## CONSTRAINT-4: 统一任务列表
- GET /api/v1/tasks 必须返回 Redis running + PG history 的合并结果
- 支持 ?status=running|completed|failed 过滤
- 结果按 updated_at DESC 排序
- INVARIANT: 新创建的任务在 1s 内出现在列表中
- FALSIFICATION: 创建诊断任务后立即查询列表，任务必须存在

## CONSTRAINT-5: 诊断结论双轨输出
- diagnosis JSON 必须保留 root_cause/confidence/severity/affected_services/suggestion 字段
- 新增 conclusion_markdown 字段（可选），包含 Markdown 格式的完整诊断报告
- INVARIANT: parseDiagnosis 对不含 conclusion_markdown 的 JSON 仍然成功
- FALSIFICATION: 旧格式 JSON 输入，解析不报错

## CONSTRAINT-6: 前端断连保持状态
- SSE 断连时，已收到的 steps/diagnosis 不得清空
- 重连后从 Last-Event-ID 继续接收
- INVARIANT: 用户在断连期间看到的步骤数 ≥ 断连前的步骤数
- FALSIFICATION: 模拟网络断开 3s，验证 steps.length 不减少
