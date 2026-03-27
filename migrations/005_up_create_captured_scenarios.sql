CREATE TABLE captured_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    source_task_id VARCHAR(64),
    root_cause TEXT,
    confidence DECIMAL(3,2),
    log_patterns JSONB NOT NULL DEFAULT '[]',
    affected_namespaces TEXT[] DEFAULT '{}',
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
