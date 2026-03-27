package handler

import (
	"net/http"

	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// DashboardSummaryHandler 返回总览页统计数据（mock）
type DashboardSummaryHandler struct{}

// NewDashboardSummaryHandler 创建总览页 mock 处理器
func NewDashboardSummaryHandler() *DashboardSummaryHandler {
	return &DashboardSummaryHandler{}
}

type dashboardSummary struct {
	TotalServices        int              `json:"total_services"`
	ActiveAlerts         int              `json:"active_alerts"`
	TodayDiagnoses       int              `json:"today_diagnoses"`
	AvgDiagnoseTimeSec   int              `json:"avg_diagnose_time_seconds"`
	ServiceHealth        []serviceHealth  `json:"service_health"`
	RecentAlerts         []recentAlert    `json:"recent_alerts"`
	RecentDiagnoses      []recentDiagnose `json:"recent_diagnoses"`
}

type serviceHealth struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	ErrorRate    float64 `json:"error_rate"`
	P99LatencyMs int     `json:"p99_latency_ms"`
}

type recentAlert struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Service  string `json:"service"`
	Message  string `json:"message"`
	Time     string `json:"time"`
}

type recentDiagnose struct {
	TaskID          string  `json:"task_id"`
	Status          string  `json:"status"`
	RootCause       string  `json:"root_cause"`
	DurationSeconds int     `json:"duration_seconds"`
	Confidence      float64 `json:"confidence"`
}

// ServeHTTP GET /api/v1/dashboard/summary
func (h *DashboardSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}

	data := dashboardSummary{
		TotalServices:      6,
		ActiveAlerts:       3,
		TodayDiagnoses:     12,
		AvgDiagnoseTimeSec: 45,
		ServiceHealth: []serviceHealth{
			{Name: "apigateway", Status: "healthy", ErrorRate: 0.01, P99LatencyMs: 120},
			{Name: "ubill", Status: "healthy", ErrorRate: 0.02, P99LatencyMs: 85},
			{Name: "uresource", Status: "degraded", ErrorRate: 0.15, P99LatencyMs: 450},
			{Name: "uhost", Status: "healthy", ErrorRate: 0.01, P99LatencyMs: 95},
			{Name: "unet", Status: "healthy", ErrorRate: 0.03, P99LatencyMs: 110},
			{Name: "udb", Status: "critical", ErrorRate: 0.35, P99LatencyMs: 2800},
		},
		RecentAlerts: []recentAlert{
			{ID: "a1", Severity: "critical", Service: "udb", Message: "数据库连接池耗尽，活跃连接数超过阈值", Time: "2026-03-20T10:15:00Z"},
			{ID: "a2", Severity: "warning", Service: "uresource", Message: "资源服务 P99 延迟超过 400ms", Time: "2026-03-20T10:12:00Z"},
			{ID: "a3", Severity: "warning", Service: "apigateway", Message: "网关 5xx 错误率上升至 2%", Time: "2026-03-20T10:08:00Z"},
			{ID: "a4", Severity: "info", Service: "ubill", Message: "计费服务实例扩容完成", Time: "2026-03-20T09:55:00Z"},
			{ID: "a5", Severity: "critical", Service: "udb", Message: "慢查询数量激增，平均耗时 > 3s", Time: "2026-03-20T09:50:00Z"},
		},
		RecentDiagnoses: []recentDiagnose{
			{TaskID: "t1", Status: "completed", RootCause: "udb 连接池配置不足导致级联超时", DurationSeconds: 38, Confidence: 0.92},
			{TaskID: "t2", Status: "completed", RootCause: "uresource 内存泄漏触发 GC 停顿", DurationSeconds: 52, Confidence: 0.85},
			{TaskID: "t3", Status: "completed", RootCause: "apigateway 限流规则配置错误", DurationSeconds: 41, Confidence: 0.88},
			{TaskID: "t4", Status: "running", RootCause: "", DurationSeconds: 0, Confidence: 0},
			{TaskID: "t5", Status: "completed", RootCause: "unet DNS 解析超时导致服务发现失败", DurationSeconds: 35, Confidence: 0.91},
		},
	}

	httputil.WriteJSON(w, http.StatusOK, data)
}
