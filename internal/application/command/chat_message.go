package command

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/agent"
	"github.com/Tsukikage7/argus/internal/domain/chat"
	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/google/uuid"
)

// SendChatMessageCommand 发送聊天消息命令
type SendChatMessageCommand struct {
	TenantID  string
	SessionID string
	Content   string
}

// SendChatMessageResult 发送消息结果
type SendChatMessageResult struct {
	SessionID string `json:"session_id"`
	MessageID string `json:"message_id"`
	RunID     string `json:"run_id"`
}

// SendChatMessageHandler 发送聊天消息处理器
type SendChatMessageHandler struct {
	agent      *agent.Agent
	sessionRepo ChatSessionRepository
	messageRepo ChatMessageRepository
	runRepo     ChatRunRepository
	events      EventPublisher
}

// NewSendChatMessageHandler 创建处理器
func NewSendChatMessageHandler(
	ag *agent.Agent,
	sessionRepo ChatSessionRepository,
	messageRepo ChatMessageRepository,
	runRepo ChatRunRepository,
	events EventPublisher,
) *SendChatMessageHandler {
	return &SendChatMessageHandler{
		agent:       ag,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		runRepo:     runRepo,
		events:      events,
	}
}

// Handle 执行发送消息
func (h *SendChatMessageHandler) Handle(ctx context.Context, cmd SendChatMessageCommand) (*SendChatMessageResult, error) {
	now := time.Now()

	// 创建用户消息
	userMsg := &chat.ChatMessage{
		ID:        uuid.New().String(),
		SessionID: cmd.SessionID,
		TenantID:  cmd.TenantID,
		Role:      chat.RoleUser,
		Content:   cmd.Content,
		Status:    chat.MessageCompleted,
		CreatedAt: now,
	}
	if err := h.messageRepo.CreateMessage(ctx, userMsg); err != nil {
		return nil, err
	}

	// 创建 Run
	run := &chat.ChatRun{
		ID:               uuid.New().String(),
		SessionID:        cmd.SessionID,
		TenantID:         cmd.TenantID,
		TriggerMessageID: userMsg.ID,
		Status:           chat.RunPending,
		StartedAt:        &now,
	}
	if err := h.runRepo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// 异步执行 Agent
	go h.executeAgent(cmd.TenantID, cmd.SessionID, run.ID, userMsg.ID)

	return &SendChatMessageResult{
		SessionID: cmd.SessionID,
		MessageID: userMsg.ID,
		RunID:     run.ID,
	}, nil
}

// executeAgent 异步执行 Agent 推理
func (h *SendChatMessageHandler) executeAgent(tenantID, sessionID, runID, triggerMsgID string) {
	ctx := context.Background()

	// 更新 run 状态为 running
	run, err := h.runRepo.GetRun(ctx, runID)
	if err != nil {
		return
	}
	run.Status = chat.RunRunning
	_ = h.runRepo.UpdateRun(ctx, run)

	// 加载历史消息构建对话上下文
	messages, err := h.messageRepo.ListBySession(ctx, sessionID, "", 20)
	if err != nil {
		h.failRun(ctx, run, err.Error())
		return
	}

	// 构建 Agent 消息
	var agentMessages []agent.Message
	for _, m := range messages {
		role := string(m.Role)
		if role == "tool" {
			continue // 跳过 tool 消息，Agent 会自己管理
		}
		agentMessages = append(agentMessages, agent.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	// 事件回调：发布 SSE 事件
	eventHandler := func(event agent.ChatEvent) {
		event.RunID = runID
		event.SessionID = sessionID
		if h.events != nil {
			h.events.Publish(runID, task.TaskEvent{
				TaskID:   runID,
				TenantID: tenantID,
				Type:     event.Type,
				Data:     event.Data,
			})
		}
	}

	// 执行 Agent Chat
	result, err := h.agent.Chat(ctx, tenantID, agentMessages, "", eventHandler)
	if err != nil {
		h.failRun(ctx, run, err.Error())
		return
	}

	// 创建 assistant 消息
	assistantMsg := &chat.ChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		TenantID:  tenantID,
		Role:      chat.RoleAssistant,
		Content:   result.Content,
		Status:    chat.MessageCompleted,
		RunID:     runID,
		CreatedAt: time.Now(),
	}
	_ = h.messageRepo.CreateMessage(ctx, assistantMsg)

	// 更新 run 状态
	now := time.Now()
	stepsJSON, _ := json.Marshal(result.Steps)
	run.Status = chat.RunCompleted
	run.CompletedAt = &now
	run.Steps = stepsJSON
	_ = h.runRepo.UpdateRun(ctx, run)

	// 更新会话标题（首条消息时自动设置）
	session, err := h.sessionRepo.Get(ctx, tenantID, sessionID)
	if err == nil && session.Title == "新会话" {
		title := result.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		session.Title = title
		session.UpdatedAt = time.Now()
		_ = h.sessionRepo.Update(ctx, session)
	}
}

// failRun 标记 run 失败
func (h *SendChatMessageHandler) failRun(ctx context.Context, run *chat.ChatRun, errMsg string) {
	now := time.Now()
	run.Status = chat.RunFailed
	run.CompletedAt = &now
	run.ErrorMessage = errMsg
	_ = h.runRepo.UpdateRun(ctx, run)

	if h.events != nil {
		h.events.Publish(run.ID, task.TaskEvent{
			TaskID:   run.ID,
			TenantID: run.TenantID,
			Type:     "run.failed",
			Data:     errMsg,
		})
	}
}
