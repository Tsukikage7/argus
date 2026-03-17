package agent

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

var jsonBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*({.*?})\\s*```")

// parseDiagnosis 从 LLM 文本响应中提取 JSON 格式的诊断结论
func parseDiagnosis(content string) (*task.Diagnosis, error) {
	// 尝试从 markdown code block 提取
	matches := jsonBlockRe.FindStringSubmatch(content)
	var jsonStr string
	if len(matches) >= 2 {
		jsonStr = matches[1]
	} else {
		// 尝试直接找 JSON
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			jsonStr = content[start : end+1]
		}
	}

	if jsonStr == "" {
		return nil, nil
	}

	var d task.Diagnosis
	if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
		return nil, err
	}

	// 校验必要字段
	if d.RootCause == "" {
		return nil, nil
	}

	return &d, nil
}

// parseParams 将 JSON string 解析为 map
func parseParams(s string) map[string]any {
	m := make(map[string]any)
	_ = json.Unmarshal([]byte(s), &m)
	return m
}
