<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'

const store = useTaskStore()

const scrollContainer = ref<HTMLElement | null>(null)

watch(
  () => store.timeline.length,
  async () => {
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  }
)

function formatTime(date: Date): string {
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const dotColor: Record<string, string> = {
  error: 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.6)]',
  warn: 'bg-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.5)]',
  info: 'bg-blue-500',
  ok: 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.5)]',
}
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden flex flex-col">
    <div class="px-4 py-2.5 border-b border-base-300/50 flex items-center gap-1.5
                text-[0.8125rem] font-semibold text-base-content/70">
      <svg class="w-3.5 h-3.5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"/>
      </svg>
      事件时间线
    </div>

    <div ref="scrollContainer" class="p-3 overflow-y-auto scroller" style="max-height: 30vh">
      <div v-if="store.timeline.length === 0" class="text-center text-xs py-6 text-base-content/30">
        暂无事件
      </div>

      <!-- 垂直时间轴 -->
      <div v-else class="relative pl-5">
        <!-- 连线 -->
        <div class="absolute left-[5px] top-1 bottom-1 w-[2px] bg-base-300/40"></div>

        <div
          v-for="(item, i) in store.timeline"
          :key="i"
          class="relative flex items-start gap-3 pb-3 last:pb-0 animate-[fadeIn_0.3s_ease_both]"
        >
          <!-- 圆点 -->
          <div class="absolute -left-5 top-0.5 w-[12px] h-[12px] rounded-full border-2 border-base-100 flex items-center justify-center">
            <div class="w-[8px] h-[8px] rounded-full" :class="dotColor[item.level] || 'bg-base-content/30'"></div>
          </div>

          <!-- 内容 -->
          <div class="flex-1 min-w-0">
            <div class="text-xs text-base-content/70 leading-relaxed">{{ item.text }}</div>
          </div>

          <!-- 时间戳 -->
          <span class="text-[0.6rem] text-base-content/25 flex-shrink-0 tabular-nums mt-0.5">
            {{ formatTime(item.time) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
