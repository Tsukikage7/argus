<script setup lang="ts">
/**
 * TopologyView - 拓扑监控页面 (Phase 3)
 * 
 * 功能：
 * - 全屏交互式拓扑图渲染
 * - 服务状态监控（健康度、错误率、告警数）
 * - 节点点击下钻详情面板
 * - 快速跳转日志查询
 */
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import { getTopologyGraph } from '@/composables/useApi'
import type { TopologyGraph } from '@/types'
import GraphRenderer from '@/components/Topology/GraphRenderer.vue'

const router = useRouter()

// 状态
const loading = ref(true)
const graphData = ref<TopologyGraph | null>(null)
const selectedNodeId = ref<string | null>(null)
const error = ref<string | null>(null)
const graphContainerRef = ref<HTMLDivElement | null>(null)
const containerHeight = ref(500)

// 监听容器高度
let resizeObs: ResizeObserver | null = null

// 选中的节点详情
const selectedNode = computed(() => {
  if (!selectedNodeId.value || !graphData.value) return null
  return graphData.value.nodes.find(n => n.id === selectedNodeId.value) || null
})

// 加载数据
async function fetchData() {
  loading.value = true
  error.value = null
  try {
    graphData.value = await getTopologyGraph()
  } catch (err) {
    console.error('Failed to fetch topology:', err)
    error.value = '无法加载拓扑数据，请检查服务状态'
  } finally {
    loading.value = false
  }
}

// 节点点击处理
function handleNodeClick(nodeId: string) {
  selectedNodeId.value = nodeId
}

// 跳转到日志
function goToLogs(serviceName: string) {
  // 服务名通常是 prj-xxx，查询时可能需要去掉前缀，根据 useTopology.ts 的逻辑统一处理
  router.push({
    path: '/logs',
    query: { namespace: serviceName } // 假设日志页面接受 namespace 参数
  })
}

onMounted(() => {
  fetchData()
  // 监听图容器高度变化
  if (graphContainerRef.value) {
    containerHeight.value = graphContainerRef.value.clientHeight || 500
    resizeObs = new ResizeObserver((entries) => {
      for (const entry of entries) {
        containerHeight.value = entry.contentRect.height
      }
    })
    resizeObs.observe(graphContainerRef.value)
  }
})

onBeforeUnmount(() => {
  resizeObs?.disconnect()
})
</script>

<template>
  <div class="h-[calc(100vh-64px)] flex overflow-hidden bg-base-200/50 relative">
    <!-- 主渲染区域 -->
    <div class="flex-1 relative flex flex-col">
      <!-- 顶部工具栏 -->
      <div class="absolute top-4 left-4 z-10 flex gap-2">
        <div class="bg-base-100/80 backdrop-blur border border-base-300 rounded-lg px-3 py-1.5 shadow-sm flex items-center gap-4 text-xs font-medium">
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
            <span>健康</span>
          </div>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-amber-500"></span>
            <span>亚健康</span>
          </div>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-red-500"></span>
            <span>故障</span>
          </div>
        </div>
        
        <button 
          @click="fetchData" 
          class="btn btn-sm btn-circle btn-ghost bg-base-100/80 backdrop-blur border border-base-300 shadow-sm"
          :class="{ 'loading': loading }"
        >
          <svg v-if="!loading" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      <!-- 拓扑图 -->
      <div v-if="error" class="flex-1 flex items-center justify-center">
        <div class="text-center">
          <p class="text-error font-medium">{{ error }}</p>
          <button @click="fetchData" class="btn btn-outline btn-sm mt-4">重试</button>
        </div>
      </div>
      
      <div ref="graphContainerRef" class="flex-1 w-full bg-base-100 relative">
        <GraphRenderer
          v-if="graphData"
          :graph-data="graphData"
          :height="containerHeight"
          :interacting="true"
          @node-click="handleNodeClick"
        />
        <div v-else-if="loading" class="w-full h-full flex items-center justify-center">
          <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
      </div>
    </div>

    <!-- 右侧详情面板 (Drawer-like) -->
    <Transition name="slide-fade">
      <div 
        v-if="selectedNode"
        class="w-80 h-full bg-base-100 border-l border-base-300 shadow-xl z-20 flex flex-col"
      >
        <!-- 面板头部 -->
        <div class="p-4 border-b border-base-300 flex items-center justify-between bg-base-200/50">
          <h3 class="font-bold text-sm">服务详情</h3>
          <button @click="selectedNodeId = null" class="btn btn-sm btn-ghost btn-circle">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- 服务基本信息 -->
        <div class="p-5 flex-1 overflow-y-auto">
          <div class="flex items-center gap-3 mb-6">
            <div 
              class="w-12 h-12 rounded-xl flex items-center justify-center text-white font-bold text-xl shadow-lg"
              :class="{
                'bg-emerald-500': selectedNode.health === 'healthy',
                'bg-amber-500': selectedNode.health === 'degraded',
                'bg-red-500': selectedNode.health === 'down' || selectedNode.health === 'critical'
              }"
            >
              {{ selectedNode.label[0].toUpperCase() }}
            </div>
            <div>
              <div class="text-xs opacity-50 font-medium">Service Name</div>
              <div class="font-bold text-lg leading-tight">{{ selectedNode.label }}</div>
            </div>
          </div>

          <!-- 指标卡片 -->
          <div class="grid grid-cols-2 gap-3 mb-6">
            <div class="bg-base-200 p-3 rounded-lg border border-base-300">
              <div class="text-[10px] uppercase opacity-50 font-bold mb-1">Health</div>
              <div 
                class="font-bold text-xs flex items-center gap-1.5"
                :class="{
                  'text-emerald-500': selectedNode.health === 'healthy',
                  'text-amber-500': selectedNode.health === 'degraded',
                  'text-red-500': selectedNode.health === 'down' || selectedNode.health === 'critical'
                }"
              >
                <span class="w-2 h-2 rounded-full bg-current"></span>
                {{ selectedNode.health.toUpperCase() }}
              </div>
            </div>
            <div class="bg-base-200 p-3 rounded-lg border border-base-300">
              <div class="text-[10px] uppercase opacity-50 font-bold mb-1">Error Rate</div>
              <div class="font-bold text-xs" :class="{ 'text-red-500': selectedNode.error_rate > 0 }">
                {{ (selectedNode.error_rate * 100).toFixed(2) }}%
              </div>
            </div>
          </div>

          <!-- 告警信息 -->
          <div v-if="selectedNode.alert_count > 0" class="mb-6">
            <div class="text-xs font-bold mb-3 flex items-center justify-between">
              <span>当前告警</span>
              <span class="badge badge-error badge-sm text-[10px] font-bold">{{ selectedNode.alert_count }}</span>
            </div>
            <div class="space-y-2">
              <div class="text-xs p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 flex gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-3.5 w-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                <span>HTTP 5xx 错误率突增，当前 {{ (selectedNode.error_rate * 100).toFixed(1) }}%</span>
              </div>
            </div>
          </div>

          <!-- 快速操作 -->
          <div class="mt-8 space-y-2">
            <button 
              @click="goToLogs(selectedNode.id)" 
              class="btn btn-primary btn-block btn-sm gap-2"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              查看日志
            </button>
            <button class="btn btn-outline btn-block btn-sm gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              查看指标
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.slide-fade-enter-active,
.slide-fade-leave-active {
  transition: all 0.3s ease-out;
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

.glass-card {
  background: rgba(var(--b1) / 0.8);
  backdrop-filter: blur(8px);
}
</style>
