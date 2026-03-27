package chat

import (
	"encoding/json"
	"time"
)

// ArtifactType 产物类型
type ArtifactType string

const (
	ArtifactDiagnosis  ArtifactType = "diagnosis"
	ArtifactLogTable   ArtifactType = "log_table"
	ArtifactTraceGraph ArtifactType = "trace_graph"
)

// ChatArtifact 表示一个结构化产物（诊断结论、日志表、链路图等）
type ChatArtifact struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	RunID     string          `json:"run_id,omitempty"`
	MessageID string          `json:"message_id,omitempty"`
	Type      ArtifactType    `json:"type"`
	Title     string          `json:"title"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}
