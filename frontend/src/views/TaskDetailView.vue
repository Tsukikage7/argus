<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { getTask, getReplaySession } from '@/composables/useApi'
import type { Task, ReplaySession } from '@/types'
import StepCard from '@/components/Inference/StepCard.vue'

const props = defineProps<{ id: string }>()

const task = ref<Task | null>(null)
const replay = ref<ReplaySession | null>(null)
const loading = ref(true)
const error = ref('')

async function loadData(id: string) {
  loading.value = true
  error.value = ''
  task.value = null
  replay.value = null

  try {
    task.value = await getTask(id)
  } catch {
    try {
      replay.value = await getReplaySession(id)
    } catch (e: any) {
      error.value = e.message || '加载失败'
    }
  } finally {
    loading.value = false
  }
}

// 监听 id 变化（含首次），同路由切换参数时也能重新加载
watch(() => props.id, (id) => loadData(id), { immediate: true })

const confidencePercent = computed(() =>
  task.value?.diagnosis ? Math.round(task.value.diagnosis.confidence * 100) : null
)

function confColor(conf: number): string {
  if (conf >= 90) return '#eab308'
  if (conf >= 60) return '#3b82f6'
  return '#f97316'
}

function conicGradient(conf: number): string {
  const color = confColor(conf)
  return `conic-gradient(${color} ${conf * 3.6}deg, oklch(var(--b3) / 0.3) ${conf * 3.6}deg)`
}

function formatTime(t: string) {
  return new Date(t).toLocaleString('zh-CN')
}

function statusBadge(s: string) {
  const map: Record<string, string> = {
    completed: 'bg-emerald-500/20 text-emerald-400',
    running: 'bg-indigo-500/20 text-indigo-400',
    generating: 'bg-indigo-500/20 text-indigo-400',
    diagnosing: 'bg-indigo-500/20 text-indigo-400',
    failed: 'bg-red-500/20 text-red-400',
    pending: 'bg-base-300/50 text-base-content/50',
  }
  return map[s] || 'bg-base-300/50 text-base-content/50'
}

function replayStatusLabel(s: string) {
  const map: Record<string, string> = {
    pending: '等待中', generating: '生成数据中', diagnosing: '诊断中',
    completed: '已完成', failed: '失败',
  }
  return map[s] || s
}
</script>

