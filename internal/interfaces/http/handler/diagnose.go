// Package handler 提供 HTTP 请求处理器
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// DiagnoseHandler 处理诊断 API 请求
type DiagnoseHandler struct {
	diagnoseCmd *command.DiagnoseHandler
	tokenStore  *StreamTokenStore
}

// NewDiagnoseHandler 创建诊断 HTTP 处理器
func NewDiagnoseHandler(cmd *command.DiagnoseHandler, tokenStore *StreamTokenStore) *DiagnoseHandler {
	return &DiagnoseHandler{diagnoseCmd: cmd, tokenStore: tokenStore}
}

type diagnoseRequest struct {
	Input   string                  `json:"input"`
	Source  string                  `json:"source"`
	Context *command.DiagnoseContext `json:"context,omitempty"`
}

type diagnoseResponse struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	StreamToken string `json:"stream_token,omitempty"` // SSE 流令牌（单次使用，TTL=5min）
}

// ServeHTTP POST /api/v1/diagnose
func (h *DiagnoseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, httputil.CodeValidation, "method not allowed")
		return
	}

	var req diagnoseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "invalid request body")
		return
	}

	if req.Input == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "input is required")
		return
	}

	if req.Source == "" {
		req.Source = "web"
	}

	p := task.PrincipalFrom(r.Context())
	t, err := h.diagnoseCmd.Handle(r.Context(), command.DiagnoseCommand{
		TenantID: p.TenantID,
		Input:    req.Input,
		Source:   req.Source,
		Context:  req.Context,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}

	// 生成 SSE 流令牌
	streamToken := h.tokenStore.Issue(p.TenantID, t.ID)

	httputil.WriteJSON(w, http.StatusAccepted, diagnoseResponse{
		TaskID:      t.ID,
		Status:      string(t.Status),
		StreamToken: streamToken,
	})
}
