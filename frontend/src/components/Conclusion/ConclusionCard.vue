<script setup lang="ts">
import { ref, computed } from 'vue'
import { useTaskStore } from '@/store/useTaskStore'
import { createScenario } from '@/composables/useApi'

const store = useTaskStore()
const copyText = ref('复制报告')

// 简易 Markdown 渲染
function renderMarkdown(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\*\*(.+?)\*\*/g, '<strong class="text-base-content/90">$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="px-1 py-0.5 rounded bg-base-300/50 text-[0.75rem] font-mono">$1</code>')
    .replace(/\n/g, '<br/>')
}

// 场景保存相关
const showSaveModal = ref(false)
const scenarioName = ref('')
const scenarioDesc = ref('')
const scenarioTags = ref('')
const saving = ref(false)
const saved = ref(false)

const canSaveScenario = computed(() =>
  store.diagnosis && store.diagnosis.confidence >= 0.7 && !saved.value
)

async function saveAsScenario() {
  if (!store.diagnosis || !scenarioName.value || !scenarioDesc.value) return
  saving.value = true
  try {
    const tags = scenarioTags.value.split(',').map(t => t.trim()).filter(Boolean)
    await createScenario({
      name: scenarioName.value,
      description: scenarioDesc.value,
      root_cause: store.diagnosis.root_cause,
      log_patterns: tags.length > 0 ? tags : undefined,
      affected_namespaces: store.diagnosis.affected_services,
    })
    saved.value = true
    showSaveModal.value = false
    scenarioName.value = ''
    scenarioDesc.value = ''
    scenarioTags.value = ''
  } catch (e) {
    console.error('保存场景失败:', e)
  } finally {
    saving.value = false
  }
}

// 置信度颜色（升级版：>90% 金色、60-90% 蓝色、<60% 橙色）
function confColor(conf: number): string {
  if (conf >= 90) return '#eab308'  // 金色
  if (conf >= 60) return '#3b82f6'  // 蓝色
  return '#f97316'                   // 橙色
}

// 置信度标题色
function confTitleColor(conf: number): string {
  if (conf >= 90) return 'text-yellow-400'
  if (conf >= 60) return 'text-blue-400'
  return 'text-orange-400'
}

// 环形进度条 conic-gradient
function conicGradient(conf: number): string {
  const color = confColor(conf)
  return `conic-gradient(${color} ${conf * 3.6}deg, oklch(var(--b3) / 0.3) ${conf * 3.6}deg)`
}

// 复制诊断报告
async function copyDiagnosis() {
  const d = store.diagnosis
  if (!d) return

  const text = `【Argus 诊断报告】
根因: ${d.root_cause}
置信度: ${Math.round(d.confidence * 100)}%
影响服务: ${(d.affected_services || []).join(', ')}
影响范围: ${d.impact || '—'}
恢复建议:
${(d.suggestions || []).map((s, i) => `${i + 1}. ${s}`).join('\n')}`

  try {
    await navigator.clipboard.writeText(text)
    copyText.value = '已复制'
    setTimeout(() => {
      copyText.value = '复制报告'
    }, 1500)
  } catch (e) {
    console.error('复制失败:', e)
  }
}
</script>

