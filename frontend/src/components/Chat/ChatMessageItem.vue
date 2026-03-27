<script setup lang="ts">
import type { ChatMessage, Step } from '@/types'
import ChatAssistantContent from './ChatAssistantContent.vue'

defineProps<{
  message: ChatMessage
  streamingSteps?: Step[]
  isStreaming?: boolean
}>()
</script>

<template>
  <div class="flex gap-3" :class="message.role === 'user' ? 'flex-row-reverse' : 'flex-row'">
    <!-- 头像 -->
    <div class="shrink-0 mt-1">
      <!-- 用户头像 -->
      <div
        v-if="message.role === 'user'"
        class="w-8 h-8 rounded-full bg-indigo-500/20 flex items-center justify-center"
      >
        <svg class="w-4 h-4 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <!-- AI 头像 -->
      <div
        v-else
        class="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold"
        style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
      >
        A
      </div>
    </div>

    <!-- 消息内容 -->
    <div class="flex-1 min-w-0" :class="message.role === 'user' ? 'flex flex-col items-end' : ''">
      <!-- 用户消息 -->
      <div
        v-if="message.role === 'user'"
        class="inline-block max-w-[80%] px-4 py-2.5 rounded-2xl rounded-tr-sm
               bg-indigo-500/20 text-sm text-base-content/80 leading-relaxed"
      >
        {{ message.content }}
      </div>

      <!-- AI 消息 -->
      <ChatAssistantContent
        v-else
        :content="message.content"
        :steps="streamingSteps"
        :is-streaming="isStreaming"
      />
    </div>
  </div>
</template>
