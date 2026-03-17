package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
)

// ReplayHandler 处理回放相关 HTTP 请求
type ReplayHandler struct {
	replayCmd  *command.ReplayHandler
	replayRepo command.ReplayRepository
	engine     *mock.ReplayEngine
	hub        *SSEHub
}

// NewReplayHandler 创建回放 HTTP 处理器
func NewReplayHandler(
	cmd *command.ReplayHandler,
	repo command.ReplayRepository,
	engine *mock.ReplayEngine,
	hub *SSEHub,
) *ReplayHandler {
	return &ReplayHandler{
		replayCmd:  cmd,
		replayRepo: repo,
		engine:     engine,
		hub:        hub,
	}
}

type replayRequest struct {
	Type     string `json:"type"`
	Scenario string `json:"scenario"`
	Config   struct {
		TrafficRateMultiplier float64 `json:"traffic_rate_multiplier"`
		FaultIntensity        float64 `json:"fault_intensity"`
		Duration              string  `json:"duration"`
		FaultDelay            string  `json:"fault_delay"`
		AutoDiagnose          bool    `json:"auto_diagnose"`
	} `json:"config"`
}

type replayResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

// ServeHTTP 处理回放请求
func (h *ReplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleGet(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *ReplayHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req replayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Scenario == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}

	replayType := task.ReplayTypeFault
	if req.Type == "traffic" {
		replayType = task.ReplayTypeTraffic
	}

	cfg := task.ReplayConfig{
		FaultIntensity:        req.Config.FaultIntensity,
		TrafficRateMultiplier: req.Config.TrafficRateMultiplier,
		AutoDiagnose:          req.Config.AutoDiagnose,
	}
	if cfg.FaultIntensity <= 0 {
		cfg.FaultIntensity = 1.0
	}
	if cfg.TrafficRateMultiplier <= 0 {
		cfg.TrafficRateMultiplier = 1.0
	}
	if req.Config.Duration != "" {
		if d, err := time.ParseDuration(req.Config.Duration); err == nil {
			cfg.Duration = d
		}
	}

	session, err := h.replayCmd.Handle(r.Context(), command.ReplayCommand{
		Type:         replayType,
		ScenarioName: req.Scenario,
		Config:       cfg,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(replayResponse{
		SessionID: session.ID,
		Status:    string(session.Status),
	})
}

func (h *ReplayHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		// 列表模式：返回空数组（简单实现）
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[]")
		return
	}

	session, err := h.replayRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"replay session not found: %s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// ScenarioHandler 处理场景列表请求
type ScenarioHandler struct {
	engine *mock.ReplayEngine
}

// NewScenarioHandler 创建场景 HTTP 处理器
func NewScenarioHandler(engine *mock.ReplayEngine) *ScenarioHandler {
	return &ScenarioHandler{engine: engine}
}

type scenarioItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ServeHTTP GET /api/v1/scenarios
func (h *ScenarioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	scenarios := h.engine.ListScenarios()
	items := make([]scenarioItem, 0, len(scenarios))
	for _, s := range scenarios {
		items = append(items, scenarioItem{
			Name:        s.Name,
			Description: s.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// ReplayStreamHandler 处理回放 SSE 流
type ReplayStreamHandler struct {
	hub *SSEHub
}

// NewReplayStreamHandler 创建回放 SSE 处理器
func NewReplayStreamHandler(hub *SSEHub) *ReplayStreamHandler {
	return &ReplayStreamHandler{hub: hub}
}

// ServeHTTP GET /api/v1/replay/{id}/stream
func (h *ReplayStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, `{"error":"session id is required"}`, http.StatusBadRequest)
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	// 订阅 replay:sessionID 前缀的事件
	ch := h.hub.Subscribe("replay:" + sessionID)
	defer h.hub.Unsubscribe("replay:"+sessionID, ch)

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

			// 终态关闭
			if event.Type == "status" {
				if status, ok := event.Data.(task.ReplayStatus); ok {
					if status == task.ReplayStatusCompleted || status == task.ReplayStatusFailed {
						return
					}
				}
			}
		}
	}
}
