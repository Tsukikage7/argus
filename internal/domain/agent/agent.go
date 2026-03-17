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
	llm      LLMClient
	tools    *tool.Registry
	config   Config
	onEvent  EventHandler
}

// New 创建 Agent
func New(llm LLMClient, tools *tool.Registry, cfg Config) *Agent {
	return &Agent{
		llm:    llm,
		tools:  tools,
		config: cfg,
	}
}

// OnEvent 设置事件回调
func (a *Agent) OnEvent(handler EventHandler) {
	a.onEvent = handler
}

func (a *Agent) emit(event task.TaskEvent) {
	if a.onEvent != nil {
		a.onEvent(event)
	}
}

// Run 执行完整的诊断流程
func (a *Agent) Run(ctx context.Context, t *task.Task) error {
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	t.Status = task.StatusRunning
	t.UpdatedAt = time.Now()
	a.emit(task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})

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
			a.emit(task.TaskEvent{TaskID: t.ID, Type: "step", Data: s})

			// 解析诊断结论
			diagnosis, err := parseDiagnosis(resp.Content)
			if err == nil && diagnosis != nil {
				t.Diagnosis = diagnosis
				t.Status = task.StatusCompleted
				now := time.Now()
				t.CompletedAt = &now
				a.emit(task.TaskEvent{TaskID: t.ID, Type: "diagnosis", Data: diagnosis})
			} else {
				t.Status = task.StatusCompleted
				now := time.Now()
				t.CompletedAt = &now
			}
			t.UpdatedAt = time.Now()
			a.emit(task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})
			return nil
		}

		// 处理 tool calls
		messages = append(messages, Message{
			Role:      "assistant",
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			s := task.Step{
				Index: step,
				Think: resp.Content,
				Action: &task.Action{
					Tool:   tc.Function.Name,
					Params: parseParams(tc.Function.Arguments),
				},
				Timestamp: time.Now(),
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
			a.emit(task.TaskEvent{TaskID: t.ID, Type: "step", Data: s})
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

const systemPrompt = `你是 Argus，一个智能运维诊断 AI Agent。你的任务是分析微服务系统的故障，找到根因并给出恢复方案。

## 工作流程
1. 收到告警或用户描述后，使用工具查询相关服务的日志
2. 根据日志中的 trace_id 追踪完整调用链路
3. 逐步缩小范围，定位根因服务和具体错误
4. 给出诊断结论和恢复建议

## 输出格式
当你完成诊断后，请以如下 JSON 格式输出结论：
` + "```json" + `
{
  "root_cause": "简明的根因描述",
  "confidence": 0.0-1.0,
  "affected_services": ["service1", "service2"],
  "impact": "影响范围描述",
  "suggestions": ["恢复建议1", "恢复建议2"]
}
` + "```" + `

## 注意事项
- 每一步都先思考再行动，不要跳过分析直接下结论
- 如果一个工具调用结果不够，继续调用更多工具收集信息
- 关注日志中的 trace_id、error 级别日志和关键指标
- 置信度要基于证据充分程度客观评估`
