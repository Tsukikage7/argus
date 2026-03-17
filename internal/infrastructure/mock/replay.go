package mock

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// ReplayEngine 回放引擎：数据注入 + 影响面计算
type ReplayEngine struct {
	es        *es.Client
	scenarios map[string]Scenario
}

// NewReplayEngine 创建回放引擎
func NewReplayEngine(esClient *es.Client) *ReplayEngine {
	scenarios := make(map[string]Scenario)
	for _, s := range AllScenarios() {
		scenarios[s.Name] = s
	}
	return &ReplayEngine{
		es:        esClient,
		scenarios: scenarios,
	}
}

// GetScenario 获取指定名称的场景
func (e *ReplayEngine) GetScenario(name string) (Scenario, bool) {
	s, ok := e.scenarios[name]
	return s, ok
}

// ListScenarios 列出所有可用场景
func (e *ReplayEngine) ListScenarios() []Scenario {
	return AllScenarios()
}

// ReplayResult 数据注入结果
type ReplayResult struct {
	LogsWritten   int
	TracesWritten int
}

// RunFaultReplay 执行故障回放：生成故障数据并注入 ES
func (e *ReplayEngine) RunFaultReplay(ctx context.Context, session *task.ReplaySession) (*ReplayResult, error) {
	scenario, ok := e.scenarios[session.ScenarioName]
	if !ok {
		return nil, fmt.Errorf("scenario %q not found", session.ScenarioName)
	}

	baseTime := time.Now()
	logs, traces := scenario.GenerateLogs(baseTime)

	// 按 FaultIntensity 缩放错误日志数量
	intensity := session.Config.FaultIntensity
	if intensity <= 0 {
		intensity = 1.0
	}
	logs = scaleLogs(logs, intensity)

	// 给每条 doc 注入 replay_session_id
	tagDocs(logs, session.ID)
	tagTraceDocs(traces, session.ID)

	// 写入 ES
	result, err := e.writeDocs(ctx, logs, traces, baseTime)
	if err != nil {
		return nil, fmt.Errorf("write replay data: %w", err)
	}
	return result, nil
}

// RunTrafficReplay 执行流量回放：按倍率重放正常+异常混合流量
func (e *ReplayEngine) RunTrafficReplay(ctx context.Context, session *task.ReplaySession) (*ReplayResult, error) {
	scenario, ok := e.scenarios[session.ScenarioName]
	if !ok {
		return nil, fmt.Errorf("scenario %q not found", session.ScenarioName)
	}

	baseTime := time.Now()
	logs, traces := scenario.GenerateLogs(baseTime)

	// 按 TrafficRateMultiplier 缩放正常日志数量
	rate := session.Config.TrafficRateMultiplier
	if rate <= 0 {
		rate = 1.0
	}

	// 按 FaultIntensity 缩放错误日志数量
	intensity := session.Config.FaultIntensity
	if intensity <= 0 {
		intensity = 1.0
	}

	logs = scaleTrafficLogs(logs, rate, intensity)

	// 打 replay tag
	tagDocs(logs, session.ID)
	tagTraceDocs(traces, session.ID)

	result, err := e.writeDocs(ctx, logs, traces, baseTime)
	if err != nil {
		return nil, fmt.Errorf("write replay data: %w", err)
	}
	return result, nil
}

// ComputeImpact 基于 ES 聚合计算影响面
func (e *ReplayEngine) ComputeImpact(ctx context.Context, sessionID string) (*task.ImpactReport, error) {
	stats, err := e.es.QueryReplayStats(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query replay stats: %w", err)
	}

	report := &task.ImpactReport{
		AffectedServices: make([]task.ServiceImpact, 0),
		ErrorRate:        make(map[string]float64),
		LatencyImpact:    make(map[string]int),
		TimeWindow:       "last 30m",
	}

	totalRequests := 0
	failedRequests := 0
	affectedCount := 0

	for _, svc := range stats.Services {
		total := svc.InfoCount + svc.WarnCount + svc.ErrorCount
		totalRequests += total
		failedRequests += svc.ErrorCount

		errRate := 0.0
		if total > 0 {
			errRate = float64(svc.ErrorCount) / float64(total)
		}

		status := "healthy"
		isDirect := false
		if errRate >= 0.8 {
			status = "down"
			isDirect = true
			affectedCount++
		} else if errRate >= 0.3 {
			status = "degraded"
			affectedCount++
		}

		impact := task.ServiceImpact{
			Name:         svc.ServiceName,
			Status:       status,
			ErrorCount:   svc.ErrorCount,
			ErrorRate:    errRate,
			AvgLatencyMs: svc.AvgLatencyMs,
			P99LatencyMs: svc.P99LatencyMs,
			IsDirect:     isDirect,
		}
		report.AffectedServices = append(report.AffectedServices, impact)
		report.ErrorRate[svc.ServiceName] = errRate
		if svc.P99LatencyMs > 0 {
			report.LatencyImpact[svc.ServiceName] = svc.P99LatencyMs
		}
	}

	report.TotalRequests = totalRequests
	report.FailedRequests = failedRequests

	// 确定 blast radius
	totalServices := len(Topology())
	report.BlastRadius = computeBlastRadius(affectedCount, totalServices, failedRequests, totalRequests)

	return report, nil
}

