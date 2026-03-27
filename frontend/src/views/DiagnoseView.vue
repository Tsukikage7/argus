<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'
import { useTaskSSE, useReplaySSE } from '@/composables/useSSE'
import { startDiagnose, getTask } from '@/composables/useApi'

// 控件
import DiagnoseInput from '@/components/Control/DiagnoseInput.vue'

// 统计栏
import StatsBar from '@/components/Dashboard/StatsBar.vue'

// 推理面板
import InferencePanel from '@/components/Inference/InferencePanel.vue'

// 拓扑 + 结论
import TopologyPanel from '@/components/Topology/TopologyPanel.vue'
import ConclusionCard from '@/components/Conclusion/ConclusionCard.vue'

// 时间线 + 历史
import TimelinePanel from '@/components/Monitor/TimelinePanel.vue'
import HistoryPanel from '@/components/Monitor/HistoryPanel.vue'
import LogExplorer from '@/components/Monitor/LogExplorer.vue'

const store = useTaskStore()
const taskSSE = useTaskSSE()
const replaySSE = useReplaySSE()

// 加载状态
const diagLoading = ref(false)

// URL 参数管理
function updateUrlParams(params: Record<string, string>) {
  const url = new URL(window.location.href)
  // 清除旧参数
  url.searchParams.delete('taskId')
  url.searchParams.delete('sessionId')
  url.searchParams.delete('mode')
  for (const [k, v] of Object.entries(params)) {
    url.searchParams.set(k, v)
  }
  window.history.replaceState({}, '', url.toString())
}

function clearUrlParams() {
  const url = new URL(window.location.href)
  url.searchParams.delete('taskId')
  url.searchParams.delete('sessionId')
  url.searchParams.delete('mode')
  window.history.replaceState({}, '', url.toString())
}

// 回放影响面报告
const impactReport = ref<import('@/types').ImpactReport | null>(null)

// 统计栏是否显示
const showStats = ref(false)

// 右栏 Tab
const rightTab = ref<'timeline' | 'logs' | 'history'>('timeline')

// 提取链路信息（从 action 参数）
function extractTraceInfo(step: import('@/types').Step) {
  if (!step.action) return
  const p = step.action.params || {}
  if (step.action.tool === 'trace_analyze') {
    const id = (p.request_uuid || p.trace_id) as string | undefined
    if (id) store.addTimeline('info', '追踪链路 ' + id.substring(0, 12) + '…')
  }
  if (step.action.tool === 'es_query_logs') {
    if (p.namespace) store.addTimeline('info', '查询 namespace: ' + p.namespace)
    if (p.request_uuid) {
      store.addTimeline('info', '追踪 request_uuid: ' + (p.request_uuid as string).substring(0, 12) + '…')
    }
  }
}

// 从 observe 提取关键事件（per-step 去重）
function extractObserveEvents(obs: string) {
  const seen = new Set<string>()
  function addOnce(level: import('@/types').TimelineEvent['level'], msg: string) {
    const key = level + '|' + msg
    if (seen.has(key)) return
    seen.add(key)
    store.addTimeline(level, msg)
  }
  // 连接与资源类
  if (obs.includes('pool exhausted')) addOnce('error', '检测到连接池耗尽')
  if (obs.includes('OOM') || obs.includes('out of memory')) addOnce('error', '检测到 OOM')
  if (obs.includes('no space left') || obs.includes('disk')) addOnce('error', '检测到磁盘空间不足')
  if (obs.includes('timeout') || obs.includes('TIMEOUT')) addOnce('warn', '检测到超时')
  if (obs.includes('connection refused')) addOnce('warn', '检测到连接拒绝')
  if (obs.includes('slow query')) addOnce('warn', '检测到慢查询')
  // UCloud 链路追踪类
  if (obs.includes('request_uuid')) addOnce('info', '检测到 request_uuid 追踪标识')
  if (obs.includes('trace-line') || obs.includes('Trace Line')) addOnce('info', '检测到网关链路 trace-line')
  if (obs.includes('trace_analyze')) addOnce('info', '执行链路分析')
  // Namespace / K8s 类
  if (obs.includes('namespace') || obs.includes('kubernetes_namespace')) addOnce('info', '检测到 namespace 信息')
  if (obs.includes('prj-apigateway')) addOnce('warn', '网关 prj-apigateway 异常')
  if (obs.includes('prj-ubill')) addOnce('warn', '计费服务 prj-ubill 异常')
  if (obs.includes('prj-uresource')) addOnce('warn', '资源服务 prj-uresource 异常')
  // 错误级别
  if (obs.includes('ERROR')) addOnce('error', '检测到 ERROR 级别日志')
}

// 诊断流程
async function handleDiagnose(input: string, diagContext?: { time_range?: string; namespaces?: string[] }) {
  diagLoading.value = true
  showStats.value = true
  impactReport.value = null

  // 先断开 SSE 连接，消除异步写入竞态
  taskSSE.disconnect()
  replaySSE.disconnect()
  store.reset()
  clearUrlParams()
  store.startTimer()
  store.setStatus('running')
  store.addTimeline('info', '开始诊断: ' + input)

  try {
    const { task_id, stream_token } = await startDiagnose(input, diagContext)
    store.currentTaskId = task_id
    updateUrlParams({ taskId: task_id })

    // 连接 SSE
    taskSSE.connect(task_id, stream_token)

    // 启动轮询作为兜底
    pollResult(task_id)
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    store.addTimeline('error', '请求失败: ' + msg)
    store.setStatus('failed')
    diagLoading.value = false
    store.stopTimer()
  }
}

