package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/chat"
)

// ChatPGRepository 基于 PostgreSQL 的聊天系统存储
type ChatPGRepository struct {
	db *sql.DB
}

// NewChatPGRepository 创建聊天 PG 仓储（自动建表）
func NewChatPGRepository(db *sql.DB) *ChatPGRepository {
	repo := &ChatPGRepository{db: db}
	// 自动建表（幂等），与 HistoryPGRepository 保持一致
	_ = repo.ensureTables()
	return repo
}

// ensureTables 确保聊天系统表存在
func (r *ChatPGRepository) ensureTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id             TEXT PRIMARY KEY,
			tenant_id      TEXT NOT NULL,
			title          TEXT NOT NULL DEFAULT '',
			source         TEXT NOT NULL DEFAULT 'web',
			status         TEXT NOT NULL DEFAULT 'active',
			last_intent    TEXT NOT NULL DEFAULT '',
			summary        TEXT NOT NULL DEFAULT '',
			working_memory JSONB DEFAULT '{}',
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			archived_at    TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_tenant ON chat_sessions (tenant_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions (status) WHERE status != 'deleted'`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			tenant_id  TEXT NOT NULL,
			role       TEXT NOT NULL,
			content    TEXT NOT NULL DEFAULT '',
			status     TEXT NOT NULL DEFAULT 'pending',
			run_id     TEXT,
			artifacts  JSONB DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages (session_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_run ON chat_messages (run_id) WHERE run_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS chat_runs (
			id                 TEXT PRIMARY KEY,
			session_id         TEXT NOT NULL,
			tenant_id          TEXT NOT NULL,
			trigger_message_id TEXT,
			intent             TEXT NOT NULL DEFAULT '',
			status             TEXT NOT NULL DEFAULT 'pending',
			steps              JSONB DEFAULT '[]',
			started_at         TIMESTAMPTZ,
			completed_at       TIMESTAMPTZ,
			error_message      TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_runs_session ON chat_runs (session_id)`,
		`CREATE TABLE IF NOT EXISTS chat_artifacts (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			run_id     TEXT,
			message_id TEXT,
			type       TEXT NOT NULL,
			title      TEXT NOT NULL DEFAULT '',
			payload    JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_artifacts_run ON chat_artifacts (run_id) WHERE run_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_chat_artifacts_message ON chat_artifacts (message_id) WHERE message_id IS NOT NULL`,
	}
	for _, stmt := range statements {
		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure chat tables: %w", err)
		}
	}
	return nil
}

// ── ChatSessionRepository 实现 ────────────────────────────────────────

// CreateSession 创建聊天会话
func (r *ChatPGRepository) Create(ctx context.Context, s *chat.ChatSession) error {
	wm, _ := json.Marshal(s.WorkingMemory)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_sessions (id, tenant_id, title, source, status, last_intent, summary, working_memory, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, s.ID, s.TenantID, s.Title, s.Source, s.Status, s.LastIntent, s.Summary, wm, s.CreatedAt, s.UpdatedAt)
	return err
}

// ── ChatRunRepository 实现 ────────────────────────────────────────────

// CreateRun 创建执行记录
func (r *ChatPGRepository) CreateRun(ctx context.Context, run *chat.ChatRun) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_runs (id, session_id, tenant_id, trigger_message_id, intent, status, steps, started_at, completed_at, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, run.ID, run.SessionID, run.TenantID, run.TriggerMessageID, run.Intent, run.Status, run.Steps, run.StartedAt, run.CompletedAt, run.ErrorMessage)
	return err
}

