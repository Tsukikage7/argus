package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// EventHandler 处理告警事件 Webhook
type EventHandler struct {
	alertHandler *command.AlertEventHandler
}

// NewEventHandler 创建告警事件 HTTP 处理器
func NewEventHandler(ah *command.AlertEventHandler) *EventHandler {
	return &EventHandler{alertHandler: ah}
}

// ServeHTTP POST /api/v1/events
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 POST")
		return
	}

	var event command.AlertEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request body")
		return
	}

	// 必填字段校验
	if event.AlertName == "" || event.Service == "" {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", "alert_name and service are required")
		return
	}

	// 安全：始终使用认证 Principal 的 tenant_id，忽略 body 中的值
	if p := task.PrincipalFrom(r.Context()); p != nil {
		event.TenantID = p.TenantID
	}

	taskID, err := h.alertHandler.Handle(r.Context(), event)
	if err != nil {
		// 区分去重冲突和内部错误
		if strings.Contains(err.Error(), "告警去重") {
			httputil.WriteError(w, http.StatusConflict, "conflict", err.Error())
		} else {
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "诊断任务创建失败")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
}
