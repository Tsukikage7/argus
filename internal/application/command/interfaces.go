// Package command 定义应用层命令及处理器
package command

import (
	"context"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// TaskRepository 任务状态存储接口
type TaskRepository interface {
	Save(ctx context.Context, t *task.Task) error
	Get(ctx context.Context, id string) (*task.Task, error)
}

// HistoryRepository 诊断历史存储接口
type HistoryRepository interface {
	Save(ctx context.Context, t *task.Task) error
	ListRecent(ctx context.Context, limit int) ([]*task.Task, error)
}

// EventPublisher SSE 事件发布接口
type EventPublisher interface {
	Publish(taskID string, event task.TaskEvent)
}
