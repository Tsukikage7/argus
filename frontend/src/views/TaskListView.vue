<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listTasks, listReplaySessions } from '@/composables/useApi'
import { useRouter } from 'vue-router'
import type { Task, ReplaySession, TaskStatus, ReplayStatus } from '@/types'

const router = useRouter()
const loading = ref(true)

// 统一列表项
interface UnifiedTask {
  id: string
  description: string
  status: TaskStatus
  source: 'diagnose' | 'replay'
  created_at: string
  // 诊断任务专属
  diagnosis_root_cause?: string
  // 回放任务专属
  replay_scenario?: string
  replay_task_id?: string
}

const items = ref<UnifiedTask[]>([])
const sourceFilter = ref<'' | 'diagnose' | 'replay'>('')
const statusFilter = ref<TaskStatus | ''>('')
const sortBy = ref<'newest' | 'oldest'>('newest')
const page = ref(1)
const pageSize = 10

// 回放状态 → 任务状态映射
function mapReplayStatus(s: ReplayStatus): TaskStatus {
  const map: Record<ReplayStatus, TaskStatus> = {
    pending: 'pending',
    generating: 'running',
    diagnosing: 'running',
    completed: 'completed',
    failed: 'failed',
  }
  return map[s] || 'pending'
}

onMounted(async () => {
  try {
    const [tasks, sessions] = await Promise.all([
      listTasks().catch(() => [] as Task[]),
      listReplaySessions().catch(() => [] as ReplaySession[]),
    ])

    const diagnoseItems: UnifiedTask[] = tasks.map(t => ({
      id: t.id,
      description: t.input,
      status: t.status,
      source: 'diagnose' as const,
      created_at: t.created_at,
      diagnosis_root_cause: t.diagnosis?.root_cause,
    }))

    const replayItems: UnifiedTask[] = sessions.map(s => ({
      id: s.id,
      description: `${s.scenario_name}（${s.type === 'fault' ? '故障' : '流量'}回放）`,
      status: mapReplayStatus(s.status),
      source: 'replay' as const,
      created_at: s.created_at,
      replay_scenario: s.scenario_name,
      replay_task_id: s.task_id,
    }))

    items.value = [...diagnoseItems, ...replayItems]
  } catch (e) {
    console.error('加载任务列表失败:', e)
  } finally {
    loading.value = false
  }
})

const filtered = computed(() => {
  let list = [...items.value]
  if (sourceFilter.value) {
    list = list.filter(t => t.source === sourceFilter.value)
  }
  if (statusFilter.value) {
    list = list.filter(t => t.status === statusFilter.value)
  }
  list.sort((a, b) => {
    const da = new Date(a.created_at).getTime()
    const db = new Date(b.created_at).getTime()
    return sortBy.value === 'newest' ? db - da : da - db
  })
  return list
})

const paged = computed(() => {
  const start = (page.value - 1) * pageSize
  return filtered.value.slice(start, start + pageSize)
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)))

function statusBadge(s: TaskStatus) {
  const map: Record<string, string> = {
    completed: 'bg-emerald-500/20 text-emerald-400',
    running: 'bg-indigo-500/20 text-indigo-400',
    failed: 'bg-red-500/20 text-red-400',
    pending: 'bg-base-300/50 text-base-content/50',
    recovering: 'bg-amber-500/20 text-amber-400',
    recovered: 'bg-cyan-500/20 text-cyan-400',
  }
  return map[s] || 'bg-base-300/50 text-base-content/50'
}

function statusLabel(s: TaskStatus) {
  const map: Record<string, string> = {
    completed: '已完成', running: '运行中', failed: '失败',
    pending: '等待中', recovering: '恢复中', recovered: '已恢复',
  }
  return map[s] || s
}

function sourceBadge(source: 'diagnose' | 'replay') {
  return source === 'diagnose'
    ? 'bg-blue-500/20 text-blue-400'
    : 'bg-purple-500/20 text-purple-400'
}

function sourceLabel(source: 'diagnose' | 'replay') {
  return source === 'diagnose' ? '诊断' : '回放'
}

function formatTime(t: string) {
  return new Date(t).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function goDetail(item: UnifiedTask) {
  router.push({ name: 'task-detail', params: { id: item.id } })
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-semibold text-base-content/80">任务列表</h2>
      <div class="flex gap-2">
        <select
          v-model="sourceFilter"
          class="select select-sm select-bordered bg-base-200 text-xs"
        >
          <option value="">全部类型</option>
          <option value="diagnose">诊断</option>
          <option value="replay">回放</option>
        </select>
        <select
          v-model="statusFilter"
          class="select select-sm select-bordered bg-base-200 text-xs"
        >
          <option value="">全部状态</option>
          <option value="completed">已完成</option>
          <option value="running">运行中</option>
          <option value="failed">失败</option>
          <option value="pending">等待中</option>
        </select>
        <select
          v-model="sortBy"
          class="select select-sm select-bordered bg-base-200 text-xs"
        >
          <option value="newest">最新优先</option>
          <option value="oldest">最早优先</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="text-center py-12 text-base-content/30 text-sm">加载中…</div>

    <div v-else-if="paged.length === 0" class="text-center py-12 text-base-content/30 text-sm">
      暂无任务
    </div>

    <div v-else class="space-y-2">
      <div
        v-for="item in paged"
        :key="`${item.source}-${item.id}`"
        class="glass-card rounded-xl p-4 cursor-pointer transition-all hover:border-indigo-500/30"
        @click="goDetail(item)"
      >
        <div class="flex items-center justify-between mb-1">
          <div class="flex items-center gap-2">
            <span class="text-xs font-mono text-base-content/40">{{ item.id.slice(0, 8) }}</span>
            <span
              class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold"
              :class="sourceBadge(item.source)"
            >
              {{ sourceLabel(item.source) }}
            </span>
            <span
              class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold"
              :class="statusBadge(item.status)"
            >
              {{ statusLabel(item.status) }}
            </span>
          </div>
          <span class="text-[0.625rem] text-base-content/40">{{ formatTime(item.created_at) }}</span>
        </div>
        <div class="text-sm text-base-content/70 truncate">{{ item.description }}</div>
        <div v-if="item.diagnosis_root_cause" class="text-xs text-base-content/50 mt-1 truncate">
          根因: {{ item.diagnosis_root_cause }}
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 mt-4">
      <button
        class="px-2 py-1 rounded text-xs border border-base-300 text-base-content/50
               disabled:opacity-30"
        :disabled="page <= 1"
        @click="page--"
      >
        上一页
      </button>
      <span class="text-xs text-base-content/40">{{ page }} / {{ totalPages }}</span>
      <button
        class="px-2 py-1 rounded text-xs border border-base-300 text-base-content/50
               disabled:opacity-30"
        :disabled="page >= totalPages"
        @click="page++"
      >
        下一页
      </button>
    </div>
  </div>
</template>
