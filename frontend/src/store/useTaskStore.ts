import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AppMode, Task, Step, Diagnosis, TimelineEvent, TaskStatus, TopologyConfig, ImpactReport } from '@/types'
import { apiFetch } from '@/composables/useApi'

/** 默认拓扑（后端不可用时的回退） */
const DEFAULT_TOPOLOGY: TopologyConfig = {
  services: ['prj-apigateway', 'prj-ubill', 'prj-uresource', 'prj-uhost', 'prj-unet', 'prj-udb'],
  edges: [
    ['prj-apigateway', 'prj-ubill'],
    ['prj-apigateway', 'prj-uresource'],
    ['prj-apigateway', 'prj-uhost'],
    ['prj-ubill', 'prj-uresource'],
    ['prj-uresource', 'prj-unet'],
    ['prj-uhost', 'prj-udb'],
  ],
  chains: {
    'prj-ubill': ['prj-apigateway', 'prj-ubill'],
    'prj-uresource': ['prj-apigateway', 'prj-uresource'],
    'prj-uhost': ['prj-apigateway', 'prj-uhost'],
    'prj-unet': ['prj-apigateway', 'prj-uresource', 'prj-unet'],
    'prj-udb': ['prj-apigateway', 'prj-uhost', 'prj-udb'],
    'prj-apigateway': ['prj-apigateway'],
  },
}

/** 动态拓扑（从后端 API 获取） */
export const TOPOLOGY = ref<TopologyConfig>({ ...DEFAULT_TOPOLOGY })

/** 从后端加载拓扑配置 */
export async function loadTopology(): Promise<void> {
  try {
    const data = await apiFetch<TopologyConfig>('/api/v1/topology')
    TOPOLOGY.value = data
  } catch {
    // 后端不可用时使用默认拓扑
    TOPOLOGY.value = { ...DEFAULT_TOPOLOGY }
  }
}

export const useTaskStore = defineStore('task', () => {
  // 应用模式
  const mode = ref<AppMode>('diagnose')

  // 当前任务
  const currentTaskId = ref<string | null>(null)
  const taskStatus = ref<TaskStatus>('pending')
  const steps = ref<Step[]>([])
  const diagnosis = ref<Diagnosis | null>(null)

  // 统计
  const stepCount = computed(() => steps.value.length)
  const toolCount = computed(() => steps.value.filter(s => s.action).length)
  const confidencePercent = computed(() =>
    diagnosis.value ? Math.round(diagnosis.value.confidence * 100) : null
  )

  // 计时
  const startTime = ref<number | null>(null)
  const elapsedSeconds = ref(0)
  let timerHandle: ReturnType<typeof setInterval> | null = null

  // 时间线
  const timeline = ref<TimelineEvent[]>([])

  // 诊断历史（内存缓存）
  const history = ref<Task[]>([])

  // 回放相关
  const replaySessionId = ref<string | null>(null)
  const impactReport = ref<ImpactReport | null>(null)

  function setMode(m: AppMode) {
    mode.value = m
  }

  function startTimer() {
    startTime.value = Date.now()
    elapsedSeconds.value = 0
    if (timerHandle) clearInterval(timerHandle)
    timerHandle = setInterval(() => {
      if (startTime.value) {
        elapsedSeconds.value = Math.round((Date.now() - startTime.value) / 1000)
      }
    }, 500)
  }

  function stopTimer() {
    if (timerHandle) {
      clearInterval(timerHandle)
      timerHandle = null
    }
  }

  function reset() {
    currentTaskId.value = null
    taskStatus.value = 'pending'
    steps.value = []
    diagnosis.value = null
    timeline.value = []
    replaySessionId.value = null
    impactReport.value = null
    stopTimer()
    elapsedSeconds.value = 0
  }

  function addStep(step: Step) {
    steps.value.push(step)
  }

  function setDiagnosis(d: Diagnosis) {
    diagnosis.value = d
  }

  function setStatus(s: TaskStatus) {
    taskStatus.value = s
  }

  function addTimeline(level: TimelineEvent['level'], text: string) {
    timeline.value.push({ level, text, time: new Date() })
  }

  function addHistory(task: Task) {
    history.value.unshift(task)
    if (history.value.length > 20) history.value.pop()
  }

  return {
    // state
    mode,
    currentTaskId,
    taskStatus,
    steps,
    diagnosis,
    stepCount,
    toolCount,
    confidencePercent,
    startTime,
    elapsedSeconds,
    timeline,
    history,
    replaySessionId,
    impactReport,
    // actions
    setMode,
    startTimer,
    stopTimer,
    reset,
    addStep,
    setDiagnosis,
    setStatus,
    addTimeline,
    addHistory,
  }
}, {
  persist: {
    pick: ['mode', 'history', 'currentTaskId', 'replaySessionId'],
  },
})
