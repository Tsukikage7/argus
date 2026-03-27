<script setup lang="ts">
/**
 * AlertEventTable - 告警事件列表组件
 * 展示时间/严重度/服务/描述/状态/关联诊断
 */
import type { AlertEvent } from '@/types'
import { useRouter } from 'vue-router'

defineProps<{
  alerts: AlertEvent[]
  loading: boolean
}>()

const emit = defineEmits<{
  select: [alert: AlertEvent]
}>()

const router = useRouter()

const severityClass: Record<string, string> = {
  critical: 'badge-error',
  warning: 'badge-warning',
  info: 'badge-info',
}

const statusClass: Record<string, string> = {
  firing: 'badge-error badge-outline',
  acknowledged: 'badge-warning badge-outline',
  resolved: 'badge-success badge-outline',
}

const statusLabel: Record<string, string> = {
  firing: '触发中',
  acknowledged: '已确认',
  resolved: '已恢复',
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

function goToTask(taskId: string) {
  router.push({ name: 'task-detail', params: { id: taskId } })
}
</script>

<template>
  <div class="overflow-x-auto">
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg text-primary"></span>
    </div>
    <table v-else class="table table-sm">
      <thead>
        <tr>
          <th>时间</th>
          <th>严重度</th>
          <th>服务</th>
          <th>描述</th>
          <th>状态</th>
          <th>关联诊断</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="alerts.length === 0">
          <td colspan="6" class="text-center text-base-content/50 py-8">暂无告警数据</td>
        </tr>
        <tr
          v-for="alert in alerts"
          :key="alert.id"
          class="hover:bg-base-200/50 cursor-pointer"
          @click="emit('select', alert)"
        >
          <td class="whitespace-nowrap text-xs">{{ formatTime(alert.created_at) }}</td>
          <td><span class="badge badge-sm" :class="severityClass[alert.severity]">{{ alert.severity }}</span></td>
          <td class="font-mono text-sm">{{ alert.service }}</td>
          <td class="max-w-xs truncate">{{ alert.message }}</td>
          <td><span class="badge badge-sm" :class="statusClass[alert.status]">{{ statusLabel[alert.status] }}</span></td>
          <td>
            <button
              v-if="alert.task_id"
              class="btn btn-xs btn-ghost text-primary"
              @click.stop="goToTask(alert.task_id)"
            >
              {{ alert.task_id }}
            </button>
            <span v-else class="text-base-content/30 text-xs">—</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
