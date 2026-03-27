<script setup lang="ts">
/**
 * TraceTable - Trace 列表表格组件
 */
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import type { TraceListItem } from '@/types'

const props = defineProps<{
  traces: TraceListItem[]
  loading: boolean
}>()

const router = useRouter()

// 排序状态
const sortKey = ref<'duration_ms' | 'timestamp'>('timestamp')
const sortOrder = ref<'asc' | 'desc'>('desc')

function toggleSort(key: 'duration_ms' | 'timestamp') {
  if (sortKey.value === key) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortOrder.value = 'desc'
  }
}

const sortedTraces = computed(() => {
  const result = [...props.traces]
  result.sort((a, b) => {
    let valA: number, valB: number
    if (sortKey.value === 'timestamp') {
      valA = new Date(a.timestamp).getTime()
      valB = new Date(b.timestamp).getTime()
    } else {
      valA = a.duration_ms
      valB = b.duration_ms
    }
    return sortOrder.value === 'asc' ? valA - valB : valB - valA
  })
  return result
})

function goToDetail(uuid: string) {
  router.push(`/traces/${uuid}`)
}

function getStatusBadge(code: number) {
  if (code >= 200 && code < 300) return 'badge-success'
  if (code >= 400 && code < 500) return 'badge-warning'
  return 'badge-error'
}

function formatTime(ts: string) {
  return new Date(ts).toLocaleString()
}
</script>

<template>
  <div class="overflow-x-auto">
    <table class="table table-zebra w-full">
      <thead>
        <tr>
          <th>Request UUID</th>
          <th>入口服务</th>
          <th>状态码</th>
          <th class="cursor-pointer hover:bg-base-200" @click="toggleSort('duration_ms')">
            总耗时
            <span v-if="sortKey === 'duration_ms'">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
          </th>
          <th class="cursor-pointer hover:bg-base-200" @click="toggleSort('timestamp')">
            时间戳
            <span v-if="sortKey === 'timestamp'">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
          </th>
          <th>涉及服务</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading" class="text-center">
          <td colspan="6" class="py-10">
            <span class="loading loading-spinner loading-lg"></span>
          </td>
        </tr>
        <tr v-else-if="sortedTraces.length === 0" class="text-center">
          <td colspan="6" class="py-10 text-base-content/50">无链路数据</td>
        </tr>
        <tr 
          v-for="trace in sortedTraces" 
          :key="trace.request_uuid"
          class="hover cursor-pointer"
          @click="goToDetail(trace.request_uuid)"
        >
          <td class="font-mono text-xs">{{ trace.request_uuid }}</td>
          <td>
            <div class="badge badge-ghost">{{ trace.entry_service }}</div>
          </td>
          <td>
            <div class="badge" :class="getStatusBadge(trace.status_code)">
              {{ trace.status_code }}
            </div>
          </td>
          <td>
            <span :class="{ 'text-error font-bold': trace.duration_ms > 1000 }">
              {{ trace.duration_ms }}ms
            </span>
          </td>
          <td class="text-xs opacity-70">{{ formatTime(trace.timestamp) }}</td>
          <td>
            <div class="flex flex-wrap gap-1">
              <span 
                v-for="svc in trace.services.slice(0, 3)" 
                :key="svc"
                class="badge badge-sm badge-outline"
              >
                {{ svc }}
              </span>
              <span v-if="trace.services.length > 3" class="text-xs opacity-50">
                +{{ trace.services.length - 3 }}
              </span>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
