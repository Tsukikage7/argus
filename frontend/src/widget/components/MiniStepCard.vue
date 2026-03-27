<script setup lang="ts">
import type { Step } from '../composables/types'

defineProps<{ step: Step }>()

const typeConfig: Record<string, { label: string; color: string; bg: string }> = {
  think: { label: 'Think', color: 'text-indigo-300', bg: 'bg-indigo-500/20' },
  act: { label: 'Act', color: 'text-amber-300', bg: 'bg-amber-500/20' },
  observe: { label: 'Observe', color: 'text-emerald-300', bg: 'bg-emerald-500/20' },
}
</script>

<template>
  <div class="flex gap-2">
    <span
      :class="[typeConfig[step.type]?.bg, typeConfig[step.type]?.color]"
      class="mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium"
    >
      {{ typeConfig[step.type]?.label ?? step.type }}
    </span>
    <div class="min-w-0 flex-1">
      <p class="text-xs leading-relaxed text-white/70">{{ step.content }}</p>
      <p v-if="step.tool_name" class="mt-0.5 text-[10px] text-white/30">
        {{ step.tool_name }}
      </p>
    </div>
  </div>
</template>
