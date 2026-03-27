import type {
  Task,
  Scenario,
  ReplaySession,
  ReplayType,
  ReplayConfig,
  TopologyGraph,
  TraceListItem,
  TraceDetail,
  AlertsActiveResponse,
  ChatSession,
  ChatMessage,
} from '@/types'

export type { Scenario } from '@/types'

const API_KEY = 'argus-demo-key'

function headers(): HeadersInit {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${API_KEY}`,
  }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: { ...headers(), ...options?.headers },
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`API error ${res.status}: ${body}`)
  }
  // 204 No Content 无 body，直接返回
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

/** 通用 API 请求（供外部模块使用） */
export async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  return request<T>(url, options)
}

/** 链路追踪列表 */
export interface TraceListResult {
  total: number
  traces: TraceListItem[]
}
export async function queryTraces(opts: {
  service?: string
  keyword?: string
  time_range?: string
  limit?: number
}): Promise<TraceListResult> {
  const params = new URLSearchParams()
  if (opts.service) params.set('service', opts.service)
  if (opts.keyword) params.set('request_uuid', opts.keyword)
  if (opts.time_range) params.set('time_range', opts.time_range)
  if (opts.limit) params.set('limit', String(opts.limit))
  return request(`/api/v1/traces?${params}`)
}

/** 链路详情 */
export async function getTraceDetail(uuid: string): Promise<TraceDetail> {
  return request(`/api/v1/traces/${uuid}`)
}

/** 火焰图数据（mock） */
export interface FlameGraphResponse {
  root: import('@/types').FlameNode
}
export async function getTraceFlameGraph(uuid: string): Promise<FlameGraphResponse> {
  return request(`/api/v1/traces/${uuid}/flamegraph`)
}

/** 触发诊断，返回 task_id */
export async function startDiagnose(
  input: string,
  context?: { time_range?: string; namespaces?: string[] },
): Promise<{ task_id: string; stream_token: string }> {
  return request('/api/v1/diagnose', {
    method: 'POST',
    body: JSON.stringify({ input, source: 'web', context }),
  })
}

/** 查询任务详情 */
export async function getTask(taskId: string): Promise<Task> {
  return request(`/api/v1/tasks/${taskId}`)
}

/** 查询历史任务列表 */
export async function listTasks(): Promise<Task[]> {
  return request('/api/v1/tasks')
}

/** 获取可用故障场景 */
export async function listScenarios(): Promise<Scenario[]> {
  return request('/api/v1/scenarios')
}

/** 手动创建沉淀场景 */
export async function createScenario(params: {
  name: string
  description: string
  root_cause?: string
  log_patterns?: string[]
  affected_namespaces?: string[]
}): Promise<Scenario> {
  return request('/api/v1/scenarios', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

/** 创建回放会话 */
export async function startReplay(params: {
  type: ReplayType
  scenario: string
  config: Partial<ReplayConfig>
}): Promise<{ session_id: string; status: string; stream_token?: string }> {
  return request('/api/v1/replay', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

/** 查询回放会话详情 */
export async function getReplaySession(sessionId: string): Promise<ReplaySession> {
  return request(`/api/v1/replay/${sessionId}`)
}

/** 查询回放历史列表 */
export async function listReplaySessions(): Promise<ReplaySession[]> {
  return request('/api/v1/replay')
}

/** 触发恢复 */
export async function triggerRecover(taskId: string): Promise<unknown> {
  return request(`/api/v1/tasks/${taskId}/recover`, { method: 'POST' })
}

/** 日志聚合摘要 */
export interface LogSummaryBucket {
  namespace: string
  level: string
  count: number
}
export interface LogSummary {
  buckets: LogSummaryBucket[]
  total: number
}
export async function getLogSummary(timeRange?: string): Promise<LogSummary> {
  const params = new URLSearchParams()
  if (timeRange) params.set('time_range', timeRange)
  return request(`/api/v1/logs/summary?${params}`)
}

/** 日志条件查询 */
export interface LogEntry {
  '@timestamp': string
  message: string
  kubernetes_namespace: string
  kubernetes_labels_app?: string
  kubernetes_pod: string
  kubernetes_node: string
  kubernetes_container: string
  host: string
  stream: string
}
export async function queryLogs(opts: {
  namespace?: string
  keyword?: string
  level?: string
  time_range?: string
  limit?: number
}): Promise<LogEntry[]> {
  const params = new URLSearchParams()
  if (opts.namespace) params.set('namespace', opts.namespace)
  if (opts.keyword) params.set('keyword', opts.keyword)
  if (opts.level) params.set('level', opts.level)
  if (opts.time_range) params.set('time_range', opts.time_range)
  if (opts.limit) params.set('limit', String(opts.limit))
  return request(`/api/v1/logs?${params}`)
}

/** 日志分面聚合 */
export interface FacetBucket {
  name: string
  count: number
}
export interface LogFacets {
  namespaces: FacetBucket[]
  services: FacetBucket[]
  levels: FacetBucket[]
  pods: FacetBucket[]
}
export async function getLogFacets(timeRange?: string): Promise<LogFacets> {
  const params = new URLSearchParams()
  if (timeRange) params.set('time_range', timeRange)
  return request(`/api/v1/logs/facets?${params}`)
}

/** 故障日志查询 */
export interface FaultLogEntry {
  id: string
  timestamp: string
  level: string
  service: string
  message: string
  request_uuid?: string
  namespace: string
  pod?: string
}
export interface FaultLogResult {
  total: number
  logs: FaultLogEntry[]
}
export async function queryFaultLogs(opts: {
  namespace?: string
  service?: string
  keyword?: string
  level?: string
  time_range?: string
  limit?: number
}): Promise<FaultLogResult> {
  const params = new URLSearchParams()
  if (opts.namespace) params.set('namespace', opts.namespace)
  if (opts.service) params.set('service', opts.service)
  if (opts.keyword) params.set('keyword', opts.keyword)
  if (opts.level) params.set('level', opts.level)
  if (opts.time_range) params.set('time_range', opts.time_range)
  if (opts.limit) params.set('limit', String(opts.limit))
  return request(`/api/v1/logs/faults?${params}`)
}

/** 日志上下文查询 */
export interface LogContextResult {
  request_uuid: string
  logs: FaultLogEntry[]
}
export async function getLogContext(requestUUID: string, timeRange?: string): Promise<LogContextResult> {
  const params = new URLSearchParams()
  params.set('request_uuid', requestUUID)
  if (timeRange) params.set('time_range', timeRange)
  return request(`/api/v1/logs/context?${params}`)
}

/** Dashboard 总览数据 */
export interface DashboardSummary {
  total_services: number
  active_alerts: number
  today_diagnoses: number
  avg_diagnose_time_seconds: number
  service_health: Array<{
    name: string
    status: 'healthy' | 'degraded' | 'critical'
    error_rate: number
    p99_latency_ms: number
  }>
  recent_alerts: Array<{
    id: string
    severity: 'critical' | 'warning' | 'info'
    service: string
    message: string
    time: string
  }>
  recent_diagnoses: Array<{
    task_id: string
    status: string
    root_cause: string
    duration_seconds: number
    confidence: number
  }>
}

export async function getDashboardSummary(): Promise<DashboardSummary> {
  return request('/api/v1/dashboard/summary')
}

/** 获取拓扑全量数据 */
export async function getTopologyGraph(): Promise<TopologyGraph> {
  return request('/api/v1/topology/graph')
}

/** 活跃告警列表（mock） */
export async function getActiveAlerts(): Promise<AlertsActiveResponse> {
  return request('/api/v1/alerts/active')
}

/** 效率统计数据（mock） */
export interface EfficiencyStats {
  ai_avg_time_seconds: number
  manual_avg_time_seconds: number
  ai_avg_steps: number
  manual_avg_steps: number
  ai_accuracy: number
  scenarios_covered: number
  total_diagnoses: number
  time_saved_hours: number
}
export async function getEfficiencyStats(): Promise<EfficiencyStats> {
  return request('/api/v1/stats/efficiency')
}

export function useApi() {
  return {
    startDiagnose,
    getTask,
    listTasks,
    listScenarios,
    createScenario,
    startReplay,
    getReplaySession,
    listReplaySessions,
    triggerRecover,
    getLogSummary,
    queryLogs,
    getDashboardSummary,
    queryTraces,
    getTraceDetail,
    getTraceFlameGraph,
    getActiveAlerts,
  }
}

// ── 聊天 API ──────────────────────────────────────────────────────────

export async function createChatSession(params?: { title?: string; source?: string }): Promise<ChatSession> {
  return request('/api/v1/chat/sessions', {
    method: 'POST',
    body: JSON.stringify(params || { source: 'web' }),
  })
}

export async function listChatSessions(limit = 20, offset = 0): Promise<ChatSession[]> {
  return request(`/api/v1/chat/sessions?limit=${limit}&offset=${offset}`)
}

export async function getChatSession(id: string): Promise<ChatSession> {
  return request(`/api/v1/chat/sessions/${id}`)
}

export async function updateChatSession(id: string, params: { title?: string; archived?: boolean }): Promise<void> {
  await request(`/api/v1/chat/sessions/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(params),
  })
}

export async function deleteChatSession(id: string): Promise<void> {
  await request(`/api/v1/chat/sessions/${id}`, { method: 'DELETE' })
}

export async function sendChatMessage(sessionId: string, content: string): Promise<{
  session_id: string
  message_id: string
  run_id: string
  stream_token: string
}> {
  return request(`/api/v1/chat/sessions/${sessionId}/messages`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export async function listChatMessages(sessionId: string, cursor?: string, limit = 50): Promise<ChatMessage[]> {
  const params = new URLSearchParams()
  if (cursor) params.set('cursor', cursor)
  params.set('limit', String(limit))
  return request(`/api/v1/chat/sessions/${sessionId}/messages?${params}`)
}
