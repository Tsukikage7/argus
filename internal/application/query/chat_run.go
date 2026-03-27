package query

import (
	"context"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/chat"
)

// GetRunHandler 获取执行详情查询处理器
type GetRunHandler struct {
	repo command.ChatRunRepository
}

// NewGetRunHandler 创建处理器
func NewGetRunHandler(repo command.ChatRunRepository) *GetRunHandler {
	return &GetRunHandler{repo: repo}
}

// Handle 执行查询
func (h *GetRunHandler) Handle(ctx context.Context, id string) (*chat.ChatRun, error) {
	return h.repo.GetRun(ctx, id)
}
