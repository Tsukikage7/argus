// Package mock 提供微服务 Mock 日志数据生成
package mock

import "math/rand"

// LogType 日志类型
type LogType int

const (
	LogTypeGateway    LogType = iota // Type A: 网关 JSON
	LogTypeText                      // Type B: 文本日志
	LogTypeStructured                // Type C: 结构化 JSON
)

// ServiceInstance 服务实例变体（模拟多副本部署）
type ServiceInstance struct {
	Pod  string // pod 名称
	Node string // 节点 IP
	Host string // 主机 IP
}

// UCloudService 模拟 UCloud K8s 服务
type UCloudService struct {
	Namespace  string            // K8s namespace，如 "prj-apigateway"
	LabelsApp  string            // kubernetes_labels_app，如 "gray-gateway-gw"
	Pod        string            // 默认 pod 名称
	Node       string            // 默认节点 IP
	Container  string            // 容器名
	Host       string            // 默认主机 IP
	LogType    LogType           // 日志格式类型
	Instances  []ServiceInstance // 多实例变体（模拟生产环境多副本）
}

// PickInstance 随机选择一个服务实例，返回带有该实例 Pod/Node/Host 的副本
func (s *UCloudService) PickInstance() *UCloudService {
	if len(s.Instances) == 0 {
		return s
	}
	inst := s.Instances[rand.Intn(len(s.Instances))]
	clone := *s
	clone.Pod = inst.Pod
	clone.Node = inst.Node
	clone.Host = inst.Host
	return &clone
}

// Topology 返回 6 个模拟 UCloud 微服务的拓扑（每个服务含多实例变体）
func Topology() []UCloudService {
	return []UCloudService{
		{
			Namespace: "prj-apigateway", LabelsApp: "gray-gateway-gw",
			Pod: "gray-gateway-gw-deployment-7d8f9a1b2c", Node: "10.69.202.25",
			Container: "gray-gateway-gw", Host: "10.69.202.25",
			LogType: LogTypeGateway,
			Instances: []ServiceInstance{
				{Pod: "gray-gateway-gw-deployment-7d8f9a1b2c", Node: "10.69.202.25", Host: "10.69.202.25"},
				{Pod: "gray-gateway-gw-deployment-3e5a7c9d1f", Node: "10.69.202.26", Host: "10.69.202.26"},
			},
		},
		{
			Namespace: "prj-uhost", LabelsApp: "go-uhost-http",
			Pod: "go-uhost-http-6c9c86d7cf-abc12", Node: "10.69.186.10",
			Container: "go-uhost-http", Host: "10.69.186.10",
			LogType: LogTypeText,
			Instances: []ServiceInstance{
				{Pod: "go-uhost-http-6c9c86d7cf-abc12", Node: "10.69.186.10", Host: "10.69.186.10"},
				{Pod: "go-uhost-http-6c9c86d7cf-def34", Node: "10.69.186.11", Host: "10.69.186.11"},
				{Pod: "go-uhost-http-6c9c86d7cf-ghi56", Node: "10.69.186.12", Host: "10.69.186.12"},
			},
		},
		{
			Namespace: "prj-ubill", LabelsApp: "go-ubill-http",
			Pod: "go-ubill-http-6c9c86d7cf-mvrs9", Node: "10.69.186.2",
			Container: "go-ubill-http", Host: "10.69.186.2",
			LogType: LogTypeText,
			Instances: []ServiceInstance{
				{Pod: "go-ubill-http-6c9c86d7cf-mvrs9", Node: "10.69.186.2", Host: "10.69.186.2"},
				{Pod: "go-ubill-http-6c9c86d7cf-nwst3", Node: "10.69.186.3", Host: "10.69.186.3"},
				{Pod: "go-ubill-http-6c9c86d7cf-pxuv7", Node: "10.69.186.4", Host: "10.69.186.4"},
			},
		},
		{
			Namespace: "prj-uresource", LabelsApp: "go-uresource-http",
			Pod: "go-uresource-http-5b8d4f2e1a-xyz34", Node: "10.69.188.5",
			Container: "go-uresource-http", Host: "10.69.188.5",
			LogType: LogTypeStructured,
			Instances: []ServiceInstance{
				{Pod: "go-uresource-http-5b8d4f2e1a-xyz34", Node: "10.69.188.5", Host: "10.69.188.5"},
				{Pod: "go-uresource-http-5b8d4f2e1a-uvw12", Node: "10.69.188.6", Host: "10.69.188.6"},
			},
		},
		{
			Namespace: "prj-unet", LabelsApp: "go-unet-http",
			Pod: "go-unet-http-7a3e5c9d8b-def56", Node: "10.69.190.3",
			Container: "go-unet-http", Host: "10.69.190.3",
			LogType: LogTypeText,
			Instances: []ServiceInstance{
				{Pod: "go-unet-http-7a3e5c9d8b-def56", Node: "10.69.190.3", Host: "10.69.190.3"},
				{Pod: "go-unet-http-7a3e5c9d8b-jkl89", Node: "10.69.190.4", Host: "10.69.190.4"},
			},
		},
		{
			Namespace: "prj-udb", LabelsApp: "go-udb-http",
			Pod: "go-udb-http-4f6b2a7c9e-ghi78", Node: "10.69.192.7",
			Container: "go-udb-http", Host: "10.69.192.7",
			LogType: LogTypeText,
			Instances: []ServiceInstance{
				{Pod: "go-udb-http-4f6b2a7c9e-ghi78", Node: "10.69.192.7", Host: "10.69.192.7"},
				{Pod: "go-udb-http-4f6b2a7c9e-mno12", Node: "10.69.192.8", Host: "10.69.192.8"},
			},
		},
	}
}

// ServiceByNamespace 按 namespace 查找服务
func ServiceByNamespace(namespace string) *UCloudService {
	for _, s := range Topology() {
		if s.Namespace == namespace {
			return &s
		}
	}
	return nil
}
