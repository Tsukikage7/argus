package chat

import "time"

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// MessageStatus 消息状态
type MessageStatus string

const (
	MessagePending   MessageStatus = "pending"
	MessageStreaming  MessageStatus = "streaming"
	MessageCompleted MessageStatus = "completed"
	MessageFailed    MessageStatus = "failed"
)

// ChatMessage 表示一条聊天消息
type ChatMessage struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	TenantID  string        `json:"tenant_id"`
	Role      MessageRole   `json:"role"`
	Content   string        `json:"content"`
	Status    MessageStatus `json:"status"`
	RunID     string        `json:"run_id,omitempty"`
	Artifacts []ChatArtifact `json:"artifacts,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}
