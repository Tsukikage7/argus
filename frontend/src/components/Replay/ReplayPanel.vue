<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listScenarios, listReplaySessions } from '@/composables/useApi'
import type { Scenario, ReplaySession } from '@/types'

// 组件事件：选中场景时向父组件传递场景名
const emit = defineEmits<{
  (e: 'select-scenario', name: string): void
}>()

// 场景列表与回放历史数据
const scenarios = ref<Scenario[]>([])
const sessions = ref<ReplaySession[]>([])
// 当前选中的场景名
const selected = ref('')
// 加载状态
const loadingScenarios = ref(false)
const loadingSessions = ref(false)

// 状态徽章颜色映射
const statusClass: Record<string, string> = {
  pending:    'bg-base-300 text-base-content/50',
  generating: 'bg-blue-900/50 text-blue-300',
  diagnosing: 'bg-indigo-900/50 text-indigo-300',
  completed:  'bg-emerald-900/50 text-emerald-300',
  failed:     'bg-red-900/50 text-red-300',
}

// 状态中文标签
const statusLabel: Record<string, string> = {
  pending:    '等待中',
  generating: '生成中',
  diagnosing: '诊断中',
  completed:  '已完成',
  failed:     '失败',
}

// 格式化时间为本地时间字符串
function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// 点击场景卡片时选中并 emit
function selectScenario(name: string) {
  selected.value = name
  emit('select-scenario', name)
}

// 挂载时加载场景列表和回放历史
onMounted(async () => {
  loadingScenarios.value = true
  try {
    scenarios.value = await listScenarios()
    // 默认选中第一个场景
    if (scenarios.value.length > 0) {
      selected.value = scenarios.value[0].name
      emit('select-scenario', selected.value)
    }
  } catch (e) {
    console.error('获取场景列表失败:', e)
  } finally {
    loadingScenarios.value = false
  }

  loadingSessions.value = true
  try {
    const all = await listReplaySessions()
    // 最近 5 条历史
    sessions.value = all.slice(0, 5)
  } catch (e) {
    console.error('获取回放历史失败:', e)
  } finally {
    loadingSessions.value = false
  }
})
</script>

<template>
  <!-- 回放场景选择面板 -->
  <div class="bg-base-100 border border-base-300 rounded-xl overflow-hidden">

    <!-- 面板标题 -->
    <div class="px-4 py-2.5 border-b border-base-300 flex items-center gap-1.5
                text-[0.8125rem] font-semibold text-base-content/70">
      <svg class="w-3.5 h-3.5 text-indigo-400" fill="none" viewBox="0 0 24 24"
           stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round"
              d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
        <path stroke-linecap="round" stroke-linejoin="round"
              d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      故障场景
    </div>

    <!-- 场景卡片列表 -->
    <div class="p-3">
      <!-- 加载占位 -->
      <div v-if="loadingScenarios" class="text-center py-4 text-xs text-base-content/30">
        加载中…
      </div>

      <!-- 空状态 -->
      <div v-else-if="scenarios.length === 0" class="text-center py-4 text-xs text-base-content/30">
        暂无可用场景
      </div>

      <!-- 场景卡片网格 -->
      <div v-else class="grid grid-cols-1 gap-2">
        <div
          v-for="scenario in scenarios"
          :key="scenario.name"
          class="flex items-start gap-3 px-3 py-2.5 rounded-lg border cursor-pointer transition-all"
          :class="selected === scenario.name
            ? 'border-indigo-500 bg-indigo-950/40 text-base-content'
            : 'border-base-300 bg-base-200 text-base-content/70 hover:border-indigo-400 hover:bg-indigo-950/20'"
          @click="selectScenario(scenario.name)"
        >
          <!-- 选中指示圆点 -->
          <div class="flex-shrink-0 mt-0.5">
            <div
              class="w-3 h-3 rounded-full border-2 transition-all"
              :class="selected === scenario.name
                ? 'border-indigo-500 bg-indigo-500'
                : 'border-base-content/20 bg-transparent'"
            ></div>
          </div>

          <!-- 场景信息 -->
          <div class="flex-1 min-w-0">
            <div class="text-xs font-semibold truncate"
                 :class="selected === scenario.name ? 'text-indigo-300' : 'text-base-content/80'">
              {{ scenario.name }}
            </div>
            <div class="text-[0.6875rem] text-base-content/40 mt-0.5 leading-snug">
              {{ scenario.description }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 分隔线 + 历史标题 -->
    <div class="px-4 py-2 border-t border-base-300 flex items-center gap-1.5
                text-[0.75rem] font-semibold text-base-content/50">
      <svg class="w-3 h-3 text-base-content/30" fill="none" viewBox="0 0 24 24"
           stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round"
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      最近回放历史
    </div>

    <!-- 历史列表 -->
    <div class="px-3 pb-3">
      <!-- 加载占位 -->
      <div v-if="loadingSessions" class="text-center py-3 text-xs text-base-content/30">
        加载中…
      </div>

      <!-- 空状态 -->
      <div v-else-if="sessions.length === 0" class="text-center py-3 text-xs text-base-content/30">
        暂无记录
      </div>

      <!-- 历史条目 -->
      <div
        v-for="session in sessions"
        :key="session.id"
        class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-base-200 transition-colors"
      >
        <!-- 场景名 -->
        <span class="text-xs flex-1 truncate text-base-content/60">
          {{ session.scenario_name }}
        </span>

        <!-- 状态徽章 -->
        <span
          class="flex-shrink-0 inline-flex items-center px-1.5 py-0.5 rounded text-[0.625rem] font-semibold"
          :class="statusClass[session.status] ?? statusClass.pending"
        >
          {{ statusLabel[session.status] ?? session.status }}
        </span>

        <!-- 创建时间 -->
        <span class="flex-shrink-0 text-[0.625rem] text-base-content/30 tabular-nums">
          {{ formatTime(session.created_at) }}
        </span>
      </div>
    </div>

  </div>
</template>
