// Package task 回放相关领域模型
package task

import "time"

// ReplayType 回放类型
type ReplayType string

const (
	ReplayTypeFault   ReplayType = "fault"
	ReplayTypeTraffic ReplayType = "traffic"
)

// ReplayStatus 回放状态
type ReplayStatus string

const (
	ReplayStatusPending    ReplayStatus = "pending"
	ReplayStatusGenerating ReplayStatus = "generating"
	ReplayStatusDiagnosing ReplayStatus = "diagnosing"
	ReplayStatusCompleted  ReplayStatus = "completed"
	ReplayStatusFailed     ReplayStatus = "failed"
)

// ReplaySession 表示一次回放会话
type ReplaySession struct {
	ID           string        `json:"id"`
	TenantID     string        `json:"tenant_id"`   // 所属租户 ID
	Type         ReplayType    `json:"type"`
	ScenarioName string        `json:"scenario_name"`
	Config       ReplayConfig  `json:"config"`
	Status       ReplayStatus  `json:"status"`
	TaskID       string        `json:"task_id,omitempty"`       // 关联的诊断任务 ID
	ImpactReport *ImpactReport `json:"impact_report,omitempty"`
	Error        string        `json:"error,omitempty"`
	LogsWritten  int           `json:"logs_written"`
	TracesWritten int          `json:"traces_written"`
	CreatedAt    time.Time     `json:"created_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
}

// ReplayConfig 回放配置参数
type ReplayConfig struct {
	TrafficRateMultiplier float64       `json:"traffic_rate_multiplier"` // 流量倍率，1.0=正常
	Duration              time.Duration `json:"duration"`                // 模拟时长
	FaultIntensity        float64       `json:"fault_intensity"`         // 故障强度 0.1~2.0
	FaultDelay            time.Duration `json:"fault_delay"`             // 故障注入延迟
	AutoDiagnose          bool          `json:"auto_diagnose"`           // 生成数据后自动触发诊断
}

// ImpactReport 影响面分析报告
type ImpactReport struct {
	AffectedServices []ServiceImpact    `json:"affected_services"`
	BlastRadius      string             `json:"blast_radius"`       // low / medium / high / critical
	TotalRequests    int                `json:"total_requests"`
	FailedRequests   int                `json:"failed_requests"`
	ErrorRate        map[string]float64 `json:"error_rate"`         // 服务 → 错误率
	LatencyImpact    map[string]int     `json:"latency_impact"`     // 服务 → P99 延迟(ms)
	TimeWindow       string             `json:"time_window"`
	Summary          string             `json:"summary"`            // LLM 生成的影响面分析
}

// ServiceImpact 单个服务的影响详情
type ServiceImpact struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`        // healthy / degraded / down
	ErrorCount   int     `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMs int     `json:"avg_latency_ms"`
	P99LatencyMs int     `json:"p99_latency_ms"`
	IsDirect     bool    `json:"is_direct"`     // 直接故障 vs 级联影响
}

// ReplayEvent 回放 SSE 推送事件
type ReplayEvent struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"`   // status / progress / impact / error
	Data      any    `json:"data"`
}
