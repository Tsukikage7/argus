// Package chat 定义聊天会话的领域模型
package chat

import "time"

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionActive   SessionStatus = "active"
	SessionArchived SessionStatus = "archived"
	SessionDeleted  SessionStatus = "deleted"
)

// SessionSource 会话来源
type SessionSource string

const (
	SourceWeb     SessionSource = "web"
	SourceReplay  SessionSource = "replay"
	SourceWechat  SessionSource = "wechat"
	SourceAPI     SessionSource = "api"
)

// ChatSession 表示一次聊天会话
type ChatSession struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	Title         string            `json:"title"`
	Source        SessionSource     `json:"source"`
	Status        SessionStatus     `json:"status"`
	LastIntent    string            `json:"last_intent,omitempty"`
	Summary       string            `json:"summary,omitempty"`
	WorkingMemory map[string]any    `json:"working_memory,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ArchivedAt    *time.Time        `json:"archived_at,omitempty"`
}
