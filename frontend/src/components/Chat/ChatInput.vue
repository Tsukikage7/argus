<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

const emit = defineEmits<{
  (e: 'send', content: string): void
}>()

const props = defineProps<{
  disabled?: boolean
}>()

const input = ref('')
const textarea = ref<HTMLTextAreaElement | null>(null)

// 自动调整高度
function autoResize() {
  nextTick(() => {
    if (textarea.value) {
      textarea.value.style.height = 'auto'
      textarea.value.style.height = Math.min(textarea.value.scrollHeight, 200) + 'px'
    }
  })
}

watch(input, autoResize)

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

function send() {
  const content = input.value.trim()
  if (!content || props.disabled) return
  emit('send', content)
  input.value = ''
  // 重置高度
  nextTick(() => {
    if (textarea.value) {
      textarea.value.style.height = 'auto'
    }
  })
}
</script>

<template>
  <div class="border-t border-base-300/50 px-4 py-3">
    <div class="max-w-3xl mx-auto flex items-end gap-3">
      <div class="flex-1 glass-card rounded-xl px-4 py-2.5">
        <textarea
          ref="textarea"
          v-model="input"
          :disabled="disabled"
          rows="1"
          placeholder="描述你遇到的问题，例如：最近 payment-service 出现大量 504 错误"
          class="w-full bg-transparent text-sm text-base-content/80 placeholder-base-content/30
                 resize-none focus:outline-none leading-relaxed"
          style="max-height: 200px"
          @keydown="handleKeydown"
        ></textarea>
      </div>
      <button
        class="shrink-0 w-10 h-10 rounded-xl flex items-center justify-center transition-all"
        :class="input.trim() && !disabled
          ? 'bg-indigo-500 text-white cursor-pointer hover:brightness-110'
          : 'bg-base-300/30 text-base-content/20 cursor-not-allowed'"
        :disabled="!input.trim() || disabled"
        @click="send"
      >
        <svg v-if="!disabled" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 19V5m0 0l-7 7m7-7l7 7" />
        </svg>
        <!-- 发送中 loading -->
        <svg v-else class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
        </svg>
      </button>
    </div>
  </div>
</template>
