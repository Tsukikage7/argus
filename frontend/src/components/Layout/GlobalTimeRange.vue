<script setup lang="ts">
import { useTimeRange, type TimeRangePreset } from '@/composables/useTimeRange'

const { activePreset, presets, presetLabels, isCustom, setPreset } = useTimeRange()

function handleSelect(preset: TimeRangePreset) {
  setPreset(preset)
}
</script>

<template>
  <div class="flex items-center gap-1">
    <button
      v-for="p in presets"
      :key="p"
      class="px-2 py-1 rounded text-[0.6875rem] font-medium transition-all"
      :class="!isCustom && activePreset === p
        ? 'bg-indigo-500/20 text-indigo-400 border border-indigo-500/30'
        : 'text-base-content/40 hover:text-base-content/60 border border-transparent'"
      @click="handleSelect(p)"
    >
      {{ presetLabels[p] }}
    </button>
  </div>
</template>
