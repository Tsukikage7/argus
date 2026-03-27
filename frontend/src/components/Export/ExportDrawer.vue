<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

// 组件属性
const props = defineProps<{
  visible: boolean
  taskId: string
}>()

const emit = defineEmits<{
  close: []
}>()

// 状态
const loading = ref(false)
const markdownContent = ref('')
const error = ref('')

// Markdown 渲染结果（经 DOMPurify 过滤，防止 XSS）
const sanitizedHtml = computed(() => {
  if (!markdownContent.value) return ''
  const raw = marked(markdownContent.value) as string
  return DOMPurify.sanitize(raw)
})

// 获取 Markdown 预览内容
async function fetchMarkdownPreview() {
  if (!props.taskId) return

  loading.value = true
  error.value = ''
  markdownContent.value = ''

  try {
    const res = await fetch(`/api/v1/tasks/${props.taskId}/export?format=markdown`, {
      headers: { Authorization: 'Bearer argus-demo-key' },
    })
    if (!res.ok) throw new Error(`请求失败: ${res.status}`)
    markdownContent.value = await res.text()
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
}

// 触发文件下载
async function download(format: 'markdown' | 'json') {
  try {
    const res = await fetch(`/api/v1/tasks/${props.taskId}/export?format=${format}`, {
      headers: { Authorization: 'Bearer argus-demo-key' },
    })
    if (!res.ok) throw new Error(`下载失败: ${res.status}`)

    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    // 从响应头获取文件名，兜底自行拼接
    const disposition = res.headers.get('Content-Disposition') || ''
    const match = disposition.match(/filename="([^"]+)"/)
    a.download = match ? match[1] : `argus-report-${props.taskId}.${format === 'json' ? 'json' : 'md'}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e: any) {
    error.value = e.message || '下载失败'
  }
}

// 抽屉打开时自动拉取预览内容
watch(
  () => props.visible,
  (v) => {
    if (v) fetchMarkdownPreview()
  },
)
</script>

<template>
  <!-- 遮罩层 -->
  <Transition name="fade">
    <div
      v-if="visible"
      class="fixed inset-0 z-40 bg-black/50"
      @click="emit('close')"
    />
  </Transition>

  <!-- 抽屉主体：从右侧滑入 -->
  <Transition name="slide">
    <div
      v-if="visible"
      class="fixed top-0 right-0 bottom-0 z-50 flex flex-col bg-base-100 border-l border-base-300 shadow-2xl"
      style="width: min(480px, 90vw)"
    >
      <!-- 头部 -->
      <div class="flex items-center justify-between px-4 py-3 border-b border-base-300 flex-shrink-0">
        <div class="flex items-center gap-2 text-sm font-semibold text-base-content/80">
          <svg class="w-4 h-4 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round"
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          诊断报告导出
        </div>
        <!-- 关闭按钮 -->
        <button
          class="w-7 h-7 flex items-center justify-center rounded hover:bg-base-200 text-base-content/40
                 hover:text-base-content/80 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- 内容区：Markdown 预览 -->
      <div class="flex-1 overflow-y-auto p-4 scroller">
        <!-- 加载中 -->
        <div v-if="loading" class="flex items-center justify-center py-16 text-sm text-base-content/40">
          <svg class="animate-spin w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
          </svg>
          加载中...
        </div>

        <!-- 错误提示 -->
        <div v-else-if="error" class="text-sm text-red-400 text-center py-8">
          {{ error }}
        </div>

        <!-- Markdown 渲染结果 -->
        <div
          v-else-if="markdownContent"
          class="prose prose-sm prose-invert max-w-none text-base-content/80
                 prose-headings:text-base-content/90 prose-code:text-indigo-300
                 prose-strong:text-base-content prose-li:text-base-content/70"
          v-html="sanitizedHtml"
        />

        <!-- 空状态 -->
        <div v-else class="text-sm text-base-content/30 text-center py-8">
          暂无内容
        </div>
      </div>

      <!-- 底部工具栏 -->
      <div class="flex items-center gap-2 px-4 py-3 border-t border-base-300 flex-shrink-0">
        <!-- 下载 Markdown 按钮 -->
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium
                 bg-indigo-600 hover:bg-indigo-500 text-white transition-colors cursor-pointer"
          @click="download('markdown')"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
          </svg>
          下载 Markdown
        </button>

        <!-- 下载 JSON 按钮 -->
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium
                 border border-base-300 text-base-content/60 hover:border-indigo-500
                 hover:text-indigo-400 transition-colors cursor-pointer"
          @click="download('json')"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
          </svg>
          下载 JSON
        </button>

        <!-- 任务 ID 提示 -->
        <span class="ml-auto text-[0.625rem] text-base-content/30 truncate" style="max-width: 140px">
          {{ taskId }}
        </span>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
/* 遮罩层淡入淡出动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 抽屉从右侧滑入滑出动画 */
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.25s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}
</style>
