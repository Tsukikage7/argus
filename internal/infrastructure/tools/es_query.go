// Package tools 实现 Agent 可调用的具体工具
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// ESQueryLogsTool 查询 ES 日志
type ESQueryLogsTool struct {
	es *es.Client
}

// NewESQueryLogsTool 创建 ES 日志查询工具
func NewESQueryLogsTool(es *es.Client) *ESQueryLogsTool {
	return &ESQueryLogsTool{es: es}
}

func (t *ESQueryLogsTool) Name() string { return "es_query_logs" }

func (t *ESQueryLogsTool) Description() string {
	return "查询 Elasticsearch 中的服务日志。可按服务名、日志级别、时间范围和关键词筛选。返回最近的日志记录，包含 trace_id、错误信息等关键字段。"
}

func (t *ESQueryLogsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"service": {
				"type": "string",
				"description": "服务名称，如 payment-service, order-service"
			},
			"severity": {
				"type": "string",
				"enum": ["ERROR", "WARN", "INFO", "DEBUG"],
				"description": "日志级别"
			},
			"time_range": {
				"type": "string",
				"description": "时间范围，如 last 15m, last 1h, last 24h"
			},
			"keyword": {
				"type": "string",
				"description": "关键词过滤，在日志正文中搜索"
			}
		},
		"required": ["service"]
	}`)
}

func (t *ESQueryLogsTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	service, _ := params["service"].(string)
	severity, _ := params["severity"].(string)
	timeRange, _ := params["time_range"].(string)
	keyword, _ := params["keyword"].(string)

	if service == "" {
		return &tool.Result{Error: "service parameter is required"}, nil
	}

	if timeRange == "" {
		timeRange = "last 15m"
	}

	logs, err := t.es.QueryLogs(ctx, service, severity, timeRange, keyword)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("query failed: %v", err)}, nil
	}

	if len(logs) == 0 {
		return &tool.Result{Output: fmt.Sprintf("No %s logs found for %s in %s", severity, service, timeRange)}, nil
	}

	// 格式化输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d log entries for %s:\n\n", len(logs), service))
	for i, log := range logs {
		if i >= 20 { // 限制输出条数
			sb.WriteString(fmt.Sprintf("\n... and %d more entries", len(logs)-20))
			break
		}
		sb.WriteString(fmt.Sprintf("[%s] %s | %s", log.Timestamp, log.Severity, log.Body))
		if log.TraceID != "" {
			sb.WriteString(fmt.Sprintf(" (trace_id=%s)", log.TraceID))
		}
		sb.WriteByte('\n')
	}
	return &tool.Result{Output: sb.String()}, nil
}

var _ tool.Tool = (*ESQueryLogsTool)(nil)
