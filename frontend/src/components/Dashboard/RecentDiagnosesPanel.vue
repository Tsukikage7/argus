<script setup lang="ts">
export interface RecentDiagnoseItem {
  task_id: string
  status: string
  root_cause: string
  duration_seconds: number
  confidence: number
}

defineProps<{
  diagnoses: RecentDiagnoseItem[]
}>()

const statusConfig: Record<string, { badge: string; label: string }> = {
  completed: { badge: 'badge-success', label: '完成' },
  running:   { badge: 'badge-info', label: '运行中' },
  failed:    { badge: 'badge-error', label: '失败' },
  pending:   { badge: 'badge-ghost', label: '等待' },
}
</script>

<template>
  <div>
    <h3 class="text-sm font-semibold text-base-content/70 mb-3">最近诊断</h3>
    <div class="space-y-2">
      <router-link
        v-for="d in diagnoses.slice(0, 5)"
        :key="d.task_id"
        :to="`/tasks/${d.task_id}`"
        class="block glass-card rounded-lg p-3 hover:shadow-md transition-all cursor-pointer"
      >
        <div class="flex items-center justify-between mb-1">
          <span class="badge badge-sm" :class="statusConfig[d.status]?.badge || 'badge-ghost'">
            {{ statusConfig[d.status]?.label || d.status }}
          </span>
          <span v-if="d.duration_seconds > 0" class="text-xs text-base-content/40 font-mono">
            {{ d.duration_seconds }}s
          </span>
        </div>
        <p v-if="d.root_cause" class="text-xs text-base-content/60 truncate">
          {{ d.root_cause }}
        </p>
        <p v-else class="text-xs text-base-content/30 italic">诊断进行中...</p>
        <div v-if="d.confidence > 0" class="mt-1.5">
          <div class="flex items-center gap-2">
            <div class="flex-1 h-1 bg-base-300 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all"
                :class="d.confidence >= 0.8 ? 'bg-success' : d.confidence >= 0.5 ? 'bg-warning' : 'bg-error'"
                :style="{ width: `${d.confidence * 100}%` }"
              />
            </div>
            <span class="text-[0.625rem] text-base-content/40 font-mono">
              {{ (d.confidence * 100).toFixed(0) }}%
            </span>
          </div>
        </div>
      </router-link>
    </div>
  </div>
</template>
