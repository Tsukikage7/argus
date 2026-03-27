## ADDED Requirements

### Requirement: CORS middleware with tenant allowlist
The system SHALL implement a CORS middleware that validates Origin against a per-tenant domain allowlist. Requests from non-allowed origins SHALL be rejected. Admin API SHALL have a separate CORS policy.

#### Scenario: Allowed origin
- **WHEN** Widget sends request from https://app.acme.com which is in tenant "acme" allowlist
- **THEN** system responds with appropriate CORS headers (Access-Control-Allow-Origin, etc.)

#### Scenario: Disallowed origin
- **WHEN** Widget sends request from https://evil.com which is NOT in any tenant allowlist
- **THEN** system responds without CORS headers (browser blocks the request)

### Requirement: Tenant-scoped rate limiting
The system SHALL enforce rate limits per tenant using Redis sliding window counters (Sorted Set + Lua script). Default limit: 100 requests/minute. Rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset) SHALL be included in responses. Specific routes MAY have lower limits (e.g., diagnose: 20/min, replay: 10/min).

#### Scenario: Under limit
- **WHEN** tenant sends 50 requests within a minute (limit is 100)
- **THEN** all requests succeed, X-RateLimit-Remaining shows 50

#### Scenario: Over limit
- **WHEN** tenant sends 101st request within a minute
- **THEN** system returns 429 Too Many Requests with Retry-After header

#### Scenario: Window reset
- **WHEN** rate limit window expires
- **THEN** tenant's counter resets and requests succeed again

### Requirement: OpenAPI 3.0 documentation
The system SHALL provide an OpenAPI 3.0 specification document at /api/v1/openapi.json describing all business API endpoints, request/response schemas, authentication requirements, and error codes.

#### Scenario: OpenAPI spec accessible
- **WHEN** GET /api/v1/openapi.json is requested
- **THEN** system returns valid OpenAPI 3.0 JSON document

### Requirement: Standardized error responses
The system SHALL return errors in a consistent JSON format: `{"error": {"code": "<ERROR_CODE>", "message": "<human readable>", "details": {}}}`. HTTP status codes SHALL follow RFC 7231.

#### Scenario: Validation error
- **WHEN** request body is missing required field
- **THEN** system returns 400 with code "validation_error" and details listing invalid fields

#### Scenario: Not found
- **WHEN** requested resource does not exist or belongs to another tenant
- **THEN** system returns 404 with code "not_found"

#### Scenario: Rate limited
- **WHEN** tenant exceeds rate limit
- **THEN** system returns 429 with code "rate_limited" and Retry-After header

### Requirement: Paginated list responses
The system SHALL support cursor-based pagination for all list endpoints. Query parameters: `limit` (default 20, max 100), `cursor` (opaque string). Response SHALL include `next_cursor` field.

#### Scenario: First page
- **WHEN** GET /api/v1/tasks?limit=10
- **THEN** system returns up to 10 items and a next_cursor if more exist

#### Scenario: Next page
- **WHEN** GET /api/v1/tasks?limit=10&cursor=xxx
- **THEN** system returns next 10 items starting after the cursor position
