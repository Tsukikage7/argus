package mock

import (
	"encoding/json"
	"fmt"
	"time"
)

// Scenario 定义一个故障场景
type Scenario struct {
	Name        string
	Description string
	// GenerateLogs 生成该场景的日志（仅日志，无独立 trace）
	GenerateLogs func(baseTime time.Time) []map[string]any
}

// AllScenarios 返回所有预定义的故障场景
func AllScenarios() []Scenario {
	return []Scenario{
		PaymentDBPoolExhausted(),
		InventoryOOM(),
		GatewayDiskFull(),
		UDBSlowQuery(),
		CreateUHostDemo(),
	}
}

// makeUCloudLog 生成 UCloud 格式的日志文档
func makeUCloudLog(t time.Time, svc *UCloudService, message string) map[string]any {
	return map[string]any{
		"@timestamp":              t.Format(time.RFC3339Nano),
		"message":                 message,
		"kubernetes_namespace":    svc.Namespace,
		"kubernetes_labels_app":   svc.LabelsApp,
		"kubernetes_pod":          svc.Pod,
		"kubernetes_node":         svc.Node,
		"kubernetes_container":    svc.Container,
		"host":                    svc.Host,
		"stream":                  "stdout",
		"json":                    map[string]any{},
		"log": map[string]any{
			"file": map[string]any{
				"path": fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", svc.Container, svc.Container),
			},
		},
	}
}

// makeGatewayLog 生成网关 JSON 格式日志
// message 字段是完整 JSON，包含 request_uri, response_time, trace-line, input(含 request_uuid) 等
func makeGatewayLog(t time.Time, svc *UCloudService, requestUUID string, action string, responseTime int, statusCode int, traceLine string, upstreamProduct string) map[string]any {
	// 构建 message JSON
	gatewayMsg := map[string]any{
		"request_uri":     fmt.Sprintf("/?Action=%s&request_uuid=%s", action, requestUUID),
		"gateway_latency": 0,
		"request_headers": map[string]any{
			"host":       "api.ucloud.cn",
			"user_agent": "Go-http-client/1.1",
		},
		"request_time":  t.Unix(),
		"response_time": responseTime,
		"remote_ip":     "10.81.98.139",
		"remote_port":   "41325",
		"response_headers": map[string]any{
			"status":                   statusCode,
			"trace-line":               traceLine,
			"x-gray-gw-product":        upstreamProduct,
			"x-gray-gw-upstream":       "Default",
			"x-gray-gw-upstream-type":  "Stable",
		},
		"input": map[string]any{
			"Action":       action,
			"request_uuid": requestUUID,
		},
		"output":   map[string]any{},
		"end_time": t.Add(time.Duration(responseTime) * time.Millisecond).Unix(),
	}
	msgBytes, _ := json.Marshal(gatewayMsg)

	doc := makeUCloudLog(t, svc, string(msgBytes))
	// 网关日志的 json 字段包含 fluentd 解析副本
	doc["json"] = map[string]any{
		"response_time": fmt.Sprintf("%d", responseTime),
		"request_uri":   gatewayMsg["request_uri"],
		"input":         fmt.Sprintf("%v", gatewayMsg["input"]),
	}
	return doc
}

// makeTextLog 生成文本格式日志 [timestamp] [LEVEL][uuid.step] content
func makeTextLog(t time.Time, svc *UCloudService, level, requestUUID string, step int, content string) map[string]any {
	message := fmt.Sprintf("[%s] [%s][%s.%d] %s",
		t.Format("2006-01-02 15:04:05.000000"), level, requestUUID, step, content)
	return makeUCloudLog(t, svc, message)
}

// makeStructuredLog 生成结构化 JSON 日志
func makeStructuredLog(t time.Time, svc *UCloudService, level, operation string, latencyMs int, traceID, spanID string) map[string]any {
	structMsg := map[string]any{
		"level":     level,
		"operation": operation,
		"latency":   latencyMs,
		"trace_id":  traceID,
		"span_id":   spanID,
		"timestamp": t.Format(time.RFC3339Nano),
	}
	msgBytes, _ := json.Marshal(structMsg)
	return makeUCloudLog(t, svc, string(msgBytes))
}

