// Package http 提供 HTTP 层公共工具
package http

import (
	"encoding/json"
	"net/http"
)

// 标准错误码常量
const (
	CodeUnauthorized   = "unauthorized"
	CodeForbidden      = "forbidden"
	CodeNotFound       = "not_found"
	CodeValidation     = "validation_error"
	CodeConflict       = "conflict"
	CodeSlugImmutable  = "slug_immutable"
	CodeInternal       = "internal_error"
	CodeRateLimited    = "rate_limited"
)

// APIError 标准化错误响应信封
type APIError struct {
	Error APIErrorBody `json:"error"`
}

// APIErrorBody 错误详情
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// WriteError 写入标准化 JSON 错误响应
func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Error: APIErrorBody{Code: code, Message: message},
	})
}

// WriteErrorWithDetails 写入带详情的标准化 JSON 错误响应
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Error: APIErrorBody{Code: code, Message: message, Details: details},
	})
}

// WriteJSON 写入 JSON 成功响应
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
