import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatSession, ChatMessage, Step, Diagnosis } from '@/types'
import {
  createChatSession,
  listChatSessions,
  deleteChatSession,
  updateChatSession,
  sendChatMessage,
  listChatMessages,
} from '@/composables/useApi'

export const useChatStore = defineStore('chat', () => {
  // 会话列表
  const sessions = ref<ChatSession[]>([])
  const currentSessionId = ref<string | null>(null)

  // 消息缓存: sessionId -> messages
  const messagesMap = ref<Record<string, ChatMessage[]>>({})

  // 流式状态
  const streamingContent = ref('')
  const streamingSteps = ref<Step[]>([])
  const streamingDiagnosis = ref<Diagnosis | null>(null)
  const isStreaming = ref(false)

  // 计算属性
  const currentSession = computed(() =>
    sessions.value.find(s => s.id === currentSessionId.value) ?? null
  )
  const currentMessages = computed(() =>
    currentSessionId.value ? (messagesMap.value[currentSessionId.value] ?? []) : []
  )

  // 加载会话列表
  async function loadSessions() {
    sessions.value = await listChatSessions()
  }

  // 创建新会话
  async function createSession(title?: string): Promise<string> {
    const session = await createChatSession({ title, source: 'web' })
    sessions.value.unshift(session)
    currentSessionId.value = session.id
    messagesMap.value[session.id] = []
    return session.id
  }

  // 切换会话
  async function switchSession(sessionId: string) {
    currentSessionId.value = sessionId
    if (!messagesMap.value[sessionId]) {
      const msgs = await listChatMessages(sessionId)
      messagesMap.value[sessionId] = msgs
    }
  }
  // 删除会话
  async function removeSession(sessionId: string) {
    await deleteChatSession(sessionId)
    sessions.value = sessions.value.filter(s => s.id !== sessionId)
    delete messagesMap.value[sessionId]
    if (currentSessionId.value === sessionId) {
      currentSessionId.value = sessions.value[0]?.id ?? null
    }
  }

  // 重命名会话
  async function renameSession(sessionId: string, title: string) {
    await updateChatSession(sessionId, { title })
    const s = sessions.value.find(s => s.id === sessionId)
    if (s) s.title = title
  }

  // 发送消息
  async function sendMessage(content: string): Promise<{ runId: string; streamToken: string } | null> {
    if (!currentSessionId.value) return null

    // 添加用户消息到本地
    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      session_id: currentSessionId.value,
      role: 'user',
      content,
      status: 'completed',
      created_at: new Date().toISOString(),
    }
    if (!messagesMap.value[currentSessionId.value]) {
      messagesMap.value[currentSessionId.value] = []
    }
    messagesMap.value[currentSessionId.value].push(userMsg)

    // 发送到后端
    const result = await sendChatMessage(currentSessionId.value, content)

    // 初始化流式状态
    isStreaming.value = true
    streamingContent.value = ''
    streamingSteps.value = []
    streamingDiagnosis.value = null

    return { runId: result.run_id, streamToken: result.stream_token }
  }

  // 流式事件处理
  function handleStreamEvent(type: string, data: unknown) {
    switch (type) {
      case 'reasoning.think':
      case 'reasoning.act':
      case 'reasoning.observe': {
        const step = data as Step
        streamingSteps.value.push(step)
        break
      }
      case 'message.delta': {
        const delta = data as { content: string }
        streamingContent.value += delta.content
        break
      }
      case 'message.completed': {
        const msg = data as ChatMessage
        if (currentSessionId.value && messagesMap.value[currentSessionId.value]) {
          messagesMap.value[currentSessionId.value].push({ ...msg })
        }
        isStreaming.value = false
        streamingContent.value = ''
        streamingSteps.value = []
        break
      }
      case 'run.completed':
      case 'run.failed':
        isStreaming.value = false
        break
    }
  }

  // 完成流式消息（SSE 结束时调用）
  function finalizeStream() {
    if (!currentSessionId.value) return
    if (streamingContent.value || streamingSteps.value.length > 0) {
      const assistantMsg: ChatMessage = {
        id: crypto.randomUUID(),
        session_id: currentSessionId.value,
        role: 'assistant',
        content: streamingContent.value,
        status: 'completed',
        created_at: new Date().toISOString(),
      }
      if (!messagesMap.value[currentSessionId.value]) {
        messagesMap.value[currentSessionId.value] = []
      }
      messagesMap.value[currentSessionId.value].push(assistantMsg)
    }
    isStreaming.value = false
    streamingContent.value = ''
    streamingSteps.value = []
    streamingDiagnosis.value = null
  }

  return {
    sessions,
    currentSessionId,
    messagesMap,
    streamingContent,
    streamingSteps,
    streamingDiagnosis,
    isStreaming,
    currentSession,
    currentMessages,
    loadSessions,
    createSession,
    switchSession,
    removeSession,
    renameSession,
    sendMessage,
    handleStreamEvent,
    finalizeStream,
  }
}, {
  persist: {
    pick: ['currentSessionId'],
  },
})
