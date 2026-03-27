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
// 按 kubernetes_namespace 分组写入，索引名格式：{prefix}_{namespace}-{date}
func (g *Generator) GenerateScenario(ctx context.Context, scenario Scenario) error {
	baseTime := time.Now()
	logs := scenario.GenerateLogs(baseTime) // 不再有 traces

	// 按 kubernetes_namespace 分组
	logsByNamespace := make(map[string][]map[string]any)
	for _, log := range logs {
		ns, _ := log["kubernetes_namespace"].(string)
		logsByNamespace[ns] = append(logsByNamespace[ns], log)
	}

	date := baseTime.Format("2006.01.02")
	for ns, nsLogs := range logsByNamespace {
		index := fmt.Sprintf("%s_%s-%s", g.prefix, ns, date)

		// 分批写入（每批 100 条）
		for i := 0; i < len(nsLogs); i += 100 {
			end := i + 100
			if end > len(nsLogs) {
				end = len(nsLogs)
			}
			// 直接传 []map[string]any，BulkIndex 已升级为强类型签名
			if err := g.es.BulkIndex(ctx, index, nsLogs[i:end]); err != nil {
				return fmt.Errorf("bulk index logs for %s: %w", ns, err)
			}
		}
		fmt.Printf("  [mock] wrote %d logs to %s\n", len(nsLogs), index)
	}

	fmt.Printf("[mock] scenario %q generated successfully\n", scenario.Name)
	return nil
}
