<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { TOPOLOGY } from '@/store/useTaskStore'

// 快捷输入列表
const QUICK_CHIPS = [
  'prj-ubill 连接池耗尽',
  'prj-uresource 大量 504 超时',
  'prj-apigateway 间歇性 502',
  'prj-uresource OOM',
]

// 自然语言输入提示
const SUGGESTIONS = [
  '最近 15 分钟有哪些服务出现异常？',
  '分析 prj-udb 的慢查询问题',
  '为什么 prj-apigateway 返回 502？',
  '检查 prj-uresource 的内存使用情况',
]

// 从拓扑动态获取 namespace 列表
const NAMESPACES = computed(() => TOPOLOGY.value.services)

// 时间范围选项
const TIME_RANGES = [
  { label: '最近 15 分钟', value: 'last 15m' },
  { label: '最近 1 小时', value: 'last 1h' },
  { label: '最近 6 小时', value: 'last 6h' },
  { label: '最近 24 小时', value: 'last 24h' },
]

const HISTORY_KEY = 'argus-diagnose-history'
const MAX_HISTORY = 10

const props = defineProps<{
  loading: boolean
}>()

export interface DiagnoseContext {
  time_range?: string
  namespaces?: string[]
}

const emit = defineEmits<{
  (e: 'diagnose', input: string, context?: DiagnoseContext): void
}>()

const input = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const showAdvanced = ref(false)
const showSuggestions = ref(false)
const inputHistory = ref<string[]>([])

const advancedOptions = reactive<DiagnoseContext>({
  time_range: '',
  namespaces: [],
})

// 加载历史记录
onMounted(() => {
  try {
    const saved = localStorage.getItem(HISTORY_KEY)
    if (saved) inputHistory.value = JSON.parse(saved)
  } catch { /* ignore */ }
})

function saveToHistory(text: string) {
  const list = inputHistory.value.filter(h => h !== text)
  list.unshift(text)
  if (list.length > MAX_HISTORY) list.pop()
  inputHistory.value = list
  localStorage.setItem(HISTORY_KEY, JSON.stringify(list))
}

// 过滤后的建议（输入匹配）
const filteredSuggestions = computed(() => {
  const q = input.value.trim().toLowerCase()
  if (!q) return [...inputHistory.value.slice(0, 3), ...SUGGESTIONS.slice(0, 3)]
  return [
    ...inputHistory.value.filter(h => h.toLowerCase().includes(q)),
    ...SUGGESTIONS.filter(s => s.toLowerCase().includes(q)),
  ].slice(0, 5)
})

function handleDiagnose() {
  const v = input.value.trim()
  if (!v) return
  saveToHistory(v)
  showSuggestions.value = false
  const ctx: DiagnoseContext = {}
  if (advancedOptions.time_range) ctx.time_range = advancedOptions.time_range
  if (advancedOptions.namespaces && advancedOptions.namespaces.length > 0) {
    ctx.namespaces = [...advancedOptions.namespaces]
  }
  emit('diagnose', v, Object.keys(ctx).length > 0 ? ctx : undefined)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleDiagnose()
  }
}

function quickFill(text: string) {
  input.value = text
  showSuggestions.value = false
  inputRef.value?.focus()
}

function selectSuggestion(text: string) {
  input.value = text
  showSuggestions.value = false
  inputRef.value?.focus()
}

function hideSuggestionsLater() {
  window.setTimeout(() => {
    showSuggestions.value = false
  }, 200)
}

function toggleNamespace(ns: string) {
  const idx = advancedOptions.namespaces!.indexOf(ns)
  if (idx >= 0) {
    advancedOptions.namespaces!.splice(idx, 1)
  } else {
    advancedOptions.namespaces!.push(ns)
  }
}
</script>

