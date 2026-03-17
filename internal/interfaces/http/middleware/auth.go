// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"
)

// APIKeyAuth 返回 API Key 认证中间件
func APIKeyAuth(validKeys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]bool, len(validKeys))
	for _, k := range validKeys {
		keySet[k] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Authorization")
			key = strings.TrimPrefix(key, "Bearer ")

			if key == "" {
				key = r.URL.Query().Get("api_key")
			}

			if !keySet[key] {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
