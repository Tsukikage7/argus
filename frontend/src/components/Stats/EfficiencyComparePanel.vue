<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { getEfficiencyStats, type EfficiencyStats } from '@/composables/useApi'

const stats = ref<EfficiencyStats | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    stats.value = await getEfficiencyStats()
  } catch (e) {
    console.error('加载效率数据失败:', e)
  } finally {
    loading.value = false
  }
})

function formatDuration(sec: number): string {
  if (sec >= 60) return `${Math.round(sec / 60)}min`
  return `${sec}s`
}

const speedup = computed(() => {
  if (!stats.value) return '-'
  return `${Math.round(stats.value.manual_avg_time_seconds / stats.value.ai_avg_time_seconds)}x`
})

const stepsReduced = computed(() => {
  if (!stats.value) return 0
  return stats.value.manual_avg_steps - stats.value.ai_avg_steps
})
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden">
    <div class="px-4 py-2.5 border-b border-base-300 text-[0.8125rem] font-semibold text-base-content/70">
      AI 诊断 vs 人工诊断
    </div>
    <div class="p-4">
      <div v-if="loading" class="text-center text-sm py-6 text-base-content/30">加载中…</div>
      <template v-else-if="stats">
        <!-- 对比卡片 -->
        <div class="grid grid-cols-2 gap-3 mb-4">
          <!-- AI -->
          <div class="rounded-lg p-3 bg-indigo-500/10 border border-indigo-500/20">
            <div class="text-[0.625rem] font-bold uppercase tracking-wider text-indigo-400 mb-2">AI Agent</div>
            <div class="text-2xl font-bold text-indigo-300">{{ formatDuration(stats.ai_avg_time_seconds) }}</div>
            <div class="text-[0.7rem] text-base-content/50 mt-1">平均 {{ stats.ai_avg_steps }} 步</div>
            <div class="text-[0.7rem] text-base-content/50">准确率 {{ Math.round(stats.ai_accuracy * 100) }}%</div>
          </div>
          <!-- 人工 -->
          <div class="rounded-lg p-3 bg-orange-500/10 border border-orange-500/20">
            <div class="text-[0.625rem] font-bold uppercase tracking-wider text-orange-400 mb-2">人工排查</div>
            <div class="text-2xl font-bold text-orange-300">{{ formatDuration(stats.manual_avg_time_seconds) }}</div>
            <div class="text-[0.7rem] text-base-content/50 mt-1">平均 {{ stats.manual_avg_steps }} 步</div>
          </div>
        </div>

        <!-- 提速指标 -->
        <div class="flex items-center justify-center gap-4 py-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
          <div class="text-center">
            <div class="text-xl font-bold text-emerald-400">{{ speedup }}</div>
            <div class="text-[0.625rem] text-base-content/50">提速倍率</div>
          </div>
          <div class="w-px h-8 bg-base-300"></div>
          <div class="text-center">
            <div class="text-xl font-bold text-emerald-400">{{ stepsReduced }}</div>
            <div class="text-[0.625rem] text-base-content/50">步骤减少</div>
          </div>
          <div class="w-px h-8 bg-base-300"></div>
          <div class="text-center">
            <div class="text-xl font-bold text-emerald-400">{{ stats.time_saved_hours }}h</div>
            <div class="text-[0.625rem] text-base-content/50">累计节省</div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
