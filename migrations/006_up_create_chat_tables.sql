-- 006: 聊天系统表结构
-- 支持智能体聊天中心的会话、消息、执行和产物持久化

-- 聊天会话
CREATE TABLE IF NOT EXISTS chat_sessions (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    source       TEXT NOT NULL DEFAULT 'web',
    status       TEXT NOT NULL DEFAULT 'active',
    last_intent  TEXT NOT NULL DEFAULT '',
    summary      TEXT NOT NULL DEFAULT '',
    working_memory JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_tenant
    ON chat_sessions (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_status
    ON chat_sessions (status) WHERE status != 'deleted';

-- 聊天消息
CREATE TABLE IF NOT EXISTS chat_messages (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES chat_sessions(id),
    tenant_id    TEXT NOT NULL,
    role         TEXT NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    run_id       TEXT,
    artifacts    JSONB DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session
    ON chat_messages (session_id, created_at);

CREATE INDEX IF NOT EXISTS idx_chat_messages_run
    ON chat_messages (run_id) WHERE run_id IS NOT NULL;

-- 聊天执行（Agent Run）
CREATE TABLE IF NOT EXISTS chat_runs (
    id                 TEXT PRIMARY KEY,
    session_id         TEXT NOT NULL REFERENCES chat_sessions(id),
    tenant_id          TEXT NOT NULL,
    trigger_message_id TEXT,
    intent             TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'pending',
    steps              JSONB DEFAULT '[]',
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    error_message      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_chat_runs_session
    ON chat_runs (session_id);

-- 聊天产物（结构化附件）
CREATE TABLE IF NOT EXISTS chat_artifacts (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES chat_sessions(id),
    run_id       TEXT REFERENCES chat_runs(id),
    message_id   TEXT REFERENCES chat_messages(id),
    type         TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    payload      JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_artifacts_run
    ON chat_artifacts (run_id) WHERE run_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_artifacts_message
    ON chat_artifacts (message_id) WHERE message_id IS NOT NULL;
