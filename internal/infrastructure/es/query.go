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

// ReplayServiceStats 单个 namespace 在回放中的统计数据（按 kubernetes_namespace 聚合）
// 注意：UCloud 格式没有独立的 severity 字段，级别需从 message 中提取
type ReplayServiceStats struct {
	Namespace string
	DocCount  int
}

// ReplayStats 回放聚合统计结果
type ReplayStats struct {
	Services []ReplayServiceStats
}

// LogSummaryBucket namespace × 级别聚合桶
type LogSummaryBucket struct {
	Namespace string `json:"namespace"`
	Level     string `json:"level"`
	Count     int    `json:"count"`
}

// LogSummary 日志聚合摘要
type LogSummary struct {
	Buckets []LogSummaryBucket `json:"buckets"`
	Total   int                `json:"total"`
}

// TopologyNodeMetric 拓扑节点在指定时间窗口内的健康指标
type TopologyNodeMetric struct {
	Namespace  string  `json:"namespace"`
	Health     string  `json:"health"`
	ErrorRate  float64 `json:"error_rate"`
	AlertCount int     `json:"alert_count"`
}

// QueryLogSummary 按 namespace × 级别聚合日志摘要（租户隔离）
// 由于 UCloud 日志无独立 severity 字段，通过 message 关键词匹配推断级别
func (c *Client) QueryLogSummary(ctx context.Context, tenantID, timeRange string) (*LogSummary, error) {
	must := []map[string]any{}
	if timeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{
					"gte": parseTimeRange(timeRange),
				},
			},
		})
	}

	queryClause := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 {
		queryClause = map[string]any{"bool": map[string]any{"must": must}}
	}

	query := map[string]any{
		"query": queryClause,
		"size":  0,
		"aggs": map[string]any{
			"by_namespace": map[string]any{
				"terms": map[string]any{
					"field": "kubernetes_namespace.keyword",
					"size":  50,
				},
				"aggs": map[string]any{
					"by_level": map[string]any{
						"filters": map[string]any{
							"filters": map[string]any{
								"ERROR": map[string]any{"bool": map[string]any{
									"should": []map[string]any{
										{"match_phrase": map[string]any{"message": "ERROR"}},
										{"match_phrase": map[string]any{"message": "error"}},
									},
									"minimum_should_match": 1,
								}},
								"WARN": map[string]any{"bool": map[string]any{
									"should": []map[string]any{
										{"match_phrase": map[string]any{"message": "WARN"}},
										{"match_phrase": map[string]any{"message": "warn"}},
									},
									"minimum_should_match": 1,
								}},
								"INFO": map[string]any{"bool": map[string]any{
									"should": []map[string]any{
										{"match_phrase": map[string]any{"message": "INFO"}},
										{"match_phrase": map[string]any{"message": "info"}},
									},
									"minimum_should_match": 1,
								}},
								"DEBUG": map[string]any{"bool": map[string]any{
									"should": []map[string]any{
										{"match_phrase": map[string]any{"message": "DEBUG"}},
										{"match_phrase": map[string]any{"message": "debug"}},
									},
									"minimum_should_match": 1,
								}},
							},
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal log summary query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: log summary search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: log summary error: %s", string(b))
	}

	return parseLogSummaryResponse(res.Body)
}

// QueryLogs 条件查询日志（租户隔离），支持 namespace/keyword/level/timeRange/limit
func (c *Client) QueryLogs(ctx context.Context, tenantID string, opts LogQueryOpts) ([]UCloudLog, error) {
	must := []map[string]any{}
	if opts.Namespace != "" {
		must = append(must, map[string]any{
			"term": map[string]any{
				"kubernetes_namespace.keyword": opts.Namespace,
			},
		})
	}
	if opts.Keyword != "" {
		must = append(must, map[string]any{
			"match_phrase": map[string]any{
				"message": opts.Keyword,
			},
		})
	}
	if opts.Level != "" {
		// 支持逗号分隔的多级别过滤，同时匹配大写和小写形式
		levels := strings.Split(opts.Level, ",")
		should := make([]map[string]any, 0, len(levels)*2)
		for _, lv := range levels {
			lv = strings.TrimSpace(lv)
			if lv == "" {
				continue
			}
			// 大写形式（文本日志 [ERROR]）
			should = append(should, map[string]any{
				"match_phrase": map[string]any{"message": strings.ToUpper(lv)},
			})
			// 小写形式（结构化 JSON "level":"error"）
			lower := strings.ToLower(lv)
			if lower != strings.ToUpper(lv) {
				should = append(should, map[string]any{
					"match_phrase": map[string]any{"message": lower},
				})
			}
		}
		if len(should) > 0 {
			must = append(must, map[string]any{
				"bool": map[string]any{
					"should":               should,
					"minimum_should_match": 1,
				},
			})
		}
	}
	if opts.TimeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{
					"gte": parseTimeRange(opts.TimeRange),
				},
			},
		})
	}

	queryClause := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 {
		queryClause = map[string]any{"bool": map[string]any{"must": must}}
	}

	limit := opts.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := map[string]any{
		"query": queryClause,
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "desc"}},
		},
		"size": limit,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal log query: %w", err)
	}

	index := c.TenantIndex(tenantID)
	if opts.Namespace != "" {
		index = c.TenantNamespaceIndex(tenantID, opts.Namespace)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: log query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: log query error: %s", string(b))
	}

	return parseUCloudLogResponse(res.Body)
}

