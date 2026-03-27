// 任务状态
export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'recovering' | 'recovered'

// 工具调用
export interface Action {
  tool: string
  params: Record<string, unknown>
}

// Agent 推理步骤
export interface Step {
  index: number
  think: string
  action?: Action
  observe?: string
  is_key_step: boolean
  tool_name?: string
  tool_params?: Record<string, unknown>
  timestamp: string
}

// 诊断结论
export interface Diagnosis {
  root_cause: string
  confidence: number
  affected_services: string[]
  impact: string
  suggestions: string[]
}

// 恢复操作
export interface RecoveryAction {
  description: string
  command?: string
  result: string
  success: boolean
}

export type RecoveryStatus = 'pending' | 'executing' | 'success' | 'failed' | 'skipped'

export interface Recovery {
  actions: RecoveryAction[]
  status: RecoveryStatus
  verified_at?: string
}

// 诊断任务
export interface Task {
  id: string
  input: string
  source: string
  status: TaskStatus
  steps: Step[]
  diagnosis?: Diagnosis
  recovery?: Recovery
  created_at: string
  updated_at: string
  completed_at?: string
}

// SSE 事件
export interface TaskEvent {
  task_id: string
  type: 'step' | 'diagnosis' | 'recovery' | 'status'
  data: unknown
}

// 回放相关
export type ReplayType = 'fault' | 'traffic'
export type ReplayStatus = 'pending' | 'generating' | 'diagnosing' | 'completed' | 'failed'

export interface ReplayConfig {
  traffic_rate_multiplier: number
  duration: number
  fault_intensity: number
  fault_delay: number
  auto_diagnose: boolean
}

export interface ServiceImpact {
  name: string
  status: 'healthy' | 'degraded' | 'down'
  error_count: number
  error_rate: number
  avg_latency_ms: number
  p99_latency_ms: number
  is_direct: boolean
}

export interface ImpactReport {
  affected_services: ServiceImpact[]
  blast_radius: 'low' | 'medium' | 'high' | 'critical'
  total_requests: number
  failed_requests: number
  error_rate: Record<string, number>
  latency_impact: Record<string, number>
  time_window: string
  summary: string
}

export interface ReplaySession {
  id: string
  type: ReplayType
  scenario_name: string
  config: ReplayConfig
  status: ReplayStatus
  task_id?: string
  impact_report?: ImpactReport
  error?: string
  logs_written: number
  traces_written: number
  created_at: string
  completed_at?: string
}

// 场景
export interface Scenario {
  name: string
  description: string
  type?: 'preset' | 'captured'
  source_task_id?: string
  confidence?: number
}

// 时间线事件
export interface TimelineEvent {
  level: 'info' | 'warn' | 'error' | 'ok'
  text: string
  time: Date
}

// 应用模式
export type AppMode = 'diagnose' | 'replay'

// 服务拓扑
export interface TopologyConfig {
  services: string[]
  edges: [string, string][]
  chains: Record<string, string[]>
}

export interface TopologyNode {
  id: string
  label: string
  health: 'healthy' | 'degraded' | 'down' | 'critical'
  error_rate: number
  alert_count: number
}

export interface TopologyEdge {
  source: string
  target: string
  weight: number
}

export interface TopologyGraph {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
}

// 链路追踪相关
export interface TraceListItem {
  request_uuid: string
  entry_service: string
  status_code: number
  duration_ms: number
  timestamp: string
  services: string[]
}

export interface TraceSpan {
  service: string
  operation: string
  start_ms: number
  duration_ms: number
  status: 'ok' | 'error' | 'slow'
  logs: Array<{
    timestamp: string
    level: string
    message: string
  }>
}

export interface TraceDetail {
  request_uuid: string
  entry_service: string
  total_duration_ms: number
  timestamp: string
  spans: TraceSpan[]
}

export interface FlameNode {
  name: string
  value: number
  children: FlameNode[]
}

// 告警相关
export type AlertSeverity = 'critical' | 'warning' | 'info'
export type AlertStatus = 'firing' | 'acknowledged' | 'resolved'

export interface AlertEvent {
  id: string
  fingerprint: string
  severity: AlertSeverity
  service: string
  message: string
  description?: string
  status: AlertStatus
  starts_at: string
  task_id?: string
  created_at: string
  resolved_at?: string
}

export interface AlertSeveritySummary {
  critical: number
  warning: number
  info: number
}

export interface AlertsActiveResponse {
  total: number
  alerts: AlertEvent[]
  summary: AlertSeveritySummary
}

// ── 聊天系统类型 ──────────────────────────────────────────────────────
export type ChatSessionStatus = 'active' | 'archived' | 'deleted'
export type ChatMessageRole = 'user' | 'assistant' | 'system' | 'tool'
export type ChatMessageStatus = 'pending' | 'streaming' | 'completed' | 'failed'
export type ChatRunStatus = 'pending' | 'running' | 'completed' | 'failed'
export type ChatArtifactType = 'diagnosis' | 'log_table' | 'trace_graph'

export interface ChatSession {
  id: string
  tenant_id: string
  title: string
  source: string
  status: ChatSessionStatus
  last_intent?: string
  summary?: string
  working_memory?: Record<string, unknown>
  created_at: string
  updated_at: string
  archived_at?: string
}

export interface ChatMessage {
  id: string
  session_id: string
  role: ChatMessageRole
  content: string
  status: ChatMessageStatus
  run_id?: string
  artifacts?: ChatArtifact[]
  created_at: string
}

export interface ChatRun {
  id: string
  session_id: string
  trigger_message_id: string
  intent?: string
  status: ChatRunStatus
  steps?: Step[]
  started_at?: string
  completed_at?: string
  error_message?: string
}

export interface ChatArtifact {
  id: string
  session_id: string
  run_id?: string
  message_id?: string
  type: ChatArtifactType
  title: string
  payload: Record<string, unknown>
  created_at: string
}

export interface ChatSSEEvent {
  type: string
  data: unknown
  run_id?: string
  session_id?: string
}
