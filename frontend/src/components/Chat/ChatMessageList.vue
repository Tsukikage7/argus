<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useChatStore } from '@/store/useChatStore'
import ChatMessageItem from './ChatMessageItem.vue'

const store = useChatStore()
const scrollContainer = ref<HTMLElement | null>(null)

// 新消息自动滚动到底部
watch(
  () => [store.currentMessages.length, store.streamingContent],
  async () => {
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  },
)
</script>

<template>
  <div ref="scrollContainer" class="flex-1 overflow-y-auto scroller px-4 py-6">
    <div class="max-w-3xl mx-auto space-y-4">
      <!-- 历史消息 -->
      <ChatMessageItem
        v-for="msg in store.currentMessages"
        :key="msg.id"
        :message="msg"
      />

      <!-- 流式消息（AI 正在回复） -->
      <ChatMessageItem
        v-if="store.isStreaming"
        :message="{
          id: '__streaming__',
          session_id: store.currentSessionId ?? '',
          role: 'assistant',
          content: store.streamingContent,
          status: 'streaming',
          created_at: new Date().toISOString(),
        }"
        :streaming-steps="store.streamingSteps"
        :is-streaming="true"
      />
    </div>
  </div>
</template>
