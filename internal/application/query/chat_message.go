package query

import (
	"context"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/chat"
)

// ListMessagesHandler 列出消息查询处理器
type ListMessagesHandler struct {
	repo command.ChatMessageRepository
}

// NewListMessagesHandler 创建处理器
func NewListMessagesHandler(repo command.ChatMessageRepository) *ListMessagesHandler {
	return &ListMessagesHandler{repo: repo}
}

// Handle 执行查询
func (h *ListMessagesHandler) Handle(ctx context.Context, sessionID, cursor string, limit int) ([]*chat.ChatMessage, error) {
	return h.repo.ListBySession(ctx, sessionID, cursor, limit)
}
