<script setup lang="ts">
const emit = defineEmits<{
  (e: 'quick-action', text: string): void
}>()

const templates = [
  { icon: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z', text: '诊断最近的 504 错误', color: 'text-red-400 bg-red-500/10' },
  { icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z', text: '查看 payment-service 日志', color: 'text-blue-400 bg-blue-500/10' },
  { icon: 'M13 10V3L4 14h7v7l9-11h-7z', text: '分析链路延迟问题', color: 'text-amber-400 bg-amber-500/10' },
  { icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z', text: '检查服务健康状态', color: 'text-emerald-400 bg-emerald-500/10' },
]
</script>

<template>
  <div class="flex flex-col items-center justify-center h-full px-8 max-w-2xl mx-auto">
    <!-- Logo + 欢迎语 -->
    <div class="mb-10 text-center">
      <div
        class="w-16 h-16 rounded-2xl flex items-center justify-center text-white font-bold text-2xl mx-auto mb-4"
        style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
      >
        A
      </div>
      <h1 class="text-xl font-bold text-base-content/90 mb-2">Argus 智能诊断助手</h1>
      <p class="text-sm text-base-content/40">描述你遇到的问题，我会自动分析日志和链路数据，定位故障根因</p>
    </div>

    <!-- 快速模板 -->
    <div class="grid grid-cols-2 gap-3 w-full">
      <button
        v-for="tpl in templates"
        :key="tpl.text"
        class="flex items-start gap-3 p-4 rounded-xl glass-card text-left
               hover:border-indigo-500/30 transition-all cursor-pointer group"
        @click="emit('quick-action', tpl.text)"
      >
        <div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" :class="tpl.color">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" :d="tpl.icon" />
          </svg>
        </div>
        <span class="text-sm text-base-content/60 group-hover:text-base-content/80 transition-colors">
          {{ tpl.text }}
        </span>
      </button>
    </div>
  </div>
</template>
