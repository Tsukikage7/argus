import { ref } from 'vue'
import type { Step, Diagnosis } from './types'

export type WidgetStatus = 'idle' | 'diagnosing' | 'completed' | 'failed'

/** Widget API 通信层：诊断 + SSE 流 */
export function useWidgetApi(apiKey: string, baseUrl: string) {
  const status = ref<WidgetStatus>('idle')
  const steps = ref<Step[]>([])
  const diagnosis = ref<Diagnosis | null>(null)
  const error = ref<string>('')
  const taskId = ref<string>('')

  let eventSource: EventSource | null = null

  function headers(): HeadersInit {
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    }
  }

  async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const url = `${baseUrl}${path}`
    const res = await fetch(url, {
      ...options,
      headers: { ...headers(), ...options?.headers },
    })
    if (!res.ok) {
      const body = await res.text()
      throw new Error(`API ${res.status}: ${body}`)
    }
    return res.json() as Promise<T>
  }

  function disconnectSSE() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  }

  function connectSSE(id: string) {
    disconnectSSE()
    const url = `${baseUrl}/api/v1/stream/${id}`
    eventSource = new EventSource(url)

    eventSource.addEventListener('step', (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { data: Step }
      steps.value.push(event.data)
    })

    eventSource.addEventListener('diagnosis', (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { data: Diagnosis }
      diagnosis.value = event.data
    })

    eventSource.addEventListener('status', (e) => {
      const event = JSON.parse((e as MessageEvent).data) as { data: unknown }
      const s = typeof event.data === 'string' ? event.data : String(event.data)
      if (s === 'completed' || s === 'recovered') {
        status.value = 'completed'
        disconnectSSE()
      } else if (s === 'failed') {
        status.value = 'failed'
        disconnectSSE()
      }
    })

    eventSource.onerror = () => {
      disconnectSSE()
      if (status.value === 'diagnosing') {
        status.value = 'failed'
        error.value = '诊断连接中断'
      }
    }
  }

  async function diagnose(input: string) {
    // 重置状态
    status.value = 'diagnosing'
    steps.value = []
    diagnosis.value = null
    error.value = ''
    taskId.value = ''

    try {
      const res = await request<{ task_id: string }>('/api/v1/diagnose', {
        method: 'POST',
        body: JSON.stringify({ input, source: 'widget' }),
      })
      taskId.value = res.task_id
      connectSSE(res.task_id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      status.value = 'failed'
    }
  }

  function reset() {
    disconnectSSE()
    status.value = 'idle'
    steps.value = []
    diagnosis.value = null
    error.value = ''
    taskId.value = ''
  }

  return { status, steps, diagnosis, error, taskId, diagnose, reset, disconnectSSE }
}
