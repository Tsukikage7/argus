package handler

import (
	"net/http"

	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// StatsEfficiencyHandler 返回效率对比数据（mock）
type StatsEfficiencyHandler struct{}

// NewStatsEfficiencyHandler 创建效率统计 mock 处理器
func NewStatsEfficiencyHandler() *StatsEfficiencyHandler {
	return &StatsEfficiencyHandler{}
}

// efficiencyStats spec 要求的扁平响应结构
type efficiencyStats struct {
	AIAvgTimeSec       int     `json:"ai_avg_time_seconds"`
	ManualAvgTimeSec   int     `json:"manual_avg_time_seconds"`
	AIAvgSteps         int     `json:"ai_avg_steps"`
	ManualAvgSteps     int     `json:"manual_avg_steps"`
	AIAccuracy         float64 `json:"ai_accuracy"`
	ScenariosCovered   int     `json:"scenarios_covered"`
	TotalDiagnoses     int     `json:"total_diagnoses"`
	TimeSavedHours     float64 `json:"time_saved_hours"`
}

// ServeHTTP GET /api/v1/stats/efficiency
func (h *StatsEfficiencyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}

	data := efficiencyStats{
		AIAvgTimeSec:     45,
		ManualAvgTimeSec: 1800,
		AIAvgSteps:       8,
		ManualAvgSteps:   25,
		AIAccuracy:       0.85,
		ScenariosCovered: 3,
		TotalDiagnoses:   47,
		TimeSavedHours:   12.5,
	}

	httputil.WriteJSON(w, http.StatusOK, data)
}
