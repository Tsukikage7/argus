package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	LogsWritten int
}

// RunFaultReplay 执行故障回放：生成故障数据并注入 ES
func (e *ReplayEngine) RunFaultReplay(ctx context.Context, session *task.ReplaySession) (*ReplayResult, error) {
	scenario, ok := e.scenarios[session.ScenarioName]
	if !ok {
		return nil, fmt.Errorf("scenario %q not found", session.ScenarioName)
	}

	baseTime := time.Now()
	logs := scenario.GenerateLogs(baseTime)

	// 按 FaultIntensity 缩放错误日志数量
	intensity := session.Config.FaultIntensity
	if intensity <= 0 {
		intensity = 1.0
	}
	logs = scaleLogs(logs, intensity)

	// 给每条 doc 注入 replay_session_id
	tagDocs(logs, session.ID)

	// 写入 ES
	result, err := e.writeDocs(ctx, logs, baseTime)
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
	logs := scenario.GenerateLogs(baseTime)

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

	result, err := e.writeDocs(ctx, logs, baseTime)
	if err != nil {
		return nil, fmt.Errorf("write replay data: %w", err)
	}
	return result, nil
}

// ComputeImpact 基于 ES 查询计算影响面
// 先用聚合取各 namespace 总文档数，再抽样原始文档统计真实错误率
func (e *ReplayEngine) ComputeImpact(ctx context.Context, tenantID, sessionID string) (*task.ImpactReport, error) {
	stats, err := e.es.QueryReplayStats(ctx, tenantID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query replay stats: %w", err)
	}

	// 获取原始文档用于真实错误统计
	docs, err := e.queryReplayDocs(ctx, tenantID, sessionID)
	if err != nil {
		// 查询失败不阻断，降级为仅使用聚合数据
		docs = nil
	}

	// 按 namespace 建立错误文档计数表
	errCountByNS := make(map[string]int)
	for _, doc := range docs {
		if isErrorOrWarn(doc) {
			ns, _ := doc["kubernetes_namespace"].(string)
			errCountByNS[ns]++
		}
	}

	report := &task.ImpactReport{
		AffectedServices: make([]task.ServiceImpact, 0),
		ErrorRate:        make(map[string]float64),
		LatencyImpact:    make(map[string]int),
		TimeWindow:       "last 30m",
	}

	totalRequests := 0
	totalFailed := 0
	affectedCount := 0

	for _, svc := range stats.Services {
		totalRequests += svc.DocCount

		// 使用真实错误计数；若无原始文档数据则退化为 0
		errCount := errCountByNS[svc.Namespace]
		totalFailed += errCount

		status := "healthy"
		if errCount > 0 {
			status = "degraded"
			affectedCount++
		}

		var errRate float64
		if svc.DocCount > 0 {
			errRate = float64(errCount) / float64(svc.DocCount)
		}

		impact := task.ServiceImpact{
			Name:       svc.Namespace,
			Status:     status,
			ErrorCount: errCount,
			ErrorRate:  errRate,
			IsDirect:   errCount > 10,
		}
		report.AffectedServices = append(report.AffectedServices, impact)
		report.ErrorRate[svc.Namespace] = errRate
	}

	report.TotalRequests = totalRequests
	report.FailedRequests = totalFailed

	// 计算 blast radius
	totalServices := len(Topology())
	report.BlastRadius = computeBlastRadius(affectedCount, totalServices, totalFailed, totalRequests)

	return report, nil
}

// queryReplayDocs 通过 replay_session_id 查询原始文档（最多 1000 条）
// 用于 ComputeImpact 中统计真实 error/warn 数量
func (e *ReplayEngine) queryReplayDocs(ctx context.Context, tenantID, sessionID string) ([]map[string]any, error) {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"replay_session_id.keyword": sessionID,
			},
		},
		"size": 1000,
		"_source": []string{"message", "kubernetes_namespace"},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	res, err := e.es.Raw().Search(
		e.es.Raw().Search.WithContext(ctx),
		e.es.Raw().Search.WithIndex(e.es.TenantIndex(tenantID)),
		e.es.Raw().Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("search replay docs: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search error: %s", string(b))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	docs := make([]map[string]any, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		docs = append(docs, h.Source)
	}
	return docs, nil
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

// isErrorOrWarn 判断日志是否为 ERROR 或 WARN 级别
// 委托给 es.ExtractLogLevel 做统一的级别提取，保证与 ES 层逻辑一致
func isErrorOrWarn(log map[string]any) bool {
	message, _ := log["message"].(string)
	level := es.ExtractLogLevel(&es.UCloudLog{Message: message})
	return level == "ERROR" || level == "WARN"
}

// scaleLogs 按故障强度缩放错误/警告日志数量
func scaleLogs(logs []map[string]any, intensity float64) []map[string]any {
	if intensity == 1.0 {
		return logs
	}

	var result []map[string]any
	for _, log := range logs {
		if isErrorOrWarn(log) {
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

// scaleTrafficLogs 按流量倍率缩放正常日志，按故障强度缩放错误/警告日志
func scaleTrafficLogs(logs []map[string]any, rate, intensity float64) []map[string]any {
	var result []map[string]any
	for _, log := range logs {
		if isErrorOrWarn(log) {
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

// tagDocs 在每条文档顶层注入 replay_session_id 字段
func tagDocs(docs []map[string]any, sessionID string) {
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

// writeDocs 按 kubernetes_namespace 分组将日志批量写入 ES
// 索引格式：{prefix}_{namespace}-{date}
func (e *ReplayEngine) writeDocs(ctx context.Context, logs []map[string]any, baseTime time.Time) (*ReplayResult, error) {
	prefix := e.es.Prefix()
	date := baseTime.Format("2006.01.02")

	// 按 kubernetes_namespace 分组
	logsByNamespace := make(map[string][]map[string]any)
	for _, log := range logs {
		ns, _ := log["kubernetes_namespace"].(string)
		logsByNamespace[ns] = append(logsByNamespace[ns], log)
	}

	totalLogs := 0
	for ns, nsLogs := range logsByNamespace {
		index := fmt.Sprintf("%s_%s-%s", prefix, ns, date)
		for i := 0; i < len(nsLogs); i += 100 {
			end := i + 100
			if end > len(nsLogs) {
				end = len(nsLogs)
			}
			// 直接传 []map[string]any，BulkIndex 已升级为强类型签名
			if err := e.es.BulkIndex(ctx, index, nsLogs[i:end]); err != nil {
				return nil, fmt.Errorf("bulk index logs for %s: %w", ns, err)
			}
		}
		totalLogs += len(nsLogs)
	}

	return &ReplayResult{LogsWritten: totalLogs}, nil
}
