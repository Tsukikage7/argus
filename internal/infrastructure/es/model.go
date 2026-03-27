package es

// UCloudLog 表示 UCloud K8s fluentd 采集的日志文档（ES _source）
type UCloudLog struct {
	Timestamp           string         `json:"@timestamp"`
	Message             string         `json:"message"`
	KubernetesNamespace string         `json:"kubernetes_namespace"`
	KubernetesLabelsApp string         `json:"kubernetes_labels_app,omitempty"`
	KubernetesPod       string         `json:"kubernetes_pod"`
	KubernetesNode      string         `json:"kubernetes_node"`
	KubernetesContainer string         `json:"kubernetes_container"`
	Host                string         `json:"host"`
	Stream              string         `json:"stream"`
	JSON                map[string]any `json:"json,omitempty"`
	Log                 *LogFile       `json:"log,omitempty"`
}

// LogFile 日志文件路径信息
type LogFile struct {
	File struct {
		Path string `json:"path"`
	} `json:"file"`
}

// GatewayMessage 网关 JSON 日志解析结果（Type A: prj-apigateway）
// message 字段是完整 JSON，包含 request_uri, response_time, trace-line 等
type GatewayMessage struct {
	RequestURI      string         `json:"request_uri"`
	GatewayLatency  int            `json:"gateway_latency"`
	RequestHeaders  map[string]any `json:"request_headers"`
	RequestTime     int64          `json:"request_time"`
	ResponseTime    int            `json:"response_time"`    // 响应时间（毫秒）
	RemoteIP        string         `json:"remote_ip"`
	RemotePort      string         `json:"remote_port"`
	ResponseHeaders map[string]any `json:"response_headers"`
	Input           map[string]any `json:"input"`
	Output          map[string]any `json:"output"`
	EndTime         int64          `json:"end_time"`
}

// TextLogParsed 文本日志解析结果（Type B: prj-ubill 等服务）
// 格式: [timestamp] [LEVEL][request_uuid.step] content
// 也可能是: [timestamp] [LEVEL][HttpRequest(request_uuid.step)|(FuncName)] content
type TextLogParsed struct {
	Timestamp   string
	Level       string
	RequestUUID string
	StepNumber  string
	FuncName    string // 可选，如 "UBillGo.BuyResource"
	Content     string
}

// StructuredLogParsed 结构化 JSON 日志解析结果（Type C: prj-uresource 等）
// message 字段是 JSON，包含 level, trace_id, span_id, operation, latency 等
type StructuredLogParsed struct {
	Level     string `json:"level"`
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
	Operation string `json:"operation"`
	Latency   int    `json:"latency"`
}

// TraceLine 网关 trace-line 解析结果
// 格式: "IP:PORT T latency" 或 "IP:PORT T latency -> IP:PORT T latency -> ..."
// 也支持 IPv6: "[2002:a40:23d:1::449b]:4210 T 0.160"
type TraceLine struct {
	Hops      []TraceHop
	Functions []TraceFunction
	Raw       string // 原始文本
}

// TraceHop IP:Port 跳转节点
type TraceHop struct {
	Address    string  // IP:Port
	LatencySec float64 // 秒
}

// TraceFunction 函数调用节点
type TraceFunction struct {
	Name       string
	Hash       string  // 可选
	LatencySec float64
}

// MessageType 日志消息类型
type MessageType int

const (
	MessageTypeUnknown    MessageType = iota
	MessageTypeGateway                // Type A: 网关 JSON
	MessageTypeText                   // Type B: 文本日志
	MessageTypeStructured             // Type C: 结构化 JSON
)