func computeBlastRadius(affected, total, failed, totalReq int) string {
	if total == 0 {
		return "low"
	}
	ratio := float64(affected) / float64(total)
	errRatio := 0.0
	if totalReq > 0 {
		errRatio = float64(failed) / float64(totalReq)
	}

	switch {
	case ratio >= 0.5 || errRatio >= 0.5:
		return "critical"
	case ratio >= 0.3 || errRatio >= 0.3:
		return "high"
	case ratio >= 0.15 || errRatio >= 0.15:
		return "medium"
	default:
		return "low"
	}
}

// scaleLogs 按故障强度缩放错误日志数量
func scaleLogs(logs []map[string]any, intensity float64) []map[string]any {
	if intensity == 1.0 {
		return logs
	}

	var result []map[string]any
	for _, log := range logs {
		severity, _ := log["severity"].(string)
		if severity == "ERROR" || severity == "WARN" {
			if intensity > 1.0 {
				// 复制更多错误日志
				count := int(intensity)
				for j := 0; j < count; j++ {
					copied := copyDoc(log)
					result = append(result, copied)
				}
			} else {
				// 按概率保留
				if rand.Float64() < intensity {
					result = append(result, log)
				}
			}
		} else {
			result = append(result, log)
		}
	}
	return result
}

// scaleTrafficLogs 按流量倍率缩放正常日志，按故障强度缩放错误日志
func scaleTrafficLogs(logs []map[string]any, rate, intensity float64) []map[string]any {
	var result []map[string]any
	for _, log := range logs {
		severity, _ := log["severity"].(string)
		if severity == "ERROR" || severity == "WARN" {
			if intensity > 1.0 {
				count := int(intensity)
				for j := 0; j < count; j++ {
					result = append(result, copyDoc(log))
				}
			} else if rand.Float64() < intensity {
				result = append(result, log)
			}
		} else {
			// 正常日志按 rate 缩放
			if rate > 1.0 {
				count := int(rate)
				for j := 0; j < count; j++ {
					result = append(result, copyDoc(log))
				}
			} else if rand.Float64() < rate {
				result = append(result, log)
			}
		}
	}
	return result
}

func tagDocs(docs []map[string]any, sessionID string) {
	for _, doc := range docs {
		attrs, ok := doc["attributes"].(map[string]any)
		if !ok {
			attrs = make(map[string]any)
			doc["attributes"] = attrs
		}
		attrs["replay_session_id"] = sessionID
	}
}

func tagTraceDocs(docs []map[string]any, sessionID string) {
	for _, doc := range docs {
		doc["replay_session_id"] = sessionID
	}
}

func copyDoc(doc map[string]any) map[string]any {
	copied := make(map[string]any, len(doc))
	for k, v := range doc {
		if m, ok := v.(map[string]any); ok {
			cm := make(map[string]any, len(m))
			for mk, mv := range m {
				cm[mk] = mv
			}
			copied[k] = cm
		} else {
			copied[k] = v
		}
	}
	return copied
}

func (e *ReplayEngine) writeDocs(ctx context.Context, logs, traces []map[string]any, baseTime time.Time) (*ReplayResult, error) {
	prefix := e.es.Prefix()
	date := baseTime.Format("2006.01.02")

	// 按服务分组写入
	logsByService := make(map[string][]map[string]any)
	for _, log := range logs {
		svcInfo, ok := log["service"].(map[string]any)
		if !ok {
			continue
		}
		svcName, _ := svcInfo["name"].(string)
		logsByService[svcName] = append(logsByService[svcName], log)
	}

	totalLogs := 0
	for svc, svcLogs := range logsByService {
		index := fmt.Sprintf("%s-logs-%s-%s", prefix, svc, date)
		for i := 0; i < len(svcLogs); i += 100 {
			end := i + 100
			if end > len(svcLogs) {
				end = len(svcLogs)
			}
			if err := e.es.BulkIndex(ctx, index, svcLogs[i:end]); err != nil {
				return nil, fmt.Errorf("bulk index logs for %s: %w", svc, err)
			}
		}
		totalLogs += len(svcLogs)
	}

	totalTraces := 0
	if len(traces) > 0 {
		traceIndex := fmt.Sprintf("%s-traces-%s", prefix, date)
		for i := 0; i < len(traces); i += 100 {
			end := i + 100
			if end > len(traces) {
				end = len(traces)
			}
			if err := e.es.BulkIndex(ctx, traceIndex, traces[i:end]); err != nil {
				return nil, fmt.Errorf("bulk index traces: %w", err)
			}
		}
		totalTraces = len(traces)
	}

	return &ReplayResult{
		LogsWritten:   totalLogs,
		TracesWritten: totalTraces,
	}, nil
}
