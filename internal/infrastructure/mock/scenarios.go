package mock

import (
	"fmt"
	"time"
)

// Scenario 定义一个故障场景
type Scenario struct {
	Name        string
	Description string
	// GenerateLogs 生成该场景的日志（正常 + 异常）
	GenerateLogs func(baseTime time.Time) (logs []map[string]any, traces []map[string]any)
}

// AllScenarios 返回所有预定义的故障场景
func AllScenarios() []Scenario {
	return []Scenario{
		PaymentDBPoolExhausted(),
		InventoryOOM(),
		GatewayDiskFull(),
	}
}

// PaymentDBPoolExhausted 场景 1: payment-service 数据库连接池耗尽
func PaymentDBPoolExhausted() Scenario {
	return Scenario{
		Name:        "payment-db-pool-exhausted",
		Description: "payment-service 数据库连接池耗尽，导致 order-service 调用超时",
		GenerateLogs: func(baseTime time.Time) ([]map[string]any, []map[string]any) {
			var logs []map[string]any
			var traces []map[string]any

			payment := ServiceByName("payment-service")
			order := ServiceByName("order-service")
			gw := ServiceByName("gateway")

			// === 正常基线日志（故障前 30 分钟） ===
			for i := 0; i < 200; i++ {
				t := baseTime.Add(-30*time.Minute + time.Duration(i)*9*time.Second)
				traceID := fmt.Sprintf("normal%032d", i)

				// gateway 正常
				logs = append(logs, makeLog(t, gw, "INFO",
					fmt.Sprintf("request completed: POST /api/v1/orders, status=200, duration=120ms"),
					traceID, fmt.Sprintf("gw%016d", i), "",
					map[string]any{"http.method": "POST", "http.url": "/api/v1/orders", "http.status_code": 200},
				))
				// order-service 正常
				logs = append(logs, makeLog(t.Add(5*time.Millisecond), order, "INFO",
					"order created successfully, calling payment-service",
					traceID, fmt.Sprintf("ord%015d", i), fmt.Sprintf("gw%016d", i),
					map[string]any{"http.method": "POST", "http.url": "/api/v1/pay", "http.status_code": 200},
				))
				// payment-service 正常
				logs = append(logs, makeLog(t.Add(50*time.Millisecond), payment, "INFO",
					"payment processed successfully, amount=99.00",
					traceID, fmt.Sprintf("pay%015d", i), fmt.Sprintf("ord%015d", i),
					map[string]any{"http.method": "POST", "http.url": "/api/v1/pay", "http.status_code": 200, "db.connection_pool.active": 10, "db.connection_pool.idle": 40},
				))
			}

			// === 慢查询开始（故障前 10 分钟） ===
			for i := 0; i < 5; i++ {
				t := baseTime.Add(-10*time.Minute + time.Duration(i)*2*time.Minute)
				logs = append(logs, makeLog(t, payment, "WARN",
					fmt.Sprintf("slow query detected: SELECT * FROM orders WHERE created_at > '2025-01-01' AND status IN ('pending','processing') ORDER BY amount DESC, duration=%ds", 30+i*5),
					"", "", "",
					map[string]any{"db.system": "mysql", "db.statement": "SELECT * FROM orders WHERE ...", "db.duration_ms": (30 + i*5) * 1000},
				))
			}

			// === 连接池逐渐耗尽 ===
			for i := 0; i < 20; i++ {
				t := baseTime.Add(-5*time.Minute + time.Duration(i)*15*time.Second)
				active := 30 + i
				idle := 20 - i
				if idle < 0 {
					idle = 0
				}
				logs = append(logs, makeLog(t, payment, "WARN",
					fmt.Sprintf("db connection pool pressure: active=%d, idle=%d, max=50", active, idle),
					"", "", "",
					map[string]any{"db.system": "mysql", "db.connection_pool.active": active, "db.connection_pool.idle": idle},
				))
			}

			// === 故障高峰（最近 5 分钟） ===
			for i := 0; i < 50; i++ {
				t := baseTime.Add(-time.Duration(50-i) * 6 * time.Second)
				traceID := fmt.Sprintf("4bf92f3577b34da6a3ce929d0e0e%04x", i)
				gwSpan := fmt.Sprintf("gwerr%011d", i)
				ordSpan := fmt.Sprintf("orderrr%09d", i)
				paySpan := fmt.Sprintf("payerr%010d", i)

				// payment-service ERROR
				logs = append(logs, makeLog(t, payment, "ERROR",
					"failed to acquire db connection: pool exhausted, active=50, idle=0, wait_timeout=30s",
					traceID, paySpan, ordSpan,
					map[string]any{
						"http.method": "POST", "http.url": "/api/v1/pay", "http.status_code": 500,
						"db.system": "mysql", "db.connection_pool.active": 50, "db.connection_pool.idle": 0,
						"error.type": "ConnectionPoolExhausted",
					},
				))

				// order-service ERROR
				logs = append(logs, makeLog(t.Add(30*time.Second), order, "ERROR",
					fmt.Sprintf("timeout calling payment-service /api/v1/pay: context deadline exceeded after 30s, trace_id=%s", traceID),
					traceID, ordSpan, gwSpan,
					map[string]any{"http.method": "POST", "http.url": "/api/v1/orders", "http.status_code": 504, "error.type": "UpstreamTimeout"},
				))

				// gateway 504
				logs = append(logs, makeLog(t.Add(30*time.Second+100*time.Millisecond), gw, "ERROR",
					fmt.Sprintf("upstream timeout: POST /api/v1/orders → order-service, status=504"),
					traceID, gwSpan, "",
					map[string]any{"http.method": "POST", "http.url": "/api/v1/orders", "http.status_code": 504},
				))

				// 完整 trace
				traces = append(traces, makeTrace(traceID, gwSpan, "", gw, 30100, "ERROR"))
				traces = append(traces, makeTrace(traceID, ordSpan, gwSpan, order, 30050, "ERROR"))
				traces = append(traces, makeTrace(traceID, paySpan, ordSpan, payment, 30000, "ERROR"))
			}

			return logs, traces
		},
	}
}

