package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/infrastructure/es"
)

// Generator 生成 mock 日志数据并写入 ES
type Generator struct {
	es     *es.Client
	prefix string
}

// NewGenerator 创建 mock 数据生成器
func NewGenerator(esClient *es.Client) *Generator {
	return &Generator{
		es:     esClient,
		prefix: esClient.Prefix(),
	}
}

// GenerateAll 生成所有场景的数据
func (g *Generator) GenerateAll(ctx context.Context) error {
	for _, scenario := range AllScenarios() {
		if err := g.GenerateScenario(ctx, scenario); err != nil {
			return fmt.Errorf("generate %s: %w", scenario.Name, err)
		}
	}
	return nil
}

// GenerateScenario 生成单个场景的数据
func (g *Generator) GenerateScenario(ctx context.Context, scenario Scenario) error {
	baseTime := time.Now()
	logs, traces := scenario.GenerateLogs(baseTime)

	// 按服务分组写入日志索引
	logsByService := make(map[string][]map[string]any)
	for _, log := range logs {
		svcInfo, ok := log["service"].(map[string]any)
		if !ok {
			continue
		}
		svcName, _ := svcInfo["name"].(string)
		logsByService[svcName] = append(logsByService[svcName], log)
	}

	date := baseTime.Format("2006.01.02")
	for svc, svcLogs := range logsByService {
		index := fmt.Sprintf("%s-logs-%s-%s", g.prefix, svc, date)

		// 分批写入（每批 100 条）
		for i := 0; i < len(svcLogs); i += 100 {
			end := i + 100
			if end > len(svcLogs) {
				end = len(svcLogs)
			}
			if err := g.es.BulkIndex(ctx, index, svcLogs[i:end]); err != nil {
				return fmt.Errorf("bulk index logs for %s: %w", svc, err)
			}
		}
		fmt.Printf("  [mock] wrote %d logs to %s\n", len(svcLogs), index)
	}

	// 写入 trace 索引
	if len(traces) > 0 {
		traceIndex := fmt.Sprintf("%s-traces-%s", g.prefix, date)
		for i := 0; i < len(traces); i += 100 {
			end := i + 100
			if end > len(traces) {
				end = len(traces)
			}
			if err := g.es.BulkIndex(ctx, traceIndex, traces[i:end]); err != nil {
				return fmt.Errorf("bulk index traces: %w", err)
			}
		}
		fmt.Printf("  [mock] wrote %d traces to %s\n", len(traces), fmt.Sprintf("%s-traces-%s", g.prefix, date))
	}

	fmt.Printf("[mock] scenario %q generated successfully\n", scenario.Name)
	return nil
}
