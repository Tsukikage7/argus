<script setup lang="ts">
/**
 * TraceTimeline - 链路时间线瀑布图
 */
import { computed } from 'vue'
import type { TraceSpan } from '@/types'

const props = defineProps<{
  spans: TraceSpan[]
  totalDuration: number
}>()

const sortedSpans = computed(() => {
  return [...props.spans].sort((a, b) => a.start_ms - b.start_ms)
})

function getStatusClass(status: string) {
  switch (status) {
    case 'error': return 'bg-error text-error-content'
    case 'slow': return 'bg-warning text-warning-content'
    default: return 'bg-success text-success-content'
  }
}

function getWidth(duration: number) {
  const percentage = (duration / props.totalDuration) * 100
  return `${Math.max(percentage, 0.5)}%` // 保证最小可见宽度
}

function getLeft(start: number) {
  return `${(start / props.totalDuration) * 100}%`
}
</script>

<template>
  <div class="flex flex-col gap-1 p-4 bg-base-200/30 rounded-lg">
    <div class="flex border-b border-base-300 pb-2 text-xs opacity-50 font-mono">
      <div class="w-1/4">Service & Operation</div>
      <div class="w-3/4 relative h-4">
        <div class="absolute left-0">0ms</div>
        <div class="absolute right-0">{{ totalDuration }}ms</div>
        <div class="absolute left-1/2 -translate-x-1/2">{{ totalDuration / 2 }}ms</div>
      </div>
    </div>
    
    <div v-for="(span, index) in sortedSpans" :key="index" class="flex items-center group">
      <!-- 左侧标签 -->
      <div class="w-1/4 pr-4 truncate">
        <span class="text-sm font-bold">{{ span.service }}</span>
        <span class="text-xs opacity-60 ml-2">{{ span.operation }}</span>
      </div>
      
      <!-- 右侧时间轴 -->
      <div class="w-3/4 relative py-2 h-10 border-l border-base-300">
        <div 
          class="absolute h-6 rounded flex items-center px-2 text-[10px] font-bold shadow-sm transition-all group-hover:scale-y-110"
          :class="getStatusClass(span.status)"
          :style="{
            left: getLeft(span.start_ms),
            width: getWidth(span.duration_ms)
          }"
          :title="`${span.service}: ${span.duration_ms}ms`"
        >
          <span class="truncate">{{ span.duration_ms }}ms</span>
        </div>
      </div>
    </div>
  </div>
</template>
