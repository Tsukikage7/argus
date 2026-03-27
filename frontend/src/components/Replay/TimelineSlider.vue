<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import type { ReplayStatus } from '@/types'

// 组件 Props 定义
const props = defineProps<{
  /** 当前回放状态 */
  status: ReplayStatus
  /** 回放开始时间（ISO 字符串） */
  startTime?: string
  /** 回放结束时间（ISO 字符串） */
  endTime?: string
  /** SSE 推送的进度文本 */
  progress?: string
}>()

// 回放状态步骤定义（顺序）
const STEPS: { key: string; label: string }[] = [
  { key: 'generating',       label: '生成数据' },
  { key: 'diagnosing',       label: '诊断分析' },
  { key: 'computing_impact', label: '计算影响' },
  { key: 'completed',        label: '完成' },
]

// 状态步骤索引映射（用于高亮判断）
const STATUS_STEP_INDEX: Record<string, number> = {
  pending:    -1,
  generating:  0,
  diagnosing:  1,
  // computing_impact 在 SSE progress 中推断，视为 diagnosing 之后
  completed:   3,
  failed:      3,
}

// 实时计时器：每秒更新 now 驱动 elapsedSeconds 响应式刷新
const now = ref(new Date())
let tickTimer: ReturnType<typeof setInterval> | null = null

