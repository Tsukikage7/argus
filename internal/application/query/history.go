package query

import (
	"context"

	"github.com/Tsukikage7/argus/internal/application/command"
	"github.com/Tsukikage7/argus/internal/domain/task"
)

// HistoryQuery 诊断历史查询
type HistoryQuery struct {
	TenantID string
	Limit    int
}

// HistoryHandler 处理历史查询
type HistoryHandler struct {
	history command.HistoryRepository
}

// NewHistoryHandler 创建历史查询处理器
func NewHistoryHandler(history command.HistoryRepository) *HistoryHandler {
	return &HistoryHandler{history: history}
}

// Handle 查询诊断历史
func (h *HistoryHandler) Handle(ctx context.Context, q HistoryQuery) ([]*task.Task, error) {
	return h.history.ListRecent(ctx, q.TenantID, q.Limit)
}
