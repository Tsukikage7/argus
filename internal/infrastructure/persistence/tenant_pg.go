package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Tsukikage7/argus/internal/domain/tenant"
	"github.com/lib/pq"
)

// TenantPGRepository 基于 PostgreSQL 的租户仓储
type TenantPGRepository struct {
	db *sql.DB
}

// NewTenantPGRepository 创建租户 PG 仓储
func NewTenantPGRepository(db *sql.DB) *TenantPGRepository {
	return &TenantPGRepository{db: db}
}

// Create 创建租户
func (r *TenantPGRepository) Create(ctx context.Context, t *tenant.Tenant) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tenants (slug, name, status, allowed_origins)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, t.Slug, t.Name, t.Status, pq.Array(t.AllowedOrigins)).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

// GetByID 按 ID 查询租户
func (r *TenantPGRepository) GetByID(ctx context.Context, id string) (*tenant.Tenant, error) {
	t := &tenant.Tenant{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, status, allowed_origins, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id).Scan(&t.ID, &t.Slug, &t.Name, &t.Status, pq.Array(&t.AllowedOrigins), &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

// GetBySlug 按 slug 查询租户
func (r *TenantPGRepository) GetBySlug(ctx context.Context, slug string) (*tenant.Tenant, error) {
	t := &tenant.Tenant{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, slug, name, status, allowed_origins, created_at, updated_at
		FROM tenants WHERE slug = $1
	`, slug).Scan(&t.ID, &t.Slug, &t.Name, &t.Status, pq.Array(&t.AllowedOrigins), &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return t, nil
}

// List 列出所有活跃租户
func (r *TenantPGRepository) List(ctx context.Context) ([]*tenant.Tenant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, slug, name, status, allowed_origins, created_at, updated_at
		FROM tenants WHERE status = 'active'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*tenant.Tenant
	for rows.Next() {
		t := &tenant.Tenant{}
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Status, pq.Array(&t.AllowedOrigins), &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}
