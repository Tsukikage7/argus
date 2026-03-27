package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Tsukikage7/argus/internal/domain/tenant"
)

// APIKeyPGRepository 基于 PostgreSQL 的 API Key 仓储
type APIKeyPGRepository struct {
	db *sql.DB
}

// NewAPIKeyPGRepository 创建 API Key PG 仓储
func NewAPIKeyPGRepository(db *sql.DB) *APIKeyPGRepository {
	return &APIKeyPGRepository{db: db}
}

// Create 创建 API Key
func (r *APIKeyPGRepository) Create(ctx context.Context, key *tenant.APIKey) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO tenant_api_keys (tenant_id, prefix, key_hash, salt, name, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, key.TenantID, key.Prefix, key.KeyHash, key.Salt, key.Name, key.Status, key.ExpiresAt).
		Scan(&key.ID, &key.CreatedAt)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

// GetByPrefix 按前缀查询所有活跃 API Key（用于认证时逐一验证哈希）
func (r *APIKeyPGRepository) GetByPrefix(ctx context.Context, prefix string) ([]*tenant.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, prefix, key_hash, salt, name, status, expires_at, created_at
		FROM tenant_api_keys
		WHERE prefix = $1 AND status IN ('active', 'rotating')
	`, prefix)
	if err != nil {
		return nil, fmt.Errorf("get api keys by prefix: %w", err)
	}
	defer rows.Close()

	var keys []*tenant.APIKey
	for rows.Next() {
		key := &tenant.APIKey{}
		if err := rows.Scan(&key.ID, &key.TenantID, &key.Prefix, &key.KeyHash, &key.Salt,
			&key.Name, &key.Status, &key.ExpiresAt, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// ListByTenant 列出租户的所有 API Key
func (r *APIKeyPGRepository) ListByTenant(ctx context.Context, tenantID string) ([]*tenant.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, prefix, key_hash, salt, name, status, expires_at, created_at
		FROM tenant_api_keys
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*tenant.APIKey
	for rows.Next() {
		key := &tenant.APIKey{}
		if err := rows.Scan(&key.ID, &key.TenantID, &key.Prefix, &key.KeyHash, &key.Salt,
			&key.Name, &key.Status, &key.ExpiresAt, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}
