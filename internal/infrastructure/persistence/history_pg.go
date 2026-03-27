package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	_ "github.com/lib/pq"
)

// HistoryPGRepository 基于 PostgreSQL 的诊断历史存储
type HistoryPGRepository struct {
	db *sql.DB
}

// DB 返回底层数据库连接（供其他 PG 仓储复用）
func (r *HistoryPGRepository) DB() *sql.DB {
	return r.db
}

// NewHistoryPGRepository 创建 PG 历史仓储
func NewHistoryPGRepository(ctx context.Context, dsn string) (*HistoryPGRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// 自动建表 + 迁移
	if err := createHistoryTable(ctx, db); err != nil {
		return nil, err
	}

	return &HistoryPGRepository{db: db}, nil
}

func createHistoryTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS diagnosis_history (
			id           TEXT PRIMARY KEY,
			tenant_id    TEXT NOT NULL DEFAULT 'default',
			input        TEXT NOT NULL,
			source       TEXT NOT NULL DEFAULT 'cli',
			status       TEXT NOT NULL,
			steps        JSONB,
			diagnosis    JSONB,
			recovery     JSONB,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		)
	`)
	if err != nil {
		return err
	}

	// 为租户查询创建索引（幂等）
	_, err = db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_diagnosis_history_tenant
		ON diagnosis_history (tenant_id, created_at DESC)
	`)
	return err
}

// Save 保存诊断历史
func (r *HistoryPGRepository) Save(ctx context.Context, t *task.Task) error {
	steps, _ := json.Marshal(t.Steps)
	diagnosis, _ := json.Marshal(t.Diagnosis)
	recovery, _ := json.Marshal(t.Recovery)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO diagnosis_history (id, tenant_id, input, source, status, steps, diagnosis, recovery, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			steps = EXCLUDED.steps,
			diagnosis = EXCLUDED.diagnosis,
			recovery = EXCLUDED.recovery,
			completed_at = EXCLUDED.completed_at
	`, t.ID, t.TenantID, t.Input, t.Source, t.Status, steps, diagnosis, recovery, t.CreatedAt, t.CompletedAt)
	return err
}

// ListRecent 查询指定租户最近的诊断历史
func (r *HistoryPGRepository) ListRecent(ctx context.Context, tenantID string, limit int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 20
	}

	// 默认租户跳过 tenant_id 过滤，避免 UUID 类型不匹配
	var rows *sql.Rows
	var err error
	if tenantID == "" || tenantID == "default" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, tenant_id, input, source, status, steps, diagnosis, recovery, created_at, completed_at
			FROM diagnosis_history
			ORDER BY created_at DESC
			LIMIT $1
		`, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, tenant_id, input, source, status, steps, diagnosis, recovery, created_at, completed_at
			FROM diagnosis_history
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`, tenantID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*task.Task
	for rows.Next() {
		t := &task.Task{}
		var steps, diagnosis, recovery []byte
		var completedAt *time.Time
		var tenantIDNull sql.NullString
		if err := rows.Scan(&t.ID, &tenantIDNull, &t.Input, &t.Source, &t.Status, &steps, &diagnosis, &recovery, &t.CreatedAt, &completedAt); err != nil {
			return nil, err
		}
		t.TenantID = tenantIDNull.String
		_ = json.Unmarshal(steps, &t.Steps)
		_ = json.Unmarshal(diagnosis, &t.Diagnosis)
		_ = json.Unmarshal(recovery, &t.Recovery)
		t.CompletedAt = completedAt
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetByID 按 ID 查询单条诊断历史（不限租户，用于 Redis 未命中时的 fallback）
func (r *HistoryPGRepository) GetByID(ctx context.Context, id string) (*task.Task, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, input, source, status, steps, diagnosis, recovery, created_at, completed_at
		FROM diagnosis_history
		WHERE id = $1
	`, id)

	t := &task.Task{}
	var steps, diagnosis, recovery []byte
	var completedAt *time.Time
	var tenantIDNull sql.NullString
	if err := row.Scan(&t.ID, &tenantIDNull, &t.Input, &t.Source, &t.Status, &steps, &diagnosis, &recovery, &t.CreatedAt, &completedAt); err != nil {
		return nil, err
	}
	t.TenantID = tenantIDNull.String
	_ = json.Unmarshal(steps, &t.Steps)
	_ = json.Unmarshal(diagnosis, &t.Diagnosis)
	_ = json.Unmarshal(recovery, &t.Recovery)
	t.CompletedAt = completedAt
	return t, nil
}

// Close 关闭数据库连接
func (r *HistoryPGRepository) Close() error {
	return r.db.Close()
}