// LogQueryOpts 日志条件查询参数
type LogQueryOpts struct {
	Namespace string
	Service   string
	Keyword   string
	Level     string
	TimeRange string
	Limit     int
}

// QueryByRequestUUID 通过 request_uuid 全文搜索租户范围内的日志
// 使用 match_phrase 匹配 message 字段，限定租户索引范围
func (c *Client) QueryByRequestUUID(ctx context.Context, tenantID, requestUUID string, timeRange string) ([]UCloudLog, error) {
	must := []map[string]any{
		{
			"match_phrase": map[string]any{
				"message": requestUUID,
			},
		},
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

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{"must": must},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "desc"}},
		},
		"size": 200,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search by request_uuid: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: search error: %s", string(b))
	}

	return parseUCloudLogResponse(res.Body)
}

// QueryByNamespace 按指定 namespace 查询租户范围内的日志
func (c *Client) QueryByNamespace(ctx context.Context, tenantID, namespace, keyword, timeRange string) ([]UCloudLog, error) {
	must := []map[string]any{
		{
			"term": map[string]any{
				"kubernetes_namespace.keyword": namespace,
			},
		},
	}
	if keyword != "" {
		must = append(must, map[string]any{
			"match_phrase": map[string]any{
				"message": keyword,
			},
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

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{"must": must},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "desc"}},
		},
		"size": 50,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantNamespaceIndex(tenantID, namespace)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search by namespace: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: namespace search error: %s", string(b))
	}

	return parseUCloudLogResponse(res.Body)
}

// QueryByKeyword 在租户范围内全文搜索关键词
func (c *Client) QueryByKeyword(ctx context.Context, tenantID, keyword, timeRange string) ([]UCloudLog, error) {
	// 第一轮：match_phrase 精确匹配
	logs, err := c.queryByKeywordMode(ctx, tenantID, keyword, timeRange, "match_phrase")
	if err != nil {
		return nil, err
	}
	if len(logs) > 0 {
		return logs, nil
	}

	// 第二轮：降级为 match 分词匹配
	logs, err = c.queryByKeywordMode(ctx, tenantID, keyword, timeRange, "match")
	if err != nil {
		return nil, err
	}
	// 标记为模糊匹配结果
	for i := range logs {
		logs[i].Message = "[fuzzy_match] " + logs[i].Message
	}
	return logs, nil
}

// queryByKeywordMode 按指定匹配模式执行关键词搜索
func (c *Client) queryByKeywordMode(ctx context.Context, tenantID, keyword, timeRange, matchMode string) ([]UCloudLog, error) {
	must := []map[string]any{
		{
			matchMode: map[string]any{
				"message": keyword,
			},
		},
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

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{"must": must},
		},
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "desc"}},
		},
		"size": 50,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: search by keyword (%s): %w", matchMode, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: keyword search error (%s): %s", matchMode, string(b))
	}

	return parseUCloudLogResponse(res.Body)
}

