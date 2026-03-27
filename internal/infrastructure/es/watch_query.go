package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SearchRecentLogs 增量查询指定 namespace 在 since 之后的日志
// 按 @timestamp 降序排列，最多返回 limit 条
func (c *Client) SearchRecentLogs(ctx context.Context, namespaces []string, since time.Time, limit int) ([]UCloudLog, error) {
	if limit <= 0 {
		limit = 200
	}

	// 构建时间范围过滤
	must := []map[string]any{
		{
			"range": map[string]any{
				"@timestamp": map[string]any{
					"gt": since.Format(time.RFC3339Nano),
				},
			},
		},
	}

	// 如果指定了 namespace 列表，添加 terms 过滤
	if len(namespaces) > 0 {
		must = append(must, map[string]any{
			"terms": map[string]any{
				"kubernetes_namespace.keyword": namespaces,
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
		"size": limit,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("es: marshal watch query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(c.allIndex()),
		c.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("es: watch search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("es: watch search error: status %s", res.Status())
	}

	return parseUCloudLogResponse(res.Body)
}
