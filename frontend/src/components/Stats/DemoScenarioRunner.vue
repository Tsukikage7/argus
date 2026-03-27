<script setup lang="ts">
import { ref, computed } from 'vue'
import { startReplay, listScenarios, type Scenario } from '@/composables/useApi'
import { useRouter } from 'vue-router'

const router = useRouter()
const scenarios = ref<Scenario[]>([])
const selectedScenario = ref('')
const running = ref(false)
const currentPhase = ref('')
const phases = ['生成故障数据', '触发告警', '自动诊断', '展示结果']
const phaseIndex = ref(-1)
const error = ref('')

async function loadScenarios() {
  try {
    scenarios.value = await listScenarios()
    if (scenarios.value.length > 0) {
      selectedScenario.value = scenarios.value[0].name
    }
  } catch (e) {
    console.error('加载场景失败:', e)
  }
}
loadScenarios()

const progress = computed(() => {
  if (phaseIndex.value < 0) return 0
  return Math.round(((phaseIndex.value + 1) / phases.length) * 100)
})

async function runDemo() {
  if (!selectedScenario.value || running.value) return
  running.value = true
  error.value = ''

  try {
    // 阶段 1: 生成故障
    phaseIndex.value = 0
    currentPhase.value = phases[0]
    await sleep(800)

    // 阶段 2: 触发告警（通过回放引擎）
    phaseIndex.value = 1
    currentPhase.value = phases[1]
    const result = await startReplay({
      type: 'fault',
      scenario: selectedScenario.value,
      config: { auto_diagnose: true, fault_intensity: 0.8 },
    })
    await sleep(600)

    // 阶段 3: 自动诊断
    phaseIndex.value = 2
    currentPhase.value = phases[2]
    await sleep(1000)

    // 阶段 4: 展示结果
    phaseIndex.value = 3
    currentPhase.value = phases[3]
    await sleep(500)

    // 跳转到诊断页面
    if (result.session_id) {
      router.push({ name: 'replay', query: { session: result.session_id } })
    }
  } catch (e: any) {
    error.value = e.message || '演示执行失败'
  } finally {
    running.value = false
    phaseIndex.value = -1
    currentPhase.value = ''
  }
}

function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms))
}
</script>

<template>
  <div class="glass-card rounded-xl overflow-hidden">
    <div class="px-4 py-2.5 border-b border-base-300 text-[0.8125rem] font-semibold text-base-content/70">
      一键演示
    </div>
    <div class="p-4 space-y-3">
      <!-- 场景选择 -->
      <div>
        <label class="text-xs text-base-content/50 mb-1 block">选择故障场景</label>
        <select
          v-model="selectedScenario"
          class="select select-sm select-bordered w-full bg-base-200 text-sm"
          :disabled="running"
        >
          <option v-for="s in scenarios" :key="s.name" :value="s.name">
            {{ s.name }} — {{ s.description }}
          </option>
        </select>
      </div>

      <!-- 进度 -->
      <div v-if="running" class="space-y-2">
        <div class="flex items-center gap-2">
          <span class="loading loading-spinner loading-xs text-indigo-400"></span>
          <span class="text-xs text-base-content/60">{{ currentPhase }}</span>
        </div>
        <div class="h-1.5 rounded-full bg-base-300/50 overflow-hidden">
          <div
            class="h-full bg-indigo-500 transition-all duration-500"
            :style="{ width: `${progress}%` }"
          ></div>
        </div>
        <div class="flex justify-between text-[0.5625rem] text-base-content/40">
          <span
            v-for="(phase, i) in phases"
            :key="phase"
            :class="{ 'text-indigo-400 font-semibold': i <= phaseIndex }"
          >
            {{ phase }}
          </span>
        </div>
      </div>

      <!-- 错误 -->
      <div v-if="error" class="text-xs text-red-400 bg-red-500/10 rounded-lg p-2">
        {{ error }}
      </div>

      <!-- 启动按钮 -->
      <button
        class="w-full px-4 py-2 rounded-lg text-sm font-semibold text-white transition-all
               hover:brightness-110 active:scale-[.97] disabled:opacity-40 disabled:pointer-events-none"
        style="background: linear-gradient(135deg, #6366f1, #818cf8)"
        :disabled="running || !selectedScenario"
        @click="runDemo"
      >
        {{ running ? '演示进行中…' : '开始一键演示' }}
      </button>
    </div>
  </div>
</template>
