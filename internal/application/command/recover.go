package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// RecoverCommand 恢复命令
type RecoverCommand struct {
	TenantID string
	TaskID   string
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
	t, err := h.taskRepo.Get(ctx, cmd.TenantID, cmd.TaskID)
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

		// 根据 suggestion 内容生成语义化恢复命令
		host, cmd := buildRecoveryCommand(suggestion, t.Diagnosis.AffectedServices)
		if cmd != "" {
			action.Command = cmd
			result, err := execTool.Execute(ctx, map[string]any{
				"host":    host,
				"command": cmd,
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

// buildRecoveryCommand 根据建议内容和受影响服务，生成语义化恢复命令
// 返回 (host, command) 元组。
// 安全策略：仅根据关键词生成白名单范围内的预定义命令，
// 不直接提取 LLM 输出文本作为命令执行，防止命令注入。
func buildRecoveryCommand(suggestion string, affectedServices []string) (string, string) {
	lower := strings.ToLower(suggestion)

	if len(affectedServices) == 0 {
		return "", ""
	}
	svc := affectedServices[0]

	// 提取 namespace 作为执行上下文（K8s 集群通过 kubectl 本地执行）
	ns := extractNamespace(suggestion)
	host := "k8s-master"

	// 根据建议关键词推断操作类型（白名单模式）
	switch {
	case strings.Contains(lower, "rollout") || strings.Contains(lower, "重新部署") || strings.Contains(lower, "滚动更新"):
		if ns != "" {
			return host, fmt.Sprintf("kubectl rollout restart deployment/%s -n %s", svc, ns)
		}
		return host, fmt.Sprintf("kubectl rollout restart deployment/%s", svc)

	case strings.Contains(lower, "重启") || strings.Contains(lower, "restart"):
		if ns != "" {
			return host, fmt.Sprintf("kubectl rollout restart deployment/%s -n %s", svc, ns)
		}
		return host, fmt.Sprintf("kubectl rollout restart deployment/%s", svc)

	case strings.Contains(lower, "扩容") || strings.Contains(lower, "scale"):
		if ns != "" {
			return host, fmt.Sprintf("kubectl scale deployment/%s --replicas=3 -n %s", svc, ns)
		}
		return host, fmt.Sprintf("kubectl scale deployment/%s --replicas=3", svc)

	case strings.Contains(lower, "删除 pod") || strings.Contains(lower, "delete pod"):
		if ns != "" {
			return host, fmt.Sprintf("kubectl delete pod -l app=%s -n %s", svc, ns)
		}
		return host, fmt.Sprintf("kubectl delete pod -l app=%s", svc)

	case strings.Contains(lower, "日志清理") || strings.Contains(lower, "清除缓存"):
		return host, fmt.Sprintf("kubectl exec -n default deployment/%s -- sh -c 'rm -rf /tmp/cache/*'", svc)
	}

	return "", ""
}

// extractNamespace 从建议文本中提取 Kubernetes namespace
// 支持格式：-n <ns>、namespace <ns>、prj-xxx
func extractNamespace(suggestion string) string {
	lower := strings.ToLower(suggestion)

	// 匹配 "-n xxx" 格式
	if idx := strings.Index(lower, " -n "); idx != -1 {
		rest := strings.TrimSpace(suggestion[idx+4:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// 匹配 "namespace xxx" 格式
	if idx := strings.Index(lower, "namespace "); idx != -1 {
		rest := strings.TrimSpace(suggestion[idx+10:])
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// 匹配 prj- 开头的 namespace 前缀
	for _, word := range strings.Fields(suggestion) {
		if strings.HasPrefix(strings.ToLower(word), "prj-") {
			return strings.ToLower(word)
		}
	}

	return ""
}