// BulkIndex 批量写入文档到指定索引
// 签名使用强类型 []map[string]any，避免调用方不必要的类型转换
// 写入部分失败时（bulk 响应 errors=true）返回 error
func (c *Client) BulkIndex(ctx context.Context, index string, docs []map[string]any) error {
	var buf bytes.Buffer
	for i, doc := range docs {
		meta := map[string]any{"index": map[string]any{"_index": index}}
		metaLine, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("es: bulk marshal meta[%d]: %w", i, err)
		}
		docLine, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("es: bulk marshal doc[%d]: %w", i, err)
		}
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

	// 解析 bulk 响应，检查 item 级别的写入失败
	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int `json:"status"`
			Error  *struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"items"`
	}
	bodyBytes, _ := io.ReadAll(res.Body)
	if jsonErr := json.Unmarshal(bodyBytes, &bulkResp); jsonErr != nil {
		// 解析失败不阻断，以响应状态码为准
		return nil
	}
	if bulkResp.Errors {
		// 收集所有失败 item 的错误信息
		var errMsgs []string
		for _, item := range bulkResp.Items {
			for action, result := range item {
				if result.Error != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("[%s] %s: %s", action, result.Error.Type, result.Error.Reason))
				}
			}
		}
		if len(errMsgs) > 0 {
			return fmt.Errorf("es: bulk partial failure (%d errors): %s", len(errMsgs), strings.Join(errMsgs, "; "))
		}
	}
	return nil
}

// QueryReplayStats 按 replay_session_id 聚合查询回放影响面数据（租户隔离）
func (c *Client) QueryReplayStats(ctx context.Context, tenantID, replaySessionID string) (*ReplayStats, error) {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"replay_session_id.keyword": replaySessionID,
			},
		},
		"size": 0,
		"aggs": map[string]any{
			"by_namespace": map[string]any{
				"terms": map[string]any{
					// 使用 .keyword 子字段聚合，避免分词器将 namespace 拆分为多个 token
					"field": "kubernetes_namespace.keyword",
					"size":  20,
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal replay query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
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

// QueryTopologyNodeMetrics 按 namespace 聚合最近时间窗口内的错误率和告警数
func (c *Client) QueryTopologyNodeMetrics(ctx context.Context, tenantID, timeRange string) (map[string]TopologyNodeMetric, error) {
	if strings.TrimSpace(timeRange) == "" {
		timeRange = "last 1h"
	}

	query := map[string]any{
		"query": map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{
					"gte": parseTimeRange(timeRange),
				},
			},
		},
		"size": 0,
		"aggs": map[string]any{
			"by_namespace": map[string]any{
				"terms": map[string]any{
					"field": "kubernetes_namespace.keyword",
					"size":  50,
				},
				"aggs": map[string]any{
					"error_docs": map[string]any{
						"filter": topologyErrorFilter(),
					},
					"alert_docs": map[string]any{
						"filter": topologyAlertFilter(),
					},
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal topology metrics query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: topology metrics search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: topology metrics error: %s", string(b))
	}

	return parseTopologyNodeMetricsResponse(res.Body)
}

// parseTimeRange 将 "last 15m"、"last 1h" 转为 ES range 表达式
func parseTimeRange(tr string) string {
	tr = strings.TrimSpace(strings.ToLower(tr))
	tr = strings.TrimPrefix(tr, "last ")
	return "now-" + tr
}

// parseUCloudLogResponse 解析 ES 响应到 []UCloudLog
func parseUCloudLogResponse(body io.Reader) ([]UCloudLog, error) {
	var result struct {
		Hits struct {
			Hits []struct {
				Source UCloudLog `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: decode response: %w", err)
	}
	logs := make([]UCloudLog, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		logs = append(logs, h.Source)
	}
	return logs, nil
}

// parseReplayStatsResponse 解析回放聚合统计响应
func parseReplayStatsResponse(body io.Reader) (*ReplayStats, error) {
	var result struct {
		Aggregations struct {
			ByNamespace struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_namespace"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: parse replay stats: %w", err)
	}

	stats := &ReplayStats{}
	for _, bucket := range result.Aggregations.ByNamespace.Buckets {
		stats.Services = append(stats.Services, ReplayServiceStats{
			Namespace: bucket.Key,
			DocCount:  bucket.DocCount,
		})
	}
	return stats, nil
}

