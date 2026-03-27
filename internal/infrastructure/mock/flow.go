package mock

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateCallChain 生成一次完整 API 调用链路的日志
// scenario 为 nil 时生成正常流量，非 nil 时按故障场景生成差异化错误日志
// 模拟: apigateway → 后端服务（ubill/uresource/uhost）→ 下游服务（unet/udb）
func GenerateCallChain(baseTime time.Time, requestUUID string, scenario *Scenario) []map[string]any {
	var logs []map[string]any

	gw := ServiceByNamespace("prj-apigateway").PickInstance()

	// 随机选择一个主要后端服务
	backends := []string{"prj-ubill", "prj-uhost", "prj-uresource"}
	backendNS := backends[rand.Intn(len(backends))]
	backend := ServiceByNamespace(backendNS).PickInstance()

	// 随机选择一个下游服务
	downstreams := []string{"prj-unet", "prj-udb"}
	downstreamNS := downstreams[rand.Intn(len(downstreams))]
	downstream := ServiceByNamespace(downstreamNS).PickInstance()

	// 对应的 API Action
	actions := map[string]string{
		"prj-ubill":     "BuyResource",
		"prj-uhost":     "DescribeUHostInstance",
		"prj-uresource": "DescribeResource",
	}
	action := actions[backendNS]
	product := map[string]string{
		"prj-ubill": "UBill", "prj-uhost": "UHost", "prj-uresource": "UResource",
	}[backendNS]

	if scenario != nil {
		// 根据场景名称生成差异化故障日志
		switch scenario.Name {
		case "payment-db-pool-exhausted":
			// 场景 1: 数据库连接池耗尽 → 超时 30s+，504
			ubill := ServiceByNamespace("prj-ubill").PickInstance()
			responseTime := 30000 + rand.Intn(5000)
			traceLine := fmt.Sprintf("%s:8080 T %d.%03d", ubill.Host, responseTime/1000, responseTime%1000)
			logs = append(logs, makeTextLog(baseTime, ubill, "ERROR", requestUUID, 1,
				"failed to acquire db connection: pool exhausted, active=50, idle=0, wait_timeout=30s"))
			logs = append(logs, makeGatewayLog(
				baseTime.Add(time.Duration(responseTime)*time.Millisecond),
				gw, requestUUID, "BuyResource", responseTime, 504, traceLine, "UBill"))

		case "inventory-oom":
			// 场景 2: OOM → 连接拒绝，500
			uresource := ServiceByNamespace("prj-uresource").PickInstance()
			uhost := ServiceByNamespace("prj-uhost").PickInstance()
			traceLine := fmt.Sprintf("%s:8080 T 0.200", uhost.Host)
			logs = append(logs, makeTextLog(baseTime, uresource, "ERROR", requestUUID, 1,
				"process killed by OOM killer: out of memory, exit code 137"))
			logs = append(logs, makeTextLog(baseTime.Add(50*time.Millisecond), uhost, "ERROR", requestUUID, 1,
				fmt.Sprintf("connection refused to uresource: dial tcp %s:8080: connect: connection refused", uresource.Host)))
			logs = append(logs, makeGatewayLog(
				baseTime.Add(200*time.Millisecond),
				gw, requestUUID, "DescribeUHostInstance", 200, 500, traceLine, "UHost"))

		case "gateway-disk-full":
			// 场景 3: 磁盘满 → 快速失败 502
			traceLine := fmt.Sprintf("%s:80 T 0.005", gw.Host)
			logs = append(logs, makeTextLog(baseTime, gw, "ERROR", requestUUID, 1,
				"failed to handle request: write /var/log/access.log: no space left on device, returning 502"))
			logs = append(logs, makeGatewayLog(
				baseTime.Add(5*time.Millisecond),
				gw, requestUUID, action, 5, 502, traceLine, product))

		default:
			// 未知场景：回退到通用超时错误
			responseTime := 30000 + rand.Intn(5000)
			traceLine := fmt.Sprintf("%s:8080 T %d.%03d", backend.Host, responseTime/1000, responseTime%1000)
			logs = append(logs, makeTextLog(baseTime, backend, "ERROR", requestUUID, 1,
				fmt.Sprintf("request timeout: operation=%s, duration=%dms, err=context deadline exceeded", action, responseTime)))
			logs = append(logs, makeGatewayLog(
				baseTime.Add(time.Duration(responseTime)*time.Millisecond),
				gw, requestUUID, action, responseTime, 504, traceLine, product))
		}
	} else {
		// 正常链路
		backendLatency := 50 + rand.Intn(200)   // 50-250ms
		downstreamLatency := 10 + rand.Intn(50) // 10-60ms
		totalLatency := backendLatency + downstreamLatency
		traceLine := fmt.Sprintf("%s:8080 T 0.%03d", backend.Host, totalLatency)

		// 下游正常处理
		logs = append(logs, makeTextLog(
			baseTime.Add(time.Duration(downstreamLatency)*time.Millisecond),
			downstream, "INFO", requestUUID, 1,
			fmt.Sprintf("request processed: latency=%dms", downstreamLatency)))
		// 后端正常处理
		logs = append(logs, makeTextLog(
			baseTime.Add(time.Duration(backendLatency)*time.Millisecond),
			backend, "INFO", requestUUID, 1,
			fmt.Sprintf("%s completed: latency=%dms", action, backendLatency)))
		// 网关 200
		logs = append(logs, makeGatewayLog(
			baseTime.Add(time.Duration(totalLatency)*time.Millisecond),
			gw, requestUUID, action, totalLatency, 200, traceLine, product))
	}

	return logs
}
