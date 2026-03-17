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

const taskKeyPrefix = "argus:task:"
const taskTTL = 24 * time.Hour

// TaskRedisRepository 基于 Redis 的任务状态存储
type TaskRedisRepository struct {
	cache cache.Cache
}

// NewTaskRedisRepository 创建 Redis 任务仓储
func NewTaskRedisRepository(c cache.Cache) *TaskRedisRepository {
	return &TaskRedisRepository{cache: c}
}

// Save 保存任务
func (r *TaskRedisRepository) Save(ctx context.Context, t *task.Task) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return r.cache.Set(ctx, taskKeyPrefix+t.ID, string(data), taskTTL)
}

// Get 获取任务
func (r *TaskRedisRepository) Get(ctx context.Context, id string) (*task.Task, error) {
	data, err := r.cache.Get(ctx, taskKeyPrefix+id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	var t task.Task
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &t, nil
}
