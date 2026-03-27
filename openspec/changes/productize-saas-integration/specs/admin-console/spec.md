## ADDED Requirements

### Requirement: Admin console layout
The admin console SHALL be a Vue 3 SPA integrated into the existing frontend/ project, accessible at /admin/* routes. Layout SHALL consist of a fixed left sidebar (brand logo, navigation menu, logout button), top breadcrumb bar, and right content area. Visual style SHALL match the main diagnostic panel's Indigo/Purple gradient theme.

#### Scenario: Admin authentication
- **WHEN** unauthenticated user navigates to /admin/tenants
- **THEN** system redirects to /admin/login page
- **WHEN** user enters valid AdminKey and submits
- **THEN** system validates key by calling GET /admin/v1/tenants, stores key in localStorage, and redirects to /admin/tenants
- **WHEN** user clicks logout
- **THEN** system clears localStorage and redirects to /admin/login

### Requirement: Tenant management page
The system SHALL provide a tenant list page at /admin/tenants with DaisyUI table component, search bar, status filter dropdown, and cursor-based pagination controls.

#### Scenario: View tenant list
- **WHEN** admin navigates to /admin/tenants
- **THEN** page displays table with columns: Name, Slug, Status (badge-success for active, badge-error for deleted), Key Count, Created At, Actions
- **THEN** page supports cursor-based pagination (Previous/Next buttons)

#### Scenario: Create tenant
- **WHEN** admin clicks "Create Tenant" button
- **THEN** modal appears with form: Name input + Slug input (with format validation hint: 3-32 chars, lowercase alphanumeric + hyphens)
- **WHEN** admin submits valid form
- **THEN** system creates tenant and navigates to tenant detail page

#### Scenario: Delete tenant
- **WHEN** admin clicks "Delete" on a tenant row
- **THEN** danger confirmation modal appears (red background, requires typing slug to confirm)
- **WHEN** admin confirms deletion
- **THEN** system soft-deletes tenant, all API Keys are invalidated, table refreshes

### Requirement: Tenant detail page
The system SHALL provide a tenant detail page at /admin/tenants/:id with tabs for API Key Management and Usage Statistics.

#### Scenario: View tenant detail
- **WHEN** admin navigates to /admin/tenants/:id
- **THEN** page displays: info card (name, slug, status, created_at, allowed_origins editor), and tabbed content below

### Requirement: API Key management page
The tenant detail page SHALL include an API Key management tab with card-based key list.

#### Scenario: View key list
- **WHEN** admin views API Keys tab
- **THEN** page displays cards for each key showing: prefix (e.g., `arg_acme_...xxxx`), status badge, created_at, last_used_at timestamp

#### Scenario: Create key ("read-once" display)
- **WHEN** admin clicks "Create Key" button
- **THEN** system generates new key and displays it in a prominent modal with:
  - Full plaintext key in large monospace font
  - One-click copy button
  - Warning banner: "This key will only be shown once. Copy it now."
  - Checkbox: "I have saved this key" (required to dismiss)

#### Scenario: Rotate key
- **WHEN** admin clicks "Rotate" on an active key
- **THEN** confirmation modal explains 24h grace period
- **WHEN** admin confirms rotation
- **THEN** system creates new key (shown once in modal), old key marked as "rotating"
- **THEN** old key card shows countdown timer for grace period expiry

#### Scenario: Revoke key
- **WHEN** admin clicks "Revoke" on a key
- **THEN** danger confirmation dialog (red theme, requires typing "REVOKE" to confirm)
- **WHEN** admin confirms
- **THEN** key is immediately revoked, card status updates to "revoked" badge

### Requirement: Usage statistics dashboard
The tenant detail page SHALL include a usage statistics tab with ApexCharts visualizations.

#### Scenario: View usage overview
- **WHEN** admin views Usage tab
- **THEN** page displays overview stats row (DaisyUI stats component with count-up animation):
  - Total Diagnoses count
  - Total Replays count
  - API Requests (last 24h) count
  - Active Keys count

#### Scenario: View usage trends
- **WHEN** admin views Usage tab
- **THEN** page displays ApexCharts area chart: daily API call volume (last 30 days), dual Y-axis (call count + avg latency)

#### Scenario: View rate limit status
- **WHEN** admin views Usage tab
- **THEN** page displays ApexCharts radial bar chart showing current rate limit window utilization percentage

### Requirement: Integration guide page
The system SHALL provide an integration guide page at /admin/integration with code snippets, step-by-step instructions, and API examples.

#### Scenario: View integration guide
- **WHEN** admin navigates to /admin/integration
- **THEN** page displays:
  1. Step-by-step cards (Step 1: Create Tenant → Step 2: Generate API Key → Step 3: Embed Widget)
  2. Widget embed code snippet (Shiki syntax highlighting + copy button):
     ```html
     <script src="https://cdn.example.com/argus-widget.es.js"
       data-api-key="YOUR_API_KEY"
       data-base-url="https://api.example.com">
     </script>
     ```
  3. REST API examples in three languages (curl / JavaScript fetch / Python requests)
  4. Webhook configuration documentation

### Requirement: Admin console component tree
```
src/views/admin/
├── AdminLayout.vue              # Sidebar + topbar + router-view
├── AdminLogin.vue               # Key input + validation
├── TenantList.vue               # Table + search + filter + pagination
├── TenantDetail.vue             # Info card + tabs (Keys / Usage)
├── IntegrationGuide.vue         # Steps + code snippets + API examples
└── components/
    ├── TenantCreateModal.vue    # Name + Slug form modal
    ├── TenantDeleteModal.vue    # Danger confirmation modal
    ├── KeyCard.vue              # Individual key display card
    ├── KeyCreateModal.vue       # "Read-once" key display modal
    ├── KeyRotateModal.vue       # Rotation confirmation + grace period
    ├── KeyRevokeModal.vue       # Danger revoke confirmation
    ├── UsageOverview.vue        # Stats row with count-up
    ├── UsageTrendChart.vue      # ApexCharts area chart
    ├── RateLimitGauge.vue       # ApexCharts radial bar
    └── CodeSnippet.vue          # Shiki syntax highlight + copy button
```
