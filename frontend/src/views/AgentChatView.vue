<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/store/useChatStore'
import { useChatSSE } from '@/composables/useChatSSE'
import ChatSidebar from '@/components/Chat/ChatSidebar.vue'
import ChatWelcome from '@/components/Chat/ChatWelcome.vue'
import ChatMessageList from '@/components/Chat/ChatMessageList.vue'
import ChatInput from '@/components/Chat/ChatInput.vue'

const route = useRoute()
const router = useRouter()
const store = useChatStore()
const chatSSE = useChatSSE()

// 初始化
onMounted(async () => {
  await store.loadSessions()
  // 从路由参数恢复会话
  const sessionId = route.params.sessionId as string | undefined
  if (sessionId) {
    await store.switchSession(sessionId)
  }
  // 从 query.input 自动触发诊断（告警中心跳转）
  const inputFromQuery = route.query.input as string | undefined
  if (inputFromQuery && typeof inputFromQuery === 'string') {
    await handleQuickAction(inputFromQuery)
    router.replace({ query: { ...route.query, input: undefined } })
  }
})

// 发送消息
async function handleSend(content: string) {
  // 如果没有当前会话，先创建一个
  if (!store.currentSessionId) {
    await store.createSession(content.slice(0, 30))
  }
  const result = await store.sendMessage(content)
  if (result) {
    chatSSE.connect(store.currentSessionId!, result.runId, result.streamToken)
  }
}

// 快速模板
async function handleQuickAction(text: string) {
  await store.createSession(text.slice(0, 30))
  await handleSend(text)
}
</script>

<template>
  <div class="flex h-[calc(100vh-3.5rem)] -m-5">
    <!-- 左侧：会话列表 -->
    <div class="w-64 border-r border-base-300/50 bg-base-200/30 shrink-0">
      <ChatSidebar />
    </div>

    <!-- 中间：聊天区域 -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- 无会话时显示欢迎页 + 输入框 -->
      <template v-if="!store.currentSessionId || store.currentMessages.length === 0">
        <ChatWelcome @quick-action="handleQuickAction" />
        <ChatInput :disabled="store.isStreaming" @send="handleSend" />
      </template>

      <!-- 有消息时显示消息列表 + 输入框 -->
      <template v-else>
        <!-- 会话标题栏 -->
        <div class="h-12 flex items-center px-4 border-b border-base-300/50 shrink-0">
          <h2 class="text-sm font-medium text-base-content/70 truncate">
            {{ store.currentSession?.title || '新会话' }}
          </h2>
        </div>

        <ChatMessageList />
        <ChatInput :disabled="store.isStreaming" @send="handleSend" />
      </template>
    </div>
  </div>
</template>
