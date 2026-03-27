package command

import (
	"context"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/chat"
	"github.com/google/uuid"
)

// CreateChatSessionCommand 创建会话命令
type CreateChatSessionCommand struct {
	TenantID string
	Title    string
	Source   string
}

// CreateChatSessionHandler 创建会话处理器
type CreateChatSessionHandler struct {
	repo ChatSessionRepository
}

// NewCreateChatSessionHandler 创建处理器
func NewCreateChatSessionHandler(repo ChatSessionRepository) *CreateChatSessionHandler {
	return &CreateChatSessionHandler{repo: repo}
}

// Handle 执行创建会话
func (h *CreateChatSessionHandler) Handle(ctx context.Context, cmd CreateChatSessionCommand) (*chat.ChatSession, error) {
	now := time.Now()
	source := chat.SessionSource(cmd.Source)
	if source == "" {
		source = chat.SourceWeb
	}

	session := &chat.ChatSession{
		ID:        uuid.New().String(),
		TenantID:  cmd.TenantID,
		Title:     cmd.Title,
		Source:    source,
		Status:    chat.SessionActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if session.Title == "" {
		session.Title = "新会话"
	}

	if err := h.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// UpdateChatSessionCommand 更新会话命令
type UpdateChatSessionCommand struct {
	TenantID string
	ID       string
	Title    *string
	Archived *bool
}

// UpdateChatSessionHandler 更新会话处理器
type UpdateChatSessionHandler struct {
	repo ChatSessionRepository
}

// NewUpdateChatSessionHandler 创建处理器
func NewUpdateChatSessionHandler(repo ChatSessionRepository) *UpdateChatSessionHandler {
	return &UpdateChatSessionHandler{repo: repo}
}

// Handle 执行更新会话
func (h *UpdateChatSessionHandler) Handle(ctx context.Context, cmd UpdateChatSessionCommand) error {
	session, err := h.repo.Get(ctx, cmd.TenantID, cmd.ID)
	if err != nil {
		return err
	}

	if cmd.Title != nil {
		session.Title = *cmd.Title
	}
	if cmd.Archived != nil && *cmd.Archived {
		session.Status = chat.SessionArchived
		now := time.Now()
		session.ArchivedAt = &now
	}
	session.UpdatedAt = time.Now()

	return h.repo.Update(ctx, session)
}

// DeleteChatSessionHandler 删除会话处理器
type DeleteChatSessionHandler struct {
	repo ChatSessionRepository
}

// NewDeleteChatSessionHandler 创建处理器
func NewDeleteChatSessionHandler(repo ChatSessionRepository) *DeleteChatSessionHandler {
	return &DeleteChatSessionHandler{repo: repo}
}

// Handle 执行软删除会话
func (h *DeleteChatSessionHandler) Handle(ctx context.Context, tenantID, id string) error {
	return h.repo.SoftDelete(ctx, tenantID, id)
}
