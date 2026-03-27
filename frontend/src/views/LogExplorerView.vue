<script setup lang="ts">
/**
 * LogExplorerView - 日志探索器页面 (Phase 4)
 *
 * 功能：
 * - 左侧分面过滤（namespace/service/level/pod）
 * - 中间日志列表（虚拟滚动、行展开）
 * - 右侧上下文抽屉（按 request_uuid 展示前后文）
 * - 搜索栏（关键词 + 级别 + 时间范围联动全局）
 */
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useTimeRange } from '@/composables/useTimeRange'
import { queryFaultLogs, getLogFacets } from '@/composables/useApi'
import type { FaultLogEntry, LogFacets } from '@/composables/useApi'
import LogSearchBar from '@/components/Logs/LogSearchBar.vue'
import LogFacetPanel from '@/components/Logs/LogFacetPanel.vue'
import LogTable from '@/components/Logs/LogTable.vue'
import LogContextDrawer from '@/components/Logs/LogContextDrawer.vue'

const route = useRoute()
const { queryParam } = useTimeRange()

// 状态
const loading = ref(false)
const facetsLoading = ref(false)
const logs = ref<FaultLogEntry[]>([])
const total = ref(0)
const facets = ref<LogFacets | null>(null)
const error = ref<string | null>(null)

// 过滤条件
const selectedNamespace = ref('')
const selectedService = ref('')
const selectedLevel = ref('')
const keyword = ref('')
const activeLevels = ref<string[]>(['ERROR', 'WARN'])

// 上下文抽屉
const contextUUID = ref<string | null>(null)

// 加载日志
async function fetchLogs() {
  loading.value = true
  error.value = null
  try {
    const levelFilter = activeLevels.value.length > 0 ? activeLevels.value.join(',') : undefined
    const result = await queryFaultLogs({
      namespace: selectedNamespace.value || undefined,
      service: selectedService.value || undefined,
      keyword: keyword.value || undefined,
      level: selectedLevel.value || levelFilter,
      time_range: `last ${queryParam.value}`,
      limit: 100,
    })
    logs.value = result.logs
    total.value = result.total
  } catch (err) {
    console.error('Failed to fetch logs:', err)
    error.value = '无法加载日志数据'
  } finally {
    loading.value = false
  }
}

// 加载分面
async function fetchFacets() {
  facetsLoading.value = true
  try {
    facets.value = await getLogFacets(`last ${queryParam.value}`)
  } catch (err) {
    console.error('Failed to fetch facets:', err)
  } finally {
    facetsLoading.value = false
  }
}

// 事件处理
function onSearch(kw: string) {
  keyword.value = kw
  fetchLogs()
}

function onToggleLevel(level: string) {
  const idx = activeLevels.value.indexOf(level)
  if (idx >= 0) {
    activeLevels.value.splice(idx, 1)
  } else {
    activeLevels.value.push(level)
  }
  fetchLogs()
}

function onClear() {
  keyword.value = ''
  activeLevels.value = ['ERROR', 'WARN']
  selectedNamespace.value = ''
  selectedService.value = ''
  selectedLevel.value = ''
  fetchLogs()
  fetchFacets()
}

function onSelectNamespace(ns: string) {
  selectedNamespace.value = ns
  fetchLogs()
}

function onSelectService(svc: string) {
  selectedService.value = svc
  fetchLogs()
}

function onSelectLevel(level: string) {
  selectedLevel.value = level
  fetchLogs()
}

function onViewContext(uuid: string) {
  contextUUID.value = uuid
}

// 从路由参数初始化过滤条件（支持拓扑跳转）
onMounted(() => {
  if (route.query.namespace) {
    selectedNamespace.value = route.query.namespace as string
  }
  if (route.query.service) {
    selectedService.value = route.query.service as string
  }
  fetchLogs()
  fetchFacets()
})

// 时间范围变化时重新加载
watch(queryParam, () => {
  fetchLogs()
  fetchFacets()
})
</script>

<template>
  <div class="h-[calc(100vh-64px)] flex flex-col overflow-hidden">
    <!-- 搜索栏 -->
    <div class="p-3 border-b border-base-300 bg-base-100">
      <LogSearchBar
        :active-levels="activeLevels"
        @search="onSearch"
        @toggle-level="onToggleLevel"
        @clear="onClear"
      />
    </div>

    <!-- 主内容区 -->
    <div class="flex-1 flex min-h-0">
      <!-- 左侧分面 -->
      <div class="border-r border-base-300 bg-base-100 p-3">
        <LogFacetPanel
          :facets="facets"
          :loading="facetsLoading"
          :selected-namespace="selectedNamespace"
          :selected-service="selectedService"
          :selected-level="selectedLevel"
          @select-namespace="onSelectNamespace"
          @select-service="onSelectService"
          @select-level="onSelectLevel"
        />
      </div>

      <!-- 中间日志列表 -->
      <div class="flex-1 flex flex-col min-w-0">
        <!-- 错误提示 -->
        <div v-if="error" class="p-3">
          <div class="alert alert-error text-sm">
            <span>{{ error }}</span>
            <button class="btn btn-sm btn-ghost" @click="fetchLogs">重试</button>
          </div>
        </div>

        <!-- 空状态提示 -->
        <div v-else-if="!loading && logs.length === 0" class="flex-1 flex items-center justify-center">
          <div class="text-center space-y-3">
            <div class="text-4xl opacity-30">📋</div>
            <p class="text-base-content/60">未找到匹配的日志</p>
            <p class="text-sm text-base-content/40">尝试调整过滤条件或扩大时间范围</p>
            <div class="flex gap-2 justify-center">
              <button class="btn btn-sm btn-outline" @click="onClear">重置过滤条件</button>
              <a href="/replay" class="btn btn-sm btn-ghost">前往回放中心生成数据</a>
            </div>
          </div>
        </div>

        <LogTable
          v-else
          :logs="logs"
          :loading="loading"
          :total="total"
          @view-context="onViewContext"
        />
      </div>

      <!-- 右侧上下文抽屉 -->
      <LogContextDrawer
        :request-u-u-i-d="contextUUID"
        @close="contextUUID = null"
      />
    </div>
  </div>
</template>
