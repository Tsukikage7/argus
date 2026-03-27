<script setup lang="ts">
/**
 * TraceDetailView - 链路详情页
 */
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getTraceDetail, getTraceFlameGraph } from '@/composables/useApi'
import type { TraceDetail, FlameNode } from '@/types'
import TraceTimeline from '@/components/Traces/TraceTimeline.vue'
import MockFlameGraph from '@/components/Traces/MockFlameGraph.vue'

const props = defineProps<{
  uuid: string
}>()

const router = useRouter()

// 状态
const loading = ref(false)
const detail = ref<TraceDetail | null>(null)
const error = ref<string | null>(null)
const activeTab = ref<'timeline' | 'flame' | 'logs'>('timeline')
const flameData = ref<FlameNode | null>(null)

async function fetchDetail() {
  loading.value = true
  error.value = null
  try {
    detail.value = await getTraceDetail(props.uuid)
    // 异步加载火焰图数据
    getTraceFlameGraph(props.uuid)
      .then(resp => { flameData.value = resp.root })
      .catch(() => {
        // 降级：从 spans 构建简易火焰图
        if (detail.value) {
          flameData.value = {
            name: detail.value.entry_service,
            value: detail.value.total_duration_ms,
            children: detail.value.spans.slice(1).map(s => ({
              name: s.service,
              value: s.duration_ms,
              children: []
            }))
          }
        }
      })
  } catch (err) {
    console.error('Failed to fetch trace detail:', err)
    error.value = '无法加载链路详情'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchDetail()
})

function goBack() {
  router.back()
}

function getStatusBadge(status: string) {
  switch (status) {
    case 'error': return 'badge-error'
    case 'slow': return 'badge-warning'
    default: return 'badge-success'
  }
}
</script>

<template>
  <div class="p-6 flex flex-col gap-6">
    <!-- 头部 & 返回 -->
    <div class="flex items-center gap-4">
      <button class="btn btn-sm btn-ghost" @click="goBack">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
        </svg>
        列表
      </button>
      <h1 class="text-xl font-bold flex items-center gap-2">
        链路详情: <span class="font-mono text-primary">{{ uuid }}</span>
      </h1>
    </div>

    <!-- 加载 & 错误 -->
    <div v-if="loading" class="flex justify-center py-20">
      <span class="loading loading-spinner loading-lg text-primary"></span>
    </div>

    <div v-else-if="error" class="alert alert-error">
      <span>{{ error }}</span>
      <button class="btn btn-sm" @click="fetchDetail">重试</button>
    </div>

    <div v-else-if="detail" class="flex flex-col gap-6">
      <!-- 概况卡片 -->
      <div class="card bg-base-100 shadow-sm p-4">
        <div class="flex justify-between items-center">
          <div class="flex gap-8">
            <div>
              <div class="text-xs opacity-50 uppercase font-bold">Entry Service</div>
              <div class="font-bold text-lg">{{ detail.entry_service }}</div>
            </div>
            <div>
              <div class="text-xs opacity-50 uppercase font-bold">Total Duration</div>
              <div class="font-bold text-lg text-primary">{{ detail.total_duration_ms }}ms</div>
            </div>
            <div>
              <div class="text-xs opacity-50 uppercase font-bold">Timestamp</div>
              <div class="text-sm opacity-80">{{ new Date(detail.timestamp).toLocaleString() }}</div>
            </div>
          </div>
          <div class="badge badge-outline">Spans: {{ detail.spans.length }}</div>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="tabs tabs-boxed bg-transparent border border-base-300">
        <a 
          class="tab" 
          :class="{ 'tab-active': activeTab === 'timeline' }"
          @click="activeTab = 'timeline'"
        >
          时间线 (Timeline)
        </a>
        <a 
          class="tab" 
          :class="{ 'tab-active': activeTab === 'flame' }"
          @click="activeTab = 'flame'"
        >
          火焰图 (Flame Graph)
        </a>
        <a 
          class="tab" 
          :class="{ 'tab-active': activeTab === 'logs' }"
          @click="activeTab = 'logs'"
        >
          关联日志 (Related Logs)
        </a>
      </div>

      <!-- 内容区域 -->
      <div class="card bg-base-100 shadow-sm p-4 min-h-[400px]">
        <!-- 时间线视图 -->
        <TraceTimeline 
          v-if="activeTab === 'timeline'" 
          :spans="detail.spans" 
          :total-duration="detail.total_duration_ms" 
        />

        <!-- 火焰图视图 -->
        <MockFlameGraph 
          v-if="activeTab === 'flame' && flameData" 
          :data="flameData" 
        />

        <!-- 日志视图 (简易列表) -->
        <div v-if="activeTab === 'logs'" class="flex flex-col gap-2">
          <div v-for="(span, sIdx) in detail.spans" :key="sIdx" class="collapse collapse-arrow bg-base-200/20">
            <input type="radio" name="log-accordion" :checked="sIdx === 0" /> 
            <div class="collapse-title flex items-center gap-2">
              <span class="badge badge-sm badge-ghost">{{ span.service }}</span>
              <span class="text-xs font-mono">{{ span.operation }}</span>
              <div class="badge badge-sm ml-auto" :class="getStatusBadge(span.status)">{{ span.status }}</div>
            </div>
            <div class="collapse-content"> 
              <div v-if="span.logs.length === 0" class="text-xs opacity-40 text-center py-4">无日志数据</div>
              <div v-else class="flex flex-col gap-1">
                <div v-for="(log, lIdx) in span.logs" :key="lIdx" class="flex gap-2 text-xs font-mono hover:bg-base-300 p-1 rounded">
                  <span class="opacity-40">{{ log.timestamp }}</span>
                  <span :class="{ 'text-error': log.level === 'ERROR', 'text-warning': log.level === 'WARN' }">[{{ log.level }}]</span>
                  <span class="opacity-80">{{ log.message }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
