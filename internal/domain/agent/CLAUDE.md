[根目录](../../../CLAUDE.md) > [internal](../../) > [domain](../) > **agent**

# agent -- ReAct Agent 核心推理引擎

## 模块职责

实现 ReAct（Reasoning + Acting）推理循环，是 Argus 的核心智能模块。负责：

1. 接收诊断任务，构建 LLM 对话上下文（system prompt + 用户输入）
2. 循环调用 LLM 获取推理结果，执行 function calling（Tool 调用）
3. 将 Tool 执行结果反馈给 LLM，直到 LLM 输出诊断结论
4. 解析 JSON 格式的诊断结论（root_cause / confidence / suggestions）
5. 恢复后验证（Verifier）：查询目标服务日志确认错误消失

## 入口与启动

- **Agent.Run(ctx, task)** -- 主入口，执行完整诊断流程
- **Agent.OnEvent(handler)** -- 注册 SSE 事件回调
- **Verifier.Verify(ctx, task)** -- 恢复后验证

## 对外接口

### LLMClient 接口（由 infrastructure/llm 实现）

```go
type LLMClient interface {
    ChatWithTools(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

### 数据结构

- `ChatRequest` -- 包含 Model、System prompt、Messages、Tools
- `ChatResponse` -- 包含 Content（文本）或 ToolCalls（工具调用）
- `Message` -- 对话消息（role/content/tool_calls/tool_call_id）
- `ToolCall` -- LLM 返回的工具调用请求（id/function.name/function.arguments）
- `Config` -- Agent 配置（MaxSteps/Timeout/Model/置信度阈值）

## 关键依赖与配置

- 依赖 `domain/tool.Registry` -- 获取已注册的 Tool 列表和执行工具
- 依赖 `domain/task` -- Task、Step、Diagnosis 等领域模型
- 配置项：`MaxSteps`（默认 15）、`Timeout`（默认 5min）、`AutoRecoverThreshold`（0.8）、`ConfirmThreshold`（0.5）

## 数据模型

本模块不直接持有数据模型，使用 `domain/task` 中定义的：
- `task.Task` -- 诊断任务
- `task.Step` -- 推理步骤（Think/Action/Observe）
- `task.Diagnosis` -- 诊断结论（从 LLM 输出的 JSON 解析）
- `task.TaskEvent` -- SSE 推送事件

## 测试与质量

> 当前无测试文件。

建议测试方向：
- mock LLMClient 测试 Agent.Run 的多步推理流程
- 测试 parseDiagnosis 对各种 JSON 格式的兼容性
- 测试 parseParams 的 edge case
- 测试 Verifier 的成功/失败路径

## 常见问题 (FAQ)

**Q: Agent 如何知道何时停止推理？**
A: 两种情况停止：(1) LLM 返回纯文本（无 tool_calls），说明推理完成；(2) 达到 MaxSteps 上限，强制终止。

**Q: System Prompt 在哪里定义？**
A: 在 `agent.go` 底部的 `systemPrompt` 常量中，指导 LLM 按 ReAct 模式工作。

**Q: 如何修改 LLM 的推理行为？**
A: 修改 `systemPrompt` 常量，或调整 Agent.Config 中的 MaxSteps / Timeout。

## 相关文件清单

| 文件 | 职责 |
|------|------|
| `agent.go` | Agent 结构体、Run 循环、LLMClient 接口、ChatRequest/Response 定义、system prompt |
| `planner.go` | parseDiagnosis（从 LLM 文本提取 JSON 诊断）、parseParams |
| `verifier.go` | Verifier -- 恢复后验证（查 ES 确认错误日志消失） |

## 变更记录 (Changelog)

| 时间 | 操作 | 说明 |
|------|------|------|
| 2026-03-18T00:09:25 | 初始生成 | 扫描生成模块文档 |
