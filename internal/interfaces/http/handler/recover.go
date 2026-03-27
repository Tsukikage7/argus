// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
)

// RecoverHandler 处理手动触发恢复的 API 请求
type RecoverHandler struct {
	recoverCmd *command.RecoverHandler
}

// NewRecoverHandler 创建恢复 HTTP 处理器
func NewRecoverHandler(cmd *command.RecoverHandler) *RecoverHandler {
	return &RecoverHandler{recoverCmd: cmd}
}

type recoverResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// ServeHTTP 处理 POST /api/v1/tasks/{id}/recover
func (h *RecoverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, `{"error":"task id is required"}`, http.StatusBadRequest)
		return
	}

	p := task.PrincipalFrom(r.Context())
	if err := h.recoverCmd.Handle(r.Context(), command.RecoverCommand{TenantID: p.TenantID, TaskID: taskID}); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(recoverResponse{
		TaskID:  taskID,
		Message: "recovery started",
	})
}
