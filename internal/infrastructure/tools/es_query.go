// Package tools 实现 Agent 可调用的具体工具
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// ESQueryLogsTool 查询 UCloud K8s ES 日志
type ESQueryLogsTool struct {
	es *es.Client
}

// NewESQueryLogsTool 创建 ES 日志查询工具
func NewESQueryLogsTool(esClient *es.Client) *ESQueryLogsTool {
	return &ESQueryLogsTool{es: esClient}
}

func (t *ESQueryLogsTool) Name() string { return "es_query_logs" }

func (t *ESQueryLogsTool) Description() string {
	return `查询 Elasticsearch 中的 UCloud K8s 日志。支持三种查询模式：
1. 通过 request_uuid 查找跨 namespace 的所有相关日志（推荐）
2. 通过 namespace + keyword 查询特定服务的日志
3. 通过 keyword 全局搜索所有 namespace 的日志
返回日志记录包含 namespace、app、日志级别和 request_uuid 等关键信息。`
}

func (t *ESQueryLogsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"request_uuid": {
				"type": "string",
				"description": "请求追踪 UUID，用于查找跨 namespace 的所有相关日志"
			},
			"namespace": {
				"type": "string",
				"description": "K8s namespace，如 prj-ubill、prj-apigateway，限定搜索范围"
			},
			"keyword": {
				"type": "string",
				"description": "关键词，在日志 message 中全文搜索"
			},
			"time_range": {
				"type": "string",
				"description": "时间范围，如 last 15m, last 1h, last 24h"
			}
		}
	}`)
}

func (t *ESQueryLogsTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	requestUUID, _ := params["request_uuid"].(string)
	namespace, _ := params["namespace"].(string)
	keyword, _ := params["keyword"].(string)
	timeRange, _ := params["time_range"].(string)
	tenantID := task.TenantIDFrom(ctx)

	if timeRange == "" {
		timeRange = "last 1h"
	}

	// 至少需要提供一个搜索条件
	if requestUUID == "" && namespace == "" && keyword == "" {
		return &tool.Result{Error: "至少需要提供 request_uuid、namespace 或 keyword 之一"}, nil
	}

	var logs []es.UCloudLog
	var err error
	var queryDesc string

	// 优先级：request_uuid > namespace+keyword > keyword
	if requestUUID != "" {
		logs, err = t.es.QueryByRequestUUID(ctx, tenantID, requestUUID, timeRange)
		queryDesc = fmt.Sprintf("request_uuid=%s", requestUUID)
	} else if namespace != "" {
		logs, err = t.es.QueryByNamespace(ctx, tenantID, namespace, keyword, timeRange)
		if keyword != "" {
			queryDesc = fmt.Sprintf("namespace=%s, keyword=%s", namespace, keyword)
		} else {
			queryDesc = fmt.Sprintf("namespace=%s", namespace)
		}
	} else {
		logs, err = t.es.QueryByKeyword(ctx, tenantID, keyword, timeRange)
		queryDesc = fmt.Sprintf("keyword=%s", keyword)
	}

	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("查询失败: %v", err)}, nil
	}

	if len(logs) == 0 {
		return &tool.Result{Output: fmt.Sprintf("未找到日志（查询条件：%s，时间范围：%s）", queryDesc, timeRange)}, nil
	}

	// 格式化输出
	var sb strings.Builder
	total := len(logs)
	limit := 30
	if total < limit {
		limit = total
	}

	sb.WriteString(fmt.Sprintf("Found %d log entries (query: %s):\n\n", total, queryDesc))

	for i := 0; i < limit; i++ {
		log := logs[i]

		// 提取 app 名称（优先 kubernetes_labels_app，回落 kubernetes_container）
		app := log.KubernetesLabelsApp
		if app == "" {
			app = log.KubernetesContainer
		}

		// 提取日志级别
		level := es.ExtractLogLevel(&log)

		// 提取 request_uuid（如果查询本身不是按 uuid 查的，也尝试从 message 提取）
		uuid := requestUUID
		if uuid == "" {
			uuid = es.ExtractRequestUUID(log.Message)
		}

		// message 截断到 200 字符
		msg := log.Message
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}

		// 输出一行：[timestamp] namespace/app [LEVEL] message (request_uuid=xxx)
		line := fmt.Sprintf("[%s] %s/%s [%s] %s", log.Timestamp, log.KubernetesNamespace, app, level, msg)
		if uuid != "" {
			line += fmt.Sprintf(" (request_uuid=%s)", uuid)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	if total > 30 {
		sb.WriteString(fmt.Sprintf("\n... and %d more entries", total-30))
	}

	return &tool.Result{Output: sb.String()}, nil
}

var _ tool.Tool = (*ESQueryLogsTool)(nil)
