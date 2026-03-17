package command

import (
	"context"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/google/uuid"
)

// DiagnoseCommand 诊断命令
type DiagnoseCommand struct {
	Input  string // 用户输入或告警内容
	Source string // 来源: cli / web / webhook
}

// DiagnoseHandler 处理诊断命令
type DiagnoseHandler struct {
	agent   *agent.Agent
	taskRepo TaskRepository
	history  HistoryRepository
	events   EventPublisher
}

// NewDiagnoseHandler 创建诊断命令处理器
func NewDiagnoseHandler(
	ag *agent.Agent,
	taskRepo TaskRepository,
	history HistoryRepository,
	events EventPublisher,
) *DiagnoseHandler {
	return &DiagnoseHandler{
		agent:    ag,
		taskRepo: taskRepo,
		history:  history,
		events:   events,
	}
}

// Handle 执行诊断
func (h *DiagnoseHandler) Handle(ctx context.Context, cmd DiagnoseCommand) (*task.Task, error) {
	t := &task.Task{
		ID:        uuid.New().String(),
		Input:     cmd.Input,
		Source:     cmd.Source,
		Status:    task.StatusPending,
		Steps:     []task.Step{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存初始任务
	if err := h.taskRepo.Save(ctx, t); err != nil {
		return nil, err
	}

	// 异步执行 Agent（使用独立 context，不受 HTTP 请求生命周期影响）
	go func() {
		bgCtx := context.Background()

		// 设置 SSE 事件回调
		h.agent.OnEvent(func(event task.TaskEvent) {
			if h.events != nil {
				h.events.Publish(t.ID, event)
			}
			// 每次事件都更新任务状态到 Redis
			_ = h.taskRepo.Save(bgCtx, t)
		})

		if err := h.agent.Run(bgCtx, t); err != nil {
			t.Status = task.StatusFailed
			t.UpdatedAt = time.Now()
		}
		_ = h.taskRepo.Save(bgCtx, t)
		if h.history != nil {
			_ = h.history.Save(bgCtx, t)
		}
	}()

	return t, nil
}