// PaymentDBPoolExhausted 场景 1: prj-ubill 数据库连接池耗尽
// 模拟 ubill 的数据库连接池耗尽，导致 BuyResource 操作超时
// prj-apigateway 的 trace-line 显示高延迟，使用 request_uuid 作为追踪主键
func PaymentDBPoolExhausted() Scenario {
	return Scenario{
		Name:        "payment-db-pool-exhausted",
		Description: "prj-ubill 数据库连接池耗尽，导致 BuyResource 操作超时，网关 trace-line 显示 30+ 秒延迟",
		GenerateLogs: func(baseTime time.Time) []map[string]any {
			var logs []map[string]any

			ubill := ServiceByNamespace("prj-ubill")
			gw := ServiceByNamespace("prj-apigateway")

			// === 正常基线（故障前 30 分钟）: 200 组正常请求 ===
			for i := 0; i < 200; i++ {
				t := baseTime.Add(-30*time.Minute + time.Duration(i)*9*time.Second)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-929d0e0e%04x", i+10000)
				traceLine := fmt.Sprintf("10.69.186.2:8080 T 0.150")

				// 网关正常日志
				logs = append(logs, makeGatewayLog(t, gw, uuid, "BuyResource", 200, 200, traceLine, "UBill"))
				// ubill 正常文本日志
				logs = append(logs, makeTextLog(t.Add(5*time.Millisecond), ubill, "INFO", uuid, 1, "BuyResource start"))
				logs = append(logs, makeTextLog(t.Add(100*time.Millisecond), ubill, "INFO", uuid, 2, "latency: 150ms"))
			}

			// === 慢查询预兆（故障前 10 分钟）: 5 条 ubill WARN ===
			for i := 0; i < 5; i++ {
				t := baseTime.Add(-10*time.Minute + time.Duration(i)*2*time.Minute)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-51ow0000%04x", i)
				logs = append(logs, makeTextLog(t, ubill, "WARN", uuid, 1,
					fmt.Sprintf("slow query detected: SELECT * FROM orders WHERE created_at > '2025-01-01' duration=%ds", 30+i*5),
				))
			}

			// === 连接池压力（故障前 5 分钟）: 20 条 ubill WARN ===
			for i := 0; i < 20; i++ {
				t := baseTime.Add(-5*time.Minute + time.Duration(i)*15*time.Second)
				active := 30 + i
				idle := 20 - i
				if idle < 0 {
					idle = 0
				}
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-900100000%03x", i)
				logs = append(logs, makeTextLog(t, ubill, "WARN", uuid, 1,
					fmt.Sprintf("db connection pool pressure: active=%d, idle=%d, max=50", active, idle),
				))
			}

			// === 故障高峰（最近 5 分钟）: 50 组完整链路 ===
			for i := 0; i < 50; i++ {
				t := baseTime.Add(-time.Duration(50-i) * 6 * time.Second)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-929d0e0e%04x", i)
				// trace-line 显示 30+ 秒延迟
				traceLine := fmt.Sprintf("10.69.186.2:8080 T 30.%03d", i)

				// ubill ERROR: 连接池耗尽
				logs = append(logs, makeTextLog(t, ubill, "ERROR", uuid, 1,
					"failed to acquire db connection: pool exhausted, active=50, idle=0, wait_timeout=30s",
				))
				// 网关 504 日志
				logs = append(logs, makeGatewayLog(
					t.Add(30*time.Second), gw, uuid, "BuyResource", 30000+i*10, 504, traceLine, "UBill",
				))
			}

			return logs
		},
	}
}

