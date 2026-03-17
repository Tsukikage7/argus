package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
	"github.com/google/uuid"
)

// ReplayRepository 回放会话存储接口
type ReplayRepository interface {
	Save(ctx context.Context, s *task.ReplaySession) error
	Get(ctx context.Context, id string) (*task.ReplaySession, error)
}

// ReplayCommand 回放命令
type ReplayCommand struct {
	Type         task.ReplayType
	ScenarioName string
	Config       task.ReplayConfig
}

// ReplayHandler 处理回放命令
type ReplayHandler struct {
	engine     *mock.ReplayEngine
	diagnoseH  *DiagnoseHandler
	replayRepo ReplayRepository
	events     EventPublisher
	llm        agent.LLMClient
	model      string
}

// NewReplayHandler 创建回放命令处理器
func NewReplayHandler(
	engine *mock.ReplayEngine,
	diagnoseH *DiagnoseHandler,
	replayRepo ReplayRepository,
	events EventPublisher,
	llm agent.LLMClient,
	model string,
) *ReplayHandler {
	return &ReplayHandler{
		engine:     engine,
		diagnoseH:  diagnoseH,
		replayRepo: replayRepo,
		events:     events,
		llm:        llm,
		model:      model,
	}
}

// Handle 执行回放，异步运行，立即返回 session
func (h *ReplayHandler) Handle(ctx context.Context, cmd ReplayCommand) (*task.ReplaySession, error) {
	// 验证场景存在
	scenario, ok := h.engine.GetScenario(cmd.ScenarioName)
	if !ok {
		return nil, fmt.Errorf("scenario %q not found", cmd.ScenarioName)
	}

	session := &task.ReplaySession{
		ID:           uuid.New().String(),
		Type:         cmd.Type,
		ScenarioName: cmd.ScenarioName,
		Config:       cmd.Config,
		Status:       task.ReplayStatusPending,
		CreatedAt:    time.Now(),
	}

	if err := h.replayRepo.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("save replay session: %w", err)
	}

	// 异步执行回放流程
	go h.runAsync(session, scenario)

	return session, nil
}

// HandleSync 同步执行回放（供 CLI 使用）
func (h *ReplayHandler) HandleSync(ctx context.Context, cmd ReplayCommand, onProgress func(string)) (*task.ReplaySession, error) {
	scenario, ok := h.engine.GetScenario(cmd.ScenarioName)
	if !ok {
		return nil, fmt.Errorf("scenario %q not found", cmd.ScenarioName)
	}

	session := &task.ReplaySession{
		ID:           uuid.New().String(),
		Type:         cmd.Type,
		ScenarioName: cmd.ScenarioName,
		Config:       cmd.Config,
		Status:       task.ReplayStatusPending,
		CreatedAt:    time.Now(),
	}

	_ = h.replayRepo.Save(ctx, session)

	// 1. 生成数据
	session.Status = task.ReplayStatusGenerating
	_ = h.replayRepo.Save(ctx, session)
	if onProgress != nil {
		onProgress("generating")
	}

	var result *mock.ReplayResult
	var err error
	switch cmd.Type {
	case task.ReplayTypeFault:
		result, err = h.engine.RunFaultReplay(ctx, session)
	case task.ReplayTypeTraffic:
		result, err = h.engine.RunTrafficReplay(ctx, session)
	default:
		return nil, fmt.Errorf("unknown replay type: %s", cmd.Type)
	}
	if err != nil {
		session.Status = task.ReplayStatusFailed
		session.Error = err.Error()
		_ = h.replayRepo.Save(ctx, session)
		return session, err
	}
	session.LogsWritten = result.LogsWritten
	session.TracesWritten = result.TracesWritten

	if onProgress != nil {
		onProgress(fmt.Sprintf("data_written: %d logs, %d traces", result.LogsWritten, result.TracesWritten))
	}

	// 2. 自动诊断
	if cmd.Config.AutoDiagnose && h.diagnoseH != nil {
		session.Status = task.ReplayStatusDiagnosing
		_ = h.replayRepo.Save(ctx, session)
		if onProgress != nil {
			onProgress("diagnosing")
		}

		diagTask, diagErr := h.diagnoseH.Handle(ctx, DiagnoseCommand{
			Input:  fmt.Sprintf("[回放诊断] 场景: %s (%s)", scenario.Name, scenario.Description),
			Source: "replay",
		})
		if diagErr == nil && diagTask != nil {
			session.TaskID = diagTask.ID
		}
	}

	// 3. 计算影响面
	if onProgress != nil {
		onProgress("computing_impact")
	}

	impact, impactErr := h.engine.ComputeImpact(ctx, session.ID)
	if impactErr == nil && impact != nil {
		// 4. LLM 生成影响面总结
		summary := h.generateImpactSummary(ctx, impact, scenario.Name, scenario.Description)
		impact.Summary = summary
		session.ImpactReport = impact
	}

	session.Status = task.ReplayStatusCompleted
	now := time.Now()
	session.CompletedAt = &now
	_ = h.replayRepo.Save(ctx, session)

	if onProgress != nil {
		onProgress("completed")
	}

	return session, nil
}

