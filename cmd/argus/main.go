// Package main 是 Argus CLI 入口
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	"github.com/Tsukikage7/argus/internal/infrastructure/llm"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
	"github.com/Tsukikage7/argus/internal/infrastructure/persistence"
	"github.com/Tsukikage7/argus/internal/infrastructure/tools"
	"github.com/Tsukikage7/argus/internal/interfaces/config"
	httphandler "github.com/Tsukikage7/argus/internal/interfaces/http/handler"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	servexcfg "github.com/Tsukikage7/servex/config"
	"github.com/Tsukikage7/servex/config/source/file"
	"github.com/Tsukikage7/servex/storage/cache"
)

var cfgPath string

func main() {
	rootCmd := &cobra.Command{
		Use:   "argus",
		Short: "Argus - AI 驱动的智能运维诊断与自愈平台",
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "configs/config.yaml", "配置文件路径")

	rootCmd.AddCommand(diagnoseCmd())
	rootCmd.AddCommand(mockCmd())
	rootCmd.AddCommand(scenariosCmd())
	rootCmd.AddCommand(replayCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	if p := os.Getenv("ARGUS_CONFIG"); p != "" {
		cfgPath = p
	}
	cfgMgr, err := servexcfg.NewManager[config.Config](
		servexcfg.WithSource[config.Config](file.New(cfgPath)),
	)
	if err != nil {
		return nil, err
	}
	if err := cfgMgr.Load(); err != nil {
		return nil, err
	}
	return cfgMgr.Get(), nil
}

func diagnoseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose [description]",
		Short: "触发一次诊断",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			// ES
			esClient, err := es.New(&cfg.Elasticsearch)
			if err != nil {
				return fmt.Errorf("elasticsearch: %w", err)
			}

			// LLM
			llmClient, err := llm.NewRouter(cfg.Providers)
			if err != nil {
				return fmt.Errorf("llm: %w", err)
			}

			// Tools
			toolRegistry := tool.NewRegistry()
			toolRegistry.Register(tools.NewESQueryLogsTool(esClient))
			toolRegistry.Register(tools.NewTraceAnalyzeTool(esClient))
			toolRegistry.Register(tools.NewExecCommandTool(true))
			toolRegistry.Register(tools.NewNotifyTool())

			// Agent
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

			// 设置实时输出（per-task handler 直接传给 Run）
			cliHandler := func(event task.TaskEvent) {
				switch event.Type {
				case "step":
					if step, ok := event.Data.(task.Step); ok {
						fmt.Printf("\n💭 Step %d - Think: %s\n", step.Index, truncate(step.Think, 200))
						if step.Action != nil {
							fmt.Printf("🔧 Action: %s(%v)\n", step.Action.Tool, step.Action.Params)
						}
						if step.Observe != "" {
							fmt.Printf("👁 Observe: %s\n", truncate(step.Observe, 300))
						}
					}
				case "diagnosis":
					if d, ok := event.Data.(*task.Diagnosis); ok {
						fmt.Println("\n" + formatDiagnosis(d))
					}
				case "status":
					fmt.Printf("📋 Status: %v\n", event.Data)
				}
			}

			// 创建任务
			t := &task.Task{
				ID:        uuid.New().String(),
				Input:     args[0],
				Source:    "cli",
				Status:    task.StatusPending,
				Steps:     []task.Step{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			fmt.Printf("🚀 Starting diagnosis: %s\n", args[0])
			fmt.Printf("📝 Task ID: %s\n", t.ID)

			ctx := context.Background()
			if err := diagAgent.Run(ctx, t, cliHandler); err != nil {
				return fmt.Errorf("diagnosis failed: %w", err)
			}

			// 输出最终结果
			if t.Diagnosis != nil {
				fmt.Println("\n" + formatDiagnosis(t.Diagnosis))
			}

			return nil
		},
	}
}

func mockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mock",
		Short: "Mock 数据管理",
	}

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "生成 mock 日志数据到 ES",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			esClient, err := es.New(&cfg.Elasticsearch)
			if err != nil {
				return fmt.Errorf("elasticsearch: %w", err)
			}

			gen := mock.NewGenerator(esClient)
			fmt.Println("🔧 Generating mock data...")
			if err := gen.GenerateAll(context.Background()); err != nil {
				return fmt.Errorf("generate failed: %w", err)
			}
			fmt.Println("✅ Mock data generated successfully!")
			return nil
		},
	}

	cmd.AddCommand(generateCmd)

	// live 子命令
	var liveRPS int
	var liveFaultRate float64
	var liveDuration string

	liveCmd := &cobra.Command{
		Use:   "live",
		Short: "持续生成实时日志数据到 ES",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			esClient, err := es.New(&cfg.Elasticsearch)
			if err != nil {
				return fmt.Errorf("elasticsearch: %w", err)
			}

			liveCfg := mock.LiveConfig{
				RPS:       liveRPS,
				FaultRate: liveFaultRate,
				Scenarios: mock.AllScenarios(),
			}
			if liveDuration != "" {
				if d, err := time.ParseDuration(liveDuration); err == nil {
					liveCfg.Duration = d
				}
			}

			gen := mock.NewLiveGenerator(esClient, mock.AllScenarios())
			fmt.Printf("🔴 Live generating: RPS=%d, FaultRate=%.1f%%\n", liveRPS, liveFaultRate*100)
			fmt.Println("Press Ctrl+C to stop...")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// 捕获中断信号
			go func() {
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, os.Interrupt)
				<-sigCh
				fmt.Println("\n⏹ Stopping...")
				cancel()
			}()

			return gen.Run(ctx, liveCfg)
		},
	}
	liveCmd.Flags().IntVar(&liveRPS, "rps", 5, "每秒请求数")
	liveCmd.Flags().Float64Var(&liveFaultRate, "fault-rate", 0.1, "故障概率 (0.0-1.0)")
	liveCmd.Flags().StringVar(&liveDuration, "duration", "", "持续时间（如 30m, 1h），空表示无限")

	cmd.AddCommand(liveCmd)
	return cmd
}

func scenariosCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scenarios",
		Short: "列出可用的故障场景",
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarios := mock.AllScenarios()
			fmt.Printf("📋 Available Scenarios (%d):\n\n", len(scenarios))
			for i, s := range scenarios {
				fmt.Printf("  %d. %s\n     %s\n\n", i+1, s.Name, s.Description)
			}
			return nil
		},
	}
}

func replayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay",
		Short: "故障/流量回放",
	}

	// replay fault
	var faultScenario string
	var faultIntensity float64
	var faultAutoDiagnose bool

	faultCmd := &cobra.Command{
		Use:   "fault",
		Short: "故障回放 — 注入故障数据并分析影响面",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReplay(task.ReplayTypeFault, faultScenario, faultIntensity, 1.0, faultAutoDiagnose)
		},
	}
	faultCmd.Flags().StringVar(&faultScenario, "scenario", "", "故障场景名称（必填）")
	faultCmd.Flags().Float64Var(&faultIntensity, "intensity", 1.0, "故障强度 (0.1~2.0)")
	faultCmd.Flags().BoolVar(&faultAutoDiagnose, "auto-diagnose", false, "自动触发诊断")
	_ = faultCmd.MarkFlagRequired("scenario")

	// replay traffic
	var trafficScenario string
	var trafficRate float64
	var trafficIntensity float64
	var trafficDuration string

	trafficCmd := &cobra.Command{
		Use:   "traffic",
		Short: "流量回放 — 按倍率重放混合流量并分析影响面",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = trafficDuration // reserved for future use
			return runReplay(task.ReplayTypeTraffic, trafficScenario, trafficIntensity, trafficRate, false)
		},
	}
	trafficCmd.Flags().StringVar(&trafficScenario, "scenario", "", "故障场景名称（必填）")
	trafficCmd.Flags().Float64Var(&trafficRate, "rate", 1.0, "流量倍率")
	trafficCmd.Flags().Float64Var(&trafficIntensity, "intensity", 1.0, "故障强度 (0.1~2.0)")
	trafficCmd.Flags().StringVar(&trafficDuration, "duration", "10m", "模拟时长")
	_ = trafficCmd.MarkFlagRequired("scenario")

	cmd.AddCommand(faultCmd)
	cmd.AddCommand(trafficCmd)
	return cmd
}