<template>
  <!-- 诊断模式输入区 -->
  <div>
    <div class="flex gap-3 relative">
      <div class="flex-1 relative">
        <input
          ref="inputRef"
          v-model="input"
          type="text"
          placeholder="用自然语言描述问题，如：prj-ubill 连接池耗尽、最近有哪些异常？"
          class="w-full rounded-lg px-4 py-2 text-sm bg-base-200 border border-base-300 text-base-content
                 placeholder:text-base-content/30 focus:outline-none focus:border-indigo-500
                 focus:ring-2 focus:ring-indigo-500/20 transition-all"
          :disabled="props.loading"
          @keydown="handleKeydown"
          @focus="showSuggestions = true"
          @blur="hideSuggestionsLater"
        />
        <!-- 建议下拉 -->
        <div
          v-if="showSuggestions && filteredSuggestions.length > 0 && !props.loading"
          class="absolute z-20 top-full left-0 right-0 mt-1 rounded-lg bg-base-200 border border-base-300
                 shadow-lg overflow-hidden"
        >
          <div
            v-for="(s, i) in filteredSuggestions"
            :key="i"
            class="px-3 py-2 text-xs text-base-content/60 cursor-pointer transition-colors
                   hover:bg-indigo-500/10 hover:text-indigo-400 flex items-center gap-2"
            @mousedown.prevent="selectSuggestion(s)"
          >
            <svg v-if="inputHistory.includes(s)" class="w-3 h-3 text-base-content/30 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
            <svg v-else class="w-3 h-3 text-base-content/30 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707"/>
            </svg>
            {{ s }}
          </div>
        </div>
      </div>
      <button
        class="shrink-0 px-5 py-2 rounded-lg text-white text-sm font-semibold transition-all
               hover:brightness-110 active:scale-[.97] disabled:opacity-40 disabled:pointer-events-none"
        style="background: linear-gradient(135deg, #6366f1, #818cf8)"
        :disabled="props.loading"
        @click="handleDiagnose"
      >
        {{ props.loading ? '诊断中…' : '开始诊断' }}
      </button>
    </div>

    <!-- 快捷输入 chip -->
    <div class="flex flex-wrap gap-1.5 mt-2.5">
      <span
        v-for="chip in QUICK_CHIPS"
        :key="chip"
        class="inline-flex items-center border border-base-300 text-base-content/50 rounded-full
               px-3 py-1 text-[0.7rem] cursor-pointer transition-all whitespace-nowrap
               hover:border-indigo-500 hover:text-indigo-400 hover:bg-indigo-500/5"
        @click="quickFill(chip)"
      >
        {{ chip }}
      </span>
    </div>

    <!-- 高级选项折叠面板 -->
    <div class="mt-3">
      <button
        class="text-xs text-base-content/40 hover:text-base-content/60 transition-colors flex items-center gap-1"
        @click="showAdvanced = !showAdvanced"
      >
        <span class="transition-transform" :class="{ 'rotate-90': showAdvanced }">▶</span>
        高级选项
      </button>

      <div v-if="showAdvanced" class="mt-2 p-3 rounded-lg bg-base-200/50 border border-base-300 space-y-3">
        <!-- 时间范围 -->
        <div>
          <label class="text-xs text-base-content/50 mb-1 block">时间范围</label>
          <select
            v-model="advancedOptions.time_range"
            class="select select-sm select-bordered w-full max-w-xs bg-base-200 text-sm"
          >
            <option value="">默认（last 1h）</option>
            <option v-for="tr in TIME_RANGES" :key="tr.value" :value="tr.value">
              {{ tr.label }}
            </option>
          </select>
        </div>

        <!-- Namespace 多选 -->
        <div>
          <label class="text-xs text-base-content/50 mb-1 block">限定 Namespace</label>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="ns in NAMESPACES"
              :key="ns"
              class="flex items-center gap-1.5 cursor-pointer"
            >
              <input
                type="checkbox"
                class="checkbox checkbox-xs checkbox-primary"
                :checked="advancedOptions.namespaces!.includes(ns)"
                @change="toggleNamespace(ns)"
              />
              <span class="text-xs text-base-content/60">{{ ns }}</span>
            </label>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
