// Package llm 提供基于 servex/ai 的 LLM 客户端适配层.
// 将 servex/ai/openai + servex/ai/router 适配为 agent.LLMClient 接口.
package llm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/interfaces/config"
	"github.com/Tsukikage7/servex/ai"
	aiopenai "github.com/Tsukikage7/servex/ai/openai"
	"github.com/Tsukikage7/servex/ai/router"
)

// NewRouter 根据 providers 配置创建 LLM 路由客户端.
// 第一个 provider 作为默认（无匹配时的兜底），所有 provider 均按 models 列表路由.
func NewRouter(providers []config.ProviderConfig) (agent.LLMClient, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("llm router: 至少需要一个 provider")
	}

	routes := make([]router.Route, 0, len(providers))
	for i := range providers {
		p := &providers[i]
		if len(p.Models) > 0 {
			routes = append(routes, router.Route{
				Models: p.Models,
				Model:  buildClient(p),
			})
		}
	}

	r := router.New(buildClient(&providers[0]), routes...)
	return &chatModelAdapter{model: r}, nil
}

// buildClient 从 ProviderConfig 创建 servex OpenAI 兼容客户端.
func buildClient(p *config.ProviderConfig) ai.ChatModel {
	opts := []aiopenai.Option{
		aiopenai.WithModel(p.DefaultModel),
	}
	if p.BaseURL != "" {
		opts = append(opts, aiopenai.WithBaseURL(p.BaseURL))
	}
	if p.Timeout > 0 {
		opts = append(opts, aiopenai.WithHTTPClient(&http.Client{Timeout: p.Timeout}))
	}
	client := aiopenai.New(p.APIKey, opts...)
	if p.MaxTokens <= 0 {
		return client
	}
	return &maxTokensModel{ChatModel: client, maxTokens: p.MaxTokens}
}

// maxTokensModel 为 ChatModel 注入默认 MaxTokens，调用方仍可覆盖.
type maxTokensModel struct {
	ai.ChatModel
	maxTokens int
}

func (m *maxTokensModel) Generate(ctx context.Context, messages []ai.Message, opts ...ai.CallOption) (*ai.ChatResponse, error) {
	return m.ChatModel.Generate(ctx, messages, append([]ai.CallOption{ai.WithMaxTokens(m.maxTokens)}, opts...)...)
}

func (m *maxTokensModel) Stream(ctx context.Context, messages []ai.Message, opts ...ai.CallOption) (ai.StreamReader, error) {
	return m.ChatModel.Stream(ctx, messages, append([]ai.CallOption{ai.WithMaxTokens(m.maxTokens)}, opts...)...)
}

// ─── 适配器：ai.ChatModel → agent.LLMClient ────────────────────────────────

type chatModelAdapter struct {
	model ai.ChatModel
}

var _ agent.LLMClient = (*chatModelAdapter)(nil)

// ChatWithTools 发起带 function calling 的对话.
func (a *chatModelAdapter) ChatWithTools(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	messages := toAIMessages(req)
	tools := toAITools(req.Tools)

	opts := make([]ai.CallOption, 0, 2)
	if req.Model != "" {
		opts = append(opts, ai.WithModel(req.Model))
	}
	if len(tools) > 0 {
		opts = append(opts, ai.WithTools(tools...))
	}

	resp, err := a.model.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	result := &agent.ChatResponse{
		Content: resp.Message.Content,
	}
	for _, tc := range resp.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, agent.ToolCall{
			ID: tc.ID,
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return result, nil
}

// toAIMessages 将 agent.ChatRequest 中的消息转为 []ai.Message.
func toAIMessages(req *agent.ChatRequest) []ai.Message {
	msgs := make([]ai.Message, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, ai.SystemMessage(req.System))
	}
	for _, m := range req.Messages {
		msg := ai.Message{
			Role:       ai.Role(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ai.ToolCall{
				ID: tc.ID,
				Function: struct {
					Name      string
					Arguments string
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

// toAITools 将 tool.ToolDef 转为 []ai.Tool.
func toAITools(defs []tool.ToolDef) []ai.Tool {
	tools := make([]ai.Tool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, ai.Tool{
			Function: ai.FunctionDef{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		})
	}
	return tools
}
