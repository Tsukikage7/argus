package handler

import (
	"encoding/json"
	"net/http"

	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
	"github.com/Tsukikage7/argus/internal/domain/tenant"
)

// AdminTenantHandler 管理端租户 CRUD
type AdminTenantHandler struct {
	tenantRepo tenant.TenantRepository
}

// NewAdminTenantHandler 创建管理端租户 Handler
func NewAdminTenantHandler(repo tenant.TenantRepository) *AdminTenantHandler {
	return &AdminTenantHandler{tenantRepo: repo}
}

// createTenantRequest 创建租户请求
type createTenantRequest struct {
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
}

// List 列出所有租户
func (h *AdminTenantHandler) List(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.tenantRepo.List(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to list tenants")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

// Create 创建租户
func (h *AdminTenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "invalid request body")
		return
	}

	if err := tenant.ValidateSlug(req.Slug); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, err.Error())
		return
	}
	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "name is required")
		return
	}

	// 检查 slug 唯一性
	existing, err := h.tenantRepo.GetBySlug(r.Context(), req.Slug)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to check slug")
		return
	}
	if existing != nil {
		httputil.WriteError(w, http.StatusConflict, httputil.CodeConflict, "slug already exists")
		return
	}

	t := &tenant.Tenant{
		Slug:           req.Slug,
		Name:           req.Name,
		Status:         "active",
		AllowedOrigins: req.AllowedOrigins,
	}
	if t.AllowedOrigins == nil {
		t.AllowedOrigins = []string{}
	}

	if err := h.tenantRepo.Create(r.Context(), t); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to create tenant")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, t)
}

// Get 获取单个租户
func (h *AdminTenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.tenantRepo.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "failed to get tenant")
		return
	}
	if t == nil {
		httputil.WriteError(w, http.StatusNotFound, httputil.CodeNotFound, "tenant not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, t)
}
