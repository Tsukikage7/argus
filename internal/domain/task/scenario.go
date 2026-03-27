package task

import (
	"context"
	"time"
)

// ScenarioStatus 沉淀场景状态
type ScenarioStatus string

const (
	ScenarioStatusDraft     ScenarioStatus = "draft"
	ScenarioStatusPublished ScenarioStatus = "published"
)

// CapturedScenario 从诊断结果沉淀的故障场景
type CapturedScenario struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	SourceTaskID       string         `json:"source_task_id,omitempty"`
	RootCause          string         `json:"root_cause,omitempty"`
	Confidence         float64        `json:"confidence,omitempty"`
	LogPatterns        []string       `json:"log_patterns"`
	AffectedNamespaces []string       `json:"affected_namespaces"`
	Status             ScenarioStatus `json:"status"`
	CreatedAt          time.Time      `json:"created_at"`
}

// ScenarioRepository 沉淀场景存储接口
type ScenarioRepository interface {
	// Save 创建或更新场景
	Save(ctx context.Context, s *CapturedScenario) error
	// Get 按 ID 查询场景
	Get(ctx context.Context, id string) (*CapturedScenario, error)
	// List 按状态过滤查询场景列表
	List(ctx context.Context, status ScenarioStatus) ([]*CapturedScenario, error)
	// UpdateStatus 更新场景状态（draft -> published）
	UpdateStatus(ctx context.Context, id string, status ScenarioStatus) error
}
