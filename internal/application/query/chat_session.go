package query

import (
	"context"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/chat"
)

// ListSessionsHandler 列出会话查询处理器
type ListSessionsHandler struct {
	repo command.ChatSessionRepository
}

// NewListSessionsHandler 创建处理器
func NewListSessionsHandler(repo command.ChatSessionRepository) *ListSessionsHandler {
	return &ListSessionsHandler{repo: repo}
}

// Handle 执行查询
func (h *ListSessionsHandler) Handle(ctx context.Context, tenantID string, limit, offset int) ([]*chat.ChatSession, error) {
	return h.repo.List(ctx, tenantID, limit, offset)
}

// GetSessionHandler 获取会话详情查询处理器
type GetSessionHandler struct {
	repo command.ChatSessionRepository
}

// NewGetSessionHandler 创建处理器
func NewGetSessionHandler(repo command.ChatSessionRepository) *GetSessionHandler {
	return &GetSessionHandler{repo: repo}
}

// Handle 执行查询
func (h *GetSessionHandler) Handle(ctx context.Context, tenantID, id string) (*chat.ChatSession, error) {
	return h.repo.Get(ctx, tenantID, id)
}