// GetRun 获取执行记录
func (r *ChatPGRepository) GetRun(ctx context.Context, id string) (*chat.ChatRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, tenant_id, trigger_message_id, intent, status, steps, started_at, completed_at, error_message
		FROM chat_runs WHERE id = $1
	`, id)
	run := &chat.ChatRun{}
	var steps []byte
	err := row.Scan(&run.ID, &run.SessionID, &run.TenantID, &run.TriggerMessageID, &run.Intent, &run.Status, &steps, &run.StartedAt, &run.CompletedAt, &run.ErrorMessage)
	if err != nil {
		return nil, err
	}
	run.Steps = steps
	return run, nil
}

// UpdateRun 更新执行记录
func (r *ChatPGRepository) UpdateRun(ctx context.Context, run *chat.ChatRun) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chat_runs SET status=$1, steps=$2, started_at=$3, completed_at=$4, error_message=$5, intent=$6
		WHERE id = $7
	`, run.Status, run.Steps, run.StartedAt, run.CompletedAt, run.ErrorMessage, run.Intent, run.ID)
	return err
}

// ── ChatArtifactRepository 实现 ───────────────────────────────────────

// CreateArtifact 创建产物
func (r *ChatPGRepository) CreateArtifact(ctx context.Context, a *chat.ChatArtifact) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_artifacts (id, session_id, run_id, message_id, type, title, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, a.ID, a.SessionID, nullStr(a.RunID), nullStr(a.MessageID), a.Type, a.Title, a.Payload, a.CreatedAt)
	return err
}

