package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// ChatStreamHandler 聊天 SSE 流处理器
type ChatStreamHandler struct {
	hub        *SSEHub
	tokenStore *StreamTokenStore
}

// NewChatStreamHandler 创建聊天 SSE 处理器
func NewChatStreamHandler(hub *SSEHub, tokenStore *StreamTokenStore) *ChatStreamHandler {
	return &ChatStreamHandler{hub: hub, tokenStore: tokenStore}
}

// ServeHTTP GET /api/v1/chat/sessions/{id}/stream?run_id=xxx&stream_token=xxx
func (h *ChatStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		http.Error(w, `{"error":"run_id is required"}`, http.StatusBadRequest)
		return
	}

	// 验证 stream_token
	token := r.URL.Query().Get("stream_token")
	if token == "" {
		http.Error(w, `{"error":"stream_token is required"}`, http.StatusUnauthorized)
		return
	}
	_, ok := h.tokenStore.Validate(token, runID)
	if !ok {
		http.Error(w, `{"error":"invalid or expired stream_token"}`, http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	// 订阅 run 级别的事件（复用 SSEHub，以 runID 作为 key）
	ch := h.hub.Subscribe(runID)
	defer h.hub.Unsubscribe(runID, ch)

	ctx := r.Context()
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {}\n\n")
			flusher.Flush()
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event.Data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

			// 终态关闭连接
			if event.Type == "run.completed" || event.Type == "run.failed" {
				return
			}
			// 兼容旧 status 事件
			if event.Type == "status" {
				if status, ok := event.Data.(task.Status); ok {
					if status == task.StatusCompleted || status == task.StatusFailed {
						return
					}
				}
			}
		}
	}
}
