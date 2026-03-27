<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import GlobalTimeRange from './GlobalTimeRange.vue'

const route = useRoute()

const isDark = ref(true)

const breadcrumb = computed(() => {
  const map: Record<string, string> = {
    '/dashboard': '总览',
    '/topology': '服务拓扑',
    '/logs': '日志探索',
    '/traces': '链路追踪',
    '/alerts': '告警中心',
    '/diagnose': 'AI 诊断',
    '/replay': '故障回放',
    '/tasks': '任务列表',
    '/settings': '设置',
  }
  return map[route.path] || route.path
})

function initTheme() {
  const saved = localStorage.getItem('argus-theme') || 'dark'
  isDark.value = saved === 'dark'
  document.documentElement.setAttribute('data-theme', saved)
}

function toggleTheme() {
  isDark.value = !isDark.value
  const theme = isDark.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem('argus-theme', theme)
}

onMounted(initTheme)
</script>

<template>
  <header
    class="h-14 flex items-center justify-between px-5 border-b shrink-0"
    style="border-color: oklch(var(--b3)); background: color-mix(in oklch, oklch(var(--b2)) 85%, transparent)"
  >
    <!-- 面包屑 -->
    <div class="flex items-center gap-2 text-sm">
      <span class="text-base-content/40">Argus</span>
      <span class="text-base-content/20">/</span>
      <span class="text-base-content font-medium">{{ breadcrumb }}</span>
    </div>

    <!-- 右侧：时间范围 + 主题 -->
    <div class="flex items-center gap-3">
      <GlobalTimeRange />

      <button
        class="w-7 h-7 rounded-md flex items-center justify-center border border-base-300
               bg-base-200 text-base-content/40 cursor-pointer transition-all
               hover:border-indigo-500 hover:text-indigo-400"
        :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
        @click="toggleTheme"
      >
        <svg v-if="!isDark" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
        <svg v-else class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
        </svg>
      </button>
    </div>
  </header>
</template>
