package mock

import (
	"encoding/json"
	"fmt"
	"time"
)

// makeInternalGatewayLog 生成内部网关日志（服务间调用经过的内部网关）
func makeInternalGatewayLog(t time.Time, gw *UCloudService, requestUUID, action string, responseTime, statusCode int, traceLine, product string) map[string]any {
	gatewayMsg := map[string]any{
		"request_uri":     "/",
		"gateway_latency": 0,
		"request_headers": map[string]any{
			"host":       "internal.api.ucloud.cn",
			"user_agent": "Go-http-client/1.1",
		},
		"request_time":  t.Add(-time.Duration(responseTime) * time.Millisecond).Unix(),
		"response_time": responseTime,
		"remote_ip":     "10.81.98.139",
		"remote_port":   "",
		"response_headers": map[string]any{
			"status":                  statusCode,
			"trace-line":              traceLine,
			"x-gray-gw-product":      product,
			"x-gray-gw-upstream":     "Default",
			"x-gray-gw-upstream-type": "Stable",
		},
		"input": map[string]any{
			"Action":       action,
			"request_uuid": requestUUID,
		},
		"output":   map[string]any{},
		"end_time": t.Unix(),
	}
	msgBytes, _ := json.Marshal(gatewayMsg)
	doc := makeUCloudLog(t, gw, string(msgBytes))
	doc["json"] = map[string]any{
		"response_time": fmt.Sprintf("%d", responseTime),
		"request_uri":   "/",
	}
	return doc
}

// CreateUHostDemo 场景 5: CreateUHostInstance 完整调用链演示
// 基于真实 UCloud 链路数据，模拟创建主机实例的完整微服务调用链
// 涉及 apigateway -> uhost -> uresource -> ubill -> uaccount -> unet -> udisk 等多服务协作
func CreateUHostDemo() Scenario {
	return Scenario{
		Name:        "create-uhost-demo",
		Description: "CreateUHostInstance 完整调用链：网关入口 → 资源校验 → 计费扣款 → 磁盘创建 → 网络分配，模拟真实生产链路",
		GenerateLogs: func(baseTime time.Time) []map[string]any {
			var logs []map[string]any

			gw := ServiceByNamespace("prj-apigateway")
			uhost := ServiceByNamespace("prj-uhost")
			ubill := ServiceByNamespace("prj-ubill")
			uresource := ServiceByNamespace("prj-uresource")
			udb := ServiceByNamespace("prj-udb")
			unet := ServiceByNamespace("prj-unet")

			// 生成 5 组完整调用链（模拟 5 个不同用户创建主机）
			for chain := 0; chain < 5; chain++ {
				// 每组链路间隔 3-5 分钟
				chainBase := baseTime.Add(-25*time.Minute + time.Duration(chain)*5*time.Minute)
				rootUUID := fmt.Sprintf("48e31a9f-%04x-495a-8607-ac84f939da7b", 0x1000+chain)

				logs = append(logs, generateCreateUHostChain(chainBase, rootUUID, gw, uhost, ubill, uresource, udb, unet, chain)...)
			}

			// 第 6 组：故障链路（ubill 扣款失败，余额不足）
			faultBase := baseTime.Add(-3 * time.Minute)
			faultUUID := "48e31a9f-fa11-495a-8607-ac84f939da7b"
			logs = append(logs, generateCreateUHostFaultChain(faultBase, faultUUID, gw, uhost, ubill, uresource, udb, unet)...)

			return logs
		},
	}
}

// PLACEHOLDER_CHAIN_FUNCS

