// Package task 认证主体模型
package task

import "context"

// Role 认证角色
type Role string

const (
	RoleAdmin  Role = "admin"  // 管理端 API Key
	RoleTenant Role = "tenant" // 租户业务 API Key
)

// Principal 认证主体，由认证中间件注入到 request context
type Principal struct {
	TenantID string `json:"tenant_id"` // 租户 ID
	KeyID    string `json:"key_id"`    // API Key ID
	Role     Role   `json:"role"`      // admin / tenant
}

type contextKey struct{}
type tenantIDKey struct{}

// WithPrincipal 将 Principal 注入 context
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}

// PrincipalFrom 从 context 提取 Principal，未认证时返回 nil
func PrincipalFrom(ctx context.Context) *Principal {
	p, _ := ctx.Value(contextKey{}).(*Principal)
	return p
}

// WithTenantID 将租户 ID 注入 context（用于 Agent 后台 goroutine 等无 Principal 的场景）
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey{}, tenantID)
}

// TenantIDFrom 从 context 提取租户 ID
// 优先从 Principal 提取，回落到独立的 tenantID key
func TenantIDFrom(ctx context.Context) string {
	if p := PrincipalFrom(ctx); p != nil {
		return p.TenantID
	}
	id, _ := ctx.Value(tenantIDKey{}).(string)
	return id
}
