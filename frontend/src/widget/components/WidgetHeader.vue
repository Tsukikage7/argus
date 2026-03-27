<script setup lang="ts">
import type { WidgetStatus } from '../composables/useWidgetApi'

defineProps<{
  status: WidgetStatus
  inputSummary?: string
}>()
</script>

<template>
  <div class="flex items-center justify-between px-4 py-3">
    <div class="flex items-center gap-2">
      <div class="text-base font-semibold text-white/90">Argus</div>
      <div class="text-xs text-white/50">智能诊断</div>
    </div>
    <div class="flex items-center gap-2">
      <span v-if="status === 'idle'" class="size-2 rounded-full bg-white/40" />
      <span v-else-if="status === 'diagnosing'" class="size-2 rounded-full bg-emerald-400 animate-pulse" />
      <span v-else-if="status === 'completed'" class="size-2 rounded-full bg-blue-400" />
      <span v-else class="size-2 rounded-full bg-red-400" />
      <span class="text-xs text-white/50">
        {{ status === 'idle' ? '就绪' : status === 'diagnosing' ? '诊断中...' : status === 'completed' ? '完成' : '失败' }}
      </span>
    </div>
  </div>
  <div v-if="status !== 'idle' && inputSummary" class="px-4 pb-2">
    <div class="text-xs text-white/40 truncate">{{ inputSummary }}</div>
  </div>
</template>
