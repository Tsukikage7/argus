<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getEfficiencyStats, type EfficiencyStats } from '@/composables/useApi'

const stats = ref<EfficiencyStats | null>(null)
const loading = ref(true)

onMounted(async () => {
  try {
    stats.value = await getEfficiencyStats()
  } catch (e) {
    console.error('加载统计数据失败:', e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="p-6 space-y-6">
    <h2 class="text-lg font-semibold text-base-content/80">诊断统计</h2>

    <div v-if="loading" class="text-center text-sm py-12 text-base-content/30">加载中…</div>

    <template v-else-if="stats">
      <!-- 概览卡片 -->
      <div class="grid grid-cols-4 gap-4">
        <div class="glass-card rounded-xl p-4 text-center">
          <div class="text-2xl font-bold text-indigo-400">{{ stats.total_diagnoses }}</div>
          <div class="text-xs text-base-content/50 mt-1">累计诊断</div>
        </div>
        <div class="glass-card rounded-xl p-4 text-center">
          <div class="text-2xl font-bold text-emerald-400">{{ Math.round(stats.ai_accuracy * 100) }}%</div>
          <div class="text-xs text-base-content/50 mt-1">诊断准确率</div>
        </div>
        <div class="glass-card rounded-xl p-4 text-center">
          <div class="text-2xl font-bold text-amber-400">{{ stats.ai_avg_time_seconds }}s</div>
          <div class="text-xs text-base-content/50 mt-1">平均耗时</div>
        </div>
        <div class="glass-card rounded-xl p-4 text-center">
          <div class="text-2xl font-bold text-cyan-400">{{ stats.scenarios_covered }}</div>
          <div class="text-xs text-base-content/50 mt-1">覆盖场景</div>
        </div>
      </div>

      <!-- 节省指标 -->
      <div class="glass-card rounded-xl p-4">
        <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-3">效率提升</div>
        <div class="grid grid-cols-3 gap-4 text-center">
          <div>
            <div class="text-xl font-bold text-emerald-400">{{ stats.time_saved_hours }}h</div>
            <div class="text-xs text-base-content/50 mt-1">累计节省时间</div>
          </div>
          <div>
            <div class="text-xl font-bold text-indigo-400">{{ stats.ai_avg_steps }} 步</div>
            <div class="text-xs text-base-content/50 mt-1">AI 平均步骤</div>
          </div>
          <div>
            <div class="text-xl font-bold text-orange-400">{{ stats.manual_avg_steps }} 步</div>
            <div class="text-xs text-base-content/50 mt-1">人工平均步骤</div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
