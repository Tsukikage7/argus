package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/google/uuid"
)

// DiagnoseCommand 诊断命令
type DiagnoseCommand struct {
	TenantID string // 租户 ID
	Input    string // 用户输入或告警内容
	Source   string // 来源: cli / web / webhook
	Context  *DiagnoseContext // 可选的高级搜索上下文
}

// DiagnoseContext 诊断高级选项
type DiagnoseContext struct {
	TimeRange  string   `json:"time_range,omitempty"`  // 时间范围，如 "last 1h"
	Namespaces []string `json:"namespaces,omitempty"`  // 限定 namespace 列表
}

// DiagnoseHandler 处理诊断命令
type DiagnoseHandler struct {
	agent        *agent.Agent
	taskRepo     TaskRepository
	history      HistoryRepository
	events       EventPublisher
	scenarioRepo task.ScenarioRepository
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

// SetScenarioRepo 设置场景仓储（可选依赖）
func (h *DiagnoseHandler) SetScenarioRepo(repo task.ScenarioRepository) {
	h.scenarioRepo = repo
}

// Handle 执行诊断
func (h *DiagnoseHandler) Handle(ctx context.Context, cmd DiagnoseCommand) (*task.Task, error) {
	// 将高级选项注入到用户输入中
	input := cmd.Input
	if cmd.Context != nil {
		var hints []string
		if cmd.Context.TimeRange != "" {
			hints = append(hints, fmt.Sprintf("搜索时间范围: %s", cmd.Context.TimeRange))
		}
		if len(cmd.Context.Namespaces) > 0 {
			hints = append(hints, fmt.Sprintf("优先搜索 namespace: %s", strings.Join(cmd.Context.Namespaces, ", ")))
		}
		if len(hints) > 0 {
			input = input + "\n\n[搜索上下文]\n" + strings.Join(hints, "\n")
		}
	}

	t := &task.Task{
		ID:        uuid.New().String(),
		TenantID:  cmd.TenantID,
		Input:     input,
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

		// per-task 事件回调：发布 SSE 事件并同步更新 Redis 状态
		eventHandler := func(event task.TaskEvent) {
			if h.events != nil {
				h.events.Publish(t.ID, event)
			}
			// 每次事件都更新任务状态到 Redis
			_ = h.taskRepo.Save(bgCtx, t)
		}

		if err := h.agent.Run(bgCtx, t, eventHandler); err != nil {
			t.Status = task.StatusFailed
			t.UpdatedAt = time.Now()
		}
		_ = h.taskRepo.Save(bgCtx, t)
		if h.history != nil {
			_ = h.history.Save(bgCtx, t)
		}

		// 诊断成功且置信度 >= 0.7 时，自动沉淀为 draft 场景
		h.tryCaptureScenario(bgCtx, t)
	}()

	// 返回浅拷贝快照，避免与后台 goroutine 的数据竞争
	snapshot := *t
	return &snapshot, nil
}

// tryCaptureScenario 尝试从诊断结果自动沉淀场景
func (h *DiagnoseHandler) tryCaptureScenario(ctx context.Context, t *task.Task) {
	if h.scenarioRepo == nil || t.Diagnosis == nil || t.Diagnosis.Confidence < 0.7 {
		return
	}

	// 从诊断步骤中提取错误日志模式
	patterns := extractLogPatterns(t.Steps)

	scenario := &task.CapturedScenario{
		ID:                 uuid.New().String(),
		Name:               fmt.Sprintf("auto-%s", t.ID[:8]),
		Description:        t.Diagnosis.RootCause,
		SourceTaskID:       t.ID,
		RootCause:          t.Diagnosis.RootCause,
		Confidence:         t.Diagnosis.Confidence,
		LogPatterns:        patterns,
		AffectedNamespaces: t.Diagnosis.AffectedServices,
		Status:             task.ScenarioStatusDraft,
		CreatedAt:          time.Now(),
	}

	_ = h.scenarioRepo.Save(ctx, scenario)
}

// errorPatternRe 匹配常见错误日志模式
var errorPatternRe = regexp.MustCompile(`(?i)(error|exception|fail|timeout|refused|unavailable|panic)[:\s].*`)

// extractLogPatterns 从 Agent 推理步骤中提取关键错误 message 模式
func extractLogPatterns(steps []task.Step) []string {
	seen := make(map[string]bool)
	var patterns []string

	for _, step := range steps {
		if step.Observe == "" {
			continue
		}
		matches := errorPatternRe.FindAllString(step.Observe, -1)
		for _, m := range matches {
			// 正则化：去除时间戳、UUID 等动态部分
			normalized := normalizePattern(m)
			if normalized == "" || seen[normalized] {
				continue
			}
			seen[normalized] = true
			patterns = append(patterns, normalized)
			if len(patterns) >= 10 {
				return patterns
			}
		}
	}
	return patterns
}

// normalizePattern 将具体错误消息正则化为可复用的模式
func normalizePattern(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) < 10 {
		return ""
	}
	// 替换 UUID
	msg = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).ReplaceAllString(msg, "<UUID>")
	// 替换 IP 地址
	msg = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?`).ReplaceAllString(msg, "<IP>")
	// 替换纯数字序列（>4位）
	msg = regexp.MustCompile(`\b\d{5,}\b`).ReplaceAllString(msg, "<NUM>")
	// 截断过长的模式
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}
