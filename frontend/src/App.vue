<script setup lang="ts">
import { onMounted } from 'vue'
import { loadTopology } from '@/store/useTaskStore'

// 应用启动时从后端加载拓扑配置
onMounted(() => {
  loadTopology()
})
</script>

<template>
  <router-view />
</template>
