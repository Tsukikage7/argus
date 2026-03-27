<script setup lang="ts">
import { useTaskStore } from '@/store/useTaskStore'
import type { Task } from '@/types'

const store = useTaskStore()

const emit = defineEmits<{
  (e: 'replayTask', taskId: string): void
}>()

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function rootCauseSummary(task: Task): string {
  if (!task.diagnosis?.root_cause) return '诊断中…'
  const rc = task.diagnosis.root_cause
  return rc.length <= 40 ? rc : rc.slice(0, 40) + '…'
}

function statusBadge(task: Task): { text: string; cls: string } {
  if (task.status === 'completed' || task.status === 'recovered') {
    return { text: '完成', cls: 'bg-emerald-500/15 text-emerald-400' }
  }
  if (task.status === 'failed') {
    return { text: '失败', cls: 'bg-red-500/15 text-red-400' }
  }
  return { text: '进行中', cls: 'bg-amber-500/15 text-amber-400' }
}

function confidenceText(task: Task): string {
  if (task.diagnosis) return Math.round(task.diagnosis.confidence * 100) + '%'
  return '—'
}
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden">
    <div class="px-4 py-2.5 border-b border-base-300/50 flex items-center gap-1.5
                text-[0.8125rem] font-semibold text-base-content/70">
      <svg class="w-3.5 h-3.5 text-base-content/30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      最近诊断
    </div>

    <div class="p-2.5 max-h-52 overflow-y-auto scroller space-y-2">
      <div v-if="store.history.length === 0" class="text-center text-xs py-4 text-base-content/30">
        暂无记录
      </div>

      <!-- 卡片式历史记录 -->
      <div
        v-for="task in store.history"
        :key="task.id"
        class="glass-card rounded-lg p-2.5 cursor-pointer transition-all duration-200
               hover:-translate-y-0.5 hover:shadow-md hover:border-indigo-500/30 group"
        @click="emit('replayTask', task.id)"
      >
        <!-- 顶行：状态标签 + 时间 -->
        <div class="flex items-center justify-between mb-1">
          <span class="inline-block px-1.5 py-0.5 rounded text-[0.5625rem] font-semibold"
                :class="statusBadge(task).cls">
            {{ statusBadge(task).text }}
          </span>
          <span class="text-[0.6rem] text-base-content/25 tabular-nums">{{ formatTime(task.created_at) }}</span>
        </div>
        <!-- 根因摘要 -->
        <div class="text-xs text-base-content/60 leading-relaxed group-hover:text-base-content/80 transition-colors">
          {{ rootCauseSummary(task) }}
        </div>
        <!-- 底行：置信度 -->
        <div class="mt-1 text-[0.6rem] text-base-content/30">
          置信度 {{ confidenceText(task) }}
        </div>
      </div>
    </div>
  </div>
</template>
