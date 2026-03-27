## ADDED Requirements

### Requirement: Domain model tenant isolation
The system SHALL include TenantID in all core domain entities: Task, ReplaySession, TaskEvent. All operations SHALL propagate tenant context from authentication through command/query handlers to repositories.

#### Scenario: Task created with tenant context
- **WHEN** authenticated tenant triggers a diagnose command
- **THEN** the resulting Task entity SHALL have TenantID set to the authenticated tenant's ID

#### Scenario: Cross-tenant task access denied
- **WHEN** tenant A requests GET /api/v1/tasks/{id} where task belongs to tenant B
- **THEN** system returns 404 Not Found (not 403, to avoid information leakage)

### Requirement: Redis tenant-scoped keys
The system SHALL use tenant-scoped Redis key patterns: `argus:tenant:{tenant_id}:task:{task_id}`, `argus:tenant:{tenant_id}:replay:{session_id}`, `argus:tenant:{tenant_id}:replay:recent`.

#### Scenario: Task stored in tenant namespace
- **WHEN** a task is saved to Redis for tenant "acme"
- **THEN** Redis key SHALL be `argus:tenant:acme:task:{task_id}`

#### Scenario: Cross-tenant key isolation
- **WHEN** scanning Redis keys for tenant A
- **THEN** no keys belonging to tenant B SHALL be returned

### Requirement: PostgreSQL tenant-scoped queries
The system SHALL add `tenant_id` column to `diagnosis_history` table with composite index `(tenant_id, created_at DESC)`. All queries SHALL include explicit `WHERE tenant_id = $1` condition.

#### Scenario: History list filtered by tenant
- **WHEN** tenant "acme" requests GET /api/v1/tasks
- **THEN** only diagnosis records with tenant_id = "acme" are returned

#### Scenario: History record access scoped
- **WHEN** tenant A requests a specific history record belonging to tenant B
- **THEN** system returns 404 Not Found

### Requirement: Elasticsearch tenant-scoped indices
The system SHALL use index pattern `argus-{tenant_id}-logs-{yyyy.MM.dd}` for all tenant data. The `allIndex()` function SHALL be removed from business API code paths. All ES queries SHALL accept explicit tenantID parameter.

#### Scenario: ES query scoped to tenant index
- **WHEN** tenant "acme" triggers a diagnose command
- **THEN** ES queries SHALL target `argus-acme-logs-*` only

#### Scenario: New tenant with no indices
- **WHEN** tenant "newcorp" queries logs but no ES index exists yet
- **THEN** system returns empty results (not 500 error), using `ignore_unavailable` and `allow_no_indices`

#### Scenario: allIndex removed from business path
- **WHEN** any /api/v1/* endpoint triggers an ES query
- **THEN** the query SHALL NOT use `allIndex()` or any pattern that matches other tenants' indices

### Requirement: SSE tenant isolation
The system SHALL scope SSE channels by tenant. SSEHub subscription keys SHALL use format `tenant:{tenant_id}:task:{task_id}`. A valid stream_token SHALL be required to subscribe.

#### Scenario: SSE subscription requires stream_token
- **WHEN** client connects to /api/v1/stream/{id} without stream_token
- **THEN** system returns 401 Unauthorized

#### Scenario: Stream token generation
- **WHEN** authenticated tenant requests POST /api/v1/stream-tokens with task_id
- **THEN** system returns a stream_token (TTL=5min, single-use, bound to task_id + tenant_id)

#### Scenario: Stream token single-use
- **WHEN** client uses a stream_token to establish SSE, then disconnects and reconnects with same token
- **THEN** second connection returns 401 Unauthorized

#### Scenario: Cross-tenant SSE blocked
- **WHEN** tenant A obtains stream_token for their task, tenant B tries to use it
- **THEN** system returns 401 Unauthorized (token bound to tenant_id)

### Requirement: Replay session tenant isolation
The system SHALL include TenantID in ReplaySession. Replay creation, querying, and SSE streaming SHALL all enforce tenant scope.

#### Scenario: Replay created with tenant context
- **WHEN** tenant "acme" creates a replay session
- **THEN** ReplaySession.TenantID SHALL be "acme"

#### Scenario: Cross-tenant replay access denied
- **WHEN** tenant A requests GET /api/v1/replay/{id} where session belongs to tenant B
- **THEN** system returns 404 Not Found

#### Scenario: Replay list scoped
- **WHEN** tenant "acme" requests replay list
- **THEN** only replay sessions with tenant_id = "acme" are returned
