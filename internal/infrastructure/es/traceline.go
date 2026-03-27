package es

import (
	"regexp"
	"strconv"
	"strings"
)

// hopIPv4Pattern 匹配 IPv4:Port 格式的跳转节点
// 格式: IP:PORT T latency
var hopIPv4Pattern = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+:\d+)\s+T\s+([\d.]+)$`)

// hopIPv6Pattern 匹配 [IPv6]:Port 格式的跳转节点
// 格式: [IPv6]:PORT T latency
var hopIPv6Pattern = regexp.MustCompile(`^(\[[^\]]+\]:\d+)\s+T\s+([\d.]+)$`)

// funcPattern 匹配函数调用节点
// 格式: FuncName(@hash)?: latency 或 FuncName: latency
var funcPattern = regexp.MustCompile(`^([\w.]+)(?:@([0-9a-f]+))?:\s*([\d.]+)$`)

// ParseTraceLine 解析网关 trace-line 字符串
// 支持格式：
// - IPv4 单跳: "10.69.204.214:4001 T 3.432"
// - IPv6 单跳: "[2002:a40:23d:1::449b]:4210 T 0.160"
// - 多跳: "10.1.1.1:8080 T 0.5 -> 10.2.2.2:9090 T 1.2"
// - 混合: "10.1.1.1:8080 T 0.5 -> FuncName: 0.3 -> 10.2.2.2:9090 T 1.2"
// 解析失败时优雅降级，返回只含 Raw 字段的 TraceLine，不报错
func ParseTraceLine(traceLine string) (*TraceLine, error) {
	result := &TraceLine{
		Raw: traceLine,
	}

	trimmed := strings.TrimSpace(traceLine)
	if trimmed == "" {
		return result, nil
	}

	// 以 " -> " 分割各段
	segments := strings.Split(trimmed, " -> ")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		if hop, ok := parseHopSegment(seg); ok {
			result.Hops = append(result.Hops, hop)
			continue
		}

		if fn, ok := parseFuncSegment(seg); ok {
			result.Functions = append(result.Functions, fn)
			continue
		}

		// 单段无法识别，继续解析其余段（容错）
	}

	return result, nil
}

// parseHopSegment 尝试将一段文本解析为 IP:Port 跳转节点
func parseHopSegment(seg string) (TraceHop, bool) {
	// 尝试 IPv4 格式
	if m := hopIPv4Pattern.FindStringSubmatch(seg); m != nil {
		latency, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return TraceHop{}, false
		}
		return TraceHop{Address: m[1], LatencySec: latency}, true
	}

	// 尝试 IPv6 格式
	if m := hopIPv6Pattern.FindStringSubmatch(seg); m != nil {
		latency, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return TraceHop{}, false
		}
		return TraceHop{Address: m[1], LatencySec: latency}, true
	}

	return TraceHop{}, false
}

// parseFuncSegment 尝试将一段文本解析为函数调用节点
func parseFuncSegment(seg string) (TraceFunction, bool) {
	if m := funcPattern.FindStringSubmatch(seg); m != nil {
		latency, err := strconv.ParseFloat(m[3], 64)
		if err != nil {
			return TraceFunction{}, false
		}
		return TraceFunction{
			Name:       m[1],
			Hash:       m[2],
			LatencySec: latency,
		}, true
	}
	return TraceFunction{}, false
}
