<script setup lang="ts">
import { computed } from 'vue'
import type { Step } from '@/types'

const props = defineProps<{
  step: Step
  isLatest?: boolean
}>()

function truncate(s: string, n: number): string {
  return s.length <= n ? s : s.slice(0, n) + '…'
}

function formatParams(params: Record<string, unknown> | undefined): string {
  if (!params) return '{}'
  return JSON.stringify(params, null, 2)
}

// 简易 Markdown 渲染（支持 **bold**、`code`、换行）
function renderMarkdown(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong class="text-base-content/90">$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-base-300/50 text-[0.75rem] font-mono">$1</code>')
    .replace(/\n/g, '<br/>')
}

const thinkHtml = computed(() => {
  if (!props.step.think) return ''
  return renderMarkdown(truncate(props.step.think, 600))
})
</script>

<template>
  <div class="step-cards relative pl-6">
    <!-- 时间线连线 -->
    <div class="absolute left-[7px] top-0 bottom-0 w-[2px] bg-emerald-500/20"
         style="box-shadow: 0 0 6px oklch(0.72 0.17 162 / 0.3)"></div>
    <!-- 时间线圆点 -->
    <div class="absolute left-0 top-3 w-[16px] h-[16px] rounded-full border-2 border-emerald-500/50 bg-base-100 flex items-center justify-center"
         :class="{ 'border-emerald-400 shadow-[0_0_8px_oklch(0.72_0.17_162/0.5)]': isLatest }">
      <div class="w-[6px] h-[6px] rounded-full bg-emerald-500" :class="{ 'animate-pulse': isLatest }"></div>
    </div>

    <!-- Think（Markdown 渲染） -->
    <div
      v-if="props.step.think"
      class="glass-card rounded-[0.625rem] px-3.5 py-3 text-[0.8125rem] leading-relaxed text-base-content/70
             animate-[dropIn_0.4s_ease_both] [&+div]:mt-1.5"
      :class="{ 'shimmer': isLatest && !props.step.action }"
    >
      <span class="inline-block text-[0.625rem] font-bold uppercase tracking-wider
                   px-1.5 py-0.5 rounded-[0.2rem] mb-1 bg-indigo-500/20 text-indigo-400">
        Think
      </span>
      <div v-html="thinkHtml"></div>
    </div>

    <!-- Action -->
    <div
      v-if="props.step.action"
      class="glass-card rounded-[0.625rem] px-3.5 py-3 text-[0.8125rem] leading-relaxed text-base-content/70
             animate-[dropIn_0.4s_0.1s_ease_both] mt-1.5"
      :class="{ 'shimmer': isLatest && !props.step.observe }"
    >
      <span class="inline-block text-[0.625rem] font-bold uppercase tracking-wider
                   px-1.5 py-0.5 rounded-[0.2rem] mb-1 bg-amber-500/20 text-amber-400">
        Action · {{ props.step.action.tool }}
      </span>
      <pre class="bg-base-300/30 px-2.5 py-1.5 rounded text-[0.7rem] mt-1.5 overflow-x-auto font-mono">{{ formatParams(props.step.action.params) }}</pre>
    </div>

    <!-- Observe -->
    <div
      v-if="props.step.observe"
      class="glass-card rounded-[0.625rem] px-3.5 py-3 text-[0.8125rem] leading-relaxed text-base-content/70
             animate-[dropIn_0.4s_0.2s_ease_both] mt-1.5"
    >
      <span class="inline-block text-[0.625rem] font-bold uppercase tracking-wider
                   px-1.5 py-0.5 rounded-[0.2rem] mb-1 bg-emerald-500/20 text-emerald-400">
        Observe
      </span>
      <pre class="bg-base-300/30 px-2.5 py-1.5 rounded text-[0.7rem] mt-1.5 overflow-x-auto whitespace-pre-wrap break-words font-mono">{{ truncate(props.step.observe, 500) }}</pre>
    </div>
  </div>
</template>

<style scoped>
@keyframes dropIn {
  from {
    opacity: 0;
    transform: translateY(-12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
