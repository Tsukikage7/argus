<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

defineProps<{ collapsed: boolean }>()
defineEmits<{ (e: 'toggle'): void }>()

const route = useRoute()

interface NavItem {
  path: string
  label: string
  icon: string
}

interface NavGroup {
  title: string
  items: NavItem[]
}

const groups: NavGroup[] = [
  {
    title: '概览',
    items: [
      { path: '/dashboard', label: '总览', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6' },
    ],
  },
  {
    title: '可观测',
    items: [
      { path: '/topology', label: '服务拓扑', icon: 'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1' },
      { path: '/logs', label: '日志探索', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
      { path: '/traces', label: '链路追踪', icon: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z' },
      { path: '/alerts', label: '告警中心', icon: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9' },
    ],
  },
  {
    title: '智能体',
    items: [
      { path: '/chat', label: '智能体聊天', icon: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z' },
      { path: '/replay', label: '故障回放', icon: 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15' },
      { path: '/tasks', label: '任务列表', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4' },
    ],
  },
  {
    title: '管理',
    items: [
      { path: '/capability', label: '能力展示', icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z' },
      { path: '/settings', label: '设置', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
    ],
  },
]

const currentPath = computed(() => route.path)

function isActive(path: string) {
  return currentPath.value === path || currentPath.value.startsWith(path + '/')
}
</script>

<template>
  <nav
    class="h-full flex flex-col border-r border-base-300/50 bg-base-200/50 transition-all duration-200 overflow-hidden"
    :style="{ width: collapsed ? '64px' : '240px' }"
  >
    <!-- Logo -->
    <div class="h-14 flex items-center px-4 border-b border-base-300/50 shrink-0">
      <div
        class="w-8 h-8 rounded-lg flex items-center justify-center text-white font-bold text-sm shrink-0"
        style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
      >
        A
      </div>
      <transition name="fade">
        <div v-if="!collapsed" class="ml-3 flex flex-col leading-tight overflow-hidden">
          <span class="font-bold text-sm text-base-content whitespace-nowrap">Argus</span>
          <span class="text-[9px] text-base-content/30 whitespace-nowrap">智能可观测平台</span>
        </div>
      </transition>
    </div>

    <!-- 菜单 -->
    <div class="flex-1 overflow-y-auto py-3 scroller">
      <div v-for="group in groups" :key="group.title" class="mb-4">
        <div
          v-if="!collapsed"
          class="px-4 mb-1 text-[0.625rem] font-semibold text-base-content/30 uppercase tracking-wider"
        >
          {{ group.title }}
        </div>
        <router-link
          v-for="item in group.items"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 mx-2 px-3 py-2 rounded-lg text-sm transition-all"
          :class="isActive(item.path)
            ? 'bg-indigo-500/15 text-indigo-400'
            : 'text-base-content/50 hover:text-base-content/80 hover:bg-base-300/30'"
          :title="collapsed ? item.label : undefined"
        >
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" :d="item.icon" />
          </svg>
          <span v-if="!collapsed" class="whitespace-nowrap">{{ item.label }}</span>
        </router-link>
      </div>
    </div>

    <!-- 折叠按钮 -->
    <button
      class="h-10 flex items-center justify-center border-t border-base-300/50 text-base-content/30
             hover:text-base-content/60 transition-colors shrink-0"
      @click="$emit('toggle')"
    >
      <svg
        class="w-4 h-4 transition-transform duration-200"
        :class="{ 'rotate-180': collapsed }"
        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
      </svg>
    </button>
  </nav>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