// parseLogSummaryResponse 解析日志聚合摘要响应
func parseLogSummaryResponse(body io.Reader) (*LogSummary, error) {
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations struct {
			ByNamespace struct {
				Buckets []struct {
					Key     string `json:"key"`
					ByLevel struct {
						Buckets map[string]struct {
							DocCount int `json:"doc_count"`
						} `json:"buckets"`
					} `json:"by_level"`
				} `json:"buckets"`
			} `json:"by_namespace"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: parse log summary: %w", err)
	}

	summary := &LogSummary{Total: result.Hits.Total.Value}
	for _, nsBucket := range result.Aggregations.ByNamespace.Buckets {
		for level, levelBucket := range nsBucket.ByLevel.Buckets {
			if levelBucket.DocCount > 0 {
				summary.Buckets = append(summary.Buckets, LogSummaryBucket{
					Namespace: nsBucket.Key,
					Level:     level,
					Count:     levelBucket.DocCount,
				})
			}
		}
	}
	return summary, nil
}

func topologyErrorFilter() map[string]any {
	return topologyMessageFilter([]string{
		"ERROR",
		"error",
		"status 500",
		"status 502",
		"status 503",
		"status 504",
	})
}

func topologyAlertFilter() map[string]any {
	return topologyMessageFilter([]string{
		"ERROR",
		"error",
		"WARN",
		"warn",
		"warning",
		"WARNING",
		"critical",
		"CRITICAL",
		"alert",
		"ALERT",
		"alarm",
		"ALARM",
		"status 500",
		"status 502",
		"status 503",
		"status 504",
	})
}

func topologyMessageFilter(phrases []string) map[string]any {
	should := make([]map[string]any, 0, len(phrases))
	for _, phrase := range phrases {
		should = append(should, map[string]any{
			"match_phrase": map[string]any{
				"message": phrase,
			},
		})
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func parseTopologyNodeMetricsResponse(body io.Reader) (map[string]TopologyNodeMetric, error) {
	var result struct {
		Aggregations struct {
			ByNamespace struct {
				Buckets []struct {
					Key       string `json:"key"`
					DocCount  int    `json:"doc_count"`
					ErrorDocs struct {
						DocCount int `json:"doc_count"`
					} `json:"error_docs"`
					AlertDocs struct {
						DocCount int `json:"doc_count"`
					} `json:"alert_docs"`
				} `json:"buckets"`
			} `json:"by_namespace"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: parse topology metrics: %w", err)
	}

	metrics := make(map[string]TopologyNodeMetric, len(result.Aggregations.ByNamespace.Buckets))
	for _, bucket := range result.Aggregations.ByNamespace.Buckets {
		var errorRate float64
		if bucket.DocCount > 0 {
			errorRate = float64(bucket.ErrorDocs.DocCount) / float64(bucket.DocCount)
		}

		metrics[bucket.Key] = TopologyNodeMetric{
			Namespace:  bucket.Key,
			Health:     topologyHealth(errorRate),
			ErrorRate:  errorRate,
			AlertCount: bucket.AlertDocs.DocCount,
		}
	}

	return metrics, nil
}

func topologyHealth(errorRate float64) string {
	switch {
	case errorRate >= 0.2:
		return "critical"
	case errorRate >= 0.05:
		return "degraded"
	default:
		return "healthy"
	}
}

// LogFacets 日志分面聚合结果
type LogFacets struct {
	Namespaces []FacetBucket `json:"namespaces"`
	Services   []FacetBucket `json:"services"`
	Levels     []FacetBucket `json:"levels"`
	Pods       []FacetBucket `json:"pods"`
}

