// Package agent 实现 ReAct Agent 核心循环
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// LLMClient 是 Agent 依赖的 LLM 接口
type LLMClient interface {
	// ChatWithTools 发起带 function calling 的对话
	ChatWithTools(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// ChatRequest 是发给 LLM 的请求
type ChatRequest struct {
	Model    string
	System   string
	Messages []Message
	Tools    []tool.ToolDef
}

// Message 对话消息
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall LLM 返回的工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON string
	} `json:"function"`
}

// ChatResponse LLM 响应
type ChatResponse struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// EventHandler 接收 Agent 执行过程中的事件（用于 SSE 推送）
type EventHandler func(event task.TaskEvent)

// Config Agent 配置
type Config struct {
	MaxSteps              int
	AutoRecoverThreshold  float64
	ConfirmThreshold      float64
	Timeout               time.Duration
	Model                 string
}

// Agent 实现 ReAct 推理循环
type Agent struct {
	llm    LLMClient
	tools  *tool.Registry
	config Config
}

// New 创建 Agent
func New(llm LLMClient, tools *tool.Registry, cfg Config) *Agent {
	return &Agent{
		llm:    llm,
		tools:  tools,
		config: cfg,
	}
}

// emit 向 per-task handler 发送事件（handler 为 nil 时静默忽略）
func emit(handler EventHandler, event task.TaskEvent) {
	if handler != nil {
		handler(event)
	}
}

// Run 执行完整的诊断流程，eventHandler 为 per-task 事件回调（可为 nil）
func (a *Agent) Run(ctx context.Context, t *task.Task, eventHandler EventHandler) error {
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	// 将租户 ID 注入 context，供 Tool 实现提取
	ctx = task.WithTenantID(ctx, t.TenantID)

	t.Status = task.StatusRunning
	t.UpdatedAt = time.Now()
	emit(eventHandler, task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})

	messages := a.buildInitialMessages(t.Input)
	toolDefs := tool.ToToolDefs(a.tools.List())

	for step := 0; step < a.config.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			t.Status = task.StatusFailed
			return ctx.Err()
		default:
		}

		resp, err := a.llm.ChatWithTools(ctx, &ChatRequest{
			Model:    a.config.Model,
			System:   systemPrompt,
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			t.Status = task.StatusFailed
			return fmt.Errorf("llm call failed at step %d: %w", step, err)
		}

		// 如果 LLM 返回文本（无 tool call），说明推理完成
		if len(resp.ToolCalls) == 0 {
			s := task.Step{
				Index:     step,
				Think:     resp.Content,
				Timestamp: time.Now(),
			}
			t.Steps = append(t.Steps, s)
			emit(eventHandler, task.TaskEvent{TaskID: t.ID, Type: "step", Data: s})

			// 解析诊断结论
			diagnosis, err := parseDiagnosis(resp.Content)
			if err == nil && diagnosis != nil {
				t.Diagnosis = diagnosis
				t.Status = task.StatusCompleted
				now := time.Now()
				t.CompletedAt = &now
				emit(eventHandler, task.TaskEvent{TaskID: t.ID, Type: "diagnosis", Data: diagnosis})
			} else {
				// 无法解析诊断结论，标记为失败
				t.Status = task.StatusFailed
				t.UpdatedAt = time.Now()
				emit(eventHandler, task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})
				return fmt.Errorf("agent: LLM output could not be parsed as diagnosis JSON")
			}
		}

		// 处理 tool calls
		messages = append(messages, Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			params := parseParams(tc.Function.Arguments)
			s := task.Step{
				Index: step,
				Think: resp.Content,
				Action: &task.Action{
					Tool:   tc.Function.Name,
					Params: params,
				},
				IsKeyStep:  true,
				ToolName:   tc.Function.Name,
				ToolParams: params,
				Timestamp:  time.Now(),
			}

			// 执行工具
			registeredTool, toolErr := a.tools.Get(tc.Function.Name)
			if toolErr != nil {
				s.Observe = fmt.Sprintf("error: tool %q not found", tc.Function.Name)
			} else {
				result, execErr := registeredTool.Execute(ctx, s.Action.Params)
				if execErr != nil {
					s.Observe = fmt.Sprintf("error: %v", execErr)
				} else if result.Error != "" {
					s.Observe = fmt.Sprintf("error: %s", result.Error)
				} else {
					s.Observe = result.Output
				}
			}

			messages = append(messages, Message{
				Role:       "tool",
				Content:    s.Observe,
				ToolCallID: tc.ID,
			})

			t.Steps = append(t.Steps, s)
			emit(eventHandler, task.TaskEvent{TaskID: t.ID, Type: "step", Data: s})
		}
	}

	t.Status = task.StatusFailed
	t.UpdatedAt = time.Now()
	return fmt.Errorf("agent: reached max steps (%d) without conclusion", a.config.MaxSteps)
}

