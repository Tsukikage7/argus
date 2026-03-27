package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/lib/pq"
)

// ScenarioPGRepository 基于 PostgreSQL 的沉淀场景存储
type ScenarioPGRepository struct {
	db *sql.DB
}

// NewScenarioPGRepository 创建 PG 场景仓储
func NewScenarioPGRepository(db *sql.DB) *ScenarioPGRepository {
	return &ScenarioPGRepository{db: db}
}

// 编译期接口检查
var _ task.ScenarioRepository = (*ScenarioPGRepository)(nil)

// Save 创建或更新场景
func (r *ScenarioPGRepository) Save(ctx context.Context, s *task.CapturedScenario) error {
	if s.LogPatterns == nil {
		s.LogPatterns = []string{}
	}
	if s.AffectedNamespaces == nil {
		s.AffectedNamespaces = []string{}
	}
	patterns, err := json.Marshal(s.LogPatterns)
	if err != nil {
		return fmt.Errorf("marshal log_patterns: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO captured_scenarios (id, name, description, source_task_id, root_cause, confidence, log_patterns, affected_namespaces, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			root_cause = EXCLUDED.root_cause,
			confidence = EXCLUDED.confidence,
			log_patterns = EXCLUDED.log_patterns,
			affected_namespaces = EXCLUDED.affected_namespaces,
			status = EXCLUDED.status
	`, s.ID, s.Name, s.Description, s.SourceTaskID, s.RootCause, s.Confidence,
		patterns, pq.Array(s.AffectedNamespaces), s.Status, s.CreatedAt)
	return err
}

// Get 按 ID 查询场景
func (r *ScenarioPGRepository) Get(ctx context.Context, id string) (*task.CapturedScenario, error) {
	s := &task.CapturedScenario{}
	var patterns []byte
	var sourceTaskID, rootCause sql.NullString
	var confidence sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, source_task_id, root_cause, confidence, log_patterns, affected_namespaces, status, created_at
		FROM captured_scenarios WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.Description, &sourceTaskID, &rootCause, &confidence,
		&patterns, pq.Array(&s.AffectedNamespaces), &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get scenario %s: %w", id, err)
	}
	s.SourceTaskID = sourceTaskID.String
	s.RootCause = rootCause.String
	s.Confidence = confidence.Float64
	_ = json.Unmarshal(patterns, &s.LogPatterns)
	return s, nil
}

// List 按状态过滤查询场景列表
func (r *ScenarioPGRepository) List(ctx context.Context, status task.ScenarioStatus) ([]*task.CapturedScenario, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, source_task_id, root_cause, confidence, log_patterns, affected_namespaces, status, created_at
		FROM captured_scenarios WHERE status = $1
		ORDER BY created_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("list scenarios: %w", err)
	}
	defer rows.Close()

	var scenarios []*task.CapturedScenario
	for rows.Next() {
		s := &task.CapturedScenario{}
		var patterns []byte
		var sourceTaskID, rootCause sql.NullString
		var confidence sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &sourceTaskID, &rootCause, &confidence,
			&patterns, pq.Array(&s.AffectedNamespaces), &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.SourceTaskID = sourceTaskID.String
		s.RootCause = rootCause.String
		s.Confidence = confidence.Float64
		_ = json.Unmarshal(patterns, &s.LogPatterns)
		scenarios = append(scenarios, s)
	}
	return scenarios, rows.Err()
}

// UpdateStatus 更新场景状态
func (r *ScenarioPGRepository) UpdateStatus(ctx context.Context, id string, status task.ScenarioStatus) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE captured_scenarios SET status = $1 WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update scenario status: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("scenario %s not found", id)
	}
	return nil
}