// FacetBucket 分面聚合桶
type FacetBucket struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// QueryLogFacets 按 namespace/service/level/pod 聚合日志分面（租户隔离）
func (c *Client) QueryLogFacets(ctx context.Context, tenantID, timeRange string) (*LogFacets, error) {
	must := []map[string]any{}
	if timeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{"gte": parseTimeRange(timeRange)},
			},
		})
	}

	queryClause := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 {
		queryClause = map[string]any{"bool": map[string]any{"must": must}}
	}

	query := map[string]any{
		"query": queryClause,
		"size":  0,
		"aggs": map[string]any{
			"by_namespace": map[string]any{
				"terms": map[string]any{"field": "kubernetes_namespace.keyword", "size": 50},
			},
			"by_service": map[string]any{
				"terms": map[string]any{"field": "kubernetes_labels_app.keyword", "size": 50},
			},
			"by_pod": map[string]any{
				"terms": map[string]any{"field": "kubernetes_pod.keyword", "size": 100},
			},
			"by_level": map[string]any{
				"filters": map[string]any{
					"filters": map[string]any{
						"ERROR": map[string]any{"bool": map[string]any{
							"should": []map[string]any{
								{"match_phrase": map[string]any{"message": "ERROR"}},
								{"match_phrase": map[string]any{"message": "error"}},
							},
							"minimum_should_match": 1,
						}},
						"WARN": map[string]any{"bool": map[string]any{
							"should": []map[string]any{
								{"match_phrase": map[string]any{"message": "WARN"}},
								{"match_phrase": map[string]any{"message": "warn"}},
							},
							"minimum_should_match": 1,
						}},
						"INFO": map[string]any{"bool": map[string]any{
							"should": []map[string]any{
								{"match_phrase": map[string]any{"message": "INFO"}},
								{"match_phrase": map[string]any{"message": "info"}},
							},
							"minimum_should_match": 1,
						}},
						"DEBUG": map[string]any{"bool": map[string]any{
							"should": []map[string]any{
								{"match_phrase": map[string]any{"message": "DEBUG"}},
								{"match_phrase": map[string]any{"message": "debug"}},
							},
							"minimum_should_match": 1,
						}},
					},
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal log facets query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: log facets search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: log facets error: %s", string(b))
	}

	return parseLogFacetsResponse(res.Body)
}

// FaultLogEntry 故障日志条目（前端展示用）
type FaultLogEntry struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	Level        string `json:"level"`
	Service      string `json:"service"`
	Message      string `json:"message"`
	RequestUUID  string `json:"request_uuid,omitempty"`
	Namespace    string `json:"namespace"`
	Pod          string `json:"pod,omitempty"`
}

// FaultLogResult 故障日志查询结果
type FaultLogResult struct {
	Total int             `json:"total"`
	Logs  []FaultLogEntry `json:"logs"`
}

// QueryFaultLogs 查询故障日志（ERROR/WARN 级别），支持分面过滤
func (c *Client) QueryFaultLogs(ctx context.Context, tenantID string, opts LogQueryOpts) (*FaultLogResult, error) {
	must := []map[string]any{}

	// 默认只查 ERROR + WARN
	if opts.Level == "" {
		opts.Level = "ERROR,WARN"
	}

	if opts.Namespace != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"kubernetes_namespace.keyword": opts.Namespace},
		})
	}
	if opts.Service != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"kubernetes_labels_app.keyword": opts.Service},
		})
	}
	if opts.Keyword != "" {
		must = append(must, map[string]any{
			"match_phrase": map[string]any{"message": opts.Keyword},
		})
	}

	// 级别过滤
	levels := strings.Split(opts.Level, ",")
	should := make([]map[string]any, 0, len(levels)*2)
	for _, lv := range levels {
		lv = strings.TrimSpace(lv)
		if lv == "" {
			continue
		}
		should = append(should, map[string]any{
			"match_phrase": map[string]any{"message": strings.ToUpper(lv)},
		})
		lower := strings.ToLower(lv)
		if lower != strings.ToUpper(lv) {
			should = append(should, map[string]any{
				"match_phrase": map[string]any{"message": lower},
			})
		}
	}
	if len(should) > 0 {
		must = append(must, map[string]any{
			"bool": map[string]any{"should": should, "minimum_should_match": 1},
		})
	}

	if opts.TimeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{"gte": parseTimeRange(opts.TimeRange)},
			},
		})
	}

	queryClause := map[string]any{"match_all": map[string]any{}}
	if len(must) > 0 {
		queryClause = map[string]any{"bool": map[string]any{"must": must}}
	}

	limit := opts.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := map[string]any{
		"query": queryClause,
		"sort":  []map[string]any{{"@timestamp": map[string]any{"order": "desc"}}},
		"size":  limit,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal fault logs query: %w", err)
	}

	index := c.TenantIndex(tenantID)
	if opts.Namespace != "" {
		index = c.TenantNamespaceIndex(tenantID, opts.Namespace)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: fault logs search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: fault logs error: %s", string(b))
	}

	return parseFaultLogResponse(res.Body)
}

// LogContextResult 日志上下文查询结果
type LogContextResult struct {
	RequestUUID string          `json:"request_uuid"`
	Logs        []FaultLogEntry `json:"logs"`
}

// QueryLogContext 按 request_uuid 查询上下文日志窗口
func (c *Client) QueryLogContext(ctx context.Context, tenantID, requestUUID, timeRange string) (*LogContextResult, error) {
	logs, err := c.QueryByRequestUUID(ctx, tenantID, requestUUID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("es: log context: %w", err)
	}

	entries := make([]FaultLogEntry, 0, len(logs))
	for _, log := range logs {
		entries = append(entries, ucloudLogToFaultEntry(&log))
	}

	return &LogContextResult{
		RequestUUID: requestUUID,
		Logs:        entries,
	}, nil
}

// ucloudLogToFaultEntry 将 UCloudLog 转换为 FaultLogEntry
func ucloudLogToFaultEntry(log *UCloudLog) FaultLogEntry {
	return FaultLogEntry{
		Timestamp:   log.Timestamp,
		Level:       ExtractLogLevel(log),
		Service:     log.KubernetesLabelsApp,
		Message:     log.Message,
		RequestUUID: ExtractRequestUUID(log.Message),
		Namespace:   log.KubernetesNamespace,
		Pod:         log.KubernetesPod,
	}
}

