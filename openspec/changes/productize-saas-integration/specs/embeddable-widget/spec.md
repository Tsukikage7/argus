## ADDED Requirements

### Requirement: Widget script loading
The system SHALL provide a single JavaScript file (`argus-widget.es.js` / `argus-widget.umd.js`) built via Vite library mode. The widget SHALL register an `<argus-widget>` custom element using Vue 3.5+ `defineCustomElement`. Vue SHALL be bundled into the widget (not externalized) to ensure zero host dependencies.

#### Scenario: Widget initialization
- **WHEN** host page loads `<script src="argus-widget.es.js" data-api-key="tenant-key" data-base-url="https://api.argus.com">`
- **THEN** Widget reads data attributes from the script tag, registers `<argus-widget>` custom element, auto-creates element after the script tag, Shadow DOM container opens, and diagnostic input panel renders

#### Scenario: Missing API key
- **WHEN** script tag is loaded without data-api-key attribute
- **THEN** Widget renders an error message inside its Shadow DOM container: "Missing API Key configuration"

#### Scenario: Missing base URL
- **WHEN** script tag is loaded without data-base-url attribute
- **THEN** Widget defaults to relative path `/api/v1/` (same-origin usage)

### Requirement: Widget CSS isolation via Shadow DOM (Open Mode)
The system SHALL render Widget UI inside a Shadow DOM (Open Mode) using Vue's `.ce.vue` convention. Tailwind 4 CSS SHALL be compiled into the Shadow Root via SFC `<style>` blocks (automatic with `defineCustomElement`). Host page styles SHALL NOT affect Widget, and Widget styles SHALL NOT leak to host page.

#### Scenario: No style leakage
- **WHEN** Widget is embedded in a page with its own Tailwind/Bootstrap/custom CSS
- **THEN** Widget styles do not affect host page, and host page styles do not affect Widget
- **THEN** Widget uses its own Glassmorphism design (backdrop-blur, semi-transparent backgrounds)

### Requirement: Widget three-state UI
The Widget SHALL present three distinct visual states with smooth transitions:

#### State 1: Input (idle)
- Glassmorphism container (`backdrop-blur-md bg-white/10 border-white/20 rounded-2xl`)
- Text input with placeholder "Describe the issue..."
- Gradient send button (`bg-gradient-to-r from-indigo-500 to-purple-500`)
- Compact layout: max-width 400px

#### State 2: Inference (diagnosing)
- Input collapses to header with input text summary
- Vertical timeline with streaming step cards
- Each step card shows Think/Act/Observe with color coding (Indigo/Amber/Emerald)
- Active step has shimmer scan animation (`animate-shimmer`)
- Pulse dot on header indicating processing

#### State 3: Result (completed/failed)
- Root cause card with confidence progress bar (conic-gradient)
- Confidence-based title coloring: >90% gold, 60-90% blue, <60% orange
- Recovery suggestions in collapsible panel
- "Retry" button to return to input state

### Requirement: Widget cross-origin communication
The Widget SHALL communicate with Argus API over HTTPS using fetch (REST) and EventSource (SSE). SSE authentication SHALL use stream_token obtained via REST API. Widget SHALL handle errors gracefully with user-friendly messages.

#### Scenario: Widget triggers diagnosis
- **WHEN** user enters error description in Widget and clicks diagnose
- **THEN** Widget sends POST /api/v1/diagnose with Authorization: Bearer header
- **THEN** Widget obtains task_id from response
- **THEN** Widget requests POST /api/v1/stream-tokens with task_id
- **THEN** Widget opens EventSource to /api/v1/stream/{task_id}?stream_token={token}

#### Scenario: Widget SSE real-time updates
- **WHEN** SSE connection is established with valid stream_token
- **THEN** Widget receives and renders diagnostic steps (Think/Act/Observe) in real-time with entry animations

#### Scenario: Widget SSE reconnection
- **WHEN** SSE connection drops during diagnosis
- **THEN** Widget attempts to obtain a new stream_token and reconnect (max 3 retries)
- **THEN** On final failure, Widget shows error message with manual retry button

#### Scenario: Widget CORS error
- **WHEN** Widget's host origin is not in tenant's allowed_origins list
- **THEN** Browser blocks the request, Widget displays "Origin not authorized" error

### Requirement: Widget bundle size
The Widget JavaScript + CSS bundle SHALL be less than 200KB gzip for first load. Vue 3 runtime is included in the bundle (~45KB gzip). Total estimated: ~100KB gzip.

#### Scenario: Bundle size check
- **WHEN** Widget is built with `npm run build:widget`
- **THEN** output gzip size of all widget assets combined SHALL be under 200KB
- **THEN** Build script SHALL fail if size exceeds limit

### Requirement: Widget component tree
```
src/widget/
├── widget-main.ts               # Entry: register custom element + auto-mount
├── ArgusWidget.ce.vue            # Root (.ce.vue triggers Shadow DOM)
├── components/
│   ├── WidgetHeader.vue          # Title bar + status indicator (pulse animation)
│   ├── DiagnoseInput.vue         # Glass-morphism input + gradient send button
│   ├── InferenceStream.vue       # Streaming step timeline
│   │   └── MiniStepCard.vue      # Compact step card (Think/Act/Observe color tags)
│   └── ResultCard.vue            # Root cause + confidence + recovery suggestions
├── composables/
│   └── useWidgetApi.ts           # REST + stream_token + SSE communication
└── styles/
    └── widget.css                # Tailwind 4 entry (scanned only widget dir)
```
