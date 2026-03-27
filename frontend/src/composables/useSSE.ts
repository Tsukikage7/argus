import { ref, onUnmounted } from 'vue'
import type { Step, Diagnosis, TaskStatus } from '@/types'

/** 诊断任务 SSE 连接管理 */
export function useTaskSSE() {
  const steps = ref<Step[]>([])
  const diagnosis = ref<Diagnosis | null>(null)
  const status = ref<TaskStatus>('pending')
  const isConnected = ref(false)

  let eventSource: EventSource | null = null

  function connect(taskId: string, streamToken?: string) {
    disconnect()
    steps.value = []
    diagnosis.value = null
    status.value = 'running'

    const url = streamToken
      ? `/api/v1/stream/${taskId}?stream_token=${encodeURIComponent(streamToken)}`
      : `/api/v1/stream/${taskId}`
    eventSource = new EventSource(url)
    isConnected.value = true

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
      status.value = s as TaskStatus
      if (['completed', 'failed', 'recovered'].includes(s)) {
        disconnect()
      }
    })

    eventSource.onerror = () => {
      // SSE 断连时断开连接，状态由轮询兜底恢复
      disconnect()
    }
  }

  function disconnect() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    isConnected.value = false
    steps.value = []
    diagnosis.value = null
    status.value = 'pending'
  }

  onUnmounted(disconnect)

  return { steps, diagnosis, status, isConnected, connect, disconnect }
}

/** 回放 SSE 连接管理 */
export function useReplaySSE() {
  const replayStatus = ref<string>('')
  const progress = ref<string>('')
  const impactData = ref<unknown>(null)
  const error = ref<string>('')
  const isConnected = ref(false)

  let eventSource: EventSource | null = null

  function connect(sessionId: string, streamToken?: string) {
    disconnect()
    replayStatus.value = ''
    progress.value = ''
    impactData.value = null
    error.value = ''

    const url = streamToken
      ? `/api/v1/replay/${sessionId}/stream?stream_token=${encodeURIComponent(streamToken)}`
      : `/api/v1/replay/${sessionId}/stream`
    eventSource = new EventSource(url)
    isConnected.value = true

    eventSource.addEventListener('status', (e) => {
      const d = JSON.parse((e as MessageEvent).data) as { data: unknown }
      replayStatus.value = typeof d.data === 'string' ? d.data : String(d.data)
      if (replayStatus.value === 'completed' || replayStatus.value === 'failed') {
        disconnect()
      }
    })

    eventSource.addEventListener('progress', (e) => {
      const d = JSON.parse((e as MessageEvent).data) as { data: unknown }
      progress.value = String(d.data)
    })

    eventSource.addEventListener('impact', (e) => {
      const d = JSON.parse((e as MessageEvent).data) as { data: unknown }
      if (d.data) impactData.value = d.data
    })

    eventSource.addEventListener('error', (e) => {
      try {
        const d = JSON.parse((e as MessageEvent).data) as { data: unknown }
        error.value = String(d.data)
      } catch {
        // 非 JSON 错误事件
      }
    })

    eventSource.onerror = () => {
      // 回放 SSE 断连时关闭，状态由轮询兜底
      disconnect()
    }
  }

  function disconnect() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    isConnected.value = false
    replayStatus.value = ''
    progress.value = ''
    impactData.value = null
    error.value = ''
  }

  onUnmounted(disconnect)

  return { replayStatus, progress, impactData, error, isConnected, connect, disconnect }
}