// parseLogFacetsResponse 解析日志分面聚合响应
func parseLogFacetsResponse(body io.Reader) (*LogFacets, error) {
	var result struct {
		Aggregations struct {
			ByNamespace struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_namespace"`
			ByService struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_service"`
			ByPod struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_pod"`
			ByLevel struct {
				Buckets map[string]struct {
					DocCount int `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_level"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: parse log facets: %w", err)
	}

	facets := &LogFacets{}
	for _, b := range result.Aggregations.ByNamespace.Buckets {
		facets.Namespaces = append(facets.Namespaces, FacetBucket{Name: b.Key, Count: b.DocCount})
	}
	for _, b := range result.Aggregations.ByService.Buckets {
		facets.Services = append(facets.Services, FacetBucket{Name: b.Key, Count: b.DocCount})
	}
	for _, b := range result.Aggregations.ByPod.Buckets {
		facets.Pods = append(facets.Pods, FacetBucket{Name: b.Key, Count: b.DocCount})
	}
	for level, b := range result.Aggregations.ByLevel.Buckets {
		if b.DocCount > 0 {
			facets.Levels = append(facets.Levels, FacetBucket{Name: level, Count: b.DocCount})
		}
	}
	return facets, nil
}

// parseFaultLogResponse 解析故障日志查询响应
func parseFaultLogResponse(body io.Reader) (*FaultLogResult, error) {
	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string   `json:"_id"`
				Source UCloudLog `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("es: parse fault logs: %w", err)
	}

	entries := make([]FaultLogEntry, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		entry := ucloudLogToFaultEntry(&h.Source)
		entry.ID = h.ID
		entries = append(entries, entry)
	}

	return &FaultLogResult{
		Total: result.Hits.Total.Value,
		Logs:  entries,
	}, nil
}

// ── Trace 查询 ──────────────────────────────────────────────────────────

// TraceListItem Trace 列表条目
type TraceListItem struct {
	RequestUUID  string   `json:"request_uuid"`
	EntryService string   `json:"entry_service"`
	StatusCode   int      `json:"status_code"`
	DurationMs   int      `json:"duration_ms"`
	Timestamp    string   `json:"timestamp"`
	Services     []string `json:"services"`
}

// TraceListResult Trace 列表查询结果
type TraceListResult struct {
	Total  int             `json:"total"`
	Traces []TraceListItem `json:"traces"`
}

// TraceSpan 链路中的单个服务调用
type TraceSpan struct {
	Service    string         `json:"service"`
	Operation  string         `json:"operation"`
	StartMs    int            `json:"start_ms"`
	DurationMs int            `json:"duration_ms"`
	Status     string         `json:"status"`
	Logs       []TraceSpanLog `json:"logs"`
}

// TraceSpanLog span 关联的日志条目
type TraceSpanLog struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// TraceDetail 链路详情
type TraceDetail struct {
	RequestUUID    string      `json:"request_uuid"`
	EntryService   string      `json:"entry_service"`
	TotalDurationMs int        `json:"total_duration_ms"`
	Timestamp      string      `json:"timestamp"`
	Spans          []TraceSpan `json:"spans"`
}

// TraceQueryOpts Trace 列表查询参数
type TraceQueryOpts struct {
	RequestUUID string
	Service     string
	TimeRange   string
	Limit       int
}

// QueryTraces 查询 Trace 列表（应用层聚合）
// 策略：查询包含 UUID 模式的日志，在应用层按 request_uuid 分组
func (c *Client) QueryTraces(ctx context.Context, tenantID string, opts TraceQueryOpts) (*TraceListResult, error) {
	must := []map[string]any{}

	if opts.RequestUUID != "" {
		// 指定 UUID 时直接用 match_phrase 精确匹配
		must = append(must, map[string]any{
			"match_phrase": map[string]any{"message": opts.RequestUUID},
		})
	} else {
		// 列表模式：用 match 查询匹配含 UUID 片段的日志
		// ES regexp 作用于分词后的 token，无法匹配完整 UUID，改用 bool should 匹配常见 UUID 前缀格式
		must = append(must, map[string]any{
			"bool": map[string]any{
				"should": []map[string]any{
					// 网关日志中 request_uuid 作为 JSON 字段值
					{"match_phrase": map[string]any{"message": "request_uuid"}},
					// 文本日志中 [uuid.step] 格式
					{"regexp": map[string]any{
						"message.keyword": map[string]any{
							"value": ".*[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}.*",
						},
					}},
				},
				"minimum_should_match": 1,
			},
		})
	}
	if opts.Service != "" {
		must = append(must, map[string]any{
			"term": map[string]any{"kubernetes_labels_app.keyword": opts.Service},
		})
	}
	if opts.TimeRange != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"@timestamp": map[string]any{"gte": parseTimeRange(opts.TimeRange)},
			},
		})
	}

	limit := opts.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	// 内部采样量放大以覆盖更多 trace（每个 trace 可能有多条日志）
	sampleSize := limit * 10
	if sampleSize > 2000 {
		sampleSize = 2000
	}

	query := map[string]any{
		"query": map[string]any{"bool": map[string]any{"must": must}},
		"sort":  []map[string]any{{"@timestamp": map[string]any{"order": "desc"}}},
		"size":  sampleSize,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal traces query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.TenantIndex(tenantID)),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithIgnoreUnavailable(true),
		c.es.Search.WithAllowNoIndices(true),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es: traces search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("es: traces error: %s", string(b))
	}

	logs, err := parseUCloudLogResponse(res.Body)
	if err != nil {
		return nil, err
	}

	return aggregateTraces(logs, limit), nil
}

