<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'

const store = useTaskStore()

// count-up 动画值
const animSteps = ref(0)
const animTools = ref(0)
const animConf = ref(0)
const animTime = ref(0)

// Sparkline 数据（最近 10 个步骤的工具调用数）
const sparkData = ref<number[]>([])

function animateValue(from: number, to: number, setter: (v: number) => void, duration = 600) {
  if (from === to) return
  const start = performance.now()
  function tick(now: number) {
    const t = Math.min((now - start) / duration, 1)
    const ease = 1 - Math.pow(1 - t, 3) // easeOutCubic
    setter(Math.round(from + (to - from) * ease))
    if (t < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

watch(() => store.stepCount, (n, o) => {
  animateValue(o ?? 0, n, v => { animSteps.value = v })
  // 更新 sparkline
  sparkData.value = [...sparkData.value.slice(-9), n]
})
watch(() => store.toolCount, (n, o) => animateValue(o ?? 0, n, v => { animTools.value = v }))
watch(() => store.confidencePercent, (n, o) => animateValue(o ?? 0, n ?? 0, v => { animConf.value = v }))
watch(() => store.elapsedSeconds, (n, o) => animateValue(o ?? 0, n, v => { animTime.value = v }))

onMounted(() => {
  animSteps.value = store.stepCount
  animTools.value = store.toolCount
  animConf.value = store.confidencePercent ?? 0
  animTime.value = store.elapsedSeconds
})

// Sparkline SVG path
function sparklinePath(data: number[]): string {
  if (data.length < 2) return ''
  const max = Math.max(...data, 1)
  const w = 60, h = 20
  return data.map((v, i) => {
    const x = (i / (data.length - 1)) * w
    const y = h - (v / max) * h
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

const cards = [
  { key: 'steps', label: '推理步骤', color: '#6366f1', icon: 'M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z' },
  { key: 'tools', label: '工具调用', color: '#f59e0b', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
  { key: 'conf', label: '置信度', color: '#10b981', icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' },
  { key: 'time', label: '耗时', color: '#8b5cf6', icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z' },
]
</script>

<template>
  <div class="grid grid-cols-4 gap-3 mb-5">
    <div
      v-for="card in cards"
      :key="card.key"
      class="glass-card rounded-xl p-3 transition-all duration-300 hover:-translate-y-0.5 hover:shadow-lg group"
    >
      <div class="flex items-center justify-between mb-1.5">
        <svg class="w-4 h-4 opacity-40 group-hover:opacity-70 transition-opacity" :style="{ color: card.color }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" :d="card.icon" />
        </svg>
        <!-- Sparkline（仅推理步骤卡片） -->
        <svg v-if="card.key === 'steps' && sparkData.length >= 2" class="w-[60px] h-[20px] opacity-30" viewBox="0 0 60 20">
          <path :d="sparklinePath(sparkData)" fill="none" :stroke="card.color" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </div>
      <div class="text-2xl font-bold text-base-content tabular-nums">
        <template v-if="card.key === 'steps'">{{ animSteps }}</template>
        <template v-else-if="card.key === 'tools'">{{ animTools }}</template>
        <template v-else-if="card.key === 'conf'">{{ animConf > 0 ? animConf + '%' : '—' }}</template>
        <template v-else>{{ animTime > 0 ? animTime + 's' : '—' }}</template>
      </div>
      <div class="text-[0.625rem] text-base-content/40 uppercase tracking-wider mt-0.5">{{ card.label }}</div>
    </div>
  </div>
</template>
