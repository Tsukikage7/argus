// Package service 提供应用层后台服务
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// CooldownCache 冷却缓存接口（与 servex cache.Cache 兼容）
type CooldownCache interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

// LogWatchConfig 日志监控配置
type LogWatchConfig struct {
	Interval   time.Duration // 扫描间隔，默认 30s
	Cooldown   time.Duration // 同一 namespace 冷却时间，默认 5m
	Namespaces []string      // 监控的 namespace 列表，为空则监控所有 namespace
	Threshold  int           // ERROR 数量触发阈值，默认 5
}

// LogWatchService 日志自动监控服务
// 周期扫描 ES 中的新日志，检测异常后自动触发诊断
type LogWatchService struct {
	esClient  *es.Client
	diagnoseH *command.DiagnoseHandler
	cache     CooldownCache
	cfg       LogWatchConfig
	lastScan  time.Time
}

// NewLogWatchService 创建日志监控服务
func NewLogWatchService(
	esClient *es.Client,
	diagnoseH *command.DiagnoseHandler,
	cache CooldownCache,
	cfg LogWatchConfig,
) *LogWatchService {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 5 * time.Minute
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	return &LogWatchService{
		esClient:  esClient,
		diagnoseH: diagnoseH,
		cache:     cache,
		cfg:       cfg,
		lastScan:  time.Now(),
	}
}

// Run 启动监控循环（阻塞，直到 ctx 取消）
func (s *LogWatchService) Run(ctx context.Context) {
	slog.Info("[logwatch] 启动日志监控", "interval", s.cfg.Interval, "namespaces", s.cfg.Namespaces)

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("[logwatch] 日志监控已停止")
			return
		case <-ticker.C:
			if err := s.scanOnce(ctx); err != nil {
				slog.Error("[logwatch] 扫描失败", "error", err)
			}
		}
	}
}

// scanOnce 执行一次日志扫描
func (s *LogWatchService) scanOnce(ctx context.Context) error {
	since := s.lastScan
	s.lastScan = time.Now()

	logs, err := s.esClient.SearchRecentLogs(ctx, s.cfg.Namespaces, since, 500)
	if err != nil {
		return fmt.Errorf("查询日志: %w", err)
	}

	if len(logs) == 0 {
		return nil
	}

	// 按 namespace 统计 ERROR 日志数量
	errorsByNS := s.evaluateRules(logs)

	// 达到阈值则触发诊断
	for ns, count := range errorsByNS {
		if count >= s.cfg.Threshold {
			s.triggerDiagnose(ctx, ns, count)
		}
	}

	return nil
}

// evaluateRules 统计各 namespace 的 ERROR 数量
func (s *LogWatchService) evaluateRules(logs []es.UCloudLog) map[string]int {
	counts := make(map[string]int)
	for i := range logs {
		level := es.ExtractLogLevel(&logs[i])
		if level == "ERROR" {
			counts[logs[i].KubernetesNamespace]++
		}
	}
	return counts
}

// triggerDiagnose 触发自动诊断（带冷却检查）
func (s *LogWatchService) triggerDiagnose(ctx context.Context, namespace string, errorCount int) {
	cooldownKey := fmt.Sprintf("logwatch:cooldown:%s", namespace)

	// 检查冷却期，若缓存命中则跳过
	var dummy string
	if err := s.cache.Get(ctx, cooldownKey, &dummy); err == nil {
		slog.Debug("[logwatch] 冷却中，跳过诊断", "namespace", namespace)
		return
	}

	// 设置冷却标记
	_ = s.cache.Set(ctx, cooldownKey, "1", s.cfg.Cooldown)

	slog.Info("[logwatch] 检测到异常，触发自动诊断",
		"namespace", namespace,
		"error_count", errorCount,
	)

	_, err := s.diagnoseH.Handle(ctx, command.DiagnoseCommand{
		Input:  fmt.Sprintf("[自动监控] namespace %s 检测到 %d 条 ERROR 日志，请分析根因", namespace, errorCount),
		Source: "logwatch",
	})
	if err != nil {
		slog.Error("[logwatch] 触发诊断失败", "namespace", namespace, "error", err)
	}
}
