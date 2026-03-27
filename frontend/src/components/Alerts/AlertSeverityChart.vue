<script setup lang="ts">
/**
 * AlertSeverityChart - 告警严重度分布图（ApexCharts 饼图）
 */
import { computed } from 'vue'
import type { AlertSeveritySummary } from '@/types'
import VueApexCharts from 'vue3-apexcharts'

const props = defineProps<{
  summary: AlertSeveritySummary
}>()

const chartOptions = computed(() => ({
  chart: { type: 'donut' as const },
  labels: ['Critical', 'Warning', 'Info'],
  colors: ['#ef4444', '#f59e0b', '#3b82f6'],
  legend: { position: 'bottom' as const },
  plotOptions: {
    pie: {
      donut: {
        size: '55%',
        labels: {
          show: true,
          total: {
            show: true,
            label: '总计',
            formatter: () => String(props.summary.critical + props.summary.warning + props.summary.info),
          },
        },
      },
    },
  },
  dataLabels: { enabled: false },
}))

const series = computed(() => [
  props.summary.critical,
  props.summary.warning,
  props.summary.info,
])
</script>

<template>
  <div class="card bg-base-100 shadow-sm p-4">
    <h3 class="font-semibold mb-3">严重度分布</h3>
    <VueApexCharts type="donut" height="240" :options="chartOptions" :series="series" />
  </div>
</template>
