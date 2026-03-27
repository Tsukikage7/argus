CREATE TABLE tenant_api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    prefix      VARCHAR(64) NOT NULL,
    key_hash    TEXT NOT NULL,
    salt        TEXT NOT NULL,
    name        VARCHAR(128) NOT NULL DEFAULT '',
    status      VARCHAR(16) NOT NULL DEFAULT 'active'
                CHECK (status IN ('active', 'rotating', 'revoked')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenant_api_keys_prefix ON tenant_api_keys (prefix) WHERE status IN ('active', 'rotating');
CREATE UNIQUE INDEX idx_tenant_api_keys_hash ON tenant_api_keys (key_hash);
CREATE INDEX idx_tenant_api_keys_tenant ON tenant_api_keys (tenant_id, status);
