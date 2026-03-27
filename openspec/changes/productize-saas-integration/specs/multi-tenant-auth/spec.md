## ADDED Requirements

### Requirement: Tenant identity model
Each tenant SHALL have a UUID primary key (`id`) and an immutable slug field (`slug`). The slug SHALL be 3-32 characters, lowercase alphanumeric with hyphens, and UNIQUE. The slug SHALL be used in ES index names, Redis key prefixes, and API Key prefixes. Once created, the slug SHALL NOT be modifiable.

#### Scenario: Slug immutability
- **WHEN** admin attempts to update a tenant's slug
- **THEN** system returns 400 with code "slug_immutable"

#### Scenario: Slug format validation
- **WHEN** admin creates tenant with slug "ACME" or "a" or "acme@corp"
- **THEN** system returns 400 with code "validation_error" and details about slug format requirements

### Requirement: Tenant CRUD lifecycle
The system SHALL provide an admin API to create, list, update, and soft-delete tenants. Each tenant SHALL have a UUID id, immutable slug, name, status (active/deleted), allowed_origins, and timestamps.

#### Scenario: Create tenant
- **WHEN** admin sends POST /admin/v1/tenants with name "Acme Corp" and slug "acme"
- **THEN** system creates a tenant with status "active" and returns id (UUID) and slug

#### Scenario: List tenants
- **WHEN** admin sends GET /admin/v1/tenants
- **THEN** system returns paginated list of all tenants with status and creation time

#### Scenario: Soft-delete tenant
- **WHEN** admin sends DELETE /admin/v1/tenants/{id}
- **THEN** system marks tenant as "deleted", all associated API Keys are immediately invalidated, and new requests from this tenant return 401

### Requirement: API Key management
The system SHALL support creating, listing, rotating, and revoking API Keys for a tenant. Keys SHALL be stored as SHA-256 hash with per-key salt and prefix indexing. Plaintext SHALL be returned only once at creation time.

#### Scenario: Create API Key
- **WHEN** admin sends POST /admin/v1/tenants/{id}/keys
- **THEN** system generates a key in format `arg_{tenant_slug}_{random32_base62}`, computes SHA-256(key + salt), stores hash + salt + prefix, and returns plaintext once

#### Scenario: Rotate API Key
- **WHEN** admin sends POST /admin/v1/keys/{key_id}/rotate
- **THEN** system creates a new key, marks old key as "rotating" with 24h grace period, both keys authenticate successfully during grace period

#### Scenario: Revoke API Key
- **WHEN** admin sends DELETE /admin/v1/keys/{key_id}
- **THEN** system marks key as "revoked", subsequent requests with this key return 401

#### Scenario: Grace period expiry
- **WHEN** 24 hours pass after key rotation
- **THEN** old key status changes from "rotating" to "revoked" automatically

### Requirement: Dual-level authentication
The system SHALL support two authentication levels: AdminKey for /admin/v1/* endpoints and TenantKey for /api/v1/* endpoints. Authentication SHALL inject a Principal (tenant_id, key_id, role) into request context.

#### Scenario: TenantKey authenticates business API
- **WHEN** request with valid TenantKey hits /api/v1/diagnose
- **THEN** system extracts tenant_id and key_id into request context, handler receives Principal

#### Scenario: TenantKey rejected on admin API
- **WHEN** request with TenantKey hits /admin/v1/tenants
- **THEN** system returns 403 Forbidden

#### Scenario: AdminKey rejected on business API
- **WHEN** request with AdminKey hits /api/v1/diagnose
- **THEN** system returns 403 Forbidden

#### Scenario: Invalid key
- **WHEN** request with unknown key hits any authenticated endpoint
- **THEN** system returns 401 Unauthorized

#### Scenario: Disabled key
- **WHEN** request with revoked or expired key hits any endpoint
- **THEN** system returns 401 Unauthorized with reason "key_revoked" or "key_expired"

### Requirement: Query parameter authentication deprecated
The system SHALL NOT accept API Keys via query parameter `?api_key=`. All authentication SHALL use `Authorization: Bearer <key>` header.

#### Scenario: Query parameter rejected
- **WHEN** request includes `?api_key=xxx` without Authorization header
- **THEN** system returns 401 Unauthorized
