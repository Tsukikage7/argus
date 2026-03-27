<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Step } from '@/types'
import StepCard from '@/components/Inference/StepCard.vue'

const props = defineProps<{
  content: string
  steps?: Step[]
  isStreaming?: boolean
}>()

const showSteps = ref(false)

const hasSteps = computed(() => props.steps && props.steps.length > 0)

// 简易 Markdown 渲染
function renderMarkdown(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong class="text-base-content/90">$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-base-300/50 text-[0.75rem] font-mono">$1</code>')
    .replace(/\n/g, '<br/>')
}
</script>

<template>
  <div class="max-w-[90%]">
    <!-- 推理步骤折叠区 -->
    <div v-if="hasSteps" class="mb-2">
      <button
        class="flex items-center gap-1.5 text-xs text-base-content/40 hover:text-base-content/60
               transition-colors cursor-pointer"
        @click="showSteps = !showSteps"
      >
        <svg
          class="w-3 h-3 transition-transform"
          :class="{ 'rotate-90': showSteps }"
          fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
        </svg>
        <span v-if="isStreaming" class="flex items-center gap-1">
          推理中
          <span class="inline-flex gap-0.5">
            <span class="w-1 h-1 rounded-full bg-indigo-400 animate-[pdot_1.4s_ease_infinite]"></span>
            <span class="w-1 h-1 rounded-full bg-indigo-400 animate-[pdot_1.4s_0.2s_ease_infinite]"></span>
            <span class="w-1 h-1 rounded-full bg-indigo-400 animate-[pdot_1.4s_0.4s_ease_infinite]"></span>
          </span>
          ({{ steps!.length }} 步)
        </span>
        <span v-else>查看推理过程 ({{ steps!.length }} 步)</span>
      </button>

      <div v-if="showSteps" class="mt-2 space-y-2">
        <StepCard
          v-for="(step, i) in steps"
          :key="i"
          :step="step"
          :is-latest="isStreaming && i === steps!.length - 1"
        />
      </div>
    </div>

    <!-- 文本内容 -->
    <div
      v-if="content"
      class="glass-card rounded-2xl rounded-tl-sm px-4 py-3 text-sm leading-relaxed text-base-content/80"
    >
      <div v-html="renderMarkdown(content)"></div>
      <!-- 流式光标 -->
      <span
        v-if="isStreaming"
        class="inline-block w-2 h-4 bg-indigo-400/60 animate-pulse ml-0.5 align-middle"
      ></span>
    </div>

    <!-- 空内容但正在流式（仅显示光标） -->
    <div
      v-else-if="isStreaming && !hasSteps"
      class="glass-card rounded-2xl rounded-tl-sm px-4 py-3"
    >
      <span class="inline-block w-2 h-4 bg-indigo-400/60 animate-pulse"></span>
    </div>
  </div>
</template>
