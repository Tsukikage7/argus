package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// LogEntry 表示一条 ES 日志记录
type LogEntry struct {
	Timestamp  string         `json:"@timestamp"`
	TraceID    string         `json:"trace_id"`
	SpanID     string         `json:"span_id"`
	ParentSpan string         `json:"parent_span_id"`
	Service    ServiceInfo    `json:"service"`
	Severity   string         `json:"severity"`
	Body       string         `json:"body"`
	Attributes map[string]any `json:"attributes"`
	Resource   map[string]any `json:"resource"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	InstanceID string `json:"instance_id"`
}

// TraceSpan 表示一个调用链 span
type TraceSpan struct {
	TraceID      string        `json:"trace_id"`
	SpanID       string        `json:"span_id"`
	ParentSpanID string        `json:"parent_span_id"`
	Service      string        `json:"service"`
	Duration     time.Duration `json:"duration"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
	Attributes   map[string]any `json:"attributes"`
}

// QueryLogs 按条件查询日志
func (c *Client) QueryLogs(ctx context.Context, service, severity, timeRange, keyword string) ([]LogEntry, error) {
	query := buildLogQuery(service, severity, timeRange, keyword)
	index := c.logIndex(service)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithSize(50),
		c.es.Search.WithSort("@timestamp:desc"),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: search error: %s", string(b))
	}

	return parseLogResponse(res.Body)
}

// QueryTrace 通过 trace_id 查询完整调用链
func (c *Client) QueryTrace(ctx context.Context, traceID string) ([]TraceSpan, error) {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"trace_id.keyword": traceID,
			},
		},
		"sort": []map[string]any{
			{"@timestamp": "asc"},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal query: %w", err)
	}

	index := fmt.Sprintf("%s-traces-*", c.prefix)
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithSize(100),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search trace: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: trace search error: %s", string(b))
	}

	return parseTraceResponse(res.Body)
}

// BulkIndex 批量写入文档
func (c *Client) BulkIndex(ctx context.Context, index string, docs []map[string]any) error {
	var buf bytes.Buffer
	for _, doc := range docs {
		meta := map[string]any{"index": map[string]any{"_index": index}}
		metaLine, _ := json.Marshal(meta)
		docLine, _ := json.Marshal(doc)
		buf.Write(metaLine)
		buf.WriteByte('\n')
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := c.es.Bulk(bytes.NewReader(buf.Bytes()),
		c.es.Bulk.WithContext(ctx),
		c.es.Bulk.WithIndex(index),
	)
	if err != nil {
		return fmt.Errorf("es: bulk: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("es: bulk error: %s", string(b))
	}
	return nil
}

func (c *Client) logIndex(service string) string {
	if service != "" {
		return fmt.Sprintf("%s-logs-%s-*", c.prefix, service)
	}
	return fmt.Sprintf("%s-logs-*", c.prefix)
}

func buildLogQuery(service, severity, timeRange, keyword string) map[string]any {
	must := make([]map[string]any, 0, 4)

	if service != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"service.name.keyword": service},
		})
	}
	if severity != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"severity.keyword": strings.ToUpper(severity)},
		})
	}
	if timeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{
					"gte": parseTimeRange(timeRange),
				},
			},
		})
	}
	if keyword != "" {
		must = append(must, map[string]any{
			"match": map[string]any{"body": keyword},
		})
	}

	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{"must": must},
		},
	}
}

// parseTimeRange 将 "last 15m", "last 1h" 转为 ES range 表达式
func parseTimeRange(tr string) string {
	tr = strings.TrimSpace(strings.ToLower(tr))
	tr = strings.TrimPrefix(tr, "last ")
	return "now-" + tr
}

func parseLogResponse(body io.Reader) ([]LogEntry, error) {
	var result struct {
		Hits struct {
			Hits []struct {
				Source LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, err
	}
	logs := make([]LogEntry, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		logs = append(logs, h.Source)
	}
	return logs, nil
}

// ReplayServiceStats 单个服务在回放中的统计数据
type ReplayServiceStats struct {
	ServiceName  string
	InfoCount    int
	WarnCount    int
	ErrorCount   int
	AvgLatencyMs int
	P99LatencyMs int
}

// ReplayStats 回放聚合统计结果
type ReplayStats struct {
	Services []ReplayServiceStats
}

// QueryReplayStats 按 replay_session_id 聚合查询回放影响面数据
func (c *Client) QueryReplayStats(ctx context.Context, replaySessionID string) (*ReplayStats, error) {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"attributes.replay_session_id.keyword": replaySessionID,
			},
		},
		"size": 0,
		"aggs": map[string]any{
			"by_service": map[string]any{
				"terms": map[string]any{
					"field": "service.name.keyword",
					"size":  20,
				},
				"aggs": map[string]any{
					"by_severity": map[string]any{
						"terms": map[string]any{
							"field": "severity.keyword",
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal replay query: %w", err)
	}

	index := fmt.Sprintf("%s-logs-*", c.prefix)
	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("es: replay stats search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: replay stats error: %s", string(b))
	}

	return parseReplayStatsResponse(res.Body)
}

func parseReplayStatsResponse(body io.Reader) (*ReplayStats, error) {
	var result struct {
		Aggregations struct {
			ByService struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
					BySeverity struct {
						Buckets []struct {
							Key      string `json:"key"`
							DocCount int    `json:"doc_count"`
						} `json:"buckets"`
					} `json:"by_severity"`
				} `json:"buckets"`
			} `json:"by_service"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse replay stats: %w", err)
	}

	stats := &ReplayStats{}
	for _, bucket := range result.Aggregations.ByService.Buckets {
		svc := ReplayServiceStats{ServiceName: bucket.Key}
		for _, sevBucket := range bucket.BySeverity.Buckets {
			switch strings.ToUpper(sevBucket.Key) {
			case "INFO":
				svc.InfoCount = sevBucket.DocCount
			case "WARN":
				svc.WarnCount = sevBucket.DocCount
			case "ERROR":
				svc.ErrorCount = sevBucket.DocCount
			}
		}
		stats.Services = append(stats.Services, svc)
	}
	return stats, nil
}

func parseTraceResponse(body io.Reader) ([]TraceSpan, error) {
	var result struct {
		Hits struct {
			Hits []struct {
				Source TraceSpan `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, err
	}
	spans := make([]TraceSpan, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		spans = append(spans, h.Source)
	}
	return spans, nil
}
