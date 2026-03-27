<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  trend?: 'up' | 'down' | 'flat'
  trendValue?: string
  color?: string
  icon?: string
}>()
</script>

<template>
  <div class="glass-card rounded-xl p-4 transition-all duration-300 hover:-translate-y-0.5 hover:shadow-lg group">
    <div class="flex items-center justify-between mb-2">
      <svg
        v-if="icon"
        class="w-5 h-5 opacity-40 group-hover:opacity-70 transition-opacity"
        :style="{ color: color || '#6366f1' }"
        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" :d="icon" />
      </svg>
      <!-- 趋势箭头 -->
      <span
        v-if="trend && trend !== 'flat'"
        class="text-xs font-medium flex items-center gap-0.5"
        :class="trend === 'up' ? 'text-error' : 'text-success'"
      >
        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path
            stroke-linecap="round" stroke-linejoin="round"
            :d="trend === 'up' ? 'M5 15l7-7 7 7' : 'M19 9l-7 7-7-7'"
          />
        </svg>
        {{ trendValue }}
      </span>
    </div>
    <div class="text-2xl font-bold text-base-content tabular-nums">{{ value }}</div>
    <div class="text-[0.625rem] text-base-content/40 uppercase tracking-wider mt-1">{{ label }}</div>
  </div>
</template>
