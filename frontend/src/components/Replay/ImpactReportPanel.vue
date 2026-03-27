<script setup lang="ts">
import type { ImpactReport } from '@/types'

const props = defineProps<{
  report: ImpactReport
}>()

const blastClass: Record<string, string> = {
  low: 'bg-emerald-900/50 text-emerald-300',
  medium: 'bg-amber-900/50 text-amber-300',
  high: 'bg-red-900/50 text-red-300',
  critical: 'bg-red-950 text-red-300',
}

function statusColor(status: string): string {
  if (status === 'down' || status === 'critical') return '#ef4444'
  if (status === 'degraded') return '#f59e0b'
  return '#10b981'
}

const overallErrRate = props.report.total_requests > 0
  ? Math.round(props.report.failed_requests / props.report.total_requests * 100)
  : 0

const affectedCount = props.report.affected_services
  ? props.report.affected_services.filter(s => s.status !== 'healthy').length
  : 0

const totalCount = props.report.affected_services?.length ?? 0
</script>

<template>
  <!-- 影响面报告面板 -->
  <div class="bg-base-100 border border-base-300 rounded-xl overflow-hidden">
    <!-- 卡片标题 -->
    <div class="px-4 py-2.5 border-b border-base-300 flex items-center gap-1.5
                text-[0.8125rem] font-semibold text-base-content/70">
      <svg class="w-3.5 h-3.5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round"
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
      </svg>
      影响面报告
      <span
        class="ml-auto inline-flex items-center px-3 py-1 rounded-full text-[0.6875rem] font-bold tracking-wider"
        :class="blastClass[props.report.blast_radius] ?? blastClass.low"
      >
        {{ (props.report.blast_radius || 'LOW').toUpperCase() }}
      </span>
    </div>

    <!-- 内容区域 -->
    <div class="p-4 overflow-y-auto scroller" style="max-height: 35vh">
      <!-- 3 列统计 -->
      <div class="grid grid-cols-3 gap-2 mb-3">
        <div class="bg-base-200 border border-base-300 rounded-lg p-2.5 text-center">
          <div class="text-sm font-bold text-base-content">{{ props.report.total_requests || 0 }}</div>
          <div class="text-[0.625rem] text-base-content/40 uppercase tracking-wider mt-0.5">请求总量</div>
        </div>
        <div class="bg-base-200 border border-base-300 rounded-lg p-2.5 text-center">
          <div class="text-sm font-bold text-red-400">{{ props.report.failed_requests || 0 }}</div>
          <div class="text-[0.625rem] text-base-content/40 uppercase tracking-wider mt-0.5">失败量</div>
        </div>
        <div class="bg-base-200 border border-base-300 rounded-lg p-2.5 text-center">
          <div class="text-sm font-bold text-base-content">{{ overallErrRate }}%</div>
          <div class="text-[0.625rem] text-base-content/40 uppercase tracking-wider mt-0.5">错误率</div>
        </div>
      </div>

      <!-- 受影响服务统计 -->
      <div class="text-xs mb-2 text-base-content/40">
        受影响服务: {{ affectedCount }}/{{ totalCount }}
      </div>

      <!-- 服务指标表格 -->
      <table
        v-if="props.report.affected_services && props.report.affected_services.length > 0"
        class="w-full text-xs border-collapse"
      >
        <thead>
          <tr class="text-base-content/40">
            <th class="text-left py-1 px-1">服务</th>
            <th class="text-center py-1 px-1">状态</th>
            <th class="text-right py-1 px-1">错误数</th>
            <th class="text-right py-1 px-1">错误率</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="svc in props.report.affected_services"
            :key="svc.name"
            class="border-t border-base-300"
          >
            <td class="py-1 px-1 text-base-content/70">{{ svc.name }}</td>
            <td class="text-center py-1 px-1">
              <span class="font-semibold" :style="{ color: statusColor(svc.status) }">
                {{ svc.status.toUpperCase() }}
              </span>
            </td>
            <td class="text-right py-1 px-1 text-base-content/70">{{ svc.error_count }}</td>
            <td class="text-right py-1 px-1 text-base-content/70">
              {{ Math.round((svc.error_rate || 0) * 100) }}%
            </td>
          </tr>
        </tbody>
      </table>

      <!-- LLM 总结 -->
      <div
        v-if="props.report.summary"
        class="mt-3 p-2.5 rounded-lg text-xs bg-base-200 text-base-content/70 leading-relaxed"
      >
        {{ props.report.summary }}
      </div>
    </div>
  </div>
</template>
