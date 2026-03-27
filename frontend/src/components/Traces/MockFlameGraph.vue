<script setup lang="ts">
/**
 * MockFlameGraph - 纯 SVG 实现的火焰图组件
 */
import { ref, computed } from 'vue'
import type { FlameNode } from '@/types'

const props = defineProps<{
  data: FlameNode
}>()

const width = 1000
const rowHeight = 30
const selectedNode = ref<FlameNode | null>(null)

// 递归计算层级和宽度
interface RenderNode extends FlameNode {
  x: number
  y: number
  w: number
  depth: number
}

function processNodes(node: FlameNode, x = 0, y = 0, totalW = width, depth = 0): RenderNode[] {
  const result: RenderNode[] = []
  const w = (node.value / props.data.value) * totalW
  
  result.push({
    ...node,
    x,
    y,
    w,
    depth
  })
  
  let currentX = x
  if (node.children) {
    node.children.forEach(child => {
      const childW = (child.value / node.value) * w
      result.push(...processNodes(child, currentX, y + rowHeight, w, depth + 1))
      currentX += childW
    })
  }
  
  return result
}

const renderNodes = computed(() => processNodes(props.data))

const totalHeight = computed(() => {
  const maxDepth = Math.max(...renderNodes.value.map(n => n.depth))
  return (maxDepth + 1) * rowHeight
})

function getColor(depth: number, name: string) {
  // 生成稳定的颜色
  const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
  const hue = (hash % 30) + 20 // 橙黄色系
  const saturation = 70 + (depth * 5)
  const lightness = 60 - (depth * 5)
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`
}

function selectNode(node: FlameNode) {
  selectedNode.value = node
}
</script>

<template>
  <div class="flame-graph-container w-full bg-base-100 rounded-lg p-4 shadow-inner overflow-hidden">
    <div class="flex justify-between items-center mb-4">
      <div class="text-sm font-bold">调用火焰图 (SVG)</div>
      <div v-if="selectedNode" class="text-xs badge badge-outline">
        {{ selectedNode.name }}: {{ selectedNode.value }}ms
      </div>
    </div>
    
    <div class="overflow-x-auto">
      <svg
        :width="width"
        :height="totalHeight"
        :viewBox="`0 0 ${width} ${totalHeight}`"
        class="cursor-pointer font-sans"
      >
        <g v-for="(node, index) in renderNodes" :key="index" @click="selectNode(node)">
          <rect 
            :x="node.x" 
            :y="node.y" 
            :width="node.w" 
            :height="rowHeight - 1" 
            :fill="getColor(node.depth, node.name)"
            class="transition-opacity hover:opacity-80"
          />
          <text 
            v-if="node.w > 40"
            :x="node.x + 5" 
            :y="node.y + 18" 
            class="text-[10px] fill-current pointer-events-none select-none"
          >
            {{ node.name }}
          </text>
        </g>
      </svg>
    </div>

    <div class="mt-4 flex gap-4 text-[10px] opacity-60">
      <div class="flex items-center gap-1">
        <div class="w-2 h-2 bg-warning"></div> 耗时较长
      </div>
      <div class="flex items-center gap-1">
        <div class="w-2 h-2 bg-success"></div> 正常执行
      </div>
    </div>
  </div>
</template>

<style scoped>
svg {
  shape-rendering: crispEdges;
}
</style>