// generateCreateUHostChain 生成一组完整的 CreateUHostInstance 正常调用链
func generateCreateUHostChain(base time.Time, rootUUID string, gw, uhost, ubill, uresource, udb, unet *UCloudService, idx int) []map[string]any {
	var logs []map[string]any
	t := base

	// Step 1: 网关入口 — CreateUHostInstance 请求到达
	// (网关日志最后写，因为要等所有子请求完成)

	// Step 2: uaccount 鉴权（.1）
	step1UUID := rootUUID + ".1"
	logs = append(logs, makeStructuredLog(t, udb, "INFO", "GetAccountInfo", 28, step1UUID, fmt.Sprintf("span-%d-01", idx)))
	logs = append(logs, makeInternalGatewayLog(t.Add(30*time.Millisecond), gw,
		step1UUID, "GetAccountInfo", 30, 200,
		fmt.Sprintf("10.69.192.7:4005 T 0.028"), "UAccount-INTERNAL-BACKEND"))

	// Step 3: uresource 查询资源列表（.3）
	t = t.Add(50 * time.Millisecond)
	step3UUID := rootUUID + ".3"
	logs = append(logs, makeStructuredLog(t, uresource, "INFO", "IGetResourceList", 34, step3UUID, fmt.Sprintf("span-%d-03", idx)))
	logs = append(logs, makeTextLog(t.Add(2*time.Millisecond), uresource, "INFO", step3UUID, 1,
		"body: {\"Action\":\"IGetResourceList\",\"Backend\":\"UResource\",\"Limit\":10000000,\"Offset\":0}"))
	logs = append(logs, makeInternalGatewayLog(t.Add(36*time.Millisecond), gw,
		step3UUID, "IGetResourceList", 34, 200,
		"10.69.188.5:8080 T 0.034", "UResource-INTERNAL-BACKEND"))

	// Step 5: uresource 查询资源标签（.5）
	t = t.Add(100 * time.Millisecond)
	step5UUID := rootUUID + ".5"
	logs = append(logs, makeStructuredLog(t, uresource, "INFO", "IGetResourceLabelList", 12, step5UUID, fmt.Sprintf("span-%d-05", idx)))
	logs = append(logs, makeInternalGatewayLog(t.Add(14*time.Millisecond), gw,
		step5UUID, "IGetResourceLabelList", 14, 200,
		"10.69.188.5:8080 T 0.012", "UResource-INTERNAL-BACKEND"))

	// Step 8: unet 网络分配（.8）
	t = t.Add(200 * time.Millisecond)
	step8UUID := rootUUID + ".8"
	logs = append(logs, makeTextLog(t, unet, "INFO", step8UUID, 1, "AllocateEIP start, bandwidth=1, operator=International"))
	logs = append(logs, makeTextLog(t.Add(150*time.Millisecond), unet, "INFO", step8UUID, 2, "EIP allocated: eip-demo-001, bindTo=uhost"))
	logs = append(logs, makeInternalGatewayLog(t.Add(250*time.Millisecond), gw,
		step8UUID, "AllocateEIP", 249, 200,
		"10.69.190.3:8080 T 0.249", "UNetFE-INTERNAL-BACKEND"))

	// Step 10: udisk 磁盘创建（.10）
	t = t.Add(300 * time.Millisecond)
	step10UUID := rootUUID + ".10"
	logs = append(logs, makeTextLog(t, udb, "INFO", step10UUID, 1, "CreateUDisk start, type=CLOUD_RSSD, size=20GB"))
	logs = append(logs, makeTextLog(t.Add(400*time.Millisecond), udb, "INFO", step10UUID, 2, "disk created: udisk-demo-001, latency: 469ms"))
	logs = append(logs, makeInternalGatewayLog(t.Add(470*time.Millisecond), gw,
		step10UUID, "CreateUDisk", 469, 200,
		"10.69.192.7:4540 T 0.469", "UDisk-INTERNAL-BACKEND"))

	// Step 15: uresource 创建资源记录（.15）
	t = t.Add(500 * time.Millisecond)
	step15UUID := rootUUID + ".15"
	logs = append(logs, makeStructuredLog(t, uresource, "INFO", "ICreateResource", 85, step15UUID, fmt.Sprintf("span-%d-15", idx)))
	logs = append(logs, makeTextLog(t.Add(2*time.Millisecond), uresource, "INFO", step15UUID, 1,
		"body: {\"Action\":\"ICreateResource\",\"Backend\":\"UResource\",\"Count\":1,\"ResourceType\":108}"))
	logs = append(logs, makeInternalGatewayLog(t.Add(87*time.Millisecond), gw,
		step15UUID, "ICreateResource", 85, 200,
		"10.69.188.5:8080 T 0.085", "UResource-INTERNAL-BACKEND"))

	// Step 20: ubill 计费扣款（.20）
	t = t.Add(150 * time.Millisecond)
	step20UUID := rootUUID + ".20"
	logs = append(logs, makeTextLog(t, ubill, "INFO", step20UUID, 1,
		fmt.Sprintf("req: {\"Action\":\"BuyResource\",\"ProductType\":243,\"Quantity\":1,\"request_uuid\":\"%s\"}", step20UUID)))
	logs = append(logs, makeTextLog(t.Add(time.Millisecond), ubill, "INFO", step20UUID, 2,
		fmt.Sprintf("[HttpRequest(%s)|(UBillGo.BuyResource)] req_json: {\"ChargeType\":2,\"OrderDetail\":[{\"ProductId\":2430001,\"Multiple\":20}]}", step20UUID)))
	buyLatency := 160 + idx*20
	logs = append(logs, makeTextLog(t.Add(time.Duration(buyLatency)*time.Millisecond), ubill, "INFO", step20UUID, 3,
		fmt.Sprintf("latency: %d.000ms res: {\"Action\":\"BuyResource\",\"RetCode\":0}", buyLatency)))
	// ubill 内部网关
	traceLine20 := fmt.Sprintf("10.69.186.2:4210 T 0.%03d -> CreateBuyOrder: 0.120,ICheckCompanyBuyPermission: 0.101", buyLatency)
	logs = append(logs, makeInternalGatewayLog(t.Add(time.Duration(buyLatency+2)*time.Millisecond), gw,
		step20UUID, "BuyResource", buyLatency, 200, traceLine20, "UBillGo-INTERNAL-BACKEND"))

	// Step 24: ubill 二次确认（.24）
	t = t.Add(time.Duration(buyLatency+50) * time.Millisecond)
	step24UUID := rootUUID + ".24"
	logs = append(logs, makeTextLog(t, ubill, "INFO", step24UUID, 1,
		fmt.Sprintf("req: {\"Action\":\"BuyResource\",\"Backend\":\"UBill\",\"request_uuid\":\"%s\"}", step24UUID)))
	confirmLatency := 155 + idx*10
	logs = append(logs, makeTextLog(t.Add(time.Duration(confirmLatency)*time.Millisecond), ubill, "INFO", step24UUID, 2,
		fmt.Sprintf("latency: %d.592ms res: {\"RetCode\":0}", confirmLatency)))
	logs = append(logs, makeInternalGatewayLog(t.Add(time.Duration(confirmLatency+2)*time.Millisecond), gw,
		step24UUID, "BuyResource", confirmLatency, 200,
		fmt.Sprintf("10.69.186.2:4210 T 0.%03d", confirmLatency), "UBill-INTERNAL-BACKEND"))

	// 最终：网关入口日志（总耗时）
	totalLatency := 3000 + idx*200
	topTraceLine := fmt.Sprintf("10.69.204.214:4001 T %d.%03d", totalLatency/1000, totalLatency%1000)
	logs = append(logs, makeGatewayLog(
		base.Add(time.Duration(totalLatency)*time.Millisecond), gw.PickInstance(),
		rootUUID, "CreateUHostInstance", totalLatency, 200, topTraceLine, "UHost-PUBLIC-BACKEND"))

	return logs
}

