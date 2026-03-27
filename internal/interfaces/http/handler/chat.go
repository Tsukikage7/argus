package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/application/query"
	"github.com/Tsukikage7/argus/internal/domain/task"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// ChatHandler 聊天 HTTP 处理器
type ChatHandler struct {
	createSession *command.CreateChatSessionHandler
	updateSession *command.UpdateChatSessionHandler
	deleteSession *command.DeleteChatSessionHandler
	sendMessage   *command.SendChatMessageHandler
	listSessions  *query.ListSessionsHandler
	getSession    *query.GetSessionHandler
	listMessages  *query.ListMessagesHandler
	tokenStore    *StreamTokenStore
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(
	createSession *command.CreateChatSessionHandler,
	updateSession *command.UpdateChatSessionHandler,
	deleteSession *command.DeleteChatSessionHandler,
	sendMessage *command.SendChatMessageHandler,
	listSessions *query.ListSessionsHandler,
	getSession *query.GetSessionHandler,
	listMessages *query.ListMessagesHandler,
	tokenStore *StreamTokenStore,
) *ChatHandler {
	return &ChatHandler{
		createSession: createSession,
		updateSession: updateSession,
		deleteSession: deleteSession,
		sendMessage:   sendMessage,
		listSessions:  listSessions,
		getSession:    getSession,
		listMessages:  listMessages,
		tokenStore:    tokenStore,
	}
}

// SendMessage POST /api/v1/chat/sessions/{id}/messages
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	p := task.PrincipalFrom(r.Context())
	sessionID := r.PathValue("id")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "content is required")
		return
	}

	result, err := h.sendMessage.Handle(r.Context(), command.SendChatMessageCommand{
		TenantID:  p.TenantID,
		SessionID: sessionID,
		Content:   req.Content,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}

	// 生成 SSE 流令牌
	streamToken := h.tokenStore.Issue(p.TenantID, result.RunID)

	httputil.WriteJSON(w, http.StatusAccepted, map[string]string{
		"session_id":   result.SessionID,
		"message_id":   result.MessageID,
		"run_id":       result.RunID,
		"stream_token": streamToken,
	})
}

// ListMessages GET /api/v1/chat/sessions/{id}/messages
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	messages, err := h.listMessages.Handle(r.Context(), sessionID, cursor, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}
	if messages == nil {
		httputil.WriteJSON(w, http.StatusOK, []any{})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, messages)
}

// CreateSession POST /api/v1/chat/sessions
func (h *ChatHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title  string `json:"title"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 允许空 body
		req.Source = "web"
	}

	p := task.PrincipalFrom(r.Context())
	session, err := h.createSession.Handle(r.Context(), command.CreateChatSessionCommand{
		TenantID: p.TenantID,
		Title:    req.Title,
		Source:   req.Source,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, session)
}

// ListSessions GET /api/v1/chat/sessions
func (h *ChatHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	p := task.PrincipalFrom(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	sessions, err := h.listSessions.Handle(r.Context(), p.TenantID, limit, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}
	if sessions == nil {
		httputil.WriteJSON(w, http.StatusOK, []any{})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, sessions)
}

// GetSession GET /api/v1/chat/sessions/{id}
func (h *ChatHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	p := task.PrincipalFrom(r.Context())
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "session id required")
		return
	}

	session, err := h.getSession.Handle(r.Context(), p.TenantID, id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, httputil.CodeNotFound, "session not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, session)
}

// UpdateSession PATCH /api/v1/chat/sessions/{id}
func (h *ChatHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	p := task.PrincipalFrom(r.Context())
	id := r.PathValue("id")

	var req struct {
		Title    *string `json:"title"`
		Archived *bool   `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "invalid request body")
		return
	}

	err := h.updateSession.Handle(r.Context(), command.UpdateChatSessionCommand{
		TenantID: p.TenantID,
		ID:       id,
		Title:    req.Title,
		Archived: req.Archived,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSession DELETE /api/v1/chat/sessions/{id}
func (h *ChatHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	p := task.PrincipalFrom(r.Context())
	id := r.PathValue("id")

	if err := h.deleteSession.Handle(r.Context(), p.TenantID, id); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
