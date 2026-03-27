package es

import (
	"encoding/json"
	"regexp"
	"strings"
)

// uuidPattern 匹配标准 UUID 格式（不含 step 后缀）
var uuidPattern = regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)

// textLogSimplePattern 匹配简单格式文本日志
// 格式: [timestamp] [LEVEL][uuid.step] content
var textLogSimplePattern = regexp.MustCompile(`^\[([^\]]+)\]\s*\[(\w+)\]\[([a-f0-9-]+(?:\.\d+)?)\]\s*(.*)$`)

// textLogHttpRequestPattern 匹配 HttpRequest 格式文本日志
// 格式: [timestamp] [LEVEL][HttpRequest(uuid.step)|(FuncName)] content
var textLogHttpRequestPattern = regexp.MustCompile(`^\[([^\]]+)\]\s*\[(\w+)\]\[HttpRequest\(([a-f0-9-]+(?:\.\d+)?)\)\|(?:\(([^)]+)\))?\]\s*(.*)$`)

// ParseMessage 识别 message 字段的日志类型
// 策略：
// 1. 尝试 JSON 解析，包含 request_uri → MessageTypeGateway
// 2. 尝试 JSON 解析，包含 level 或 trace_id → MessageTypeStructured
// 3. 尝试文本正则匹配 → MessageTypeText
// 4. 其余 → MessageTypeUnknown
func ParseMessage(message string) MessageType {
	trimmed := strings.TrimSpace(message)
	if strings.HasPrefix(trimmed, "{") {
		var raw map[string]any
		if err := json.Unmarshal([]byte(trimmed), &raw); err == nil {
			if _, ok := raw["request_uri"]; ok {
				return MessageTypeGateway
			}
			_, hasLevel := raw["level"]
			_, hasTraceID := raw["trace_id"]
			if hasLevel || hasTraceID {
				return MessageTypeStructured
			}
		}
	}

	if textLogSimplePattern.MatchString(trimmed) || textLogHttpRequestPattern.MatchString(trimmed) {
		return MessageTypeText
	}

	return MessageTypeUnknown
}

// ParseGatewayMessage 解析网关 JSON 日志（Type A: prj-apigateway）
func ParseGatewayMessage(message string) (*GatewayMessage, error) {
	var gm GatewayMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(message)), &gm); err != nil {
		return nil, err
	}
	return &gm, nil
}

// ParseTextLog 解析文本格式日志（Type B: prj-ubill 等服务）
// 支持两种变体：
// 1. 简单格式: [timestamp] [LEVEL][uuid.step] content
// 2. HttpRequest 格式: [timestamp] [LEVEL][HttpRequest(uuid.step)|(FuncName)] content
func ParseTextLog(message string) (*TextLogParsed, error) {
	trimmed := strings.TrimSpace(message)

	// 优先匹配 HttpRequest 格式（更具体）
	if m := textLogHttpRequestPattern.FindStringSubmatch(trimmed); m != nil {
		parsed := &TextLogParsed{
			Timestamp: m[1],
			Level:     m[2],
			FuncName:  m[4],
			Content:   m[5],
		}
		parsed.RequestUUID, parsed.StepNumber = splitUUIDStep(m[3])
		return parsed, nil
	}

	// 匹配简单格式
	if m := textLogSimplePattern.FindStringSubmatch(trimmed); m != nil {
		parsed := &TextLogParsed{
			Timestamp: m[1],
			Level:     m[2],
			Content:   m[4],
		}
		parsed.RequestUUID, parsed.StepNumber = splitUUIDStep(m[3])
		return parsed, nil
	}

	return nil, &parseError{msg: "不匹配任何文本日志格式: " + trimmed}
}

// ParseStructuredLog 解析结构化 JSON 日志（Type C: 含 level/trace_id 字段）
func ParseStructuredLog(message string) (*StructuredLogParsed, error) {
	var sl StructuredLogParsed
	if err := json.Unmarshal([]byte(strings.TrimSpace(message)), &sl); err != nil {
		return nil, err
	}
	return &sl, nil
}

// ExtractRequestUUID 从任意类型 message 中提取 request_uuid
// 策略：
// 1. 在 JSON 中查找 "request_uuid" 键
// 2. 尝试文本正则匹配 [uuid.step] 格式
// 3. 用正则搜索任意 UUID 格式
func ExtractRequestUUID(message string) string {
	trimmed := strings.TrimSpace(message)

	// 先尝试从 JSON 的 input 或顶层中提取 request_uuid
	if strings.HasPrefix(trimmed, "{") {
		var raw map[string]any
		if err := json.Unmarshal([]byte(trimmed), &raw); err == nil {
			if v := extractStringField(raw, "request_uuid"); v != "" {
				return v
			}
			// 网关日志的 request_uuid 在 input 字段中
			if input, ok := raw["input"].(map[string]any); ok {
				if v := extractStringField(input, "request_uuid"); v != "" {
					return v
				}
			}
		}
	}

	// 尝试文本日志格式解析
	if parsed, err := ParseTextLog(trimmed); err == nil && parsed.RequestUUID != "" {
		return parsed.RequestUUID
	}

	// 兜底：正则搜索 UUID
	if m := uuidPattern.FindString(trimmed); m != "" {
		return m
	}

	return ""
}

// ExtractLogLevel 从 UCloudLog 中提取日志级别
// 策略：
// 1. 先检查 log.JSON 中是否有 "level" 字段
// 2. 尝试 ParseMessage 判断类型后提取
// 3. 文本日志从 [LEVEL] 提取
// 4. 网关日志检查 response_headers 中的 status
// 5. 默认返回 "INFO"
func ExtractLogLevel(log *UCloudLog) string {
	// 优先从 ES json 字段提取
	if log.JSON != nil {
		if v := extractStringField(log.JSON, "level"); v != "" {
			return strings.ToUpper(v)
		}
	}

	msgType := ParseMessage(log.Message)
	switch msgType {
	case MessageTypeText:
		if parsed, err := ParseTextLog(log.Message); err == nil {
			return strings.ToUpper(parsed.Level)
		}
	case MessageTypeStructured:
		if parsed, err := ParseStructuredLog(log.Message); err == nil && parsed.Level != "" {
			return strings.ToUpper(parsed.Level)
		}
	case MessageTypeGateway:
		if gm, err := ParseGatewayMessage(log.Message); err == nil {
			if gm.ResponseHeaders != nil {
				if status, ok := gm.ResponseHeaders["status"]; ok {
					// HTTP 5xx 视为 ERROR，4xx 视为 WARN，其余 INFO
					switch v := status.(type) {
					case float64:
						return httpStatusToLevel(int(v))
					case int:
						return httpStatusToLevel(v)
					}
				}
			}
		}
	}

	return "INFO"
}

// splitUUIDStep 将 "uuid.step" 格式分割为 uuid 和 step
// 从最后一个 "." 分割，避免误切 UUID 内部的字符
func splitUUIDStep(raw string) (uuid, step string) {
	idx := strings.LastIndex(raw, ".")
	if idx < 0 {
		return raw, ""
	}
	return raw[:idx], raw[idx+1:]
}

// extractStringField 安全地从 map 中提取字符串字段
func extractStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// httpStatusToLevel 将 HTTP 状态码映射为日志级别
func httpStatusToLevel(status int) string {
	switch {
	case status >= 500:
		return "ERROR"
	case status >= 400:
		return "WARN"
	default:
		return "INFO"
	}
}

// parseError 解析错误类型
type parseError struct {
	msg string
}

func (e *parseError) Error() string {
	return e.msg
}
