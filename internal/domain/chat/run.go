package chat

import (
	"encoding/json"
	"time"
)

// RunStatus 执行状态
type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
)

// ChatRun 表示一次 Agent 执行（由用户消息触发）
type ChatRun struct {
	ID               string          `json:"id"`
	SessionID        string          `json:"session_id"`
	TenantID         string          `json:"tenant_id"`
	TriggerMessageID string          `json:"trigger_message_id"`
	Intent           string          `json:"intent,omitempty"`
	Status           RunStatus       `json:"status"`
	Steps            json.RawMessage `json:"steps,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
}
