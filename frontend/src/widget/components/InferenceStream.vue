<script setup lang="ts">
import type { Step } from '../composables/types'
import MiniStepCard from './MiniStepCard.vue'

defineProps<{
  steps: Step[]
  isActive: boolean
}>()
</script>

<template>
  <div class="flex-1 overflow-y-auto px-4 py-2">
    <div class="relative space-y-3 pl-4">
      <!-- 时间线竖线 -->
      <div class="absolute left-[3px] top-2 bottom-2 w-px bg-gradient-to-b from-emerald-400/60 to-transparent" />

      <div
        v-for="(step, i) in steps"
        :key="i"
        class="relative animate-[fadeSlideIn_0.3s_ease-out]"
      >
        <!-- 时间线圆点 -->
        <div
          class="absolute -left-4 top-1.5 size-[7px] rounded-full"
          :class="step.type === 'think' ? 'bg-indigo-400' : step.type === 'act' ? 'bg-amber-400' : 'bg-emerald-400'"
        />
        <MiniStepCard :step="step" />
      </div>

      <!-- 处理中指示器 -->
      <div v-if="isActive" class="relative flex items-center gap-2 py-1">
        <div class="absolute -left-4 top-1/2 -translate-y-1/2 size-[7px] rounded-full bg-white/40 animate-pulse" />
        <div class="h-1 flex-1 overflow-hidden rounded-full bg-white/5">
          <div class="h-full w-1/3 animate-[shimmer_1.5s_infinite] rounded-full bg-gradient-to-r from-transparent via-white/20 to-transparent" />
        </div>
      </div>
    </div>
  </div>
</template>
