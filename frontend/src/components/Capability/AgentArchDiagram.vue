<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

// 动画状态
const activeNode = ref(0)
let timer: ReturnType<typeof setInterval> | null = null

const nodes = [
  { id: 'user', label: '用户输入', x: 50, y: 30, color: '#6366f1' },
  { id: 'think', label: 'Think', x: 50, y: 120, color: '#818cf8' },
  { id: 'act', label: 'Act', x: 150, y: 200, color: '#f59e0b' },
  { id: 'observe', label: 'Observe', x: 50, y: 280, color: '#10b981' },
  { id: 'tools', label: 'Tools', x: 280, y: 200, color: '#ef4444' },
  { id: 'llm', label: 'LLM', x: 280, y: 120, color: '#8b5cf6' },
  { id: 'result', label: '诊断结论', x: 50, y: 370, color: '#06b6d4' },
]

// 循环动画：user → think → act → tools → observe → think → ... → result
const sequence = [0, 1, 2, 4, 3, 1, 2, 4, 3, 6]

onMounted(() => {
  let idx = 0
  timer = setInterval(() => {
    activeNode.value = sequence[idx % sequence.length]
    idx++
  }, 1200)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="glass-card rounded-xl p-6">
    <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-4">ReAct 推理循环架构</div>

    <div class="flex gap-8">
      <!-- SVG 架构图 -->
      <svg viewBox="0 0 380 420" class="w-80 h-auto flex-shrink-0">
        <!-- 连线 -->
        <line x1="80" y1="50" x2="80" y2="110" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>
        <line x1="80" y1="150" x2="170" y2="195" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>
        <line x1="210" y1="210" x2="270" y2="210" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>
        <line x1="170" y1="225" x2="80" y2="270" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>
        <line x1="80" y1="310" x2="80" y2="140" stroke="oklch(var(--bc) / 0.1)" stroke-width="1.5" stroke-dasharray="2"/>
        <line x1="80" y1="310" x2="80" y2="360" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>
        <line x1="300" y1="150" x2="300" y2="190" stroke="oklch(var(--bc) / 0.15)" stroke-width="2" stroke-dasharray="4"/>

        <!-- 节点 -->
        <g v-for="(node, i) in nodes" :key="node.id">
          <rect
            :x="node.x - 10" :y="node.y - 5"
            :width="node.id === 'result' ? 80 : 60" height="30" rx="6"
            :fill="activeNode === i ? node.color + '40' : 'oklch(var(--b2) / 0.5)'"
            :stroke="activeNode === i ? node.color : 'oklch(var(--bc) / 0.1)'"
            stroke-width="1.5"
            class="transition-all duration-500"
          />
          <text
            :x="node.id === 'result' ? node.x + 30 : node.x + 20" :y="node.y + 15"
            text-anchor="middle"
            :fill="activeNode === i ? node.color : 'oklch(var(--bc) / 0.5)'"
            font-size="11" font-weight="600"
            class="transition-all duration-500"
          >
            {{ node.label }}
          </text>
        </g>

        <!-- 循环箭头标注 -->
        <text x="20" y="210" fill="oklch(var(--bc) / 0.2)" font-size="9" transform="rotate(-90, 20, 210)">
          ReAct Loop
        </text>
      </svg>

      <!-- 说明文字 -->
      <div class="flex-1 space-y-3 text-xs text-base-content/60">
        <div>
          <div class="font-semibold text-indigo-400 mb-1">Think（推理）</div>
          <p>LLM 分析当前信息，决定下一步行动策略</p>
        </div>
        <div>
          <div class="font-semibold text-amber-400 mb-1">Act（执行）</div>
          <p>调用注册的 Tool 执行具体操作（查询日志、分析链路等）</p>
        </div>
        <div>
          <div class="font-semibold text-emerald-400 mb-1">Observe（观察）</div>
          <p>收集 Tool 执行结果，作为下一轮推理的输入</p>
        </div>
        <div>
          <div class="font-semibold text-cyan-400 mb-1">结论输出</div>
          <p>当 Agent 收集到足够信息后，输出结构化诊断结论（根因、置信度、建议）</p>
        </div>
        <div class="pt-2 border-t border-base-300">
          <div class="font-semibold text-base-content/50 mb-1">关键参数</div>
          <p>最大步数: 15 | 自动恢复阈值: 0.8 | 确认阈值: 0.5 | 超时: 5min</p>
        </div>
      </div>
    </div>
  </div>
</template>