// InventoryOOM 场景 2: inventory-service OOM
func InventoryOOM() Scenario {
	return Scenario{
		Name:        "inventory-oom",
		Description: "inventory-service 内存泄漏导致 OOM，库存扣减失败",
		GenerateLogs: func(baseTime time.Time) ([]map[string]any, []map[string]any) {
			var logs []map[string]any
			var traces []map[string]any

			inv := ServiceByName("inventory-service")
			order := ServiceByName("order-service")
			gw := ServiceByName("gateway")

			// 内存逐渐升高
			for i := 0; i < 30; i++ {
				t := baseTime.Add(-30*time.Minute + time.Duration(i)*time.Minute)
				memPct := 50.0 + float64(i)*1.5
				logs = append(logs, makeLog(t, inv, "WARN",
					fmt.Sprintf("high memory usage: %.0f%%, heap=%.1fGB", memPct, float64(memPct)/100*4),
					"", "", "",
					map[string]any{"host.memory_pct": memPct, "process.heap_bytes": int64(float64(memPct) / 100 * 4 * 1024 * 1024 * 1024)},
				))
			}

			// OOM 事件
			logs = append(logs, makeLog(baseTime.Add(-2*time.Minute), inv, "ERROR",
				"runtime: out of memory allocating 1073741824 bytes",
				"", "", "",
				map[string]any{"error.type": "OutOfMemory"},
			))
			logs = append(logs, makeLog(baseTime.Add(-2*time.Minute+time.Second), inv, "ERROR",
				"process killed by OOM killer, exit code 137",
				"", "", "",
				map[string]any{"error.type": "OOMKilled", "process.exit_code": 137},
			))

			// 级联失败
			for i := 0; i < 30; i++ {
				t := baseTime.Add(-2*time.Minute + time.Duration(i)*4*time.Second)
				traceID := fmt.Sprintf("oom%029d", i)
				logs = append(logs, makeLog(t, order, "ERROR",
					"failed to deduct inventory: connection refused to inventory-service",
					traceID, fmt.Sprintf("oord%012d", i), fmt.Sprintf("ogw%013d", i),
					map[string]any{"error.type": "ConnectionRefused"},
				))
				logs = append(logs, makeLog(t.Add(100*time.Millisecond), gw, "ERROR",
					"upstream error: POST /api/v1/orders → order-service, status=500",
					traceID, fmt.Sprintf("ogw%013d", i), "",
					map[string]any{"http.status_code": 500},
				))

				traces = append(traces, makeTrace(traceID, fmt.Sprintf("ogw%013d", i), "", gw, 200, "ERROR"))
				traces = append(traces, makeTrace(traceID, fmt.Sprintf("oord%012d", i), fmt.Sprintf("ogw%013d", i), order, 150, "ERROR"))
			}

			return logs, traces
		},
	}
}

