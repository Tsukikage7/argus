// Package main 是 Argus API Server 的启动入口，负责依赖注入组装
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/application/query"
	"github.com/Tsukikage7/argus/internal/application/service"
	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	"github.com/Tsukikage7/argus/internal/infrastructure/llm"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
	"github.com/Tsukikage7/argus/internal/infrastructure/persistence"
	"github.com/Tsukikage7/argus/internal/infrastructure/tools"
	"github.com/Tsukikage7/argus/internal/infrastructure/wechat"
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
	// PG 不可用时降级为 NoopHistoryRepository，避免 nil panic
	// 但多租户模式下 PG 是必需的，直接报错退出
	var historyRepoImpl command.HistoryRepository
	var scenarioRepo task.ScenarioRepository
	pgRepo, pgErr := persistence.NewHistoryPGRepository(
		context.Background(),
		cfg.Postgres.DSN,
	)
	if pgErr != nil {
		if cfg.MultiTenant.Enabled {
			return fmt.Errorf("postgres required for multi-tenant mode: %w", pgErr)
		}
		log.With(logger.Err(pgErr)).Warn("[argus] postgres not available, history will not be persisted")
		historyRepoImpl = &persistence.NoopHistoryRepository{}
	} else {
		historyRepoImpl = pgRepo
		scenarioRepo = persistence.NewScenarioPGRepository(pgRepo.DB())
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

	// ── 9. SSE Hub + Stream Token Store ──────────────────────────────────
	sseHub := httphandler.NewSSEHub()
	streamTokenStore := httphandler.NewStreamTokenStore()

	// ── 10. Repositories ───────────────────────────────────────────────────
	taskRepo := persistence.NewTaskRedisRepository(redisCache)

	// ── 11. Application: CQRS Handlers ─────────────────────────────────────
	diagnoseCmd := command.NewDiagnoseHandler(diagAgent, taskRepo, historyRepoImpl, sseHub)
	diagnoseCmd.SetScenarioRepo(scenarioRepo)
	alertEventCmd := command.NewAlertEventHandler(diagnoseCmd)
	verifier := agent.NewVerifier(toolRegistry)
	recoverCmd := command.NewRecoverHandler(taskRepo, toolRegistry, verifier, sseHub)

	// ── 11b. Replay Engine + Handler ──────────────────────────────────────
	replayEngine := mock.NewReplayEngine(esClient)
	replayRepo := persistence.NewReplayRedisRepository(redisCache)
	replayCmd := command.NewReplayHandler(
		replayEngine, diagnoseCmd, replayRepo, sseHub,
		llmClient, cfg.Providers[0].DefaultModel,
	)

	// ── 11c. Live Generator ───────────────────────────────────────────────
	liveGen := mock.NewLiveGenerator(esClient, mock.AllScenarios())

	// ── 11d. Log Watcher ──────────────────────────────────────────────────
	if cfg.Monitor.Enabled {
		watchCfg := service.LogWatchConfig{
			Interval:   cfg.Monitor.Interval,
			Cooldown:   cfg.Monitor.Cooldown,
			Namespaces: cfg.Elasticsearch.Namespaces,
			Threshold:  cfg.Monitor.Threshold,
		}
		// servex cache.Cache.Get 签名为 (ctx, key) (string, error)，
		// 需要适配为 CooldownCache.Get(ctx, key, dest any) error
		cooldownCache := &cacheAdapter{redisCache}
		watcher := service.NewLogWatchService(esClient, diagnoseCmd, cooldownCache, watchCfg)
		go watcher.Run(context.Background())
	}

	taskStatusQuery := query.NewTaskStatusHandler(taskRepo)
	historyQuery := query.NewHistoryHandler(historyRepoImpl)

	// ── 11e. Chat CQRS Handlers ──────────────────────────────────────────
	var chatRepo *persistence.ChatPGRepository
	if pgRepo != nil {
		chatRepo = persistence.NewChatPGRepository(pgRepo.DB())
	}

	var createSessionCmd *command.CreateChatSessionHandler
	var updateSessionCmd *command.UpdateChatSessionHandler
	var deleteSessionCmd *command.DeleteChatSessionHandler
	var sendMessageCmd *command.SendChatMessageHandler
	var listSessionsQ *query.ListSessionsHandler
	var getSessionQ *query.GetSessionHandler
	var listMessagesQ *query.ListMessagesHandler

	if chatRepo != nil {
		createSessionCmd = command.NewCreateChatSessionHandler(chatRepo)
		updateSessionCmd = command.NewUpdateChatSessionHandler(chatRepo)
		deleteSessionCmd = command.NewDeleteChatSessionHandler(chatRepo)
		sendMessageCmd = command.NewSendChatMessageHandler(diagAgent, chatRepo, chatRepo, chatRepo, sseHub)
		listSessionsQ = query.NewListSessionsHandler(chatRepo)
		getSessionQ = query.NewGetSessionHandler(chatRepo)
		listMessagesQ = query.NewListMessagesHandler(chatRepo)
	}

	// ── 12. HTTP Handlers ──────────────────────────────────────────────────
	diagnoseH := httphandler.NewDiagnoseHandler(diagnoseCmd, streamTokenStore)
	eventH := httphandler.NewEventHandler(alertEventCmd)
	taskH := httphandler.NewTaskHandler(taskStatusQuery, historyQuery, historyRepoImpl)
	streamH := httphandler.NewStreamHandler(sseHub, streamTokenStore)
	replayH := httphandler.NewReplayHandler(replayCmd, replayRepo, replayEngine, sseHub, streamTokenStore)
	scenarioH := httphandler.NewScenarioHandler(replayEngine, scenarioRepo)
	scenarioManageH := httphandler.NewScenarioManageHandler(scenarioRepo)
	scenarioPublishH := httphandler.NewScenarioPublishHandler(scenarioRepo)
	replayStreamH := httphandler.NewReplayStreamHandler(sseHub, streamTokenStore)
	recoverH := httphandler.NewRecoverHandler(recoverCmd)
	exportH := httphandler.NewExportHandler(taskRepo) // 证据包导出
	logsH := httphandler.NewLogsHandler(esClient)
	logSummaryH := httphandler.NewLogSummaryHandler(esClient)
	logFaultsH := httphandler.NewLogFaultsHandler(esClient)
	logContextH := httphandler.NewLogContextHandler(esClient)
	logFacetsH := httphandler.NewLogFacetsHandler(esClient)
	topologyH := httphandler.NewTopologyHandler()
	topologyGraphH := httphandler.NewTopologyGraphHandler(esClient)
	dashboardH := httphandler.NewDashboardSummaryHandler()
	tracesH := httphandler.NewTracesHandler(esClient)
	traceDetailH := httphandler.NewTraceDetailHandler(esClient)
	traceFlameH := httphandler.NewTraceFlameGraphHandler()
	alertsActiveH := httphandler.NewAlertsActiveHandler()
	statsEfficiencyH := httphandler.NewStatsEfficiencyHandler()

	// ── 12c. Chat Handlers ───────────────────────────────────────────────
	var chatH *httphandler.ChatHandler
	var chatStreamH *httphandler.ChatStreamHandler
	if chatRepo != nil {
		chatH = httphandler.NewChatHandler(
			createSessionCmd, updateSessionCmd, deleteSessionCmd, sendMessageCmd,
			listSessionsQ, getSessionQ, listMessagesQ, streamTokenStore,
		)
		chatStreamH = httphandler.NewChatStreamHandler(sseHub, streamTokenStore)
	}

	// ── 12b. 企微应用回调处理器 ────────────────────────────────────────────
	// 优先使用配置文件中的 Token/EncodingAESKey，回退到环境变量
	wechatAppCfg := wechat.LoadAppConfigFromEnv()
	wechatAppCfg.CorpID = cfg.Wechat.CorpID
	wechatAppCfg.AgentID = cfg.Wechat.AgentID
	wechatAppCfg.Secret = cfg.Wechat.Secret
	if cfg.Wechat.Token != "" {
		wechatAppCfg.Token = cfg.Wechat.Token
	}
	if cfg.Wechat.EncodingAESKey != "" {
		wechatAppCfg.EncodingAESKey = cfg.Wechat.EncodingAESKey
	}
	wechatApp, err := wechat.NewApp(wechatAppCfg)
	if err != nil {
		return fmt.Errorf("wechat app: %w", err)
	}
	wechatBot := wechat.NewBot(cfg.Wechat.WebhookURL)
	wechatRouter := wechat.NewCommandRouter()
	wechatCallbackH := httphandler.NewWechatCallbackHandler(wechatApp, wechatRouter, diagnoseCmd, wechatBot)

	// ── 13. 路由 ───────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// 认证：根据 multi_tenant.enabled 选择 KeyResolver
	adminKeys := cfg.MultiTenant.BootstrapAdminKeys
	if len(adminKeys) == 0 && cfg.App.AdminKey != "" {
		adminKeys = []string{cfg.App.AdminKey}
	}

	var keyResolver middleware.KeyResolver
	if cfg.MultiTenant.Enabled && pgRepo != nil {
		// 多租户模式：DB 解析 TenantKey + Config 解析 AdminKey（含 demo tenantKeys 回退）
		tenantPGRepo := persistence.NewTenantPGRepository(pgRepo.DB())
		apiKeyPGRepo := persistence.NewAPIKeyPGRepository(pgRepo.DB())
		dbResolver := middleware.NewDBKeyResolver(apiKeyPGRepo, tenantPGRepo)
		configResolver := middleware.NewConfigKeyResolver(cfg.App.APIKeys, adminKeys)
		keyResolver = middleware.NewChainResolver(dbResolver, configResolver)

		// 管理端 Handler
		adminTenantH := httphandler.NewAdminTenantHandler(tenantPGRepo)
		adminKeyH := httphandler.NewAdminAPIKeyHandler(apiKeyPGRepo, tenantPGRepo)
		adminAuth := middleware.AdminAuth(keyResolver)

		mux.Handle("GET /admin/v1/tenants", adminAuth(http.HandlerFunc(adminTenantH.List)))
		mux.Handle("POST /admin/v1/tenants", adminAuth(http.HandlerFunc(adminTenantH.Create)))
		mux.Handle("GET /admin/v1/tenants/{id}", adminAuth(http.HandlerFunc(adminTenantH.Get)))
		mux.Handle("GET /admin/v1/tenants/{tenant_id}/keys", adminAuth(http.HandlerFunc(adminKeyH.List)))
		mux.Handle("POST /admin/v1/tenants/{tenant_id}/keys", adminAuth(http.HandlerFunc(adminKeyH.Create)))

		log.Info("[argus] 多租户模式已启用")
	} else {
		// 兼容模式：仅使用配置文件中的 API Keys
		keyResolver = middleware.NewConfigKeyResolver(cfg.App.APIKeys, adminKeys)
	}
	tenantAuth := middleware.TenantAuth(keyResolver)

	// 业务 API 路由（TenantKey 认证）
	mux.Handle("POST /api/v1/diagnose", tenantAuth(diagnoseH))
	mux.Handle("POST /api/v1/events", tenantAuth(eventH))
	mux.Handle("GET /api/v1/stream/{id}", streamH) // SSE 暂无认证（后续 stream_token）
	mux.Handle("GET /api/v1/tasks/{id}", tenantAuth(taskH))
	mux.Handle("GET /api/v1/tasks", tenantAuth(taskH))
	mux.Handle("POST /api/v1/tasks/{id}/recover", tenantAuth(recoverH))
	mux.Handle("GET /api/v1/tasks/{id}/export", tenantAuth(exportH))
	mux.Handle("GET /api/v1/logs/summary", tenantAuth(logSummaryH))
	mux.Handle("GET /api/v1/logs/faults", tenantAuth(logFaultsH))
	mux.Handle("GET /api/v1/logs/context", tenantAuth(logContextH))
	mux.Handle("GET /api/v1/logs/facets", tenantAuth(logFacetsH))
	mux.Handle("GET /api/v1/logs", tenantAuth(logsH))

	// 拓扑 API（供前端动态获取服务拓扑）
	mux.Handle("GET /api/v1/topology", tenantAuth(topologyH))
	mux.Handle("GET /api/v1/topology/graph", tenantAuth(topologyGraphH))

	// Dashboard 总览 API（mock）
	mux.Handle("GET /api/v1/dashboard/summary", tenantAuth(dashboardH))

	// 链路追踪 API
	mux.Handle("GET /api/v1/traces/{uuid}/flamegraph", tenantAuth(traceFlameH))
	mux.Handle("GET /api/v1/traces/{uuid}", tenantAuth(traceDetailH))
	mux.Handle("GET /api/v1/traces", tenantAuth(tracesH))

	// 告警 API（mock）
	mux.Handle("GET /api/v1/alerts/active", tenantAuth(alertsActiveH))

	// 效率统计 API（mock）
	mux.Handle("GET /api/v1/stats/efficiency", tenantAuth(statsEfficiencyH))

	// 回放相关路由
	mux.Handle("GET /api/v1/scenarios", tenantAuth(scenarioH))

	// ── Chat API 路由 ────────────────────────────────────────────────────
	if chatH != nil {
		mux.Handle("POST /api/v1/chat/sessions", tenantAuth(http.HandlerFunc(chatH.CreateSession)))
		mux.Handle("GET /api/v1/chat/sessions", tenantAuth(http.HandlerFunc(chatH.ListSessions)))
		mux.Handle("GET /api/v1/chat/sessions/{id}", tenantAuth(http.HandlerFunc(chatH.GetSession)))
		mux.Handle("PATCH /api/v1/chat/sessions/{id}", tenantAuth(http.HandlerFunc(chatH.UpdateSession)))
		mux.Handle("DELETE /api/v1/chat/sessions/{id}", tenantAuth(http.HandlerFunc(chatH.DeleteSession)))
		mux.Handle("POST /api/v1/chat/sessions/{id}/messages", tenantAuth(http.HandlerFunc(chatH.SendMessage)))
		mux.Handle("GET /api/v1/chat/sessions/{id}/messages", tenantAuth(http.HandlerFunc(chatH.ListMessages)))
		mux.Handle("GET /api/v1/chat/sessions/{id}/stream", chatStreamH)
	}
	mux.Handle("POST /api/v1/scenarios", tenantAuth(scenarioManageH))
	mux.Handle("PATCH /api/v1/scenarios/{id}/publish", tenantAuth(scenarioPublishH))
	mux.Handle("POST /api/v1/replay", tenantAuth(replayH))
	mux.Handle("GET /api/v1/replay/{id}", tenantAuth(replayH))
	mux.Handle("GET /api/v1/replay", tenantAuth(replayH))
	mux.Handle("GET /api/v1/replay/{id}/stream", replayStreamH) // SSE 暂无认证

	// 企微应用回调（GET=URL验证，POST=消息接收，均无需 APIKey 认证）
	mux.Handle("/api/v1/wechat/callback", wechatCallbackH)

	// Live Generator 控制 + 日志流
	mux.Handle("POST /api/v1/mock/live/start", tenantAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		liveCfg := mock.LiveConfig{
			RPS:       cfg.Live.RPS,
			FaultRate: cfg.Live.FaultRate,
			Scenarios: mock.AllScenarios(),
		}
		if err := liveGen.Start(liveCfg); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"started"}`)
	})))
	mux.Handle("POST /api/v1/mock/live/stop", tenantAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		liveGen.Stop()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"stopped"}`)
	})))
	mux.Handle("GET /api/v1/mock/live/status", tenantAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"running":%t}`, liveGen.IsRunning())
	})))

	// 实时日志 SSE 流（需租户认证）
	mux.Handle("GET /api/v1/logs/stream", tenantAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// CORS 由全局 CORS 中间件处理，不再硬编码 *
		flusher.Flush()

		ch := liveGen.Subscribe()
		defer liveGen.Unsubscribe(ch)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
				flusher.Flush()
			}
		}
	})))

	// 静态文件
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// ── 14. 启动 ──────────────────────────────────────────────────────────
	// CORS 中间件（多租户模式下使用配置的 allowed_origins）
	var handler http.Handler = mux
	if cfg.MultiTenant.Enabled && len(cfg.MultiTenant.AllowedOrigins) > 0 {
		handler = middleware.CORS(cfg.MultiTenant.AllowedOrigins)(mux)
	}

	log.With(
		logger.String("addr", cfg.App.Addr),
		logger.String("name", cfg.App.Name),
	).Info("[argus] 服务启动")

	srv := &http.Server{
		Addr:    cfg.App.Addr,
		Handler: handler,
	}
	return srv.ListenAndServe()
}

// cacheAdapter 将 servex cache.Cache 适配为 service.CooldownCache 接口
// servex Cache.Get 签名为 (ctx, key) (string, error)，
// CooldownCache.Get 签名为 (ctx, key, dest any) error
type cacheAdapter struct {
	c cache.Cache
}

func (a *cacheAdapter) Get(ctx context.Context, key string, dest any) error {
	val, err := a.c.Get(ctx, key)
	if err != nil {
		return err
	}
	if dp, ok := dest.(*string); ok {
		*dp = val
	}
	return nil
}

func (a *cacheAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return a.c.Set(ctx, key, value, ttl)
}
