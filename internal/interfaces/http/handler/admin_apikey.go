package handler

import (
	"encoding/json"
	"net/http"

	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
	"github.com/Tsukikage7/argus/internal/domain/tenant"
)

// AdminAPIKeyHandler 管理端 API Key 操作
type AdminAPIKeyHandler struct {
	keyRepo    tenant.APIKeyRepository
	tenantRepo tenant.TenantRepository
}

// NewAdminAPIKeyHandler 创建管理端 API Key Handler
func NewAdminAPIKeyHandler(keyRepo tenant.APIKeyRepository, tenantRepo tenant.TenantRepository) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{keyRepo: keyRepo, tenantRepo: tenantRepo}
}

// createKeyRequest 创建 API Key 请求
type createKeyRequest struct {
	Name string `json:"name"`
}

// createKeyResponse 创建 API Key 响应（包含明文，仅此一次）
type createKeyResponse struct {
	Key       string      `json:"key"`
	APIKeyRef *tenant.APIKey `json:"api_key"`
}

// List 列出租户的所有 API Key
func (h *AdminAPIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")

	// 验证租户存在
	t, err := h.tenantRepo.GetByID(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to get tenant")
		return
	}
	if t == nil {
		httputil.WriteError(w, http.StatusNotFound, httputil.CodeNotFound, "tenant not found")
		return
	}

	keys, err := h.keyRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to list keys")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

// Create 为租户创建新的 API Key
func (h *AdminAPIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")

	// 验证租户存在
	t, err := h.tenantRepo.GetByID(r.Context(), tenantID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to get tenant")
		return
	}
	if t == nil {
		httputil.WriteError(w, http.StatusNotFound, httputil.CodeNotFound, "tenant not found")
		return
	}

	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "invalid request body")
		return
	}

	plaintext, apiKey, err := tenant.NewAPIKey(t.Slug, req.Name)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to generate key")
		return
	}
	apiKey.TenantID = tenantID

	if err := h.keyRepo.Create(r.Context(), apiKey); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to create key")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, createKeyResponse{
		Key:       plaintext,
		APIKeyRef: apiKey,
	})
}