watch(
  () => props.status,
  (s) => {
    if (s === 'generating' || s === 'diagnosing') {
      if (!tickTimer) {
        tickTimer = setInterval(() => { now.value = new Date() }, 1000)
      }
    } else {
      if (tickTimer) {
        clearInterval(tickTimer)
        tickTimer = null
      }
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  if (tickTimer) {
    clearInterval(tickTimer)
    tickTimer = null
  }
})

// 当前步骤索引
const currentStepIndex = computed((): number => {
  // 如果 progress 文本包含"影响"，判定正在计算影响面
  if (props.status === 'diagnosing' && props.progress?.includes('影响')) {
    return 2
  }
  return STATUS_STEP_INDEX[props.status] ?? -1
})

// 进度条样式类（根据状态决定颜色与动画）
const barClass = computed((): string => {
  switch (props.status) {
    case 'pending':
      return 'bg-base-300 w-0'
    case 'generating':
      return 'animate-pulse bg-blue-500 w-1/3'
    case 'diagnosing':
      return 'animate-pulse bg-indigo-500 w-2/3'
    case 'completed':
      return 'bg-emerald-500 w-full transition-all duration-700'
    case 'failed':
      return 'bg-red-500 w-full transition-all duration-500'
    default:
      return 'bg-base-300 w-0'
  }
})

// 状态标签文字颜色
const statusTextClass = computed((): string => {
  switch (props.status) {
    case 'pending':    return 'text-base-content/30'
    case 'generating': return 'text-blue-400'
    case 'diagnosing': return 'text-indigo-400'
    case 'completed':  return 'text-emerald-400'
    case 'failed':     return 'text-red-400'
    default:           return 'text-base-content/30'
  }
})

// 状态标签文字
const statusLabel = computed((): string => {
  switch (props.status) {
    case 'pending':    return '等待开始'
    case 'generating': return '生成中'
    case 'diagnosing': return '诊断中'
    case 'completed':  return '已完成'
    case 'failed':     return '失败'
    default:           return '—'
  }
})

// 显示的进度文本（优先用 SSE 推送内容）
const progressText = computed((): string => {
  if (props.progress) return props.progress
  return statusLabel.value
})

// 格式化开始时间
const startTimeStr = computed((): string => {
  if (!props.startTime) return '—'
  return new Date(props.startTime).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
})

// 计算已用时长（秒）— 使用响应式 now 驱动实时更新
const elapsedSeconds = computed((): number => {
  if (!props.startTime) return 0
  const end = props.endTime ? new Date(props.endTime) : now.value
  return Math.floor((end.getTime() - new Date(props.startTime).getTime()) / 1000)
})

// 格式化已用时长
const elapsedStr = computed((): string => {
  if (!props.startTime) return '—'
  const s = elapsedSeconds.value
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m ${s % 60}s`
})

// 判断某个步骤是否已激活（已经到达或超过该步骤）
function isStepActive(stepIndex: number): boolean {
  return currentStepIndex.value >= stepIndex
}

// 判断某个步骤是否是当前正在进行的步骤
function isStepCurrent(stepIndex: number): boolean {
  return currentStepIndex.value === stepIndex &&
    props.status !== 'completed' &&
    props.status !== 'failed'
}
</script>

<template>
  <!-- 回放时间线进度控制条（矮长条形，适合嵌入面板顶部） -->
  <div class="bg-base-100 border border-base-300 rounded-xl px-4 py-3">

    <!-- 顶部行：状态标签 + 进度文本 + 时间信息 -->
    <div class="flex items-center gap-3 mb-2">
      <!-- 状态圆点 + 标签 -->
      <div class="flex items-center gap-1.5 flex-shrink-0">
        <div
          class="w-2 h-2 rounded-full flex-shrink-0"
          :class="{
            'bg-base-300':                      status === 'pending',
            'bg-blue-500 animate-pulse':         status === 'generating',
            'bg-indigo-500 animate-pulse':       status === 'diagnosing',
            'bg-emerald-500':                    status === 'completed',
            'bg-red-500':                        status === 'failed',
          }"
        ></div>
        <span class="text-[0.6875rem] font-semibold uppercase tracking-wider" :class="statusTextClass">
          {{ statusLabel }}
        </span>
      </div>

      <!-- 进度文本（SSE 推送内容） -->
      <span class="text-[0.6875rem] text-base-content/40 flex-1 truncate">
        {{ progressText }}
      </span>

      <!-- 时间信息 -->
      <div class="flex items-center gap-3 flex-shrink-0 text-[0.625rem] text-base-content/30 tabular-nums">
        <span v-if="startTime">开始 {{ startTimeStr }}</span>
        <span v-if="startTime">用时 {{ elapsedStr }}</span>
      </div>
    </div>

    <!-- 进度条 -->
    <div class="h-1 bg-base-300 rounded-full overflow-hidden mb-3">
      <div class="h-full rounded-full transition-all duration-500" :class="barClass"></div>
    </div>

    <!-- 步骤指示器：4 个圆点 + 连线 -->
    <div class="flex items-center">
      <template v-for="(step, index) in STEPS" :key="step.key">
        <!-- 步骤圆点 -->
        <div class="flex flex-col items-center flex-shrink-0">
          <div
            class="w-2 h-2 rounded-full border transition-all duration-300"
            :class="{
              /* 已完成步骤：实心绿色（failed 时用红色） */
              'bg-emerald-500 border-emerald-500':
                isStepActive(index) && status !== 'failed',
              'bg-red-500 border-red-500':
                isStepActive(index) && status === 'failed',
              /* 当前进行中步骤：脉冲动画 */
              'animate-pulse': isStepCurrent(index),
              /* 未到达步骤：空心灰色 */
              'bg-transparent border-base-content/20': !isStepActive(index),
            }"
          ></div>
          <!-- 步骤标签 -->
          <span
            class="mt-1 text-[0.5625rem] whitespace-nowrap transition-colors duration-300"
            :class="isStepActive(index)
              ? (status === 'failed' ? 'text-red-400' : 'text-base-content/60')
              : 'text-base-content/20'"
          >
            {{ step.label }}
          </span>
        </div>

        <!-- 步骤之间的连线（最后一个步骤后不加） -->
        <div
          v-if="index < STEPS.length - 1"
          class="flex-1 h-px mx-1 mb-4 transition-colors duration-500"
          :class="isStepActive(index + 1)
            ? (status === 'failed' ? 'bg-red-500/50' : 'bg-emerald-500/50')
            : 'bg-base-content/10'"
        ></div>
      </template>
    </div>

  </div>
</template>
