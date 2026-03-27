package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/servex/storage/cache"
)

const replayTTL = 48 * time.Hour

// replayKey 生成租户隔离的 replay Redis key
func replayKey(tenantID, sessionID string) string {
	return fmt.Sprintf("argus:tenant:%s:replay:%s", tenantID, sessionID)
}

// replayRecentKey 生成租户隔离的最近回放列表 key
func replayRecentKey(tenantID string) string {
	return fmt.Sprintf("argus:tenant:%s:replay:recent", tenantID)
}

// ReplayRedisRepository 基于 Redis 的回放会话存储
type ReplayRedisRepository struct {
	cache cache.Cache
}

// NewReplayRedisRepository 创建 Redis 回放仓储
func NewReplayRedisRepository(c cache.Cache) *ReplayRedisRepository {
	return &ReplayRedisRepository{cache: c}
}

// Save 保存回放会话，并维护最近会话 ID 列表
func (r *ReplayRedisRepository) Save(ctx context.Context, s *task.ReplaySession) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal replay session: %w", err)
	}
	if err := r.cache.Set(ctx, replayKey(s.TenantID, s.ID), string(data), replayTTL); err != nil {
		return fmt.Errorf("save replay session: %w", err)
	}

	// 维护最近会话 ID 列表（逗号分隔，最多保留 50 条，最新的在最前面）
	rk := replayRecentKey(s.TenantID)
	existing, _ := r.cache.Get(ctx, rk)
	ids := []string{s.ID}
	if existing != "" {
		for _, id := range strings.Split(existing, ",") {
			if id != s.ID && id != "" {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) > 50 {
		ids = ids[:50]
	}
	_ = r.cache.Set(ctx, rk, strings.Join(ids, ","), replayTTL)
	return nil
}

// Get 获取回放会话（需要 tenantID 确保租户隔离）
func (r *ReplayRedisRepository) Get(ctx context.Context, tenantID, id string) (*task.ReplaySession, error) {
	data, err := r.cache.Get(ctx, replayKey(tenantID, id))
	if err != nil {
		return nil, fmt.Errorf("get replay session: %w", err)
	}
	var s task.ReplaySession
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, fmt.Errorf("unmarshal replay session: %w", err)
	}
	return &s, nil
}

// ListRecent 获取最近 limit 条回放会话
func (r *ReplayRedisRepository) ListRecent(ctx context.Context, tenantID string, limit int) ([]*task.ReplaySession, error) {
	raw, err := r.cache.Get(ctx, replayRecentKey(tenantID))
	if err != nil || raw == "" {
		return nil, nil
	}
	ids := strings.Split(raw, ",")
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	var sessions []*task.ReplaySession
	for _, id := range ids {
		if id == "" {
			continue
		}
		s, err := r.Get(ctx, tenantID, id)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}
