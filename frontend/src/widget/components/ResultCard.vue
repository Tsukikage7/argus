<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Diagnosis } from '../composables/types'

const props = defineProps<{
  diagnosis: Diagnosis
  error?: string
}>()

const emit = defineEmits<{ retry: [] }>()

const showSuggestions = ref(false)

const confidencePercent = computed(() => Math.round(props.diagnosis.confidence * 100))

const titleColor = computed(() => {
  const p = confidencePercent.value
  if (p >= 90) return 'text-amber-300'
  if (p >= 60) return 'text-blue-300'
  return 'text-orange-300'
})

const ringGradient = computed(() => {
  const deg = (confidencePercent.value / 100) * 360
  return `conic-gradient(from 0deg, #818cf8 0deg, #a78bfa ${deg}deg, rgba(255,255,255,0.05) ${deg}deg)`
})
</script>

<template>
  <div class="flex-1 overflow-y-auto px-4 py-3">
    <!-- 错误状态 -->
    <div v-if="error" class="mb-3 rounded-lg bg-red-500/10 p-3 text-xs text-red-300">
      {{ error }}
    </div>

    <!-- 置信度环 + 根因 -->
    <div class="flex items-start gap-3">
      <div class="relative flex size-14 shrink-0 items-center justify-center rounded-full" :style="{ background: ringGradient }">
        <div class="flex size-11 items-center justify-center rounded-full bg-[#1a1a2e]">
          <span class="text-sm font-bold text-white">{{ confidencePercent }}%</span>
        </div>
      </div>
      <div class="min-w-0 flex-1">
        <h3 :class="titleColor" class="text-sm font-semibold">根因分析</h3>
        <p class="mt-1 text-xs leading-relaxed text-white/70">{{ diagnosis.root_cause }}</p>
      </div>
    </div>

    <!-- 影响服务 -->
    <div v-if="diagnosis.affected_services?.length" class="mt-3 flex flex-wrap gap-1">
      <span
        v-for="svc in diagnosis.affected_services"
        :key="svc"
        class="rounded-full bg-white/5 px-2 py-0.5 text-[10px] text-white/50"
      >
        {{ svc }}
      </span>
    </div>

    <!-- 恢复建议折叠 -->
    <div v-if="diagnosis.recovery_suggestion" class="mt-3">
      <button
        class="flex w-full items-center gap-1 text-xs text-indigo-300 transition hover:text-indigo-200"
        @click="showSuggestions = !showSuggestions"
      >
        <span class="transition-transform" :class="showSuggestions ? 'rotate-90' : ''">▸</span>
        恢复建议
      </button>
      <div v-if="showSuggestions" class="mt-2 rounded-lg bg-white/5 p-3 text-xs leading-relaxed text-white/60">
        {{ diagnosis.recovery_suggestion }}
      </div>
    </div>

    <!-- 重试按钮 -->
    <button
      class="mt-4 w-full rounded-lg border border-white/10 py-2 text-xs text-white/60 transition hover:bg-white/5"
      @click="emit('retry')"
    >
      重新诊断
    </button>
  </div>
</template>
