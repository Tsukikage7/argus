<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface IntegrationStatus {
  name: string
  type: string
  status: 'connected' | 'disconnected' | 'unknown'
  detail: string
}

const integrations = ref<IntegrationStatus[]>([])
const loading = ref(true)

onMounted(async () => {
  // 通过 health 端点检测连接状态
  try {
    const res = await fetch('/health')
    const ok = res.ok
    integrations.value = [
      { name: 'Elasticsearch', type: '日志存储', status: ok ? 'connected' : 'unknown', detail: 'localhost:9200 · 索引前缀 argus' },
      { name: 'Redis', type: '状态缓存', status: ok ? 'connected' : 'unknown', detail: 'localhost:6379 · TTL 24h' },
      { name: 'PostgreSQL', type: '历史持久化', status: ok ? 'connected' : 'unknown', detail: 'localhost:5432/argus' },
      { name: 'DashScope', type: 'LLM Provider', status: ok ? 'connected' : 'unknown', detail: 'qwen-plus · OpenAI 兼容接口' },
    ]
  } catch {
    integrations.value = [
      { name: 'Elasticsearch', type: '日志存储', status: 'disconnected', detail: 'localhost:9200' },
      { name: 'Redis', type: '状态缓存', status: 'disconnected', detail: 'localhost:6379' },
      { name: 'PostgreSQL', type: '历史持久化', status: 'disconnected', detail: 'localhost:5432/argus' },
      { name: 'DashScope', type: 'LLM Provider', status: 'disconnected', detail: 'qwen-plus' },
    ]
  } finally {
    loading.value = false
  }
})

function statusColor(s: string) {
  if (s === 'connected') return 'bg-emerald-500'
  if (s === 'disconnected') return 'bg-red-500'
  return 'bg-amber-500'
}

function statusLabel(s: string) {
  if (s === 'connected') return '已连接'
  if (s === 'disconnected') return '未连接'
  return '未知'
}
</script>

<template>
  <div class="glass-card rounded-xl p-4">
    <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-3">集成配置</div>

    <div v-if="loading" class="text-center text-sm py-4 text-base-content/30">检测中…</div>

    <div v-else class="space-y-2">
      <div
        v-for="item in integrations"
        :key="item.name"
        class="flex items-center justify-between p-3 rounded-lg bg-base-200/50"
      >
        <div class="flex items-center gap-3">
          <div class="w-2 h-2 rounded-full" :class="statusColor(item.status)"></div>
          <div>
            <div class="text-sm font-medium text-base-content/80">{{ item.name }}</div>
            <div class="text-[0.625rem] text-base-content/40">{{ item.type }} · {{ item.detail }}</div>
          </div>
        </div>
        <span
          class="text-[0.5625rem] font-semibold px-1.5 py-0.5 rounded"
          :class="{
            'bg-emerald-500/20 text-emerald-400': item.status === 'connected',
            'bg-red-500/20 text-red-400': item.status === 'disconnected',
            'bg-amber-500/20 text-amber-400': item.status === 'unknown',
          }"
        >
          {{ statusLabel(item.status) }}
        </span>
      </div>
    </div>
  </div>
</template>