func (a *Agent) buildInitialMessages(input string) []Message {
	return []Message{
		{
			Role:    "user",
			Content: input,
		},
	}
}

const systemPrompt = `你是 Argus，一个智能运维诊断 AI Agent。你的任务是分析 UCloud 微服务系统的故障，找到根因并给出恢复方案。

## 系统环境
你正在分析 UCloud 云平台的 K8s 微服务日志。日志通过 fluentd 采集，存储在 Elasticsearch 中。
- 每个服务部署在独立的 K8s namespace 中（如 prj-apigateway、prj-ubill、prj-uresource）
- 请求追踪使用 request_uuid 作为主键，贯穿所有服务
- 网关日志（prj-apigateway）包含 trace-line，记录请求经过的后端节点和延迟

## 日志格式
1. **网关 JSON 日志**（prj-apigateway）：message 是 JSON，包含 request_uri、response_time、trace-line、input（含 request_uuid）
2. **文本日志**（prj-ubill 等）：格式 [timestamp] [LEVEL][request_uuid.step] content
3. **结构化 JSON 日志**（prj-uresource 等）：message 是 JSON，包含 level、trace_id、operation、latency

## 工作流程
1. 收到告警或用户描述后，提取关键信息（request_uuid、错误关键词、namespace）
2. 使用 es_query_logs 工具搜索：
   - 优先用 request_uuid 查找跨 namespace 的所有相关日志
   - 或用 namespace + keyword 查询特定服务的错误日志
   - 或用 keyword 全局搜索
3. 使用 trace_analyze 工具分析 request_uuid 的完整链路：
   - 查看网关 trace-line 了解请求路径和各节点延迟
   - 查看服务日志中的 uuid.step 子请求关系
4. 根据链路耗时和错误信息，定位根因 namespace/服务
5. 给出诊断结论和恢复建议

## 输出格式
当你完成诊断后，请以如下 JSON 格式输出结论：
` + "```json" + `
{
  "root_cause": "简明的根因描述",
  "confidence": 0.0-1.0,
  "affected_services": ["prj-namespace1", "prj-namespace2"],
  "impact": "影响范围描述",
  "suggestions": ["恢复建议1", "恢复建议2"]
}
` + "```" + `

## 注意事项
- 每一步都先思考再行动，不要跳过分析直接下结论
- 如果一个工具调用结果不够，继续调用更多工具收集信息
- 关注 trace-line 中的高延迟节点和 ERROR 级别日志
- request_uuid 是跨服务追踪的核心线索，优先使用
- affected_services 使用 namespace 名称（如 prj-ubill）
- 置信度要基于证据充分程度客观评估

## 搜索策略
- 如果 es_query_logs 返回 0 条结果，尝试以下策略：
  1. 扩大时间范围（如从 last 1h 改为 last 6h 或 last 24h）
  2. 简化关键词（去掉过于具体的限定词，只保留核心错误关键词）
  3. 去掉 namespace 限定，改用全局搜索
  4. 如果结果带有 [fuzzy_match] 标记，说明是模糊匹配结果，需要人工确认相关性
- 不要在同一条件上重复查询超过 2 次，换一个角度搜索

## 高级搜索上下文
- 如果用户输入中包含 [搜索上下文] 段落，务必严格遵守其中的指示
- 如果指定了"搜索时间范围"，优先使用该时间范围（而非默认的 last 1h）
- 如果指定了"优先搜索 namespace"，首次查询必须使用指定的 namespace
- 高级上下文是用户明确设置的偏好，比默认策略优先级更高`