// InventoryOOM 场景 2: prj-uresource OOM
// 模拟 uresource 内存泄漏导致 OOM 被 kill，级联影响 uhost
func InventoryOOM() Scenario {
	return Scenario{
		Name:        "inventory-oom",
		Description: "prj-uresource 内存泄漏导致 OOM kill，级联导致 prj-uhost 调用失败",
		GenerateLogs: func(baseTime time.Time) []map[string]any {
			var logs []map[string]any

			uresource := ServiceByNamespace("prj-uresource")
			uhost := ServiceByNamespace("prj-uhost")
			gw := ServiceByNamespace("prj-apigateway")

			// === 内存逐渐升高（故障前 30 分钟）: 30 条结构化 WARN ===
			for i := 0; i < 30; i++ {
				t := baseTime.Add(-30*time.Minute + time.Duration(i)*time.Minute)
				memPct := 50 + i*2
				logs = append(logs, makeStructuredLog(t, uresource, "WARN", "memory_check", 0,
					fmt.Sprintf("mem-%04d", i), fmt.Sprintf("span-%04d", i),
				))
				// 补充文本说明内存百分比
				logs = append(logs, makeTextLog(t.Add(time.Millisecond), uresource, "WARN",
					fmt.Sprintf("mem-warn-%04d", i), 1,
					fmt.Sprintf("high memory usage: %d%%, heap=%.1fGB", memPct, float64(memPct)/100*4),
				))
			}

			// === OOM 事件 ===
			logs = append(logs, makeStructuredLog(
				baseTime.Add(-2*time.Minute), uresource, "ERROR", "runtime", 0, "4bf92f35-00ab-4da6-a3ce-000100000001", "span-00ab-0001",
			))
			logs = append(logs, makeTextLog(
				baseTime.Add(-2*time.Minute+time.Second), uresource, "ERROR", "4bf92f35-00ab-4da6-a3ce-000100000002", 1,
				"process killed by OOM killer: out of memory allocating 1073741824 bytes, exit code 137",
			))

			// === 级联失败（故障前 2 分钟）: 30 组 ===
			for i := 0; i < 30; i++ {
				t := baseTime.Add(-2*time.Minute + time.Duration(i)*4*time.Second)
				uuid := fmt.Sprintf("4bf92f35-00ab-4da6-a3ce-000200000%03x", i)
				traceLine := fmt.Sprintf("10.69.186.10:8080 T 0.200")

				// uhost ERROR: 连接拒绝
				logs = append(logs, makeTextLog(t, uhost, "ERROR", uuid, 1,
					"connection refused to uresource: dial tcp 10.69.188.5:8080: connect: connection refused",
				))
				// 网关 500 日志
				logs = append(logs, makeGatewayLog(
					t.Add(100*time.Millisecond), gw, uuid, "DescribeUHostInstance", 200, 500, traceLine, "UHost",
				))
			}

			return logs
		},
	}
}

// GatewayDiskFull 场景 3: prj-apigateway 磁盘满
// 模拟网关自身日志写满磁盘，间歇性返回 502
func GatewayDiskFull() Scenario {
	return Scenario{
		Name:        "gateway-disk-full",
		Description: "prj-apigateway 日志写满磁盘，间歇性 502",
		GenerateLogs: func(baseTime time.Time) []map[string]any {
			var logs []map[string]any

			gw := ServiceByNamespace("prj-apigateway")

			// === 磁盘使用率升高（故障前 60 分钟）: 20 条 WARN ===
			for i := 0; i < 20; i++ {
				t := baseTime.Add(-60*time.Minute + time.Duration(i)*3*time.Minute)
				diskPct := 80 + i
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-d15c0000%04x", i)
				// 使用文本格式记录磁盘状态
				logs = append(logs, makeTextLog(t, gw, "WARN", uuid, 1,
					fmt.Sprintf("disk usage high: /var/log at %d%%, available=%dMB", diskPct, (100-diskPct)*100),
				))
			}

			// === 磁盘满，写入失败 ===
			logs = append(logs, makeTextLog(
				baseTime.Add(-5*time.Minute), gw, "ERROR", "4bf92f35-77b3-4da6-a3ce-d15cf0110001", 1,
				"failed to write access log: no space left on device",
			))

			// === 间歇性 502（故障前 5 分钟）: 40 条，三分之一失败 ===
			for i := 0; i < 40; i++ {
				t := baseTime.Add(-5*time.Minute + time.Duration(i)*7*time.Second)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-d15c00f0%04x", i)
				if i%3 == 0 {
					// 失败请求：502
					traceLine := fmt.Sprintf("10.69.202.25:80 T 0.005")
					logs = append(logs, makeGatewayLog(
						t, gw, uuid, "DescribeUHostInstance", 5, 502, traceLine, "UHost",
					))
					// 附加错误说明文本日志
					logs = append(logs, makeTextLog(t.Add(time.Millisecond), gw, "ERROR", uuid, 1,
						"failed to handle request: write /var/log/access.log: no space left on device, returning 502",
					))
				}
			}

			return logs
		},
	}
}

