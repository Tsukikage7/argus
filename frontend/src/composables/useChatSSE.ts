import { ref, onUnmounted } from 'vue'
import { useChatStore } from '@/store/useChatStore'

/** 聊天 SSE 连接管理 */
export function useChatSSE() {
  const isConnected = ref(false)
  let eventSource: EventSource | null = null

  function connect(sessionId: string, runId: string, streamToken: string) {
    disconnect()
    const store = useChatStore()

    const url = `/api/v1/chat/sessions/${sessionId}/stream?run_id=${encodeURIComponent(runId)}&stream_token=${encodeURIComponent(streamToken)}`
    eventSource = new EventSource(url)
    isConnected.value = true

    const eventTypes = [
      'run.started', 'intent.detected',
      'reasoning.think', 'reasoning.act', 'reasoning.observe',
      'artifact.ready', 'message.delta', 'message.completed',
      'run.completed', 'run.failed', 'heartbeat',
    ]

    for (const type of eventTypes) {
      eventSource.addEventListener(type, (e) => {
        try {
          const data = JSON.parse((e as MessageEvent).data)
          store.handleStreamEvent(type, data)

          if (type === 'run.completed' || type === 'run.failed') {
            store.finalizeStream()
            disconnect()
          }
        } catch {
          // 忽略解析错误
        }
      })
    }

    eventSource.onerror = () => {
      store.finalizeStream()
      disconnect()
    }
  }

  function disconnect() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    isConnected.value = false
  }

  onUnmounted(disconnect)

  return { isConnected, connect, disconnect }
}
