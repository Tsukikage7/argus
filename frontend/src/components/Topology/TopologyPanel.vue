<script setup lang="ts">
import { computed } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'
import type { ImpactReport } from '@/types'
import GraphRenderer from './GraphRenderer.vue'

const props = defineProps<{
  impactReport?: ImpactReport | null
}>()

const store = useTaskStore()

// 是否有诊断数据（用于判断空状态）
const hasData = computed(() => {
  return !!store.diagnosis || !!props.impactReport || store.steps.length > 0
})

// 从 steps 中提取当前正在查询的 namespace（推理过程中高亮）
const highlightNamespace = computed(() => {
  const steps = store.steps
  if (!steps || steps.length === 0) return null
  for (let i = steps.length - 1; i >= 0; i--) {
    const step = steps[i]
    if (step.action?.tool === 'es_query_logs' && step.action.params?.namespace) {
      return step.action.params.namespace as string
    }
  }
  return null
})
</script>

<template>
  <!-- 调用链路拓扑面板 -->
  <div class="glass-card rounded-xl overflow-hidden flex flex-col">
    <!-- 卡片标题 -->
    <div class="px-4 py-2.5 border-b border-base-300 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-1.5 text-[0.8125rem] font-semibold text-base-content/70">
        <svg class="w-3.5 h-3.5 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/>
        </svg>
        调用链路
      </div>

      <!-- 回放模式：blast_radius 徽章 -->
      <span
        v-if="props.impactReport?.blast_radius"
        class="inline-flex items-center px-2 py-0.5 rounded-full text-[0.625rem] font-bold tracking-wider"
        :class="{
          'bg-emerald-900/50 text-emerald-300': props.impactReport.blast_radius === 'low',
          'bg-amber-900/50 text-amber-300': props.impactReport.blast_radius === 'medium',
          'bg-red-900/50 text-red-300': props.impactReport.blast_radius === 'high',
          'bg-red-950 text-red-300': props.impactReport.blast_radius === 'critical',
        }"
      >
        {{ props.impactReport.blast_radius.toUpperCase() }}
      </span>
    </div>

    <!-- 内容区域：固定高度容器，overflow hidden 防止画布溢出 -->
    <div class="relative" style="height: 220px; min-height: 220px;">
      <!-- 空状态 -->
      <div
        v-if="!hasData"
        class="absolute inset-0 flex items-center justify-center text-sm text-base-content/30"
      >
        开始诊断后展示调用链路
      </div>

      <!-- X6 图渲染区域 -->
      <div v-else class="w-full h-full overflow-hidden">
        <GraphRenderer
          :diagnosis="store.diagnosis"
          :impact-report="props.impactReport"
          :highlight-namespace="highlightNamespace"
          :height="220"
        />
      </div>
    </div>
  </div>
</template>
