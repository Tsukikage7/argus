/** Widget 内部使用的精简类型 */
export interface Step {
  step: number
  type: 'think' | 'act' | 'observe'
  content: string
  tool_name?: string
  tool_input?: string
  tool_result?: string
  timestamp?: string
}

export interface Diagnosis {
  root_cause: string
  confidence: number
  affected_services: string[]
  recovery_suggestion: string
  summary: string
}