// GatewayDiskFull 场景 3: gateway 磁盘空间不足
func GatewayDiskFull() Scenario {
	return Scenario{
		Name:        "gateway-disk-full",
		Description: "gateway 日志写满磁盘，导致间歇性 502",
		GenerateLogs: func(baseTime time.Time) ([]map[string]any, []map[string]any) {
			var logs []map[string]any
			var traces []map[string]any

			gw := ServiceByName("gateway")

			// 磁盘使用率逐渐升高
			for i := 0; i < 20; i++ {
				t := baseTime.Add(-60*time.Minute + time.Duration(i)*3*time.Minute)
				diskPct := 80 + i
				logs = append(logs, makeLog(t, gw, "WARN",
					fmt.Sprintf("disk usage high: /var/log at %d%%, available=%dMB", diskPct, (100-diskPct)*100),
					"", "", "",
					map[string]any{"host.disk_pct": diskPct, "host.disk_path": "/var/log"},
				))
			}

			// 磁盘满，写入失败
			logs = append(logs, makeLog(baseTime.Add(-5*time.Minute), gw, "ERROR",
				"failed to write access log: no space left on device",
				"", "", "",
				map[string]any{"error.type": "ENOSPC", "host.disk_path": "/var/log/access.log"},
			))

			// 间歇性 502
			for i := 0; i < 40; i++ {
				t := baseTime.Add(-5*time.Minute + time.Duration(i)*7*time.Second)
				traceID := fmt.Sprintf("disk%028d", i)
				if i%3 == 0 { // 三分之一失败
					logs = append(logs, makeLog(t, gw, "ERROR",
						"failed to handle request: write /var/log/access.log: no space left on device, returning 502",
						traceID, fmt.Sprintf("dgw%013d", i), "",
						map[string]any{"http.status_code": 502, "error.type": "ENOSPC"},
					))
					traces = append(traces, makeTrace(traceID, fmt.Sprintf("dgw%013d", i), "", gw, 5, "ERROR"))
				}
			}

			return logs, traces
		},
	}
}

func makeLog(t time.Time, svc *Service, severity, body, traceID, spanID, parentSpanID string, attrs map[string]any) map[string]any {
	doc := map[string]any{
		"@timestamp": t.Format(time.RFC3339Nano),
		"service": map[string]any{
			"name":        svc.Name,
			"version":     svc.Version,
			"instance_id": svc.InstanceID,
		},
		"severity": severity,
		"body":     body,
		"resource": map[string]any{
			"k8s.namespace": svc.Namespace,
			"k8s.pod.name":  svc.PodName,
			"k8s.node.name": svc.NodeName,
		},
	}
	if traceID != "" {
		doc["trace_id"] = traceID
	}
	if spanID != "" {
		doc["span_id"] = spanID
	}
	if parentSpanID != "" {
		doc["parent_span_id"] = parentSpanID
	}
	if attrs != nil {
		allAttrs := map[string]any{
			"host.name": svc.InstanceID,
			"host.ip":   svc.HostIP,
		}
		for k, v := range attrs {
			allAttrs[k] = v
		}
		doc["attributes"] = allAttrs
	}
	return doc
}

func makeTrace(traceID, spanID, parentSpanID string, svc *Service, durationMs int, status string) map[string]any {
	doc := map[string]any{
		"trace_id":    traceID,
		"span_id":     spanID,
		"service":     svc.Name,
		"duration":    time.Duration(durationMs) * time.Millisecond,
		"status":      status,
		"@timestamp":  time.Now().Format(time.RFC3339Nano),
	}
	if parentSpanID != "" {
		doc["parent_span_id"] = parentSpanID
	}
	return doc
}