func runReplay(replayType task.ReplayType, scenario string, intensity, rate float64, autoDiagnose bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	esClient, err := es.New(&cfg.Elasticsearch)
	if err != nil {
		return fmt.Errorf("elasticsearch: %w", err)
	}

	// LLM (for impact summary)
	llmClient, err := llm.NewRouter(cfg.Providers)
	if err != nil {
		return fmt.Errorf("llm: %w", err)
	}

	// 构建回放依赖
	replayEngine := mock.NewReplayEngine(esClient)
	redisCache, err := cache.NewRedisCache(&cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}, nil)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	replayRepo := persistence.NewReplayRedisRepository(redisCache)

	// 构建诊断依赖（如果需要自动诊断）
	var diagnoseH *command.DiagnoseHandler
	if autoDiagnose {
		toolRegistry := tool.NewRegistry()
		toolRegistry.Register(tools.NewESQueryLogsTool(esClient))
		toolRegistry.Register(tools.NewTraceAnalyzeTool(esClient))
		toolRegistry.Register(tools.NewExecCommandTool(true))
		toolRegistry.Register(tools.NewNotifyTool())

		agentCfg := agent.Config{
			MaxSteps: cfg.Agent.MaxSteps,
			Timeout:  cfg.Agent.Timeout,
			Model:    cfg.Providers[0].DefaultModel,
		}
		if agentCfg.MaxSteps == 0 {
			agentCfg.MaxSteps = 15
		}
		if agentCfg.Timeout == 0 {
			agentCfg.Timeout = 5 * time.Minute
		}
		diagAgent := agent.New(llmClient, toolRegistry, agentCfg)

		// CLI 回放场景下的实时输出 handler 由 DiagnoseHandler 内部注入，
		// 此处无需再调用 OnEvent；若需 CLI 输出可在 DiagnoseHandler.events 侧处理

		sseHub := httphandler.NewSSEHub()
		taskRepo := persistence.NewTaskRedisRepository(redisCache)
		diagnoseH = command.NewDiagnoseHandler(diagAgent, taskRepo, nil, sseHub)
	}

	replayHandler := command.NewReplayHandler(
		replayEngine, diagnoseH, replayRepo, nil,
		llmClient, cfg.Providers[0].DefaultModel,
	)

	typeLabel := "Fault"
	if replayType == task.ReplayTypeTraffic {
		typeLabel = "Traffic"
	}
	fmt.Printf("🔄 %s Replay: %s (intensity=%.1f", typeLabel, scenario, intensity)
	if replayType == task.ReplayTypeTraffic {
		fmt.Printf(", rate=%.1f", rate)
	}
	fmt.Println(")")

	replayCfg := task.ReplayConfig{
		FaultIntensity:        intensity,
		TrafficRateMultiplier: rate,
		AutoDiagnose:          autoDiagnose,
	}

	session, err := replayHandler.HandleSync(context.Background(), command.ReplayCommand{
		Type:         replayType,
		ScenarioName: scenario,
		Config:       replayCfg,
	}, func(progress string) {
		switch progress {
		case "generating":
			fmt.Print("⏳ Generating data...")
		case "diagnosing":
			fmt.Println("\n🔍 Running diagnosis...")
		case "computing_impact":
			fmt.Println("📊 Computing impact...")
		case "completed":
			// printed below
		default:
			fmt.Printf("\n  %s\n", progress)
		}
	})
	if err != nil {
		return fmt.Errorf("replay failed: %w", err)
	}

	fmt.Printf("📝 Session: %s\n", session.ID)

	// 输出影响报告
	if session.ImpactReport != nil {
		printImpactReport(session.ImpactReport)
	}

	return nil
}

func printImpactReport(report *task.ImpactReport) {
	fmt.Println()
	fmt.Println("📊 Impact Report")
	fmt.Println("════════════════════════════════════════════")
	fmt.Printf("  Blast Radius: %s\n", formatBlastRadius(report.BlastRadius))
	fmt.Printf("  Total Requests: %d | Failed: %d\n", report.TotalRequests, report.FailedRequests)

	// 统计受影响的服务数
	affected := 0
	for _, svc := range report.AffectedServices {
		if svc.Status != "healthy" {
			affected++
		}
	}
	fmt.Printf("  Affected: %d/%d services\n", affected, len(report.AffectedServices))
	fmt.Println()

	// 表格
	fmt.Println("  ┌──────────────────────┬──────────┬────────┬─────────┐")
	fmt.Println("  │ Service              │ Status   │ Errors │ ErrRate │")
	fmt.Println("  ├──────────────────────┼──────────┼────────┼─────────┤")
	for _, svc := range report.AffectedServices {
		if svc.Status == "healthy" && svc.ErrorCount == 0 {
			continue
		}
		fmt.Printf("  │ %-20s │ %-8s │ %6d │ %5.0f%%  │\n",
			svc.Name, formatStatus(svc.Status), svc.ErrorCount, svc.ErrorRate*100)
	}
	fmt.Println("  └──────────────────────┴──────────┴────────┴─────────┘")

	if report.Summary != "" {
		fmt.Printf("\n  💡 %s\n", report.Summary)
	}
	fmt.Println("════════════════════════════════════════════")
}

func formatBlastRadius(radius string) string {
	switch radius {
	case "critical":
		return "CRITICAL"
	case "high":
		return "HIGH"
	case "medium":
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func formatStatus(status string) string {
	switch status {
	case "down":
		return "DOWN"
	case "degraded":
		return "DEGRADED"
	default:
		return "HEALTHY"
	}
}

func formatDiagnosis(d *task.Diagnosis) string {
	data, _ := json.MarshalIndent(d, "", "  ")
	return fmt.Sprintf(`
════════════════════════════════════════════
  📊 诊断结论
════════════════════════════════════════════
%s
════════════════════════════════════════════`, string(data))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
