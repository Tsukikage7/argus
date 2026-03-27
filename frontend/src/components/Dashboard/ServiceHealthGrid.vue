<script setup lang="ts">
export interface ServiceHealthItem {
  name: string
  status: 'healthy' | 'degraded' | 'critical'
  error_rate: number
  p99_latency_ms: number
}

defineProps<{
  services: ServiceHealthItem[]
}>()

const statusConfig: Record<string, { dot: string; label: string }> = {
  healthy:  { dot: 'bg-success', label: '正常' },
  degraded: { dot: 'bg-warning', label: '降级' },
  critical: { dot: 'bg-error', label: '异常' },
}
</script>

<template>
  <div>
    <h3 class="text-sm font-semibold text-base-content/70 mb-3">服务健康</h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <div
        v-for="svc in services"
        :key="svc.name"
        class="glass-card rounded-lg p-3 transition-all hover:shadow-md"
        :class="svc.status === 'critical' ? 'border-error/30' : svc.status === 'degraded' ? 'border-warning/30' : ''"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-base-content truncate">{{ svc.name }}</span>
          <span class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full" :class="statusConfig[svc.status]?.dot" />
            <span class="text-xs text-base-content/50">{{ statusConfig[svc.status]?.label }}</span>
          </span>
        </div>
        <div class="flex items-center gap-4 text-xs text-base-content/50">
          <span>
            错误率
            <span class="font-mono text-base-content" :class="svc.error_rate > 0.1 ? 'text-error' : ''">
              {{ (svc.error_rate * 100).toFixed(1) }}%
            </span>
          </span>
          <span>
            P99
            <span class="font-mono text-base-content" :class="svc.p99_latency_ms > 1000 ? 'text-warning' : ''">
              {{ svc.p99_latency_ms }}ms
            </span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
