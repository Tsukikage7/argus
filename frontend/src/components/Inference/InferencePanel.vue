<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'
import StepCard from './StepCard.vue'

const store = useTaskStore()

const scrollContainer = ref<HTMLElement | null>(null)

watch(
  () => store.steps.length,
  async () => {
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  }
)
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden flex flex-col"
       style="min-height: 400px; max-height: 75vh">
    <!-- 卡片标题 -->
    <div class="px-4 py-2.5 border-b border-base-300/50 flex items-center justify-between
                text-[0.8125rem] font-semibold text-base-content/70">
      <div class="flex items-center gap-1.5">
        <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
        </svg>
        AI 推理过程
      </div>
      <span v-if="store.stepCount > 0" class="text-xs text-base-content/30">
        Step {{ store.stepCount }}
      </span>
    </div>

    <!-- 内容区域 -->
    <div ref="scrollContainer" class="flex-1 overflow-y-auto p-3 scroller">
      <!-- 空状态 -->
      <div
        v-if="store.steps.length === 0 && store.taskStatus === 'pending'"
        class="text-center text-sm py-16 text-base-content/30"
      >
        输入告警描述，开始诊断
      </div>

      <!-- 推理中状态 -->
      <div
        v-else-if="store.steps.length === 0 && store.taskStatus === 'running'"
        class="flex items-center justify-center gap-2 text-sm py-8 text-base-content/40"
      >
        <span class="typing flex gap-1">
          <span class="animate-[blink_1.2s_infinite_both]">●</span>
          <span class="animate-[blink_1.2s_0.2s_infinite_both]">●</span>
          <span class="animate-[blink_1.2s_0.4s_infinite_both]">●</span>
        </span>
        Agent 推理中
      </div>

      <!-- 步骤列表 -->
      <div v-else class="space-y-3">
        <StepCard
          v-for="(step, idx) in store.steps"
          :key="step.index"
          :step="step"
          :is-latest="idx === store.steps.length - 1 && store.taskStatus === 'running'"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes blink {
  0%, 80%, 100% { opacity: 0.2; }
  40% { opacity: 1; }
}
</style>
