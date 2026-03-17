// Package query 定义应用层查询
package query

import (
	"context"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
)

// TaskStatusQuery 任务状态查询
type TaskStatusQuery struct {
	TaskID string
}

// TaskStatusHandler 处理任务状态查询
type TaskStatusHandler struct {
	taskRepo command.TaskRepository
}

// NewTaskStatusHandler 创建任务状态查询处理器
func NewTaskStatusHandler(taskRepo command.TaskRepository) *TaskStatusHandler {
	return &TaskStatusHandler{taskRepo: taskRepo}
}

// Handle 查询任务状态
func (h *TaskStatusHandler) Handle(ctx context.Context, q TaskStatusQuery) (*task.Task, error) {
	return h.taskRepo.Get(ctx, q.TaskID)
}