// aggregateTraces 将日志按 request_uuid 分组聚合为 Trace 列表
// limit 控制返回的最大 trace 数量，total 为去重后的实际 trace 总数
func aggregateTraces(logs []UCloudLog, limit int) *TraceListResult {
	type traceGroup struct {
		uuid      string
		logs      []UCloudLog
		services  map[string]bool
		earliest  string
		latest    string
		gateway   *GatewayMessage
	}

	groups := make(map[string]*traceGroup)
	var order []string

	for i := range logs {
		uuid := ExtractRequestUUID(logs[i].Message)
		if uuid == "" {
			continue
		}

		g, ok := groups[uuid]
		if !ok {
			g = &traceGroup{
				uuid:     uuid,
				services: make(map[string]bool),
				earliest: logs[i].Timestamp,
				latest:   logs[i].Timestamp,
			}
			groups[uuid] = g
			order = append(order, uuid)
		}

		g.logs = append(g.logs, logs[i])
		if svc := logs[i].KubernetesLabelsApp; svc != "" {
			g.services[svc] = true
		}
		if logs[i].Timestamp < g.earliest {
			g.earliest = logs[i].Timestamp
		}
		if logs[i].Timestamp > g.latest {
			g.latest = logs[i].Timestamp
		}

		// 尝试提取网关信息
		if g.gateway == nil && ParseMessage(logs[i].Message) == MessageTypeGateway {
			if gm, err := ParseGatewayMessage(logs[i].Message); err == nil {
				g.gateway = gm
			}
		}
	}

	traces := make([]TraceListItem, 0, len(order))
	for _, uuid := range order {
		g := groups[uuid]
		services := make([]string, 0, len(g.services))
		for svc := range g.services {
			services = append(services, svc)
		}

		item := TraceListItem{
			RequestUUID:  uuid,
			Timestamp:    g.earliest,
			Services:     services,
			StatusCode:   200,
			DurationMs:   0,
		}

		// 从网关日志提取状态码和耗时
		if g.gateway != nil {
			item.DurationMs = g.gateway.ResponseTime
			if g.gateway.ResponseHeaders != nil {
				if status, ok := g.gateway.ResponseHeaders["status"]; ok {
					switch v := status.(type) {
					case float64:
						item.StatusCode = int(v)
					case int:
						item.StatusCode = v
					}
				}
			}
		}

		// 入口服务：优先网关，否则取第一个服务
		if len(services) > 0 {
			item.EntryService = services[0]
			for _, svc := range services {
				if strings.Contains(strings.ToLower(svc), "gateway") {
					item.EntryService = svc
					break
				}
			}
		}

		traces = append(traces, item)
	}

	// total 为去重后的实际 trace 总数
	total := len(traces)

	// 按 limit 截断返回列表
	if limit > 0 && len(traces) > limit {
		traces = traces[:limit]
	}

	return &TraceListResult{
		Total:  total,
		Traces: traces,
	}
}

// QueryTraceDetail 查询单个 Trace 的详情（按 request_uuid 聚合 spans）
func (c *Client) QueryTraceDetail(ctx context.Context, tenantID, requestUUID, timeRange string) (*TraceDetail, error) {
	logs, err := c.QueryByRequestUUID(ctx, tenantID, requestUUID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("es: trace detail: %w", err)
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("es: trace not found: %s", requestUUID)
	}

	return buildTraceDetail(requestUUID, logs), nil
}

