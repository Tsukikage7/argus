package command

import (
	"context"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// RecoverCommand 恢复命令
type RecoverCommand struct {
	TaskID string
}

// RecoverHandler 处理恢复命令
type RecoverHandler struct {
	taskRepo TaskRepository
	tools    *tool.Registry
	verifier *agent.Verifier
	events   EventPublisher
}

// NewRecoverHandler 创建恢复命令处理器
func NewRecoverHandler(
	taskRepo TaskRepository,
	tools *tool.Registry,
	verifier *agent.Verifier,
	events EventPublisher,
) *RecoverHandler {
	return &RecoverHandler{
		taskRepo: taskRepo,
		tools:    tools,
		verifier: verifier,
		events:   events,
	}
}

// Handle 执行恢复
func (h *RecoverHandler) Handle(ctx context.Context, cmd RecoverCommand) error {
	t, err := h.taskRepo.Get(ctx, cmd.TaskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if t.Diagnosis == nil {
		return fmt.Errorf("no diagnosis available for task %s", cmd.TaskID)
	}

	t.Status = task.StatusRecovering
	t.Recovery = &task.Recovery{
		Status: task.RecoveryExecuting,
	}
	t.UpdatedAt = time.Now()
	_ = h.taskRepo.Save(ctx, t)

	if h.events != nil {
		h.events.Publish(t.ID, task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})
	}

	// 执行恢复建议中的命令
	execTool, execErr := h.tools.Get("exec_command")
	if execErr != nil {
		return execErr
	}

	for _, suggestion := range t.Diagnosis.Suggestions {
		action := task.RecoveryAction{
			Description: suggestion,
		}

		// 对包含 "重启" 的建议自动生成命令
		if len(t.Diagnosis.AffectedServices) > 0 {
			svc := t.Diagnosis.AffectedServices[0]
			result, err := execTool.Execute(ctx, map[string]any{
				"host":    svc,
				"command": fmt.Sprintf("systemctl restart %s", svc),
			})
			if err != nil {
				action.Result = fmt.Sprintf("error: %v", err)
				action.Success = false
			} else {
				action.Result = result.Output
				action.Success = result.Error == ""
			}
		}

		t.Recovery.Actions = append(t.Recovery.Actions, action)
	}

	// 验证恢复
	if verifyErr := h.verifier.Verify(ctx, t); verifyErr != nil {
		t.Recovery.Status = task.RecoveryFailed
		t.Status = task.StatusFailed
	} else {
		t.Recovery.Status = task.RecoverySuccess
		t.Status = task.StatusRecovered
	}

	t.UpdatedAt = time.Now()
	_ = h.taskRepo.Save(ctx, t)

	if h.events != nil {
		h.events.Publish(t.ID, task.TaskEvent{TaskID: t.ID, Type: "recovery", Data: t.Recovery})
		h.events.Publish(t.ID, task.TaskEvent{TaskID: t.ID, Type: "status", Data: t.Status})
	}

	return nil
}