// generateCreateUHostFaultChain 生成故障调用链：ubill 扣款失败（余额不足）
func generateCreateUHostFaultChain(base time.Time, rootUUID string, gw, uhost, ubill, uresource, udb, unet *UCloudService) []map[string]any {
	var logs []map[string]any
	t := base

	// Step 1: uaccount 鉴权正常
	step1UUID := rootUUID + ".1"
	logs = append(logs, makeStructuredLog(t, udb, "INFO", "GetAccountInfo", 25, step1UUID, "span-fault-01"))
	logs = append(logs, makeInternalGatewayLog(t.Add(27*time.Millisecond), gw,
		step1UUID, "GetAccountInfo", 27, 200, "10.69.192.7:4005 T 0.025", "UAccount-INTERNAL-BACKEND"))

	// Step 3: uresource 查询正常
	t = t.Add(50 * time.Millisecond)
	step3UUID := rootUUID + ".3"
	logs = append(logs, makeStructuredLog(t, uresource, "INFO", "IGetResourceList", 30, step3UUID, "span-fault-03"))
	logs = append(logs, makeInternalGatewayLog(t.Add(32*time.Millisecond), gw,
		step3UUID, "IGetResourceList", 30, 200, "10.69.188.5:8080 T 0.030", "UResource-INTERNAL-BACKEND"))

	// Step 8: unet 网络分配正常
	t = t.Add(200 * time.Millisecond)
	step8UUID := rootUUID + ".8"
	logs = append(logs, makeTextLog(t, unet, "INFO", step8UUID, 1, "AllocateEIP start, bandwidth=1, operator=International"))
	logs = append(logs, makeTextLog(t.Add(200*time.Millisecond), unet, "INFO", step8UUID, 2, "EIP allocated: eip-fault-001"))
	logs = append(logs, makeInternalGatewayLog(t.Add(210*time.Millisecond), gw,
		step8UUID, "AllocateEIP", 210, 200, "10.69.190.3:8080 T 0.210", "UNetFE-INTERNAL-BACKEND"))

	// Step 10: udisk 创建正常
	t = t.Add(300 * time.Millisecond)
	step10UUID := rootUUID + ".10"
	logs = append(logs, makeTextLog(t, udb, "INFO", step10UUID, 1, "CreateUDisk start, type=CLOUD_RSSD, size=20GB"))
	logs = append(logs, makeTextLog(t.Add(450*time.Millisecond), udb, "INFO", step10UUID, 2, "disk created: udisk-fault-001"))
	logs = append(logs, makeInternalGatewayLog(t.Add(460*time.Millisecond), gw,
		step10UUID, "CreateUDisk", 455, 200, "10.69.192.7:4540 T 0.455", "UDisk-INTERNAL-BACKEND"))

	// Step 15: uresource 创建资源记录正常
	t = t.Add(500 * time.Millisecond)
	step15UUID := rootUUID + ".15"
	logs = append(logs, makeStructuredLog(t, uresource, "INFO", "ICreateResource", 80, step15UUID, "span-fault-15"))
	logs = append(logs, makeInternalGatewayLog(t.Add(82*time.Millisecond), gw,
		step15UUID, "ICreateResource", 80, 200, "10.69.188.5:8080 T 0.080", "UResource-INTERNAL-BACKEND"))

	// Step 20: ubill 扣款 — 第一次尝试失败！余额不足
	t = t.Add(150 * time.Millisecond)
	step20UUID := rootUUID + ".20"
	logs = append(logs, makeTextLog(t, ubill, "INFO", step20UUID, 1,
		fmt.Sprintf("req: {\"Action\":\"BuyResource\",\"ProductType\":243,\"request_uuid\":\"%s\"}", step20UUID)))
	logs = append(logs, makeTextLog(t.Add(time.Millisecond), ubill, "INFO", step20UUID, 2,
		fmt.Sprintf("[HttpRequest(%s)|(UBillGo.BuyResource)] req_json: {\"ChargeType\":2,\"OrderDetail\":[{\"ProductId\":2430001}]}", step20UUID)))
	// 扣款失败！
	logs = append(logs, makeTextLog(t.Add(220*time.Millisecond), ubill, "WARN", step20UUID, 3,
		"ICheckCompanyBuyPermission failed: account balance insufficient, required=580.00, available=12.35"))
	logs = append(logs, makeTextLog(t.Add(221*time.Millisecond), ubill, "INFO", step20UUID, 4,
		"latency: 221.540ms res: {\"Message\":\"EC_UBILL_ACCOUNT_LESS_AMOUNT\",\"RetCode\":27013}"))
	logs = append(logs, makeTextLog(t.Add(222*time.Millisecond), ubill, "INFO", step20UUID, 5,
		fmt.Sprintf("[HttpRequest(%s)|(UBillGo.BuyResource)] res_json: {\"Message\":\"EC_UBILL_ACCOUNT_LESS_AMOUNT\",\"RetCode\":27013}", step20UUID)))
	// ubill 内部网关返回 200（业务错误码在 body 中）
	traceLine20 := "10.179.65.172:4540 T 0.155 -> CreateBuyOrder: 0.120,ICheckCompanyBuyPermission: 0.101"
	logs = append(logs, makeInternalGatewayLog(t.Add(225*time.Millisecond), gw,
		step20UUID, "BuyResource", 222, 200, traceLine20, "UBillGo-INTERNAL-BACKEND"))

	// Step 24: ubill 二次确认也失败
	t = t.Add(250 * time.Millisecond)
	step24UUID := rootUUID + ".24"
	logs = append(logs, makeTextLog(t, ubill, "INFO", step24UUID, 1,
		fmt.Sprintf("req: {\"Action\":\"BuyResource\",\"Backend\":\"UBill\",\"request_uuid\":\"%s\"}", step24UUID)))
	logs = append(logs, makeTextLog(t.Add(160*time.Millisecond), ubill, "INFO", step24UUID, 2,
		"latency: 159.592ms res: {\"Message\":\"EC_UBILL_ACCOUNT_LESS_AMOUNT\",\"RetCode\":27013}"))
	logs = append(logs, makeInternalGatewayLog(t.Add(162*time.Millisecond), gw,
		step24UUID, "BuyResource", 160, 200,
		"10.69.186.2:4210 T 0.160", "UBill-INTERNAL-BACKEND"))

	// 最终：网关入口日志（总耗时 3.4s，状态 200 但业务失败）
	totalLatency := 3435
	topTraceLine := fmt.Sprintf("10.69.204.214:4001 T %d.%03d", totalLatency/1000, totalLatency%1000)
	logs = append(logs, makeGatewayLog(
		base.Add(time.Duration(totalLatency)*time.Millisecond), gw.PickInstance(),
		rootUUID, "CreateUHostInstance", totalLatency, 200, topTraceLine, "UHost-PUBLIC-BACKEND"))

	return logs
}