// ListArtifactsByRun 按执行 ID 获取产物
func (r *ChatPGRepository) ListByRun(ctx context.Context, runID string) ([]*chat.ChatArtifact, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, run_id, message_id, type, title, payload, created_at
		FROM chat_artifacts WHERE run_id = $1 ORDER BY created_at
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*chat.ChatArtifact
	for rows.Next() {
		a := &chat.ChatArtifact{}
		var payload []byte
		var runIDNull, messageIDNull sql.NullString
		if err := rows.Scan(&a.ID, &a.SessionID, &runIDNull, &messageIDNull, &a.Type, &a.Title, &payload, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.RunID = runIDNull.String
		a.MessageID = messageIDNull.String
		a.Payload = payload
		artifacts = append(artifacts, a)
	}
	return artifacts, rows.Err()
}

// ── 辅助函数 ──────────────────────────────────────────────────────────

// nullStr 将空字符串转为 sql.NullString
func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// scanSession 从 QueryRow 扫描会话
func scanSession(row *sql.Row) (*chat.ChatSession, error) {
	s := &chat.ChatSession{}
	var wm []byte
	var archivedAt *time.Time
	err := row.Scan(&s.ID, &s.TenantID, &s.Title, &s.Source, &s.Status, &s.LastIntent, &s.Summary, &wm, &s.CreatedAt, &s.UpdatedAt, &archivedAt)
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	_ = json.Unmarshal(wm, &s.WorkingMemory)
	s.ArchivedAt = archivedAt
	return s, nil
}

// scanSessionRow 从 Rows 扫描会话
func scanSessionRow(rows *sql.Rows) (*chat.ChatSession, error) {
	s := &chat.ChatSession{}
	var wm []byte
	var archivedAt *time.Time
	err := rows.Scan(&s.ID, &s.TenantID, &s.Title, &s.Source, &s.Status, &s.LastIntent, &s.Summary, &wm, &s.CreatedAt, &s.UpdatedAt, &archivedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(wm, &s.WorkingMemory)
	s.ArchivedAt = archivedAt
	return s, nil
}

// scanMessage 从 Rows 扫描消息
func scanMessage(rows *sql.Rows) (*chat.ChatMessage, error) {
	m := &chat.ChatMessage{}
	var artifacts []byte
	var runIDNull sql.NullString
	err := rows.Scan(&m.ID, &m.SessionID, &m.TenantID, &m.Role, &m.Content, &m.Status, &runIDNull, &artifacts, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.RunID = runIDNull.String
	_ = json.Unmarshal(artifacts, &m.Artifacts)
	return m, nil
}

// scanMessageRow 从 QueryRow 扫描消息
func scanMessageRow(row *sql.Row) (*chat.ChatMessage, error) {
	m := &chat.ChatMessage{}
	var artifacts []byte
	var runIDNull sql.NullString
	err := row.Scan(&m.ID, &m.SessionID, &m.TenantID, &m.Role, &m.Content, &m.Status, &runIDNull, &artifacts, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan message: %w", err)
	}
	m.RunID = runIDNull.String
	_ = json.Unmarshal(artifacts, &m.Artifacts)
	return m, nil
}

// ── ChatMessageRepository 实现 ────────────────────────────────────────

// CreateMessage 创建聊天消息
func (r *ChatPGRepository) CreateMessage(ctx context.Context, m *chat.ChatMessage) error {
	artifacts, _ := json.Marshal(m.Artifacts)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (id, session_id, tenant_id, role, content, status, run_id, artifacts, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, m.ID, m.SessionID, m.TenantID, m.Role, m.Content, m.Status, nullStr(m.RunID), artifacts, m.CreatedAt)
	return err
}

// ListMessagesBySession 按会话分页获取消息（cursor 分页）
func (r *ChatPGRepository) ListBySession(ctx context.Context, sessionID string, cursor string, limit int) ([]*chat.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error
	if cursor == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, session_id, tenant_id, role, content, status, run_id, artifacts, created_at
			FROM chat_messages WHERE session_id = $1
			ORDER BY created_at ASC LIMIT $2
		`, sessionID, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, session_id, tenant_id, role, content, status, run_id, artifacts, created_at
			FROM chat_messages WHERE session_id = $1 AND created_at > $2
			ORDER BY created_at ASC LIMIT $3
		`, sessionID, cursor, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*chat.ChatMessage
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// GetMessageByID 按 ID 获取消息
func (r *ChatPGRepository) GetMessageByID(ctx context.Context, id string) (*chat.ChatMessage, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, tenant_id, role, content, status, run_id, artifacts, created_at
		FROM chat_messages WHERE id = $1
	`, id)
	return scanMessageRow(row)
}

// GetSession 获取聊天会话
func (r *ChatPGRepository) Get(ctx context.Context, tenantID, id string) (*chat.ChatSession, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, title, source, status, last_intent, summary, working_memory, created_at, updated_at, archived_at
		FROM chat_sessions WHERE id = $1 AND tenant_id = $2 AND status != 'deleted'
	`, id, tenantID)
	return scanSession(row)
}

// ListSessions 列出聊天会话
func (r *ChatPGRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*chat.ChatSession, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, title, source, status, last_intent, summary, working_memory, created_at, updated_at, archived_at
		FROM chat_sessions WHERE tenant_id = $1 AND status != 'deleted'
		ORDER BY updated_at DESC LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*chat.ChatSession
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// UpdateSession 更新聊天会话
func (r *ChatPGRepository) Update(ctx context.Context, s *chat.ChatSession) error {
	wm, _ := json.Marshal(s.WorkingMemory)
	_, err := r.db.ExecContext(ctx, `
		UPDATE chat_sessions SET title=$1, status=$2, last_intent=$3, summary=$4, working_memory=$5, updated_at=$6, archived_at=$7
		WHERE id = $8 AND tenant_id = $9
	`, s.Title, s.Status, s.LastIntent, s.Summary, wm, s.UpdatedAt, s.ArchivedAt, s.ID, s.TenantID)
	return err
}

// SoftDelete 软删除聊天会话
func (r *ChatPGRepository) SoftDelete(ctx context.Context, tenantID, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE chat_sessions SET status='deleted', updated_at=NOW() WHERE id = $1 AND tenant_id = $2
	`, id, tenantID)
	return err
}

// CleanExpired 清理过期会话
func (r *ChatPGRepository) CleanExpired(ctx context.Context, ttlDays int) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE chat_sessions SET status='deleted', updated_at=NOW()
		WHERE status = 'active' AND updated_at < NOW() - INTERVAL '1 day' * $1
	`, ttlDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
