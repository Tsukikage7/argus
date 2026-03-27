package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// SSEHub 管理 SSE 连接和事件分发
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan task.TaskEvent]struct{} // taskID → set of channels
}

// NewSSEHub 创建 SSE Hub
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[string]map[chan task.TaskEvent]struct{}),
	}
}

// Publish 发布事件到所有订阅该 task 的客户端
func (h *SSEHub) Publish(taskID string, event task.TaskEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if subs, ok := h.clients[taskID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default: // 丢弃慢消费者
			}
		}
	}
}

// Subscribe 订阅指定 task 的事件
func (h *SSEHub) Subscribe(taskID string) chan task.TaskEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan task.TaskEvent, 32)
	if _, ok := h.clients[taskID]; !ok {
		h.clients[taskID] = make(map[chan task.TaskEvent]struct{})
	}
	h.clients[taskID][ch] = struct{}{}
	return ch
}

// Unsubscribe 取消订阅
func (h *SSEHub) Unsubscribe(taskID string, ch chan task.TaskEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.clients[taskID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.clients, taskID)
		}
	}
	close(ch)
}

// StreamHandler 处理 SSE 连接（需要 stream_token 认证）
type StreamHandler struct {
	hub        *SSEHub
	tokenStore *StreamTokenStore
}

// NewStreamHandler 创建 SSE 处理器
func NewStreamHandler(hub *SSEHub, tokenStore *StreamTokenStore) *StreamHandler {
	return &StreamHandler{hub: hub, tokenStore: tokenStore}
}

// ServeHTTP GET /api/v1/stream/{id}?stream_token=xxx
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 提取 task ID
	taskID := r.PathValue("id")
	if taskID == "" {
		taskID = strings.TrimPrefix(r.URL.Path, "/api/v1/stream/")
	}
	if taskID == "" {
		http.Error(w, `{"error":"task_id is required"}`, http.StatusBadRequest)
		return
	}

	// 验证 stream_token（单次使用，TTL=5min，绑定 tenant+task）
	token := r.URL.Query().Get("stream_token")
	if token == "" {
		http.Error(w, `{"error":"stream_token is required"}`, http.StatusUnauthorized)
		return
	}
	_, ok := h.tokenStore.Validate(token, taskID)
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
	// CORS 不再使用 *，由上层 CORS 中间件控制
	flusher.Flush()

	ch := h.hub.Subscribe(taskID)
	defer h.hub.Unsubscribe(taskID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

			// 如果是终态，关闭连接
			if event.Type == "status" {
				if status, ok := event.Data.(task.Status); ok {
					if status == task.StatusCompleted || status == task.StatusFailed || status == task.StatusRecovered {
						return
					}
				}
			}
		}
	}
}
