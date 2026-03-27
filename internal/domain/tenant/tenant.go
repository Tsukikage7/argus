// Package tenant 租户领域模型
package tenant

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// slugPattern 合法 slug 格式：3-32 位小写字母数字和连字符
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$`)

// Tenant 租户实体
type Tenant struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	AllowedOrigins []string  `json:"allowed_origins"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// APIKey 租户 API Key 实体
type APIKey struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Prefix    string     `json:"prefix"`
	KeyHash   string     `json:"-"`
	Salt      string     `json:"-"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ValidateSlug 校验 slug 格式
func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must be 3-32 chars, lowercase alphanumeric with hyphens")
	}
	return nil
}

// NewAPIKey 生成新的 API Key，返回明文（仅此一次）和实体
func NewAPIKey(tenantSlug, name string) (plaintext string, key *APIKey, err error) {
	// 生成 32 字节随机数作为 key 后缀
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return "", nil, fmt.Errorf("generate random: %w", err)
	}
	randHex := hex.EncodeToString(randBytes)

	prefix := fmt.Sprintf("arg_%s_", tenantSlug)
	plaintext = prefix + randHex

	// 生成 salt
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", nil, fmt.Errorf("generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)

	// SHA-256 哈希
	hash := HashKey(plaintext, salt)

	key = &APIKey{
		Prefix:  prefix,
		KeyHash: hash,
		Salt:    salt,
		Name:    name,
		Status:  "active",
	}
	return plaintext, key, nil
}

// HashKey 计算 key 的 SHA-256 哈希
func HashKey(plaintext, salt string) string {
	h := sha256.Sum256([]byte(plaintext + salt))
	return hex.EncodeToString(h[:])
}

// TenantRepository 租户仓储接口
type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	List(ctx context.Context) ([]*Tenant, error)
}

// APIKeyRepository API Key 仓储接口
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByPrefix(ctx context.Context, prefix string) ([]*APIKey, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*APIKey, error)
}
