package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/servex/storage/cache"
)

const replayKeyPrefix = "argus:replay:"
const replayListKey = "argus:replay:list"
const replayTTL = 48 * time.Hour

// ReplayRedisRepository 基于 Redis 的回放会话存储
type ReplayRedisRepository struct {
	cache cache.Cache
}

// NewReplayRedisRepository 创建 Redis 回放仓储
func NewReplayRedisRepository(c cache.Cache) *ReplayRedisRepository {
	return &ReplayRedisRepository{cache: c}
}

// Save 保存回放会话
func (r *ReplayRedisRepository) Save(ctx context.Context, s *task.ReplaySession) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal replay session: %w", err)
	}
	if err := r.cache.Set(ctx, replayKeyPrefix+s.ID, string(data), replayTTL); err != nil {
		return fmt.Errorf("save replay session: %w", err)
	}
	// 维护一个简单的 ID 列表（用 set 存 ID，list recent 时逐个取）
	_ = r.cache.Set(ctx, replayListKey+":"+s.ID, s.ID, replayTTL)
	return nil
}

// Get 获取回放会话
func (r *ReplayRedisRepository) Get(ctx context.Context, id string) (*task.ReplaySession, error) {
	data, err := r.cache.Get(ctx, replayKeyPrefix+id)
	if err != nil {
		return nil, fmt.Errorf("get replay session: %w", err)
	}
	var s task.ReplaySession
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, fmt.Errorf("unmarshal replay session: %w", err)
	}
	return &s, nil
}
