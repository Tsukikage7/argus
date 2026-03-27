<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { getLogSummary, queryLogs } from '@/composables/useApi'
import type { LogSummaryBucket, LogEntry } from '@/composables/useApi'

// 时间范围选项
const TIME_RANGES = [
  { label: '最近 15 分钟', value: 'last 15m' },
  { label: '最近 1 小时', value: 'last 1h' },
  { label: '最近 6 小时', value: 'last 6h' },
  { label: '最近 24 小时', value: 'last 24h' },
]

const LEVELS = ['ERROR', 'WARN', 'INFO', 'DEBUG'] as const

const timeRange = ref('last 1h')
const activeLevels = ref<Set<string>>(new Set(['ERROR', 'WARN', 'INFO']))
const selectedNamespace = ref('')
const keyword = ref('')
const loading = ref(false)

// 聚合数据
const summaryBuckets = ref<LogSummaryBucket[]>([])
const logs = ref<LogEntry[]>([])
const expandedRows = ref<Set<number>>(new Set())

// 按 namespace 分组的摘要
const namespaceSummary = computed(() => {
  const map = new Map<string, { total: number; errors: number }>()
  for (const b of summaryBuckets.value) {
    if (!map.has(b.namespace)) {
      map.set(b.namespace, { total: 0, errors: 0 })
    }
    const entry = map.get(b.namespace)!
    entry.total += b.count
    if (b.level === 'ERROR') entry.errors = b.count
  }
  return Array.from(map.entries())
    .map(([ns, stats]) => ({ namespace: ns, ...stats }))
    .sort((a, b) => b.errors - a.errors || b.total - a.total)
})

async function fetchSummary() {
  try {
    const data = await getLogSummary(timeRange.value)
    summaryBuckets.value = data.buckets || []
  } catch {
    summaryBuckets.value = []
  }
}

