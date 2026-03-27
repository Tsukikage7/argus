package persistence

import (
	"context"
	"fmt"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// NoopHistoryRepository 空实现的诊断历史存储，用于 PG 不可用时的降级处理
// 所有方法均为空操作，不会引发 panic
type NoopHistoryRepository struct{}

// 编译期检查：确保 NoopHistoryRepository 实现了 HistoryRepository 接口
// 注意：此处无法直接引用 command 包（循环依赖），接口兼容性由调用方保证

// Save 空实现，直接丢弃诊断记录
func (r *NoopHistoryRepository) Save(_ context.Context, _ *task.Task) error {
	return nil
}

// ListRecent 空实现，始终返回空列表（非 nil，保证 JSON 编码为 [] 而非 null）
func (r *NoopHistoryRepository) ListRecent(_ context.Context, _ string, _ int) ([]*task.Task, error) {
	return []*task.Task{}, nil
}

// GetByID 空实现，始终返回未找到
func (r *NoopHistoryRepository) GetByID(_ context.Context, _ string) (*task.Task, error) {
	return nil, fmt.Errorf("task not found")
}
