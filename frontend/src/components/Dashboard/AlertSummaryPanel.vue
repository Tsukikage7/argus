<script setup lang="ts">
export interface AlertItem {
  id: string
  severity: 'critical' | 'warning' | 'info'
  service: string
  message: string
  time: string
}

defineProps<{
  alerts: AlertItem[]
}>()

const severityConfig: Record<string, { badge: string; label: string }> = {
  critical: { badge: 'badge-error', label: 'Critical' },
  warning:  { badge: 'badge-warning', label: 'Warning' },
  info:     { badge: 'badge-info', label: 'Info' },
}

function countBySeverity(alerts: AlertItem[], severity: string): number {
  return alerts.filter(a => a.severity === severity).length
}

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div>
    <h3 class="text-sm font-semibold text-base-content/70 mb-3">告警摘要</h3>

    <!-- 严重度计数 -->
    <div class="flex gap-3 mb-3">
      <div
        v-for="sev in ['critical', 'warning', 'info']"
        :key="sev"
        class="flex items-center gap-1.5"
      >
        <span class="badge badge-sm" :class="severityConfig[sev]?.badge">
          {{ countBySeverity(alerts, sev) }}
        </span>
        <span class="text-xs text-base-content/50">{{ severityConfig[sev]?.label }}</span>
      </div>
    </div>

    <!-- 最近告警列表 -->
    <div class="space-y-2">
      <div
        v-for="alert in alerts.slice(0, 5)"
        :key="alert.id"
        class="flex items-start gap-2 text-xs"
      >
        <span class="badge badge-xs mt-0.5 shrink-0" :class="severityConfig[alert.severity]?.badge" />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="font-medium text-base-content/70">{{ alert.service }}</span>
            <span class="text-base-content/30">{{ formatTime(alert.time) }}</span>
          </div>
          <p class="text-base-content/50 truncate">{{ alert.message }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
