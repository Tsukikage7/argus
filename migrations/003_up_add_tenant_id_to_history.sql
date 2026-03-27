-- diagnosis_history 表已在 history_pg.go 自动建表中包含 tenant_id 列
-- 此迁移确保索引存在（幂等）
CREATE INDEX IF NOT EXISTS idx_diagnosis_history_tenant
ON diagnosis_history (tenant_id, created_at DESC);
