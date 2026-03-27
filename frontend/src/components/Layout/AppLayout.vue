<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const collapsed = ref(false)

function handleResize() {
  collapsed.value = window.innerWidth < 768
}

function toggleSidebar() {
  collapsed.value = !collapsed.value
}

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <div class="flex h-screen overflow-hidden">
    <!-- 侧边栏 -->
    <SideNav :collapsed="collapsed" @toggle="toggleSidebar" />

    <!-- 主内容区 -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <TopBar />
      <main class="flex-1 overflow-y-auto p-5 scroller">
        <router-view />
      </main>
    </div>
  </div>
</template>
