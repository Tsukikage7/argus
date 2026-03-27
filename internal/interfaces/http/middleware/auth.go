// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// KeyResolver 抽象 API Key 查找逻辑
// 由基础设施层实现（配置文件 / 数据库）
type KeyResolver interface {
	// Resolve 根据原始 key 字符串解析出认证主体
	// 返回 nil 表示 key 无效
	Resolve(ctx context.Context, rawKey string) (*task.Principal, error)
}

// extractBearerToken 从 Authorization header 提取 Bearer token
// 不再支持 ?api_key= 查询参数
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// TenantAuth 返回租户业务 API 认证中间件（/api/v1/*）
// 仅允许 RoleTenant 角色通过，AdminKey 返回 403
func TenantAuth(resolver KeyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "missing or invalid Authorization header")
				return
			}

			principal, err := resolver.Resolve(r.Context(), token)
			if err != nil || principal == nil {
				httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "invalid API key")
				return
			}

			// AdminKey 不允许访问业务 API
			if principal.Role == task.RoleAdmin {
				httputil.WriteError(w, http.StatusForbidden, httputil.CodeForbidden, "admin keys cannot access business API")
				return
			}

			ctx := task.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuth 返回管理端 API 认证中间件（/admin/v1/*）
// 仅允许 RoleAdmin 角色通过，TenantKey 返回 403
func AdminAuth(resolver KeyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "missing or invalid Authorization header")
				return
			}

			principal, err := resolver.Resolve(r.Context(), token)
			if err != nil || principal == nil {
				httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "invalid API key")
				return
			}

			// TenantKey 不允许访问管理 API
			if principal.Role == task.RoleTenant {
				httputil.WriteError(w, http.StatusForbidden, httputil.CodeForbidden, "tenant keys cannot access admin API")
				return
			}

			ctx := task.WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
