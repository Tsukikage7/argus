<script setup lang="ts">
/**
 * TracesView - 链路追踪列表页
 */
import { ref, onMounted, watch } from 'vue'
import { useTimeRange } from '@/composables/useTimeRange'
import { queryTraces } from '@/composables/useApi'
import type { TraceListItem } from '@/types'
import TraceTable from '@/components/Traces/TraceTable.vue'

const { queryParam } = useTimeRange()

// 状态
const loading = ref(false)
const traces = ref<TraceListItem[]>([])
const error = ref<string | null>(null)

// 过滤条件
const serviceFilter = ref('')
const keyword = ref('')

async function fetchTraces() {
  loading.value = true
  error.value = null
  try {
    const result = await queryTraces({
      service: serviceFilter.value || undefined,
      keyword: keyword.value || undefined,
      time_range: `last ${queryParam.value}`,
      limit: 50
    })
    traces.value = result.traces
  } catch (err) {
    console.error('Failed to fetch traces:', err)
    error.value = '无法加载链路追踪数据'
  } finally {
    loading.value = false
  }
}

function onSearch() {
  fetchTraces()
}

function onReset() {
  serviceFilter.value = ''
  keyword.value = ''
  fetchTraces()
}

onMounted(() => {
  fetchTraces()
})

watch(queryParam, () => {
  fetchTraces()
})
</script>

<template>
  <div class="p-6 flex flex-col gap-6">
    <div class="flex justify-between items-center">
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <span class="w-8 h-8 rounded-lg bg-primary flex items-center justify-center text-primary-content">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
          </svg>
        </span>
        链路追踪 (Tracing)
      </h1>
    </div>

    <!-- 过滤器 -->
    <div class="card bg-base-100 shadow-sm p-4">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="form-control w-full max-w-xs">
          <label class="label"><span class="label-text">Request UUID</span></label>
          <input 
            v-model="keyword" 
            type="text" 
            placeholder="输入完整或部分 UUID" 
            class="input input-bordered w-full" 
            @keyup.enter="onSearch"
          />
        </div>
        
        <div class="form-control w-full max-w-xs">
          <label class="label"><span class="label-text">服务名称</span></label>
          <input 
            v-model="serviceFilter" 
            type="text" 
            placeholder="例如: order-service" 
            class="input input-bordered w-full" 
            @keyup.enter="onSearch"
          />
        </div>

        <div class="flex gap-2">
          <button class="btn btn-primary" @click="onSearch">搜索</button>
          <button class="btn btn-ghost" @click="onReset">重置</button>
        </div>
      </div>
    </div>

    <!-- 列表展示 -->
    <div class="card bg-base-100 shadow-sm overflow-hidden">
      <div v-if="error" class="alert alert-error m-4">
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        <span>{{ error }}</span>
        <button class="btn btn-sm" @click="fetchTraces">重试</button>
      </div>
      
      <TraceTable :traces="traces" :loading="loading" />
    </div>
  </div>
</template>
