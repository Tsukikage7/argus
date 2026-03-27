<script setup lang="ts">
/**
 * AlertsView - 告警仪表盘页面
 * 告警事件列表 + 严重度分布图 + 过滤 + 详情抽屉
 */
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useTimeRange } from '@/composables/useTimeRange'
import { getActiveAlerts } from '@/composables/useApi'
import type { AlertEvent, AlertSeveritySummary } from '@/types'
import AlertEventTable from '@/components/Alerts/AlertEventTable.vue'
import AlertSeverityChart from '@/components/Alerts/AlertSeverityChart.vue'
import AlertDetailDrawer from '@/components/Alerts/AlertDetailDrawer.vue'

const { queryParam } = useTimeRange()
const router = useRouter()

const loading = ref(false)
const error = ref<string | null>(null)
const alerts = ref<AlertEvent[]>([])
const summary = ref<AlertSeveritySummary>({ critical: 0, warning: 0, info: 0 })

// 过滤
const severityFilter = ref('')
const serviceFilter = ref('')
const statusFilter = ref('')

// 抽屉
const selectedAlert = ref<AlertEvent | null>(null)
const drawerOpen = ref(false)

const filteredAlerts = computed(() => {
  return alerts.value.filter(a => {
    if (severityFilter.value && a.severity !== severityFilter.value) return false
    if (serviceFilter.value && !a.service.toLowerCase().includes(serviceFilter.value.toLowerCase())) return false
    if (statusFilter.value && a.status !== statusFilter.value) return false
    return true
  })
})

async function fetchAlerts() {
  loading.value = true
  error.value = null
  try {
    const result = await getActiveAlerts()
    alerts.value = result.alerts
    summary.value = result.summary
  } catch (err) {
    console.error('Failed to fetch alerts:', err)
    error.value = '无法加载告警数据'
  } finally {
    loading.value = false
  }
}
function onSelectAlert(alert: AlertEvent) {
  selectedAlert.value = alert
  drawerOpen.value = true
}

function onDiagnose(alert: AlertEvent) {
  router.push({ name: 'chat', query: { input: `诊断告警: ${alert.message} (服务: ${alert.service})` } })
}

function onReset() {
  severityFilter.value = ''
  serviceFilter.value = ''
  statusFilter.value = ''
}

onMounted(() => fetchAlerts())
watch(queryParam, () => fetchAlerts())
</script>

<template>
  <div class="p-6 flex flex-col gap-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <span class="w-8 h-8 rounded-lg bg-error flex items-center justify-center text-error-content">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
        </span>
        告警中心 (Alerts)
      </h1>
    </div>

    <!-- 统计 + 图表 -->
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-4">
      <div class="lg:col-span-3">
        <!-- 过滤器 -->
        <div class="card bg-base-100 shadow-sm p-4 mb-4">
          <div class="flex flex-wrap gap-4 items-end">
            <div class="form-control w-full max-w-[160px]">
              <label class="label"><span class="label-text">严重度</span></label>
              <select v-model="severityFilter" class="select select-bordered select-sm w-full">
                <option value="">全部</option>
                <option value="critical">Critical</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
              </select>
            </div>
            <div class="form-control w-full max-w-[160px]">
              <label class="label"><span class="label-text">服务</span></label>
              <input v-model="serviceFilter" type="text" placeholder="服务名" class="input input-bordered input-sm w-full" />
            </div>
            <div class="form-control w-full max-w-[160px]">
              <label class="label"><span class="label-text">状态</span></label>
              <select v-model="statusFilter" class="select select-bordered select-sm w-full">
                <option value="">全部</option>
                <option value="firing">触发中</option>
                <option value="acknowledged">已确认</option>
                <option value="resolved">已恢复</option>
              </select>
            </div>
            <button class="btn btn-sm btn-ghost" @click="onReset">重置</button>
          </div>
        </div>
<!-- TABLE_PLACEHOLDER -->
        <!-- 告警列表 -->
        <div class="card bg-base-100 shadow-sm overflow-hidden">
          <div v-if="error" class="alert alert-error m-4">
            <span>{{ error }}</span>
            <button class="btn btn-sm" @click="fetchAlerts">重试</button>
          </div>
          <AlertEventTable :alerts="filteredAlerts" :loading="loading" @select="onSelectAlert" />
        </div>
      </div>

      <!-- 右侧图表 -->
      <div>
        <AlertSeverityChart :summary="summary" />
      </div>
    </div>

    <!-- 详情抽屉 -->
    <AlertDetailDrawer
      :alert="selectedAlert"
      :open="drawerOpen"
      @close="drawerOpen = false"
      @diagnose="onDiagnose"
    />
  </div>
</template>

