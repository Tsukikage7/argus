<script setup lang="ts">
import { useTaskStore } from '@/store/useTaskStore'
import type { AppMode } from '@/types'

const store = useTaskStore()

function handleSwitch(mode: AppMode) {
  store.setMode(mode)
}
</script>

<template>
  <!-- 模式切换 Tab -->
  <div class="flex gap-2">
    <button
      class="px-3.5 py-1.5 rounded-md text-xs font-medium cursor-pointer transition-all border"
      :class="store.mode === 'diagnose'
        ? 'bg-indigo-600 text-white border-indigo-600'
        : 'bg-transparent text-base-content/50 border-base-300 hover:text-base-content'"
      @click="handleSwitch('diagnose')"
    >
      诊断模式
    </button>
    <button
      class="px-3.5 py-1.5 rounded-md text-xs font-medium cursor-pointer transition-all border"
      :class="store.mode === 'replay'
        ? 'bg-indigo-600 text-white border-indigo-600'
        : 'bg-transparent text-base-content/50 border-base-300 hover:text-base-content'"
      @click="handleSwitch('replay')"
    >
      回放模式
    </button>
  </div>
</template>
