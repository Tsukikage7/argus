package middleware

import (
	"context"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// defaultTenantID 是 demo 模式下的默认租户 ID
const defaultTenantID = "default"

// ConfigKeyResolver 基于配置文件的 KeyResolver 实现
// 用于 demo / 单租户模式，后续可替换为数据库实现
type ConfigKeyResolver struct {
	tenantKeys map[string]bool // 业务 API Key 集合
	adminKeys  map[string]bool // 管理 API Key 集合
}

// NewConfigKeyResolver 从配置创建 KeyResolver
// apiKeys 用于业务 API（RoleTenant），adminKeys 用于管理 API（RoleAdmin）
func NewConfigKeyResolver(apiKeys []string, adminKeys []string) *ConfigKeyResolver {
	tk := make(map[string]bool, len(apiKeys))
	for _, k := range apiKeys {
		tk[k] = true
	}
	ak := make(map[string]bool, len(adminKeys))
	for _, k := range adminKeys {
		ak[k] = true
	}
	return &ConfigKeyResolver{tenantKeys: tk, adminKeys: ak}
}

// Resolve 解析 key 并返回 Principal
func (r *ConfigKeyResolver) Resolve(_ context.Context, rawKey string) (*task.Principal, error) {
	if r.adminKeys[rawKey] {
		return &task.Principal{
			TenantID: defaultTenantID,
			KeyID:    "config-admin",
			Role:     task.RoleAdmin,
		}, nil
	}
	if r.tenantKeys[rawKey] {
		return &task.Principal{
			TenantID: defaultTenantID,
			KeyID:    "config-tenant",
			Role:     task.RoleTenant,
		}, nil
	}
	return nil, nil
}