// buildTraceDetail 从日志构建 Trace 详情（每个服务一个 span）
func buildTraceDetail(requestUUID string, logs []UCloudLog) *TraceDetail {
	type svcGroup struct {
		service   string
		logs      []UCloudLog
		earliest  string
		latest    string
		hasError  bool
		operation string
	}

	groups := make(map[string]*svcGroup)
	var svcOrder []string
	var earliest string

	for i := range logs {
		svc := logs[i].KubernetesLabelsApp
		if svc == "" {
			svc = logs[i].KubernetesNamespace
		}

		g, ok := groups[svc]
		if !ok {
			g = &svcGroup{
				service:  svc,
				earliest: logs[i].Timestamp,
				latest:   logs[i].Timestamp,
			}
			groups[svc] = g
			svcOrder = append(svcOrder, svc)
		}

		g.logs = append(g.logs, logs[i])
		if logs[i].Timestamp < g.earliest {
			g.earliest = logs[i].Timestamp
		}
		if logs[i].Timestamp > g.latest {
			g.latest = logs[i].Timestamp
		}

		level := ExtractLogLevel(&logs[i])
		if level == "ERROR" {
			g.hasError = true
		}

		// 提取 operation
		if g.operation == "" {
			msgType := ParseMessage(logs[i].Message)
			switch msgType {
			case MessageTypeGateway:
				if gm, err := ParseGatewayMessage(logs[i].Message); err == nil && gm.RequestURI != "" {
					g.operation = gm.RequestURI
				}
			case MessageTypeText:
				if parsed, err := ParseTextLog(logs[i].Message); err == nil && parsed.FuncName != "" {
					g.operation = parsed.FuncName
				}
			case MessageTypeStructured:
				if parsed, err := ParseStructuredLog(logs[i].Message); err == nil && parsed.Operation != "" {
					g.operation = parsed.Operation
				}
			}
		}

		if earliest == "" || logs[i].Timestamp < earliest {
			earliest = logs[i].Timestamp
		}
	}

	// 构建 spans
	spans := make([]TraceSpan, 0, len(svcOrder))
	var totalDuration int
	var entryService string

	for idx, svc := range svcOrder {
		g := groups[svc]
		if idx == 0 {
			entryService = svc
		}

		// 计算 start_ms：该服务最早日志相对于 trace 起始时间的偏移
		startMs := timestampDiffMs(earliest, g.earliest)

		// 估算 duration：优先用网关 response_time，否则用时间跨度
		durationMs := 0
		for _, log := range g.logs {
			if ParseMessage(log.Message) == MessageTypeGateway {
				if gm, err := ParseGatewayMessage(log.Message); err == nil && gm.ResponseTime > 0 {
					durationMs = gm.ResponseTime
					break
				}
			}
		}
		if durationMs == 0 && g.earliest != g.latest {
			// 降级：用最早到最晚日志的时间差
			durationMs = timestampDiffMs(g.earliest, g.latest)
			if durationMs == 0 {
				durationMs = 1 // 保证最小可见宽度
			}
		}

		status := "ok"
		if g.hasError {
			status = "error"
		} else if durationMs > 1000 {
			status = "slow"
		}

		if g.operation == "" {
			g.operation = svc
		}

		spanLogs := make([]TraceSpanLog, 0, len(g.logs))
		for _, log := range g.logs {
			spanLogs = append(spanLogs, TraceSpanLog{
				Timestamp: log.Timestamp,
				Level:     ExtractLogLevel(&log),
				Message:   truncateMessage(log.Message, 200),
			})
		}

		spans = append(spans, TraceSpan{
			Service:    svc,
			Operation:  g.operation,
			StartMs:    startMs,
			DurationMs: durationMs,
			Status:     status,
			Logs:       spanLogs,
		})

		endMs := startMs + durationMs
		if endMs > totalDuration {
			totalDuration = endMs
		}
	}

	return &TraceDetail{
		RequestUUID:     requestUUID,
		EntryService:    entryService,
		TotalDurationMs: totalDuration,
		Timestamp:       earliest,
		Spans:           spans,
	}
}

// truncateMessage 截断过长的消息
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}

// timestampDiffMs 计算两个 ISO 时间戳之间的毫秒差
func timestampDiffMs(from, to string) int {
	t1, err1 := parseTimestamp(from)
	t2, err2 := parseTimestamp(to)
	if err1 != nil || err2 != nil {
		return 0
	}
	diff := t2.Sub(t1).Milliseconds()
	if diff < 0 {
		diff = -diff
	}
	return int(diff)
}

// parseTimestamp 解析 ISO 8601 时间戳
func parseTimestamp(ts string) (time.Time, error) {
	// 尝试多种常见格式
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间戳: %s", ts)
}