<template>
  <div class="p-6">
    <div v-if="loading" class="text-center py-12 text-base-content/30 text-sm">加载中…</div>
    <div v-else-if="error" class="text-center py-12">
      <div class="text-base-content/40 text-sm mb-4">未找到任务或回放记录</div>
      <router-link to="/tasks" class="text-sm text-indigo-400 hover:text-indigo-300 transition-colors">
        返回任务列表
      </router-link>
    </div>

    <!-- 诊断任务详情 -->
    <template v-else-if="task">
      <!-- 头部 -->
      <div class="flex items-center gap-3 mb-6">
        <router-link to="/tasks" class="text-base-content/40 hover:text-base-content/60 transition-colors">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7"/>
          </svg>
        </router-link>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-semibold text-base-content/80">任务详情</h2>
            <span class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold" :class="statusBadge(task.status)">
              {{ task.status }}
            </span>
          </div>
          <div class="text-xs text-base-content/40 mt-0.5">
            {{ task.id }} · {{ formatTime(task.created_at) }}
          </div>
        </div>
      </div>

      <!-- 输入 -->
      <div class="glass-card rounded-xl p-4 mb-4">
        <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">输入</div>
        <div class="text-sm text-base-content/70">{{ task.input }}</div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <!-- 推理过程 -->
        <div class="lg:col-span-2 space-y-3">
          <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-2">
            推理过程（{{ task.steps.length }} 步）
          </div>
          <StepCard
            v-for="(step, i) in task.steps"
            :key="i"
            :step="step"
            :is-latest="false"
          />
          <div v-if="task.steps.length === 0" class="text-center text-sm py-8 text-base-content/30">
            暂无推理步骤
          </div>
        </div>

        <!-- 诊断结论 -->
        <div v-if="task.diagnosis" class="glass-card rounded-xl p-4 h-fit">
          <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-3">诊断结论</div>

          <div class="mb-3">
            <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">根因</div>
            <div class="text-sm text-base-content/80">{{ task.diagnosis.root_cause }}</div>
          </div>

          <div class="flex items-center gap-3 mb-3">
            <div
              class="relative w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0"
              :style="{ background: conicGradient(confidencePercent ?? 0) }"
            >
              <div class="w-7 h-7 rounded-full bg-base-100/90 flex items-center justify-center">
                <span class="text-[0.625rem] font-bold" :style="{ color: confColor(confidencePercent ?? 0) }">
                  {{ confidencePercent }}%
                </span>
              </div>
            </div>
            <div class="text-xs text-base-content/50">置信度</div>
          </div>

          <div class="flex flex-wrap gap-1 mb-3">
            <span
              v-for="svc in task.diagnosis.affected_services"
              :key="svc"
              class="px-1.5 py-0.5 rounded text-[0.5625rem] font-semibold bg-red-900/60 text-red-300"
            >
              {{ svc }}
            </span>
          </div>

          <div v-if="task.diagnosis.suggestions?.length">
            <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">建议</div>
            <div
              v-for="(s, i) in task.diagnosis.suggestions"
              :key="i"
              class="text-xs text-base-content/60 mt-1"
            >
              {{ i + 1 }}. {{ s }}
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 回放 session 详情（fallback） -->
    <template v-else-if="replay">
      <div class="flex items-center gap-3 mb-6">
        <router-link to="/tasks" class="text-base-content/40 hover:text-base-content/60 transition-colors">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7"/>
          </svg>
        </router-link>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-semibold text-base-content/80">回放详情</h2>
            <span class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold bg-purple-500/20 text-purple-400">
              回放
            </span>
            <span class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold" :class="statusBadge(replay.status)">
              {{ replayStatusLabel(replay.status) }}
            </span>
          </div>
          <div class="text-xs text-base-content/40 mt-0.5">
            {{ replay.id }} · {{ formatTime(replay.created_at) }}
          </div>
        </div>
      </div>

      <!-- 回放信息 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <div class="glass-card rounded-xl p-4">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">场景</div>
          <div class="text-sm text-base-content/80">{{ replay.scenario_name }}</div>
          <div class="text-xs text-base-content/50 mt-1">
            {{ replay.type === 'fault' ? '故障回放' : '流量回放' }}
          </div>
        </div>
        <div class="glass-card rounded-xl p-4">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">配置</div>
          <div class="space-y-1 text-xs text-base-content/60">
            <div>故障强度: {{ replay.config.fault_intensity }}</div>
            <div>流量倍率: {{ replay.config.traffic_rate_multiplier }}</div>
            <div>自动诊断: {{ replay.config.auto_diagnose ? '是' : '否' }}</div>
          </div>
        </div>
      </div>

      <!-- 数据统计 -->
      <div class="glass-card rounded-xl p-4 mb-4">
        <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">数据统计</div>
        <div class="flex gap-6 text-sm">
          <div>
            <span class="text-base-content/50">日志写入: </span>
            <span class="text-base-content/80 font-mono">{{ replay.logs_written }}</span>
          </div>
          <div>
            <span class="text-base-content/50">链路写入: </span>
            <span class="text-base-content/80 font-mono">{{ replay.traces_written }}</span>
          </div>
        </div>
      </div>

      <!-- 影响面报告 -->
      <div v-if="replay.impact_report" class="glass-card rounded-xl p-4 mb-4">
        <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">影响面报告</div>
        <div class="text-sm text-base-content/70">{{ replay.impact_report.summary }}</div>
        <div class="flex flex-wrap gap-1 mt-2">
          <span
            v-for="svc in replay.impact_report.affected_services"
            :key="svc.name"
            class="px-1.5 py-0.5 rounded text-[0.5625rem] font-semibold"
            :class="svc.status === 'down' ? 'bg-red-900/60 text-red-300'
              : svc.status === 'degraded' ? 'bg-amber-900/60 text-amber-300'
              : 'bg-emerald-900/60 text-emerald-300'"
          >
            {{ svc.name }} ({{ svc.status }})
          </span>
        </div>
      </div>

      <!-- 错误信息 -->
      <div v-if="replay.error" class="glass-card rounded-xl p-4 mb-4 border-red-500/20">
        <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-red-400/60 mb-1">错误</div>
        <div class="text-sm text-red-400">{{ replay.error }}</div>
      </div>

      <!-- 关联诊断任务跳转 -->
      <div v-if="replay.task_id" class="glass-card rounded-xl p-4">
        <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">关联诊断</div>
        <router-link
          :to="{ name: 'task-detail', params: { id: replay.task_id } }"
          class="text-sm text-indigo-400 hover:text-indigo-300 transition-colors"
        >
          查看诊断任务 {{ replay.task_id.slice(0, 8) }}…
        </router-link>
      </div>
    </template>
  </div>
</template>
