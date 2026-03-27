package handler

import (
	"net/http"

	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// AlertsActiveHandler 返回活跃告警列表（mock）
type AlertsActiveHandler struct{}

// NewAlertsActiveHandler 创建告警列表 mock 处理器
func NewAlertsActiveHandler() *AlertsActiveHandler {
	return &AlertsActiveHandler{}
}

type alertEvent struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Severity    string `json:"severity"`
	Service     string `json:"service"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	StartsAt    string `json:"starts_at"`
	TaskID      string `json:"task_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
}

type alertsActiveResponse struct {
	Total   int                  `json:"total"`
	Alerts  []alertEvent         `json:"alerts"`
	Summary alertSeveritySummary `json:"summary"`
}

type alertSeveritySummary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// ServeHTTP GET /api/v1/alerts/active
func (h *AlertsActiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	alerts := []alertEvent{
		{ID: "alert-001", Fingerprint: "fp-udb-pool-001", Severity: "critical", Service: "go-udb-http", Message: "数据库连接池耗尽，活跃连接数超过阈值 (95/100)", Status: "firing", StartsAt: "2026-03-20T10:15:00Z", CreatedAt: "2026-03-20T10:15:00Z"},
		{ID: "alert-002", Fingerprint: "fp-udb-slow-001", Severity: "critical", Service: "go-udb-http", Message: "慢查询数量激增，平均耗时 > 3s", Status: "firing", StartsAt: "2026-03-20T09:50:00Z", CreatedAt: "2026-03-20T09:50:00Z"},
		{ID: "alert-003", Fingerprint: "fp-ures-p99-001", Severity: "warning", Service: "go-uresource-http", Message: "资源服务 P99 延迟超过 400ms", Status: "firing", StartsAt: "2026-03-20T10:12:00Z", CreatedAt: "2026-03-20T10:12:00Z"},
		{ID: "alert-004", Fingerprint: "fp-gw-5xx-001", Severity: "warning", Service: "gray-gateway-gw", Message: "网关 5xx 错误率上升至 2%", Status: "acknowledged", StartsAt: "2026-03-20T10:08:00Z", CreatedAt: "2026-03-20T10:08:00Z"},
		{ID: "alert-005", Fingerprint: "fp-ubill-scale-001", Severity: "info", Service: "go-ubill-http", Message: "计费服务实例扩容完成", Status: "resolved", StartsAt: "2026-03-20T09:55:00Z", CreatedAt: "2026-03-20T09:55:00Z", ResolvedAt: "2026-03-20T10:00:00Z"},
		{ID: "alert-006", Fingerprint: "fp-unet-dns-001", Severity: "warning", Service: "go-unet-http", Message: "DNS 解析超时率上升至 5%", Status: "firing", StartsAt: "2026-03-20T10:20:00Z", CreatedAt: "2026-03-20T10:20:00Z"},
		{ID: "alert-007", Fingerprint: "fp-ures-mem-001", Severity: "critical", Service: "go-uresource-http", Message: "内存使用率超过 90%，疑似内存泄漏", Status: "firing", StartsAt: "2026-03-20T10:25:00Z", CreatedAt: "2026-03-20T10:25:00Z"},
		{ID: "alert-008", Fingerprint: "fp-uhost-hc-001", Severity: "info", Service: "go-uhost-http", Message: "主机服务健康检查恢复正常", Status: "resolved", StartsAt: "2026-03-20T09:30:00Z", CreatedAt: "2026-03-20T09:30:00Z", ResolvedAt: "2026-03-20T09:35:00Z"},
	}

	data := alertsActiveResponse{
		Total:  len(alerts),
		Alerts: alerts,
		Summary: alertSeveritySummary{
			Critical: 3,
			Warning:  3,
			Info:     2,
		},
	}

	httputil.WriteJSON(w, http.StatusOK, data)
}
