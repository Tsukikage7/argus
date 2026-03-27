// Package command 定义应用层命令及处理器
package command

import (
	"context"

	"github.com/Tsukikage7/argus/internal/domain/chat"
	"github.com/Tsukikage7/argus/internal/domain/task"
)

// TaskRepository 任务状态存储接口
type TaskRepository interface {
	Save(ctx context.Context, t *task.Task) error
	Get(ctx context.Context, tenantID, id string) (*task.Task, error)
}

// HistoryRepository 诊断历史存储接口
type HistoryRepository interface {
	Save(ctx context.Context, t *task.Task) error
	ListRecent(ctx context.Context, tenantID string, limit int) ([]*task.Task, error)
	GetByID(ctx context.Context, id string) (*task.Task, error)
}

// EventPublisher SSE 事件发布接口
type EventPublisher interface {
	Publish(taskID string, event task.TaskEvent)
}

// ── 聊天系统 Repository 接口 ──────────────────────────────────────────

// ChatSessionRepository 聊天会话存储接口
type ChatSessionRepository interface {
	Create(ctx context.Context, s *chat.ChatSession) error
	Get(ctx context.Context, tenantID, id string) (*chat.ChatSession, error)
	List(ctx context.Context, tenantID string, limit, offset int) ([]*chat.ChatSession, error)
	Update(ctx context.Context, s *chat.ChatSession) error
	SoftDelete(ctx context.Context, tenantID, id string) error
	CleanExpired(ctx context.Context, ttlDays int) (int64, error)
}

// ChatMessageRepository 聊天消息存储接口
type ChatMessageRepository interface {
	CreateMessage(ctx context.Context, m *chat.ChatMessage) error
	ListBySession(ctx context.Context, sessionID string, cursor string, limit int) ([]*chat.ChatMessage, error)
	GetMessageByID(ctx context.Context, id string) (*chat.ChatMessage, error)
}

// ChatRunRepository 聊天执行存储接口
type ChatRunRepository interface {
	CreateRun(ctx context.Context, r *chat.ChatRun) error
	GetRun(ctx context.Context, id string) (*chat.ChatRun, error)
	UpdateRun(ctx context.Context, r *chat.ChatRun) error
}

// ChatArtifactRepository 聊天产物存储接口
type ChatArtifactRepository interface {
	CreateArtifact(ctx context.Context, a *chat.ChatArtifact) error
	ListByRun(ctx context.Context, runID string) ([]*chat.ChatArtifact, error)
}
