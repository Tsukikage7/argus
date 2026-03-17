// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/argus/internal/application/command"
)

// DiagnoseHandler 处理诊断 API 请求
type DiagnoseHandler struct {
	diagnoseCmd *command.DiagnoseHandler
}

// NewDiagnoseHandler 创建诊断 HTTP 处理器
func NewDiagnoseHandler(cmd *command.DiagnoseHandler) *DiagnoseHandler {
	return &DiagnoseHandler{diagnoseCmd: cmd}
}

type diagnoseRequest struct {
	Input  string `json:"input"`
	Source string `json:"source"`
}

type diagnoseResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// ServeHTTP POST /api/v1/diagnose
func (h *DiagnoseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req diagnoseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Input == "" {
		http.Error(w, `{"error":"input is required"}`, http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		req.Source = "web"
	}

	task, err := h.diagnoseCmd.Handle(r.Context(), command.DiagnoseCommand{
		Input:  req.Input,
		Source: req.Source,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(diagnoseResponse{
		TaskID: task.ID,
		Status: string(task.Status),
	})
}
