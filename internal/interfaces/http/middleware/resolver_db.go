package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tenant"
)

// DBKeyResolver 基于数据库的 KeyResolver 实现
// 通过 API Key 前缀索引快速查找，然后验证 SHA-256 哈希
type DBKeyResolver struct {
	keyRepo    tenant.APIKeyRepository
	tenantRepo tenant.TenantRepository
}

// NewDBKeyResolver 创建数据库版 KeyResolver
func NewDBKeyResolver(keyRepo tenant.APIKeyRepository, tenantRepo tenant.TenantRepository) *DBKeyResolver {
	return &DBKeyResolver{keyRepo: keyRepo, tenantRepo: tenantRepo}
}

// Resolve 解析 API Key 并返回 Principal
// key 格式：arg_{slug}_{random32hex}
// 前缀索引：arg_{slug}_
func (r *DBKeyResolver) Resolve(ctx context.Context, rawKey string) (*task.Principal, error) {
	// 提取前缀：找到第三个 _ 之前的部分（含尾部 _）
	prefix := extractPrefix(rawKey)
	if prefix == "" {
		return nil, nil
	}

	// 按前缀查找所有匹配的 key 记录
	apiKeys, err := r.keyRepo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	// 逐一验证哈希，找到匹配的 key
	var matched *tenant.APIKey
	for _, k := range apiKeys {
		h := tenant.HashKey(rawKey, k.Salt)
		if h == k.KeyHash {
			matched = k
			break
		}
	}
	if matched == nil {
		return nil, nil
	}

	// 检查过期时间
	if matched.ExpiresAt != nil && matched.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	// 检查租户状态
	t, err := r.tenantRepo.GetByID(ctx, matched.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != "active" {
		return nil, nil
	}

	return &task.Principal{
		TenantID: matched.TenantID,
		KeyID:    matched.ID,
		Role:     task.RoleTenant,
	}, nil
}

// extractPrefix 从 key 中提取前缀（arg_{slug}_）
func extractPrefix(key string) string {
	// 格式：arg_{slug}_{random}
	// 前缀：arg_{slug}_
	if !strings.HasPrefix(key, "arg_") {
		return ""
	}
	// 找到第三个 _ 的位置（arg_ 之后的第一个 _）
	rest := key[4:] // 去掉 "arg_"
	idx := strings.Index(rest, "_")
	if idx < 0 {
		return ""
	}
	return key[:4+idx+1] // "arg_{slug}_"
}
