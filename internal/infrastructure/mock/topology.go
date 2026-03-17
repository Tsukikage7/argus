// Package mock 提供微服务 Mock 日志数据生成
package mock

// Service 定义一个微服务
type Service struct {
	Name       string
	Port       int
	Version    string
	InstanceID string
	HostIP     string
	Namespace  string
	PodName    string
	NodeName   string
}

// Topology 返回 6 个微服务的拓扑定义
func Topology() []Service {
	return []Service{
		{
			Name: "gateway", Port: 8000, Version: "2.1.0",
			InstanceID: "gateway-pod-a1b2c", HostIP: "10.0.1.10",
			Namespace: "production", PodName: "gateway-pod-a1b2c", NodeName: "node-01",
		},
		{
			Name: "user-service", Port: 8001, Version: "1.5.2",
			InstanceID: "user-service-pod-d3e4f", HostIP: "10.0.2.10",
			Namespace: "production", PodName: "user-service-pod-d3e4f", NodeName: "node-01",
		},
		{
			Name: "order-service", Port: 8002, Version: "1.8.1",
			InstanceID: "order-service-pod-g5h6i", HostIP: "10.0.2.20",
			Namespace: "production", PodName: "order-service-pod-g5h6i", NodeName: "node-02",
		},
		{
			Name: "payment-service", Port: 8003, Version: "1.2.3",
			InstanceID: "payment-service-pod-7d8f9", HostIP: "10.0.3.15",
			Namespace: "production", PodName: "payment-service-pod-7d8f9", NodeName: "node-03",
		},
		{
			Name: "inventory-service", Port: 8004, Version: "1.3.0",
			InstanceID: "inventory-service-pod-j7k8l", HostIP: "10.0.2.30",
			Namespace: "production", PodName: "inventory-service-pod-j7k8l", NodeName: "node-02",
		},
		{
			Name: "notification-service", Port: 8005, Version: "1.1.0",
			InstanceID: "notification-service-pod-m9n0p", HostIP: "10.0.4.10",
			Namespace: "production", PodName: "notification-service-pod-m9n0p", NodeName: "node-04",
		},
	}
}

// ServiceByName 按名称查找服务
func ServiceByName(name string) *Service {
	for _, s := range Topology() {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
