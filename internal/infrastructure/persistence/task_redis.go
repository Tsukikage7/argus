// Package persistence 提供任务和历史记录的持久化实现
package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/servex/storage/cache"
)

const taskTTL = 24 * time.Hour

// taskKey 生成租户隔离的 task Redis key
// 格式：argus:tenant:{tenantID}:task:{taskID}
func taskKey(tenantID, taskID string) string {
	return fmt.Sprintf("argus:tenant:%s:task:%s", tenantID, taskID)
}

// TaskRedisRepository 基于 Redis 的任务状态存储
type TaskRedisRepository struct {
	cache cache.Cache
}

// NewTaskRedisRepository 创建 Redis 任务仓储
func NewTaskRedisRepository(c cache.Cache) *TaskRedisRepository {
	return &TaskRedisRepository{cache: c}
}

// Save 保存任务（使用 Task.TenantID 构建租户隔离 key）
func (r *TaskRedisRepository) Save(ctx context.Context, t *task.Task) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return r.cache.Set(ctx, taskKey(t.TenantID, t.ID), string(data), taskTTL)
}

// Get 获取任务（需要 tenantID 确保租户隔离）
func (r *TaskRedisRepository) Get(ctx context.Context, tenantID, id string) (*task.Task, error) {
	data, err := r.cache.Get(ctx, taskKey(tenantID, id))
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	var t task.Task
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &t, nil
}
