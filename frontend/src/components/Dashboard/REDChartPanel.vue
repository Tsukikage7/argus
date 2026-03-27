<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import { useTimeRange } from '@/composables/useTimeRange'

const { activePreset, label: timeLabel } = useTimeRange()

// Mock RED 指标数据生成
function generateMockSeries(points: number, base: number, variance: number): number[] {
  const data: number[] = []
  let val = base
  for (let i = 0; i < points; i++) {
    val += (Math.random() - 0.5) * variance
    val = Math.max(0, val)
    data.push(Math.round(val * 100) / 100)
  }
  return data
}

const PRESET_POINTS: Record<string, { points: number; intervalMs: number }> = {
  '15m': { points: 15, intervalMs: 60 * 1000 },
  '1h':  { points: 30, intervalMs: 2 * 60 * 1000 },
  '6h':  { points: 36, intervalMs: 10 * 60 * 1000 },
  '24h': { points: 48, intervalMs: 30 * 60 * 1000 },
  '7d':  { points: 42, intervalMs: 4 * 60 * 60 * 1000 },
}

const version = ref(0)

watch(activePreset, () => { version.value++ })

const chartData = computed(() => {
  // version 触发重新计算
  void version.value
  const cfg = PRESET_POINTS[activePreset.value] || PRESET_POINTS['1h']
  const now = Date.now()
  const cats = Array.from({ length: cfg.points }, (_, i) => {
    return new Date(now - (cfg.points - 1 - i) * cfg.intervalMs).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    })
  })
  return {
    categories: cats,
    series: [
      { name: 'Request Rate (req/s)', data: generateMockSeries(cfg.points, 120, 30) },
      { name: 'Error Rate (%)', data: generateMockSeries(cfg.points, 2, 1.5) },
      { name: 'Duration P99 (ms)', data: generateMockSeries(cfg.points, 200, 80) },
    ],
  }
})

const chartOptions = computed(() => ({
  chart: {
    type: 'line' as const,
    height: 240,
    toolbar: { show: false },
    background: 'transparent',
    fontFamily: 'inherit',
  },
  theme: { mode: 'dark' as const },
  colors: ['#6366f1', '#ef4444', '#f59e0b'],
  stroke: { width: 2, curve: 'smooth' as const },
  grid: {
    borderColor: 'rgba(255,255,255,0.06)',
    strokeDashArray: 4,
  },
  xaxis: {
    categories: chartData.value.categories,
    labels: {
      style: { colors: 'rgba(255,255,255,0.3)', fontSize: '10px' },
      rotate: 0,
      hideOverlappingLabels: true,
    },
    axisBorder: { show: false },
    axisTicks: { show: false },
  },
  yaxis: [
    { title: { text: 'req/s', style: { color: 'rgba(255,255,255,0.3)', fontSize: '10px' } }, labels: { style: { colors: 'rgba(255,255,255,0.3)' } } },
    { opposite: true, title: { text: '%', style: { color: 'rgba(255,255,255,0.3)', fontSize: '10px' } }, labels: { style: { colors: 'rgba(255,255,255,0.3)' } } },
    { opposite: true, show: false },
  ],
  legend: {
    labels: { colors: 'rgba(255,255,255,0.5)' },
    fontSize: '11px',
  },
  tooltip: { theme: 'dark' },
}))
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-sm font-semibold text-base-content/70">RED 指标</h3>
      <span class="text-xs text-base-content/30">{{ timeLabel }}</span>
    </div>
    <VueApexCharts
      type="line"
      height="240"
      :options="chartOptions"
      :series="chartData.series"
    />
  </div>
</template>
