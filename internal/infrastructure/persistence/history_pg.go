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

// NewHistoryPGRepository 创建 PG 历史仓储
func NewHistoryPGRepository(ctx context.Context, dsn string) (*HistoryPGRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// 自动建表
	if err := createHistoryTable(ctx, db); err != nil {
		return nil, err
	}

	return &HistoryPGRepository{db: db}, nil
}

func createHistoryTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS diagnosis_history (
			id          TEXT PRIMARY KEY,
			input       TEXT NOT NULL,
			source      TEXT NOT NULL DEFAULT 'cli',
			status      TEXT NOT NULL,
			steps       JSONB,
			diagnosis   JSONB,
			recovery    JSONB,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		)
	`)
	return err
}

// Save 保存诊断历史
func (r *HistoryPGRepository) Save(ctx context.Context, t *task.Task) error {
	steps, _ := json.Marshal(t.Steps)
	diagnosis, _ := json.Marshal(t.Diagnosis)
	recovery, _ := json.Marshal(t.Recovery)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO diagnosis_history (id, input, source, status, steps, diagnosis, recovery, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			steps = EXCLUDED.steps,
			diagnosis = EXCLUDED.diagnosis,
			recovery = EXCLUDED.recovery,
			completed_at = EXCLUDED.completed_at
	`, t.ID, t.Input, t.Source, t.Status, steps, diagnosis, recovery, t.CreatedAt, t.CompletedAt)
	return err
}

// ListRecent 查询最近的诊断历史
func (r *HistoryPGRepository) ListRecent(ctx context.Context, limit int) ([]*task.Task, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, input, source, status, steps, diagnosis, recovery, created_at, completed_at
		FROM diagnosis_history
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*task.Task
	for rows.Next() {
		t := &task.Task{}
		var steps, diagnosis, recovery []byte
		var completedAt *time.Time
		if err := rows.Scan(&t.ID, &t.Input, &t.Source, &t.Status, &steps, &diagnosis, &recovery, &t.CreatedAt, &completedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(steps, &t.Steps)
		_ = json.Unmarshal(diagnosis, &t.Diagnosis)
		_ = json.Unmarshal(recovery, &t.Recovery)
		t.CompletedAt = completedAt
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// Close 关闭数据库连接
func (r *HistoryPGRepository) Close() error {
	return r.db.Close()
}
