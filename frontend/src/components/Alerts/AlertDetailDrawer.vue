<script setup lang="ts">
/**
 * AlertDetailDrawer - 告警详情抽屉
 * 告警信息 + 关联诊断任务 + 跳转按钮
 */
import type { AlertEvent } from '@/types'
import { useRouter } from 'vue-router'

const props = defineProps<{
  alert: AlertEvent | null
  open: boolean
}>()

const emit = defineEmits<{
  close: []
  diagnose: [alert: AlertEvent]
}>()

const router = useRouter()

const severityLabel: Record<string, string> = {
  critical: '严重',
  warning: '警告',
  info: '信息',
}

const severityClass: Record<string, string> = {
  critical: 'text-error',
  warning: 'text-warning',
  info: 'text-info',
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

function goToTask() {
  if (props.alert?.task_id) {
    router.push({ name: 'task-detail', params: { id: props.alert.task_id } })
    emit('close')
  }
}

function triggerDiagnose() {
  if (props.alert) {
    emit('diagnose', props.alert)
    emit('close')
  }
}
</script>

<template>
  <div class="drawer drawer-end" :class="{ 'drawer-open': open }">
    <input type="checkbox" class="drawer-toggle" :checked="open" />
    <div class="drawer-side z-50">
      <label class="drawer-overlay" @click="emit('close')"></label>
      <div class="bg-base-100 w-96 min-h-full p-6 flex flex-col gap-4">
        <div class="flex justify-between items-center">
          <h3 class="text-lg font-bold">告警详情</h3>
          <button class="btn btn-sm btn-ghost btn-circle" @click="emit('close')">✕</button>
        </div>

        <template v-if="alert">
          <div class="space-y-3">
            <div>
              <span class="text-xs text-base-content/50">严重度</span>
              <p class="font-semibold" :class="severityClass[alert.severity]">
                {{ severityLabel[alert.severity] }}
              </p>
            </div>
            <div>
              <span class="text-xs text-base-content/50">服务</span>
              <p class="font-mono">{{ alert.service }}</p>
            </div>
            <div>
              <span class="text-xs text-base-content/50">描述</span>
              <p>{{ alert.message }}</p>
            </div>
            <div>
              <span class="text-xs text-base-content/50">触发时间</span>
              <p>{{ formatTime(alert.created_at) }}</p>
            </div>
            <div v-if="alert.resolved_at">
              <span class="text-xs text-base-content/50">恢复时间</span>
              <p>{{ formatTime(alert.resolved_at) }}</p>
            </div>
          </div>

          <div class="divider"></div>

          <div class="space-y-2">
            <h4 class="font-semibold text-sm">关联诊断</h4>
            <div v-if="alert.task_id" class="flex flex-col gap-2">
              <p class="text-sm">诊断任务: <span class="font-mono text-primary">{{ alert.task_id }}</span></p>
              <button class="btn btn-sm btn-primary" @click="goToTask">查看诊断详情</button>
            </div>
            <div v-else class="flex flex-col gap-2">
              <p class="text-sm text-base-content/50">暂无关联诊断任务</p>
              <button class="btn btn-sm btn-primary btn-outline" @click="triggerDiagnose">触发诊断</button>
            </div>
          </div>

          <div class="divider"></div>

          <button
            class="btn btn-sm btn-ghost"
            @click="$router.push({ name: 'logs', query: { service: alert.service } }); emit('close')"
          >
            查看相关日志 →
          </button>
        </template>
      </div>
    </div>
  </div>
</template>
