package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// TraceAnalyzeTool 通过 request_uuid 分析完整请求链路
type TraceAnalyzeTool struct {
	es *es.Client
}

// NewTraceAnalyzeTool 创建链路分析工具
func NewTraceAnalyzeTool(esClient *es.Client) *TraceAnalyzeTool {
	return &TraceAnalyzeTool{es: esClient}
}

func (t *TraceAnalyzeTool) Name() string { return "trace_analyze" }

func (t *TraceAnalyzeTool) Description() string {
	return "通过 request_uuid 分析完整的请求链路。从网关日志中提取 trace-line 解析链路拓扑，从服务日志中提取 uuid.step 子请求关系，按时间排序构建请求链路视图。"
}

func (t *TraceAnalyzeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"request_uuid": {
				"type": "string",
				"description": "请求追踪 UUID，用于分析完整请求链路"
			},
			"time_range": {
				"type": "string",
				"description": "时间范围，如 last 15m, last 1h"
			}
		},
		"required": ["request_uuid"]
	}`)
}

func (t *TraceAnalyzeTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	requestUUID, _ := params["request_uuid"].(string)
	if requestUUID == "" {
		return &tool.Result{Error: "request_uuid is required"}, nil
	}
	timeRange, _ := params["time_range"].(string)
	if timeRange == "" {
		timeRange = "last 1h"
	}

	// 查询所有相关日志
	tenantID := task.TenantIDFrom(ctx)
	logs, err := t.es.QueryByRequestUUID(ctx, tenantID, requestUUID, timeRange)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("查询失败: %v", err)}, nil
	}
	if len(logs) == 0 {
		return &tool.Result{Output: fmt.Sprintf("No logs found for request_uuid=%s", requestUUID)}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Request UUID: %s (%d related logs)\n\n", requestUUID, len(logs)))

	// === 提取网关 trace-line ===
	var gatewayLogs []es.UCloudLog
	for _, log := range logs {
		if strings.Contains(log.KubernetesNamespace, "apigateway") {
			gatewayLogs = append(gatewayLogs, log)
		}
	}

	if len(gatewayLogs) > 0 {
		sb.WriteString("=== Trace Line (网关链路) ===\n")
		for _, gwLog := range gatewayLogs {
			gm, err := es.ParseGatewayMessage(gwLog.Message)
			if err != nil {
				// 无法解析网关消息，优雅降级并输出告警
				sb.WriteString(fmt.Sprintf("[WARN] 无法解析网关消息: %v\n", err))
				// 输出原始消息摘要（截断到 120 字符）
				raw := strings.ReplaceAll(gwLog.Message, "\n", " ")
				if len(raw) > 120 {
					raw = raw[:120] + "..."
				}
				sb.WriteString(fmt.Sprintf("  原始消息: %s\n\n", raw))
				continue
			}

			// 提取请求动作（从 request_uri 或 input 中）
			action := extractAction(gm)
			sb.WriteString(fmt.Sprintf("Gateway: %s (response_time=%dms, status=%v)\n",
				action,
				gm.ResponseTime,
				extractStatus(gm),
			))

			// 提取 trace-line（在 response_headers 中）
			if gm.ResponseHeaders != nil {
				if tl, ok := gm.ResponseHeaders["trace-line"]; ok {
					if tlStr, ok := tl.(string); ok && tlStr != "" {
						traceLine, _ := es.ParseTraceLine(tlStr)
						if traceLine != nil {
							// trace-line 解析有效（至少含有 Hops 或 Functions）
							if len(traceLine.Hops) == 0 && len(traceLine.Functions) == 0 {
								// 无法识别任何跳转节点，回退展示原始字符串并告警
								sb.WriteString(fmt.Sprintf("[WARN] trace-line 无法解析为已知格式，原始值: %s\n", traceLine.Raw))
							} else {
								// 输出每个 hop
								for _, hop := range traceLine.Hops {
									sb.WriteString(fmt.Sprintf("→ %s [%.3fs]\n", hop.Address, hop.LatencySec))
								}
								// 输出每个 func
								for _, fn := range traceLine.Functions {
									if fn.Hash != "" {
										sb.WriteString(fmt.Sprintf("→ %s@%s [%.3fs]\n", fn.Name, fn.Hash, fn.LatencySec))
									} else {
										sb.WriteString(fmt.Sprintf("→ %s [%.3fs]\n", fn.Name, fn.LatencySec))
									}
								}
							}
						}
					}
				}
			}
			sb.WriteByte('\n')
		}
	}

	// === 按时间排序构建请求链路视图 ===
	// 对所有日志按 @timestamp 升序排序
	sorted := make([]es.UCloudLog, len(logs))
	copy(sorted, logs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp < sorted[j].Timestamp
	})

	sb.WriteString("=== 请求链路 (按时间) ===\n")
	for _, log := range sorted {
		app := log.KubernetesLabelsApp
		if app == "" {
			app = log.KubernetesContainer
		}

		level := es.ExtractLogLevel(&log)

		// 从文本日志提取 step 信息
		stepInfo := ""
		if parsed, err := es.ParseTextLog(log.Message); err == nil && parsed.StepNumber != "" {
			stepInfo = fmt.Sprintf("[uuid.%s]", parsed.StepNumber)
		}

		// message 摘要：截断到 120 字符，去除换行
		msg := strings.ReplaceAll(log.Message, "\n", " ")
		if len(msg) > 120 {
			msg = msg[:120] + "..."
		}

		sb.WriteString(fmt.Sprintf("[%s] %s/%s [%s]%s → %s\n",
			log.Timestamp,
			log.KubernetesNamespace,
			app,
			level,
			stepInfo,
			msg,
		))
	}

	return &tool.Result{Output: sb.String()}, nil
}

// extractAction 从 GatewayMessage 中提取请求动作描述
func extractAction(gm *es.GatewayMessage) string {
	if gm == nil {
		return "unknown"
	}
	// 优先从 input 中提取 Action 参数
	if gm.Input != nil {
		if action, ok := gm.Input["Action"].(string); ok && action != "" {
			return fmt.Sprintf("%s → %s", gm.RequestURI, action)
		}
	}
	return gm.RequestURI
}

// extractStatus 从 GatewayMessage 的 response_headers 中提取 HTTP 状态码
func extractStatus(gm *es.GatewayMessage) any {
	if gm == nil || gm.ResponseHeaders == nil {
		return "unknown"
	}
	if status, ok := gm.ResponseHeaders["status"]; ok {
		return status
	}
	return "unknown"
}

var _ tool.Tool = (*TraceAnalyzeTool)(nil)
