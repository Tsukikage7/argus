<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboardSummary, type DashboardSummary } from '@/composables/useApi'
import StatCard from '@/components/Dashboard/StatCard.vue'
import ServiceHealthGrid from '@/components/Dashboard/ServiceHealthGrid.vue'
import AlertSummaryPanel from '@/components/Dashboard/AlertSummaryPanel.vue'
import RecentDiagnosesPanel from '@/components/Dashboard/RecentDiagnosesPanel.vue'
import REDChartPanel from '@/components/Dashboard/REDChartPanel.vue'
import EfficiencyComparePanel from '@/components/Stats/EfficiencyComparePanel.vue'
import DemoScenarioRunner from '@/components/Stats/DemoScenarioRunner.vue'

const data = ref<DashboardSummary | null>(null)
const loading = ref(true)
const error = ref('')

const statCards = [
  { key: 'services', label: '总服务数', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2', color: '#6366f1' },
  { key: 'alerts', label: '活跃告警', icon: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z', color: '#ef4444' },
  { key: 'diagnoses', label: '今日诊断', icon: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z', color: '#10b981' },
  { key: 'avgTime', label: '平均诊断耗时', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z', color: '#f59e0b' },
]

function getStatValue(key: string): string | number {
  if (!data.value) return '—'
  switch (key) {
    case 'services': return data.value.total_services
    case 'alerts': return data.value.active_alerts
    case 'diagnoses': return data.value.today_diagnoses
    case 'avgTime': return `${data.value.avg_diagnose_time_seconds}s`
    default: return '—'
  }
}

onMounted(async () => {
  try {
    data.value = await getDashboardSummary()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div>
    <h1 class="text-xl font-bold text-base-content mb-4">总览</h1>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <span class="loading loading-spinner loading-md text-primary" />
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="glass-card rounded-xl p-6 text-center">
      <p class="text-error text-sm">{{ error }}</p>
      <p class="text-base-content/40 text-xs mt-1">Dashboard API 不可用，请检查后端服务</p>
    </div>

    <!-- 数据展示 -->
    <template v-else-if="data">
      <!-- 统计卡片 -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
        <StatCard
          v-for="card in statCards"
          :key="card.key"
          :label="card.label"
          :value="getStatValue(card.key)"
          :icon="card.icon"
          :color="card.color"
          :trend="card.key === 'alerts' && data.active_alerts > 0 ? 'up' : undefined"
          :trend-value="card.key === 'alerts' && data.active_alerts > 0 ? `${data.active_alerts}` : undefined"
        />
      </div>

      <!-- 服务健康 + 告警摘要 -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <div class="lg:col-span-2">
          <ServiceHealthGrid :services="data.service_health" />
        </div>
        <div class="glass-card rounded-xl p-4">
          <AlertSummaryPanel :alerts="data.recent_alerts" />
        </div>
      </div>

      <!-- RED 图表 + 最近诊断 -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <div class="lg:col-span-2 glass-card rounded-xl p-4">
          <REDChartPanel />
        </div>
        <div class="glass-card rounded-xl p-4">
          <RecentDiagnosesPanel :diagnoses="data.recent_diagnoses" />
        </div>
      </div>

      <!-- 效率对比 + 一键演示 -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div class="lg:col-span-2">
          <EfficiencyComparePanel />
        </div>
        <DemoScenarioRunner />
      </div>
    </template>
  </div>
</template>