// UDBSlowQuery 场景 4: prj-udb 数据库慢查询
// 模拟 go-udb-http 慢查询导致连接池压力，网关 trace-line 显示高延迟
func UDBSlowQuery() Scenario {
	return Scenario{
		Name:        "udb-slow-query",
		Description: "prj-udb 数据库慢查询导致连接池告警，go-udb-http 响应超时",
		GenerateLogs: func(baseTime time.Time) []map[string]any {
			var logs []map[string]any

			udb := ServiceByNamespace("prj-udb")
			gw := ServiceByNamespace("prj-apigateway")

			// 正常基线（故障前 30 分钟）
			for i := 0; i < 100; i++ {
				t := baseTime.Add(-30*time.Minute + time.Duration(i)*18*time.Second)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-udb00000%04x", i+5000)
				traceLine := "10.69.192.7:8080 T 0.080"
				logs = append(logs, makeGatewayLog(t, gw, uuid, "DescribeUDBInstance", 120, 200, traceLine, "UDB"))
				logs = append(logs, makeTextLog(t.Add(5*time.Millisecond), udb, "INFO", uuid, 1, "DescribeUDBInstance start"))
				logs = append(logs, makeTextLog(t.Add(80*time.Millisecond), udb, "INFO", uuid, 2, "query completed, latency: 80ms"))
			}

			// 慢查询预兆（故障前 10 分钟）
			for i := 0; i < 10; i++ {
				t := baseTime.Add(-10*time.Minute + time.Duration(i)*time.Minute)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-udbsw000%04x", i)
				logs = append(logs, makeTextLog(t, udb, "WARN", uuid, 1,
					fmt.Sprintf("slow query detected: SELECT * FROM udb_instance WHERE region_id='cn-bj2' duration=%ds", 5+i*3)))
			}

			// 连接池压力（故障前 5 分钟）
			for i := 0; i < 15; i++ {
				t := baseTime.Add(-5*time.Minute + time.Duration(i)*20*time.Second)
				active := 40 + i*3
				if active > 95 {
					active = 95
				}
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-udbpl000%04x", i)
				logs = append(logs, makeTextLog(t, udb, "WARN", uuid, 1,
					fmt.Sprintf("db connection pool pressure: active=%d, idle=%d, max=100", active, 100-active)))
			}

			// 故障高峰（最近 3 分钟）
			for i := 0; i < 30; i++ {
				t := baseTime.Add(-3*time.Minute - 15*time.Second + time.Duration(i)*6*time.Second)
				uuid := fmt.Sprintf("4bf92f35-77b3-4da6-a3ce-udber000%04x", i)
				traceLine := fmt.Sprintf("10.69.192.7:8080 T 15.%03d", i*100)
				logs = append(logs, makeTextLog(t, udb, "ERROR", uuid, 1,
					"failed to acquire db connection: pool exhausted, active=95, idle=0, wait_timeout=15s"))
				logs = append(logs, makeTextLog(t.Add(time.Second), udb, "ERROR", uuid, 2,
					"DescribeUDBInstance timeout after 15s, client_ip=10.69.202.25"))
				gwTime := t.Add(15 * time.Second)
				if gwTime.After(baseTime) {
					gwTime = baseTime.Add(-time.Second)
				}
				logs = append(logs, makeGatewayLog(
					gwTime, gw, uuid, "DescribeUDBInstance", 15000+i*50, 504, traceLine, "UDB"))
			}

			return logs
		},
	}
}