<template>
  <!-- 诊断结论面板 -->
  <div class="glass-card rounded-xl overflow-hidden flex flex-col">
    <!-- 卡片标题 -->
    <div class="px-4 py-2.5 border-b border-base-300 flex items-center justify-between
                text-[0.8125rem] font-semibold text-base-content/70">
      <div class="flex items-center gap-1.5">
        <svg class="w-3.5 h-3.5 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round"
            d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
        </svg>
        诊断结论
      </div>
      <button
        v-if="store.diagnosis"
        class="px-2 py-1 rounded text-[0.625rem] border border-base-300 text-base-content/40
               cursor-pointer transition-all hover:border-indigo-500 hover:text-indigo-400"
        @click="copyDiagnosis"
      >
        {{ copyText }}
      </button>
    </div>

    <!-- 内容区域 -->
    <div class="p-4 overflow-y-auto scroller" style="max-height: 40vh">
      <!-- 空状态 -->
      <div v-if="!store.diagnosis" class="text-center text-sm py-8 text-base-content/30">
        等待诊断完成
      </div>

      <!-- 诊断结论内容 -->
      <template v-else>
        <!-- 根因 -->
        <div class="mb-2.5">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            根因
          </div>
          <div class="text-sm leading-relaxed text-base-content/80" v-html="renderMarkdown(store.diagnosis.root_cause)"></div>
        </div>

        <!-- 置信度环形进度条 -->
        <div class="mb-2.5">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            置信度
          </div>
          <div class="flex items-center gap-3 mt-1">
            <!-- 环形进度条 -->
            <div class="relative w-12 h-12 rounded-full flex items-center justify-center flex-shrink-0"
                 :style="{ background: conicGradient(store.confidencePercent ?? 0) }">
              <div class="w-9 h-9 rounded-full bg-base-100/90 flex items-center justify-center">
                <span class="text-xs font-bold" :class="confTitleColor(store.confidencePercent ?? 0)">
                  {{ store.confidencePercent }}%
                </span>
              </div>
            </div>
            <div class="text-xs text-base-content/50">
              {{ (store.confidencePercent ?? 0) >= 90 ? '高置信度诊断' : (store.confidencePercent ?? 0) >= 60 ? '中等置信度' : '低置信度，建议人工复核' }}
            </div>
          </div>
        </div>

        <!-- 影响服务 -->
        <div class="mb-2.5">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            影响服务
          </div>
          <div class="flex flex-wrap gap-1 mt-1">
            <span
              v-if="store.diagnosis.affected_services.length === 0"
              class="text-sm text-base-content/40"
            >—</span>
            <span
              v-for="svc in store.diagnosis.affected_services"
              :key="svc"
              class="inline-block px-1.5 py-0.5 rounded text-[0.625rem] font-semibold
                     bg-red-900/60 text-red-300"
            >
              {{ svc }}
            </span>
          </div>
        </div>

        <!-- 影响范围 -->
        <div class="mb-2.5">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            影响范围
          </div>
          <div class="text-xs text-base-content/60">
            {{ store.diagnosis.impact || '—' }}
          </div>
        </div>

        <!-- 恢复建议 -->
        <div>
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-1">
            恢复建议
          </div>
          <div
            v-for="(sugg, i) in store.diagnosis.suggestions"
            :key="i"
            class="flex items-start gap-1.5 p-2 glass-card rounded-lg mt-1.5 text-xs text-base-content/70
                   transition-all hover:border-indigo-500/30 cursor-default"
          >
            <span class="w-5 h-5 rounded-full text-white text-[0.5625rem]
                         font-bold flex items-center justify-center flex-shrink-0 mt-0.5"
                  style="background: linear-gradient(135deg, #6366f1, #a78bfa)">
              {{ i + 1 }}
            </span>
            <span>{{ sugg }}</span>
          </div>
          <div v-if="!store.diagnosis.suggestions?.length" class="text-sm text-base-content/40">
            —
          </div>
        </div>

        <!-- 效率量化指标 -->
        <div class="mt-3 pt-3 border-t border-base-300">
          <div class="text-[0.625rem] font-semibold uppercase tracking-wider text-base-content/40 mb-2">
            效率提升
          </div>
          <div class="grid grid-cols-2 gap-2">
            <div class="rounded-lg p-2 bg-emerald-500/10 border border-emerald-500/20 text-center">
              <div class="text-sm font-bold text-emerald-400">
                {{ store.elapsedSeconds ? `${store.elapsedSeconds}s` : '~45s' }}
              </div>
              <div class="text-[0.5625rem] text-base-content/40">AI 诊断耗时</div>
            </div>
            <div class="rounded-lg p-2 bg-orange-500/10 border border-orange-500/20 text-center">
              <div class="text-sm font-bold text-orange-400">~30min</div>
              <div class="text-[0.5625rem] text-base-content/40">人工排查耗时</div>
            </div>
            <div class="rounded-lg p-2 bg-indigo-500/10 border border-indigo-500/20 text-center">
              <div class="text-sm font-bold text-indigo-400">{{ store.stepCount }} 步</div>
              <div class="text-[0.5625rem] text-base-content/40">AI 自动完成</div>
            </div>
            <div class="rounded-lg p-2 bg-base-300/30 border border-base-300 text-center">
              <div class="text-sm font-bold text-base-content/60">~25 步</div>
              <div class="text-[0.5625rem] text-base-content/40">人工手动排查</div>
            </div>
          </div>
        </div>

        <!-- 保存为场景按钮 -->
        <div v-if="canSaveScenario" class="mt-3 pt-3 border-t border-base-300">
          <button
            class="w-full px-3 py-1.5 rounded-lg text-xs font-medium border border-indigo-500/30
                   text-indigo-400 cursor-pointer transition-all hover:bg-indigo-500/10"
            @click="showSaveModal = true"
          >
            保存为场景
          </button>
        </div>
        <div v-if="saved" class="mt-3 pt-3 border-t border-base-300 text-center text-xs text-emerald-400">
          已保存为场景
        </div>
      </template>
    </div>

    <!-- 保存场景 Modal -->
    <dialog :open="showSaveModal" class="modal" @close="showSaveModal = false">
      <div class="modal-box bg-base-100 border border-base-300 max-w-sm">
        <h3 class="text-sm font-semibold text-base-content/80 mb-3">保存为沉淀场景</h3>
        <div class="flex flex-col gap-2.5">
          <input
            v-model="scenarioName"
            type="text"
            placeholder="场景名称"
            class="w-full px-3 py-2 rounded-lg text-sm bg-base-200 border border-base-300
                   text-base-content focus:outline-none focus:border-indigo-500"
          />
          <textarea
            v-model="scenarioDesc"
            placeholder="场景描述"
            rows="3"
            class="w-full px-3 py-2 rounded-lg text-sm bg-base-200 border border-base-300
                   text-base-content focus:outline-none focus:border-indigo-500 resize-none"
          ></textarea>
          <input
            v-model="scenarioTags"
            type="text"
            placeholder="标签（逗号分隔，如：连接池,超时,数据库）"
            class="w-full px-3 py-2 rounded-lg text-sm bg-base-200 border border-base-300
                   text-base-content focus:outline-none focus:border-indigo-500"
          />
        </div>
        <div class="modal-action mt-3">
          <button
            class="px-3 py-1.5 rounded-lg text-xs border border-base-300 text-base-content/50
                   cursor-pointer hover:text-base-content"
            @click="showSaveModal = false"
          >
            取消
          </button>
          <button
            class="px-3 py-1.5 rounded-lg text-xs text-white bg-indigo-600 cursor-pointer
                   hover:brightness-110 disabled:opacity-40"
            :disabled="!scenarioName || !scenarioDesc || saving"
            @click="saveAsScenario"
          >
            {{ saving ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showSaveModal = false">close</button>
      </form>
    </dialog>
  </div>
</template>
