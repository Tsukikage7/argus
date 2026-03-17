package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// TraceAnalyzeTool 通过 trace_id 分析完整调用链路
type TraceAnalyzeTool struct {
	es *es.Client
}

// NewTraceAnalyzeTool 创建链路分析工具
func NewTraceAnalyzeTool(es *es.Client) *TraceAnalyzeTool {
	return &TraceAnalyzeTool{es: es}
}

func (t *TraceAnalyzeTool) Name() string { return "trace_analyze" }

func (t *TraceAnalyzeTool) Description() string {
	return "通过 trace_id 获取完整的微服务调用链路。展示每个服务的耗时、状态和错误信息，帮助定位链路中的故障节点。"
}

func (t *TraceAnalyzeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"trace_id": {
				"type": "string",
				"description": "分布式追踪 ID"
			}
		},
		"required": ["trace_id"]
	}`)
}

func (t *TraceAnalyzeTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	traceID, _ := params["trace_id"].(string)
	if traceID == "" {
		return &tool.Result{Error: "trace_id is required"}, nil
	}

	spans, err := t.es.QueryTrace(ctx, traceID)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("trace query failed: %v", err)}, nil
	}

	if len(spans) == 0 {
		return &tool.Result{Output: fmt.Sprintf("No trace found for trace_id=%s", traceID)}, nil
	}

	// 构建调用链可视化
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Trace %s (%d spans):\n\n", traceID, len(spans)))

	for _, span := range spans {
		indent := ""
		if span.ParentSpanID != "" {
			indent = "  → "
		}
		status := span.Status
		if span.Error != "" {
			status = fmt.Sprintf("ERROR: %s", span.Error)
		}
		sb.WriteString(fmt.Sprintf("%s%s [%s] %v\n", indent, span.Service, status, span.Duration))
	}

	return &tool.Result{Output: sb.String()}, nil
}

var _ tool.Tool = (*TraceAnalyzeTool)(nil)
