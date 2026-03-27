<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listScenarios } from '@/composables/useApi'
import type { Scenario, ReplayType } from '@/types'

const props = defineProps<{
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'replay', params: {
    type: ReplayType
    scenario: string
    config: {
      fault_intensity: number
      traffic_rate_multiplier: number
      auto_diagnose: boolean
    }
  }): void
}>()

const scenarios = ref<Scenario[]>([])
const selectedScenario = ref('')
const replayType = ref<ReplayType>('fault')
const faultIntensity = ref(1.0)
const trafficRate = ref(1.0)
const autoDiagnose = ref(true)

// 按类型分组场景
const presetScenarios = computed(() => scenarios.value.filter(s => s.type !== 'captured'))
const capturedScenarios = computed(() => scenarios.value.filter(s => s.type === 'captured'))

// 加载场景列表
async function fetchScenarios() {
  try {
    scenarios.value = await listScenarios()
    if (scenarios.value.length > 0) {
      selectedScenario.value = scenarios.value[0].name
    }
  } catch (e) {
    console.error('获取场景列表失败:', e)
  }
}

function handleReplay() {
  if (!selectedScenario.value) return
  emit('replay', {
    type: replayType.value,
    scenario: selectedScenario.value,
    config: {
      fault_intensity: faultIntensity.value,
      traffic_rate_multiplier: trafficRate.value,
      auto_diagnose: autoDiagnose.value,
    },
  })
}

onMounted(() => {
  fetchScenarios()
})
</script>

<template>
  <!-- 回放模式输入区 -->
  <div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <!-- 左列：场景选择 + 类型 -->
      <div class="flex flex-col gap-3">
        <div>
          <label class="block text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            故障场景
          </label>
          <select
            v-model="selectedScenario"
            class="w-full rounded-lg px-3 py-2 text-sm bg-base-200 border border-base-300
                   text-base-content focus:outline-none focus:border-indigo-500 transition-all"
          >
            <option v-if="scenarios.length === 0" value="">加载中...</option>
            <optgroup v-if="presetScenarios.length > 0" label="预置场景">
              <option
                v-for="s in presetScenarios"
                :key="s.name"
                :value="s.name"
              >
                {{ s.name }} — {{ s.description }}
              </option>
            </optgroup>
            <optgroup v-if="capturedScenarios.length > 0" label="沉淀场景">
              <option
                v-for="s in capturedScenarios"
                :key="s.name"
                :value="s.name"
              >
                {{ s.name }} — {{ s.description }}
                {{ s.confidence ? ` [${Math.round(s.confidence * 100)}%]` : '' }}
              </option>
            </optgroup>
          </select>
        </div>

        <div>
          <label class="block text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            回放类型
          </label>
          <div class="flex gap-2">
            <button
              class="px-3.5 py-1.5 rounded-md text-xs font-medium cursor-pointer transition-all border"
              :class="replayType === 'fault'
                ? 'bg-indigo-600 text-white border-indigo-600'
                : 'bg-transparent text-base-content/50 border-base-300 hover:text-base-content'"
              @click="replayType = 'fault'"
            >
              故障回放
            </button>
            <button
              class="px-3.5 py-1.5 rounded-md text-xs font-medium cursor-pointer transition-all border"
              :class="replayType === 'traffic'
                ? 'bg-indigo-600 text-white border-indigo-600'
                : 'bg-transparent text-base-content/50 border-base-300 hover:text-base-content'"
              @click="replayType = 'traffic'"
            >
              流量回放
            </button>
          </div>
        </div>
      </div>

      <!-- 右列：参数配置 -->
      <div class="flex flex-col gap-3">
        <!-- 故障强度滑块 -->
        <div>
          <label class="block text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            故障强度
            <span class="text-xs font-mono normal-case text-base-content/70 ml-1">{{ faultIntensity.toFixed(1) }}</span>
          </label>
          <div class="flex items-center gap-2">
            <span class="text-xs text-base-content/30">0.1</span>
            <input
              v-model.number="faultIntensity"
              type="range"
              min="0.1"
              max="2.0"
              step="0.1"
              class="flex-1 h-1 appearance-none bg-base-300 rounded-full outline-none
                     [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3.5
                     [&::-webkit-slider-thumb]:h-3.5 [&::-webkit-slider-thumb]:rounded-full
                     [&::-webkit-slider-thumb]:bg-indigo-500 [&::-webkit-slider-thumb]:cursor-pointer"
            />
            <span class="text-xs text-base-content/30">2.0</span>
          </div>
        </div>

        <!-- 流量倍率滑块（仅流量回放时显示） -->
        <div v-if="replayType === 'traffic'">
          <label class="block text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            流量倍率
            <span class="text-xs font-mono normal-case text-base-content/70 ml-1">{{ trafficRate.toFixed(1) }}</span>
          </label>
          <div class="flex items-center gap-2">
            <span class="text-xs text-base-content/30">0.5</span>
            <input
              v-model.number="trafficRate"
              type="range"
              min="0.5"
              max="5.0"
              step="0.5"
              class="flex-1 h-1 appearance-none bg-base-300 rounded-full outline-none
                     [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3.5
                     [&::-webkit-slider-thumb]:h-3.5 [&::-webkit-slider-thumb]:rounded-full
                     [&::-webkit-slider-thumb]:bg-indigo-500 [&::-webkit-slider-thumb]:cursor-pointer"
            />
            <span class="text-xs text-base-content/30">5.0</span>
          </div>
        </div>

        <!-- 自动诊断开关 -->
        <div class="flex items-center gap-2">
          <label class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40">
            自动诊断
          </label>
          <div
            class="relative w-9 h-5 rounded-full cursor-pointer transition-colors duration-200"
            :class="autoDiagnose ? 'bg-indigo-600' : 'bg-base-300'"
            @click="autoDiagnose = !autoDiagnose"
          >
            <div
              class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform duration-200"
              :class="autoDiagnose ? 'translate-x-[18px]' : 'translate-x-0.5'"
            ></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 开始回放按钮 -->
    <div class="mt-3 flex justify-end">
      <button
        class="px-5 py-2 rounded-lg text-white text-sm font-semibold transition-all
               hover:brightness-110 active:scale-[.97] disabled:opacity-40 disabled:pointer-events-none"
        style="background: linear-gradient(135deg, #06b6d4, #3b82f6)"
        :disabled="props.loading || !selectedScenario"
        @click="handleReplay"
      >
        {{ props.loading ? '创建中…' : '创建回放任务' }}
      </button>
    </div>
  </div>
</template>