// ── Chat 多轮对话支持 ────────────────────────────────────────────────

// ChatEventHandler 聊天事件回调
type ChatEventHandler func(event ChatEvent)

// ChatEvent 聊天过程中的事件
type ChatEvent struct {
	Type      string `json:"type"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
	Data      any    `json:"data"`
}

// ChatResult 聊天执行结果
type ChatResult struct {
	Content   string          `json:"content"`
	Steps     []task.Step     `json:"steps"`
	Diagnosis *task.Diagnosis `json:"diagnosis,omitempty"`
}

// Chat 执行多轮对话中的一次 Agent 推理，接受外部构建的消息历史
func (a *Agent) Chat(ctx context.Context, tenantID string, conversationMessages []Message, sysPrompt string, handler ChatEventHandler) (*ChatResult, error) {
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	ctx = task.WithTenantID(ctx, tenantID)

	if sysPrompt == "" {
		sysPrompt = systemPrompt
	}

	emitChat := func(eventType string, data any) {
		if handler != nil {
			handler(ChatEvent{Type: eventType, Data: data})
		}
	}

	emitChat("run.started", nil)

	messages := make([]Message, len(conversationMessages))
	copy(messages, conversationMessages)
	toolDefs := tool.ToToolDefs(a.tools.List())

	var steps []task.Step

	for step := 0; step < a.config.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			emitChat("run.failed", ctx.Err().Error())
			return nil, ctx.Err()
		default:
		}

		resp, err := a.llm.ChatWithTools(ctx, &ChatRequest{
			Model:    a.config.Model,
			System:   sysPrompt,
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			emitChat("run.failed", err.Error())
			return nil, fmt.Errorf("llm call failed at step %d: %w", step, err)
		}

		// LLM 返回纯文本 → 推理完成
		if len(resp.ToolCalls) == 0 {
			s := task.Step{
				Index:     step,
				Think:     resp.Content,
				Timestamp: time.Now(),
			}
			steps = append(steps, s)
			emitChat("reasoning.think", s)

			diagnosis, _ := parseDiagnosis(resp.Content)

			result := &ChatResult{
				Content:   resp.Content,
				Steps:     steps,
				Diagnosis: diagnosis,
			}
			emitChat("message.completed", result)
			emitChat("run.completed", nil)
			return result, nil
		}

		// 处理 tool calls
		messages = append(messages, Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			params := parseParams(tc.Function.Arguments)
			s := task.Step{
				Index: step,
				Think: resp.Content,
				Action: &task.Action{
					Tool:   tc.Function.Name,
					Params: params,
				},
				IsKeyStep:  true,
				ToolName:   tc.Function.Name,
				ToolParams: params,
				Timestamp:  time.Now(),
			}

			emitChat("reasoning.act", s)

			registeredTool, toolErr := a.tools.Get(tc.Function.Name)
			if toolErr != nil {
				s.Observe = fmt.Sprintf("error: tool %q not found", tc.Function.Name)
			} else {
				result, execErr := registeredTool.Execute(ctx, s.Action.Params)
				if execErr != nil {
					s.Observe = fmt.Sprintf("error: %v", execErr)
				} else if result.Error != "" {
					s.Observe = fmt.Sprintf("error: %s", result.Error)
				} else {
					s.Observe = result.Output
				}
			}

			emitChat("reasoning.observe", s)

			messages = append(messages, Message{
				Role:       "tool",
				Content:    s.Observe,
				ToolCallID: tc.ID,
			})

			steps = append(steps, s)
		}
	}

	emitChat("run.failed", "reached max steps")
	return nil, fmt.Errorf("agent: reached max steps (%d) without conclusion", a.config.MaxSteps)
}