func (h *ReplayHandler) runAsync(session *task.ReplaySession, scenario Scenario) {
	ctx := context.Background()

	// 1. 生成数据
	session.Status = task.ReplayStatusGenerating
	_ = h.replayRepo.Save(ctx, session)
	h.publishReplayEvent(session, "status", session.Status)

	var result *mock.ReplayResult
	var err error
	switch session.Type {
	case task.ReplayTypeFault:
		result, err = h.engine.RunFaultReplay(ctx, session)
	case task.ReplayTypeTraffic:
		result, err = h.engine.RunTrafficReplay(ctx, session)
	}
	if err != nil {
		session.Status = task.ReplayStatusFailed
		session.Error = err.Error()
		_ = h.replayRepo.Save(ctx, session)
		h.publishReplayEvent(session, "error", err.Error())
		return
	}
	session.LogsWritten = result.LogsWritten
	session.TracesWritten = result.TracesWritten
	h.publishReplayEvent(session, "progress", fmt.Sprintf("data written: %d logs, %d traces", result.LogsWritten, result.TracesWritten))

	// 2. 自动诊断
	if session.Config.AutoDiagnose && h.diagnoseH != nil {
		session.Status = task.ReplayStatusDiagnosing
		_ = h.replayRepo.Save(ctx, session)
		h.publishReplayEvent(session, "status", session.Status)

		diagTask, diagErr := h.diagnoseH.Handle(ctx, DiagnoseCommand{
			Input:  fmt.Sprintf("[回放诊断] 场景: %s (%s)", scenario.Name, scenario.Description),
			Source: "replay",
		})
		if diagErr == nil && diagTask != nil {
			session.TaskID = diagTask.ID
			_ = h.replayRepo.Save(ctx, session)
		}
	}

	// 3. 计算影响面
	impact, impactErr := h.engine.ComputeImpact(ctx, session.ID)
	if impactErr == nil && impact != nil {
		// 4. LLM 生成影响面总结
		summary := h.generateImpactSummary(ctx, impact, scenario.Name, scenario.Description)
		impact.Summary = summary
		session.ImpactReport = impact
	}

	session.Status = task.ReplayStatusCompleted
	now := time.Now()
	session.CompletedAt = &now
	_ = h.replayRepo.Save(ctx, session)
	h.publishReplayEvent(session, "status", session.Status)
	if session.ImpactReport != nil {
		h.publishReplayEvent(session, "impact", session.ImpactReport)
	}
}

func (h *ReplayHandler) publishReplayEvent(session *task.ReplaySession, eventType string, data any) {
	if h.events == nil {
		return
	}
	h.events.Publish("replay:"+session.ID, task.TaskEvent{
		TaskID: session.ID,
		Type:   eventType,
		Data:   data,
	})
}

func (h *ReplayHandler) generateImpactSummary(ctx context.Context, report *task.ImpactReport, scenarioName, scenarioDesc string) string {
	if h.llm == nil {
		return ""
	}

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	prompt := fmt.Sprintf(`你是一个运维分析师。以下是故障回放场景 "%s"（%s）的影响面数据。
请用简洁的中文总结：哪些服务受到影响、影响程度如何、故障传播路径是什么。
控制在 200 字以内。

影响面数据：
%s`, scenarioName, scenarioDesc, string(reportJSON))

	resp, err := h.llm.ChatWithTools(ctx, &agent.ChatRequest{
		Model: h.model,
		Messages: []agent.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return fmt.Sprintf("影响面总结生成失败: %v", err)
	}
	return resp.Content
}

// Scenario 在这里重新定义以避免循环导入时的类型别名问题
type Scenario = mock.Scenario
