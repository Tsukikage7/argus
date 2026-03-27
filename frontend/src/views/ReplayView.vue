<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { startReplay } from '@/composables/useApi'
import type { ReplayType } from '@/types'

import ReplayInput from '@/components/Control/ReplayInput.vue'

const router = useRouter()
const loading = ref(false)
const errorMsg = ref('')

async function handleReplay(params: {
  type: ReplayType
  scenario: string
  config: { fault_intensity: number; traffic_rate_multiplier: number; auto_diagnose: boolean }
}) {
  loading.value = true
  errorMsg.value = ''

  try {
    await startReplay(params)
    // 创建成功，立即跳转任务列表
    router.push('/tasks')
  } catch (e: unknown) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
    loading.value = false
  }
}
</script>

<template>
  <div>
    <div class="glass-card rounded-xl p-4">
      <ReplayInput :loading="loading" @replay="handleReplay" />

      <!-- 错误提示 -->
      <div
        v-if="errorMsg"
        class="mt-3 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs"
      >
        {{ errorMsg }}
      </div>
    </div>
  </div>
</template>
