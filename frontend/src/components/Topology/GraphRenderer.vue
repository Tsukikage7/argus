<script setup lang="ts">
/**
 * GraphRenderer - 基于 AntV X6 的服务拓扑图渲染组件
 *
 * 三种工作模式：
 *   - 诊断模式：根据 diagnosis 高亮故障节点
 *   - 回放模式：根据 impactReport 显示各服务状态
 *   - 全量模式：通过 updateGraph 渲染后端返回的拓扑数据
 */
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useTopology } from '@/composables/useTopology'
import type { Diagnosis, ImpactReport, TopologyNode, TopologyEdge } from '@/types'

const props = defineProps<{
  impactReport?: ImpactReport | null
  diagnosis?: Diagnosis | null
  highlightNamespace?: string | null
  // 拓扑全量数据（可选，用于独立拓扑页）
  graphData?: { nodes: TopologyNode[]; edges: TopologyEdge[] } | null
  // 画布高度，默认 250
  height?: number | string
  // 是否允许交互（缩放、平移、点击）
  interacting?: boolean
}>()

const emit = defineEmits<{
  (e: 'node-click', nodeId: string): void
}>()

// 容器 DOM 引用
const containerRef = ref<HTMLDivElement | null>(null)

// topology composable
const { 
  initGraph, 
  updateNodes, 
  updateGraph, 
  highlightNode, 
  setOnNodeClick, 
  dispose, 
  resizeGraph 
} = useTopology()

// ResizeObserver 监听容器宽度变化
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (!containerRef.value) return

  // 初始化 X6 图实例
  initGraph(containerRef.value, { 
    height: props.height, 
    interacting: props.interacting 
  })

  // 设置点击回调
  if (props.interacting) {
    setOnNodeClick((nodeId) => {
      emit('node-click', nodeId)
    })
  }

  // 初始数据渲染
  if (props.graphData) {
    updateGraph(props.graphData.nodes, props.graphData.edges)
  } else if (props.diagnosis || props.impactReport) {
    updateNodes(props.diagnosis ?? null, props.impactReport ?? null)
  }

  // 监听容器宽度变化
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const h = typeof props.height === 'number' ? props.height : containerRef.value?.clientHeight || 250
      resizeGraph(entry.contentRect.width, h)
    }
  })
  resizeObserver.observe(containerRef.value)
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  dispose()
})

// 监听数据变化
watch(
  () => props.graphData,
  (data) => {
    if (data) updateGraph(data.nodes, data.edges)
  },
  { deep: true }
)

watch(
  [() => props.diagnosis, () => props.impactReport],
  ([diag, report]) => {
    if (!props.graphData) {
      updateNodes(diag ?? null, report ?? null)
    }
  }
)

watch(
  () => props.highlightNamespace,
  (ns) => {
    highlightNode(ns ?? null)
  }
)
</script>

<template>
  <div
    ref="containerRef"
    class="w-full h-full overflow-hidden"
    :style="{ height: typeof height === 'number' ? height + 'px' : height }"
  />
</template>
