package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/argus/internal/application/command"
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var event command.AlertEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	taskID, err := h.alertHandler.Handle(r.Context(), event)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
}