// 监听 SSE 推理步骤（steps 是已解包的 Step[]）
watch(taskSSE.steps, (steps) => {
  if (!steps || steps.length === 0) return
  // SSE 返回的是全量数组，增量添加到 store
  const newSteps = steps.slice(store.steps.length)
  for (const step of newSteps) {
    store.addStep(step)
    if (step.action) {
      extractTraceInfo(step)
      store.addTimeline('info', '调用 ' + step.action.tool)
    }
    if (step.observe) extractObserveEvents(step.observe)
  }
})

// 监听 SSE 诊断结论
watch(taskSSE.diagnosis, (d) => {
  if (d) {
    store.setDiagnosis(d)
    store.addTimeline('ok', '根因: ' + d.root_cause.slice(0, 60))
  }
})

// 监听 SSE 任务状态
watch(taskSSE.status, (s) => {
  store.setStatus(s)
  if (['completed', 'failed', 'recovered'].includes(s)) {
    diagLoading.value = false
    store.stopTimer()
    store.addTimeline(s === 'completed' ? 'ok' : 'error', '诊断完成: ' + s)
  }
})

// 轮询任务结果（兜底）
let pollTimer: ReturnType<typeof setInterval> | null = null
function pollResult(taskId: string) {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const task = await getTask(taskId)
      if (['completed', 'failed', 'recovered'].includes(task.status)) {
        if (pollTimer) clearInterval(pollTimer)

        if (task.diagnosis && !store.diagnosis) {
          store.setDiagnosis(task.diagnosis)
          store.addTimeline('ok', '根因: ' + task.diagnosis.root_cause.slice(0, 60))
        }
        // 如果 SSE 步骤没到，补充
        if (store.steps.length === 0 && task.steps) {
          for (const step of task.steps) {
            store.addStep(step)
            if (step.action) extractTraceInfo(step)
            if (step.observe) extractObserveEvents(step.observe)
          }
        }
        store.setStatus(task.status)
        diagLoading.value = false
        store.stopTimer()
        store.addHistory(task)
      }
    } catch {}
  }, 3000)
}

// 历史任务回放
async function handleReplayTask(taskId: string) {
  try {
    const task = await getTask(taskId)
    store.reset()
    store.setStatus(task.status)
    showStats.value = true
    if (task.steps) {
      for (const step of task.steps) {
        store.addStep(step)
      }
    }
    if (task.diagnosis) {
      store.setDiagnosis(task.diagnosis)
    }
  } catch {}
}

onMounted(() => {
  restoreFromUrl()
})

/** 从 URL query 参数恢复诊断状态 */
async function restoreFromUrl() {
  const params = new URLSearchParams(window.location.search)
  const taskId = params.get('taskId')

  if (taskId) {
    try {
      const task = await getTask(taskId)
      store.currentTaskId = taskId
      store.setStatus(task.status)
      showStats.value = true
      if (task.steps) {
        for (const step of task.steps) {
          store.addStep(step)
          if (step.action) extractTraceInfo(step)
          if (step.observe) extractObserveEvents(step.observe)
        }
      }
      if (task.diagnosis) {
        store.setDiagnosis(task.diagnosis)
      }
      if (task.status === 'running') {
        diagLoading.value = true
        store.startTimer()
        taskSSE.connect(taskId)
        pollResult(taskId)
      }
    } catch {
      // 任务不存在或已过期
    }
  }
}

// 组件卸载时清理定时器，防止内存泄漏
onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>

<template>
  <div>
    <!-- 输入区 -->
    <div class="glass-card rounded-xl p-4 mb-5">
      <DiagnoseInput
        :loading="diagLoading"
        @diagnose="handleDiagnose"
      />
    </div>

    <!-- 统计栏 -->
    <StatsBar v-if="showStats" />

    <!-- 三栏布局 -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <!-- 左栏：推理过程（5/12） -->
      <div class="lg:col-span-5">
        <InferencePanel />
      </div>

      <!-- 中栏：拓扑 + 结论（4/12） -->
      <div class="lg:col-span-4 flex flex-col gap-5">
        <TopologyPanel :impact-report="impactReport" />
        <ConclusionCard />
      </div>

      <!-- 右栏：Tab 切换（3/12） -->
      <div class="lg:col-span-3 flex flex-col gap-0">
        <div class="flex border-b border-base-300/50 mb-3">
          <button
            v-for="tab in ([
              { key: 'timeline', label: '时间线' },
              { key: 'logs', label: '日志' },
              { key: 'history', label: '历史' },
            ] as const)"
            :key="tab.key"
            class="px-3 py-1.5 text-xs font-medium transition-all border-b-2 -mb-[1px]"
            :class="rightTab === tab.key
              ? 'text-indigo-400 border-indigo-400'
              : 'text-base-content/40 border-transparent hover:text-base-content/60'"
            @click="rightTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </div>

        <TimelinePanel v-if="rightTab === 'timeline'" />
        <LogExplorer v-else-if="rightTab === 'logs'" />
        <HistoryPanel v-else @replay-task="handleReplayTask" />
      </div>
    </div>
  </div>
</template>

<style>
/* 全局动画关键帧 */
@keyframes pdot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

@keyframes fadeSlide {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
