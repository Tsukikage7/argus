<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAdminApi } from '@/composables/useAdminApi'

const route = useRoute()
const router = useRouter()
const { useMock, init } = useAdminApi()

init()

const menuItems = [
  { path: '/admin/tenants', label: '租户管理', icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z' },
  { path: '/admin/integration', label: '集成指南', icon: 'M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4' },
]

const currentPath = computed(() => route.path)

// 面包屑
const breadcrumbs = computed(() => {
  const crumbs = [{ label: '管理控制台', path: '/admin' }]
  if (route.path.startsWith('/admin/tenants/') && route.params.id) {
    crumbs.push({ label: '租户管理', path: '/admin/tenants' })
    crumbs.push({ label: '租户详情', path: route.path })
  } else if (route.path === '/admin/tenants') {
    crumbs.push({ label: '租户管理', path: '/admin/tenants' })
  } else if (route.path === '/admin/integration') {
    crumbs.push({ label: '集成指南', path: '/admin/integration' })
  }
  return crumbs
})

function logout() {
  localStorage.removeItem('argus-admin-key')
  router.push('/admin/login')
}
</script>

<template>
  <div class="min-h-screen flex">
    <!-- ═══ 左侧侧边栏 ═══ -->
    <aside
      class="w-56 flex-shrink-0 flex flex-col border-r"
      style="border-color: oklch(var(--b3)); background: oklch(var(--b2))"
    >
      <!-- Logo -->
      <div class="h-14 flex items-center gap-2.5 px-5 border-b" style="border-color: oklch(var(--b3))">
        <div
          class="w-8 h-8 rounded-lg flex items-center justify-center text-white font-bold text-sm"
          style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
        >
          A
        </div>
        <div class="flex flex-col leading-tight">
          <span class="font-bold text-sm text-base-content">Argus</span>
          <span class="text-[9px] text-base-content/30">Admin Console</span>
        </div>
      </div>

      <!-- 菜单 -->
      <nav class="flex-1 py-3 px-3 space-y-1">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-all"
          :class="currentPath.startsWith(item.path)
            ? 'bg-indigo-600/20 text-indigo-400 font-medium'
            : 'text-base-content/50 hover:text-base-content/80 hover:bg-base-300/30'"
        >
          <svg class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" :d="item.icon" />
          </svg>
          {{ item.label }}
        </router-link>
      </nav>

      <!-- 底部 -->
      <div class="px-3 py-3 border-t space-y-2" style="border-color: oklch(var(--b3))">
        <!-- Mock 指示 -->
        <div
          v-if="useMock"
          class="px-3 py-1.5 rounded-lg text-[10px] text-amber-400 bg-amber-900/20 text-center"
        >
          Mock 模式
        </div>

        <!-- 返回诊断面板 -->
        <router-link
          to="/"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs text-base-content/40 hover:text-base-content/60 transition-colors"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          诊断面板
        </router-link>

        <!-- 退出 -->
        <button
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs text-red-400/60 hover:text-red-400 transition-colors w-full"
          @click="logout"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
          退出登录
        </button>
      </div>
    </aside>

    <!-- ═══ 右侧内容区 ═══ -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- 顶栏面包屑 -->
      <header
        class="h-14 flex items-center px-6 border-b sticky top-0 z-20 backdrop-blur"
        style="border-color: oklch(var(--b3)); background: color-mix(in oklch, oklch(var(--b1)) 85%, transparent)"
      >
        <div class="flex items-center gap-1.5 text-xs text-base-content/40">
          <template v-for="(crumb, i) in breadcrumbs" :key="crumb.path">
            <span v-if="i > 0" class="mx-1">/</span>
            <router-link
              v-if="i < breadcrumbs.length - 1"
              :to="crumb.path"
              class="hover:text-base-content/60 transition-colors"
            >
              {{ crumb.label }}
            </router-link>
            <span v-else class="text-base-content/70">{{ crumb.label }}</span>
          </template>
        </div>
      </header>

      <!-- 页面内容 -->
      <main class="flex-1 p-6 overflow-y-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>