async function fetchLogs() {
  loading.value = true
  try {
    const levelFilter = activeLevels.value.size < LEVELS.length
      ? Array.from(activeLevels.value).join(',')
      : undefined
    logs.value = await queryLogs({
      namespace: selectedNamespace.value || undefined,
      keyword: keyword.value || undefined,
      level: levelFilter,
      time_range: timeRange.value,
      limit: 100,
    })
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

function toggleLevel(level: string) {
  if (activeLevels.value.has(level)) {
    activeLevels.value.delete(level)
  } else {
    activeLevels.value.add(level)
  }
  activeLevels.value = new Set(activeLevels.value) // 触发响应
  fetchLogs()
}

function selectNamespace(ns: string) {
  selectedNamespace.value = selectedNamespace.value === ns ? '' : ns
  fetchLogs()
}

function toggleExpand(idx: number) {
  if (expandedRows.value.has(idx)) {
    expandedRows.value.delete(idx)
  } else {
    expandedRows.value.add(idx)
  }
  expandedRows.value = new Set(expandedRows.value)
}

function inferLevel(message: string): string {
  if (message.includes('ERROR') || message.includes('"level":"error"')) return 'ERROR'
  if (message.includes('WARN') || message.includes('"level":"warn"')) return 'WARN'
  if (message.includes('DEBUG') || message.includes('"level":"debug"')) return 'DEBUG'
  return 'INFO'
}

const levelColor: Record<string, string> = {
  ERROR: 'text-red-400 bg-red-500/10 border-red-500/30',
  WARN: 'text-amber-400 bg-amber-500/10 border-amber-500/30',
  INFO: 'text-blue-400 bg-blue-500/10 border-blue-500/30',
  DEBUG: 'text-base-content/40 bg-base-300/20 border-base-300/30',
}

function formatTimestamp(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString('zh-CN', {
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    })
  } catch {
    return ts
  }
}

function truncateMessage(msg: string, max = 120): string {
  return msg.length > max ? msg.slice(0, max) + '…' : msg
}

watch(timeRange, () => {
  fetchSummary()
  fetchLogs()
})

onMounted(() => {
  fetchSummary()
  fetchLogs()
})
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden flex flex-col">
    <!-- 顶部工具栏 -->
    <div class="px-4 py-2.5 border-b border-base-300/50 flex items-center gap-3 flex-wrap">
      <div class="flex items-center gap-1.5 text-[0.8125rem] font-semibold text-base-content/70">
        <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
        </svg>
        日志浏览器
      </div>

      <!-- 时间范围 -->
      <select
        v-model="timeRange"
        class="select select-xs select-bordered bg-base-200 text-xs ml-auto"
      >
        <option v-for="tr in TIME_RANGES" :key="tr.value" :value="tr.value">
          {{ tr.label }}
        </option>
      </select>

      <!-- 级别 toggle -->
      <div class="flex gap-1">
        <button
          v-for="level in LEVELS"
          :key="level"
          class="px-2 py-0.5 rounded text-[0.65rem] font-medium border transition-all"
          :class="activeLevels.has(level)
            ? levelColor[level]
            : 'text-base-content/20 bg-transparent border-base-300/30 opacity-50'"
          @click="toggleLevel(level)"
        >
          {{ level }}
        </button>
      </div>
    </div>

    <!-- 主体：左侧 namespace 列表 + 右侧日志 -->
    <div class="flex" style="max-height: 40vh">
      <!-- 左侧 namespace 树 -->
      <div class="w-48 shrink-0 border-r border-base-300/50 overflow-y-auto scroller p-2 space-y-0.5">
        <div
          class="px-2 py-1.5 rounded text-xs cursor-pointer transition-all flex items-center justify-between"
          :class="selectedNamespace === '' ? 'bg-indigo-500/15 text-indigo-400' : 'text-base-content/60 hover:bg-base-200'"
          @click="selectNamespace('')"
        >
          <span>全部</span>
        </div>
        <div
          v-for="ns in namespaceSummary"
          :key="ns.namespace"
          class="px-2 py-1.5 rounded text-xs cursor-pointer transition-all flex items-center justify-between gap-1"
          :class="selectedNamespace === ns.namespace ? 'bg-indigo-500/15 text-indigo-400' : 'text-base-content/60 hover:bg-base-200'"
          @click="selectNamespace(ns.namespace)"
        >
          <span class="truncate">{{ ns.namespace }}</span>
          <span
            v-if="ns.errors > 0"
            class="shrink-0 px-1.5 py-0.5 rounded-full text-[0.6rem] font-bold bg-red-500/20 text-red-400"
          >
            {{ ns.errors }}
          </span>
        </div>
        <div v-if="namespaceSummary.length === 0" class="text-center text-[0.65rem] text-base-content/25 py-4">
          暂无数据
        </div>
      </div>

      <!-- 右侧日志列表 -->
      <div class="flex-1 overflow-y-auto scroller p-2 space-y-1">
        <div v-if="loading" class="text-center text-xs text-base-content/30 py-8">
          加载中…
        </div>
        <div v-else-if="logs.length === 0" class="text-center text-xs text-base-content/30 py-8">
          暂无日志
        </div>
        <div
          v-for="(log, i) in logs"
          v-else
          :key="i"
          class="rounded-lg px-3 py-1.5 text-xs bg-base-200/30 hover:bg-base-200/60 transition-colors cursor-pointer"
          @click="toggleExpand(i)"
        >
          <div class="flex items-center gap-2">
            <!-- 级别色标 -->
            <span
              class="shrink-0 w-1.5 h-1.5 rounded-full"
              :class="{
                'bg-red-500': inferLevel(log.message) === 'ERROR',
                'bg-amber-500': inferLevel(log.message) === 'WARN',
                'bg-blue-500': inferLevel(log.message) === 'INFO',
                'bg-base-content/30': inferLevel(log.message) === 'DEBUG',
              }"
            ></span>
            <!-- 时间 -->
            <span class="shrink-0 text-[0.6rem] text-base-content/30 tabular-nums">
              {{ formatTimestamp(log['@timestamp']) }}
            </span>
            <!-- namespace -->
            <span class="shrink-0 text-[0.6rem] text-indigo-400/60">
              {{ log.kubernetes_namespace }}
            </span>
            <!-- message 截断 -->
            <span class="truncate text-base-content/60">
              {{ expandedRows.has(i) ? log.message : truncateMessage(log.message) }}
            </span>
          </div>
          <!-- 展开详情 -->
          <div v-if="expandedRows.has(i)" class="mt-2 pl-4 text-[0.65rem] text-base-content/40 space-y-0.5">
            <div><span class="text-base-content/25">Pod:</span> {{ log.kubernetes_pod }}</div>
            <div><span class="text-base-content/25">Container:</span> {{ log.kubernetes_container }}</div>
            <div><span class="text-base-content/25">Node:</span> {{ log.kubernetes_node }}</div>
            <div class="mt-1 p-2 rounded bg-base-300/30 whitespace-pre-wrap break-all font-mono text-base-content/50">{{ log.message }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
