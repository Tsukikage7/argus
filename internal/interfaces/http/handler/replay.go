package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
)

// replayLister 定义支持列表查询的回放仓储接口
// 通过类型断言在 handler 中使用，避免修改 command.ReplayRepository 接口
type replayLister interface {
	ListRecent(ctx context.Context, tenantID string, limit int) ([]*task.ReplaySession, error)
}

// ReplayHandler 处理回放相关 HTTP 请求
type ReplayHandler struct {
	replayCmd  *command.ReplayHandler
	replayRepo command.ReplayRepository
	engine     *mock.ReplayEngine
	hub        *SSEHub
	tokenStore *StreamTokenStore
}

// NewReplayHandler 创建回放 HTTP 处理器
func NewReplayHandler(
	cmd *command.ReplayHandler,
	repo command.ReplayRepository,
	engine *mock.ReplayEngine,
	hub *SSEHub,
	tokenStore *StreamTokenStore,
) *ReplayHandler {
	return &ReplayHandler{
		replayCmd:  cmd,
		replayRepo: repo,
		engine:     engine,
		hub:        hub,
		tokenStore: tokenStore,
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
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
	StreamToken string `json:"stream_token,omitempty"` // SSE 流令牌
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
	// 解析故障注入延迟参数
	if req.Config.FaultDelay != "" {
		if d, err := time.ParseDuration(req.Config.FaultDelay); err == nil {
			cfg.FaultDelay = d
		}
	}

	p := task.PrincipalFrom(r.Context())
	session, err := h.replayCmd.Handle(r.Context(), command.ReplayCommand{
		TenantID:     p.TenantID,
		Type:         replayType,
		ScenarioName: req.Scenario,
		Config:       cfg,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// 生成 SSE 流令牌
	streamToken := h.tokenStore.Issue(session.TenantID, "replay:"+session.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(replayResponse{
		SessionID:   session.ID,
		Status:      string(session.Status),
		StreamToken: streamToken,
	})
}

func (h *ReplayHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		// 列表模式：通过类型断言调用 ListRecent（如果仓储支持）
		w.Header().Set("Content-Type", "application/json")
		if lister, ok := h.replayRepo.(replayLister); ok {
			p := task.PrincipalFrom(r.Context())
			sessions, err := lister.ListRecent(r.Context(), p.TenantID, 20)
			if err != nil || sessions == nil {
				sessions = []*task.ReplaySession{}
			}
			json.NewEncoder(w).Encode(sessions)
			return
		}
		fmt.Fprint(w, "[]")
		return
	}

	p := task.PrincipalFrom(r.Context())
	session, err := h.replayRepo.Get(r.Context(), p.TenantID, id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"replay session not found: %s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// ScenarioHandler 处理场景列表请求
type ScenarioHandler struct {
	engine       *mock.ReplayEngine
	scenarioRepo task.ScenarioRepository
}

// NewScenarioHandler 创建场景 HTTP 处理器
func NewScenarioHandler(engine *mock.ReplayEngine, scenarioRepo task.ScenarioRepository) *ScenarioHandler {
	return &ScenarioHandler{engine: engine, scenarioRepo: scenarioRepo}
}

type scenarioItem struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`                    // preset / captured
	SourceTask  string  `json:"source_task_id,omitempty"` // 仅 captured 类型
	Confidence  float64 `json:"confidence,omitempty"`     // 仅 captured 类型
}

// ServeHTTP GET /api/v1/scenarios
func (h *ScenarioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 预置场景
	scenarios := h.engine.ListScenarios()
	items := make([]scenarioItem, 0, len(scenarios))
	for _, s := range scenarios {
		items = append(items, scenarioItem{
			Name:        s.Name,
			Description: s.Description,
			Type:        "preset",
		})
	}

	// 已发布的沉淀场景
	if h.scenarioRepo != nil {
		captured, err := h.scenarioRepo.List(r.Context(), task.ScenarioStatusPublished)
		if err == nil {
			for _, c := range captured {
				items = append(items, scenarioItem{
					Name:        c.Name,
					Description: c.Description,
					Type:        "captured",
					SourceTask:  c.SourceTaskID,
					Confidence:  c.Confidence,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// ScenarioManageHandler 处理场景创建/编辑/发布请求
type ScenarioManageHandler struct {
	scenarioRepo task.ScenarioRepository
}

// NewScenarioManageHandler 创建场景管理 HTTP 处理器
func NewScenarioManageHandler(repo task.ScenarioRepository) *ScenarioManageHandler {
	return &ScenarioManageHandler{scenarioRepo: repo}
}

type createScenarioRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RootCause   string   `json:"root_cause,omitempty"`
	LogPatterns []string `json:"log_patterns,omitempty"`
	Namespaces  []string `json:"affected_namespaces,omitempty"`
}

// ServeHTTP POST /api/v1/scenarios — 手动创建沉淀场景
func (h *ScenarioManageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req createScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Description == "" {
		http.Error(w, `{"error":"name and description are required"}`, http.StatusBadRequest)
		return
	}

	scenario := &task.CapturedScenario{
		ID:                 fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:               req.Name,
		Description:        req.Description,
		RootCause:          req.RootCause,
		LogPatterns:        req.LogPatterns,
		AffectedNamespaces: req.Namespaces,
		Status:             task.ScenarioStatusDraft,
		CreatedAt:          time.Now(),
	}

	if err := h.scenarioRepo.Save(r.Context(), scenario); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"save scenario: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(scenario)
}

// ScenarioPublishHandler 处理场景发布请求
type ScenarioPublishHandler struct {
	scenarioRepo task.ScenarioRepository
}

// NewScenarioPublishHandler 创建场景发布 HTTP 处理器
func NewScenarioPublishHandler(repo task.ScenarioRepository) *ScenarioPublishHandler {
	return &ScenarioPublishHandler{scenarioRepo: repo}
}

// ServeHTTP PATCH /api/v1/scenarios/{id}/publish
func (h *ScenarioPublishHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"scenario id is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.scenarioRepo.UpdateStatus(r.Context(), id, task.ScenarioStatusPublished); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}

// ReplayStreamHandler 处理回放 SSE 流（需要 stream_token 认证）
type ReplayStreamHandler struct {
	hub        *SSEHub
	tokenStore *StreamTokenStore
}

// NewReplayStreamHandler 创建回放 SSE 处理器
func NewReplayStreamHandler(hub *SSEHub, tokenStore *StreamTokenStore) *ReplayStreamHandler {
	return &ReplayStreamHandler{hub: hub, tokenStore: tokenStore}
}

// ServeHTTP GET /api/v1/replay/{id}/stream?stream_token=xxx
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

	// 验证 stream_token
	token := r.URL.Query().Get("stream_token")
	if token == "" {
		http.Error(w, `{"error":"stream_token is required"}`, http.StatusUnauthorized)
		return
	}
	// replay 的 taskID 使用 "replay:{sessionID}" 格式
	_, ok := h.tokenStore.Validate(token, "replay:"+sessionID)
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
