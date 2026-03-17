// Package main 是 Argus API Server 的启动入口，负责依赖注入组装
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/application/query"
	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	"github.com/Tsukikage7/argus/internal/infrastructure/llm"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
	"github.com/Tsukikage7/argus/internal/infrastructure/persistence"
	"github.com/Tsukikage7/argus/internal/infrastructure/tools"
	"github.com/Tsukikage7/argus/internal/interfaces/config"
	httphandler "github.com/Tsukikage7/argus/internal/interfaces/http/handler"
	"github.com/Tsukikage7/argus/internal/interfaces/http/middleware"
	servexcfg "github.com/Tsukikage7/servex/config"
	"github.com/Tsukikage7/servex/config/source/file"
	"github.com/Tsukikage7/servex/logger"
	"github.com/Tsukikage7/servex/storage/cache"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "argus: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	// ── 1. 配置 ────────────────────────────────────────────────────────────
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("ARGUS_CONFIG"); p != "" {
		cfgPath = p
	}

	cfgMgr, err := servexcfg.NewManager[config.Config](
		servexcfg.WithSource[config.Config](file.New(cfgPath)),
	)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfgMgr.Load(); err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	cfg := cfgMgr.Get()

	// ── 2. Logger ──────────────────────────────────────────────────────────
	log := logger.MustNewLogger(&cfg.Log)

	// ── 3. Redis ───────────────────────────────────────────────────────────
	redisCache, err := cache.NewRedisCache(&cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, log)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	// ── 4. Elasticsearch ───────────────────────────────────────────────────
	esClient, err := es.New(&cfg.Elasticsearch)
	if err != nil {
		return fmt.Errorf("elasticsearch: %w", err)
	}

	// ── 5. PostgreSQL ──────────────────────────────────────────────────────
	historyRepo, err := persistence.NewHistoryPGRepository(
		context.Background(),
		cfg.Postgres.DSN,
	)
	if err != nil {
		log.With(logger.Err(err)).Warn("[argus] postgres not available, history will not be persisted")
		historyRepo = nil
	}

	// ── 6. LLM Client ─────────────────────────────────────────────────────
	llmClient, err := llm.NewRouter(cfg.Providers)
	if err != nil {
		return fmt.Errorf("llm router: %w", err)
	}

	// ── 7. Tool Registry ───────────────────────────────────────────────────
	toolRegistry := tool.NewRegistry()
	toolRegistry.Register(tools.NewESQueryLogsTool(esClient))
	toolRegistry.Register(tools.NewTraceAnalyzeTool(esClient))
	toolRegistry.Register(tools.NewExecCommandTool(true)) // dry-run mode
	toolRegistry.Register(tools.NewNotifyTool())

	// ── 8. Agent ───────────────────────────────────────────────────────────
	agentCfg := agent.Config{
		MaxSteps:             cfg.Agent.MaxSteps,
		AutoRecoverThreshold: cfg.Agent.AutoRecoverThreshold,
		ConfirmThreshold:     cfg.Agent.ConfirmThreshold,
		Timeout:              cfg.Agent.Timeout,
		Model:                cfg.Providers[0].DefaultModel,
	}
	if agentCfg.MaxSteps == 0 {
		agentCfg.MaxSteps = 15
	}
	if agentCfg.Timeout == 0 {
		agentCfg.Timeout = 5 * time.Minute
	}
	diagAgent := agent.New(llmClient, toolRegistry, agentCfg)

	// ── 9. SSE Hub ─────────────────────────────────────────────────────────
	sseHub := httphandler.NewSSEHub()

	// ── 10. Repositories ───────────────────────────────────────────────────
	taskRepo := persistence.NewTaskRedisRepository(redisCache)

	// ── 11. Application: CQRS Handlers ─────────────────────────────────────
	var historyRepoInterface command.HistoryRepository
	if historyRepo != nil {
		historyRepoInterface = historyRepo
	}

	diagnoseCmd := command.NewDiagnoseHandler(diagAgent, taskRepo, historyRepoInterface, sseHub)
	alertEventCmd := command.NewAlertEventHandler(diagnoseCmd)
	verifier := agent.NewVerifier(toolRegistry)
	_ = command.NewRecoverHandler(taskRepo, toolRegistry, verifier, sseHub)

	// ── 11b. Replay Engine + Handler ──────────────────────────────────────
	replayEngine := mock.NewReplayEngine(esClient)
	replayRepo := persistence.NewReplayRedisRepository(redisCache)
	replayCmd := command.NewReplayHandler(
		replayEngine, diagnoseCmd, replayRepo, sseHub,
		llmClient, cfg.Providers[0].DefaultModel,
	)

	taskStatusQuery := query.NewTaskStatusHandler(taskRepo)
	historyQuery := query.NewHistoryHandler(historyRepoInterface)

	// ── 12. HTTP Handlers ──────────────────────────────────────────────────
	diagnoseH := httphandler.NewDiagnoseHandler(diagnoseCmd)
	eventH := httphandler.NewEventHandler(alertEventCmd)
	taskH := httphandler.NewTaskHandler(taskStatusQuery, historyQuery)
	streamH := httphandler.NewStreamHandler(sseHub)
	replayH := httphandler.NewReplayHandler(replayCmd, replayRepo, replayEngine, sseHub)
	scenarioH := httphandler.NewScenarioHandler(replayEngine)
	replayStreamH := httphandler.NewReplayStreamHandler(sseHub)

	// ── 13. 路由 ───────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// API 路由（带 APIKey 认证中间件）
	authMW := middleware.APIKeyAuth(cfg.App.APIKeys)
	mux.Handle("POST /api/v1/diagnose", authMW(diagnoseH))
	mux.Handle("POST /api/v1/events", authMW(eventH))
	mux.Handle("GET /api/v1/stream/{id}", streamH) // SSE 无需认证
	mux.Handle("GET /api/v1/tasks/{id}", authMW(taskH))
	mux.Handle("GET /api/v1/tasks", authMW(taskH))

	// 回放相关路由
	mux.Handle("GET /api/v1/scenarios", authMW(scenarioH))
	mux.Handle("POST /api/v1/replay", authMW(replayH))
	mux.Handle("GET /api/v1/replay/{id}", authMW(replayH))
	mux.Handle("GET /api/v1/replay", authMW(replayH))
	mux.Handle("GET /api/v1/replay/{id}/stream", replayStreamH) // SSE 无需认证

	// 静态文件
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// ── 14. 启动 ──────────────────────────────────────────────────────────
	log.With(
		logger.String("addr", cfg.App.Addr),
		logger.String("name", cfg.App.Name),
	).Info("[argus] 服务启动")

	srv := &http.Server{
		Addr:    cfg.App.Addr,
		Handler: mux,
	}
	return srv.ListenAndServe()
}
