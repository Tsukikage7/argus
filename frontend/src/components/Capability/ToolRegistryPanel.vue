<script setup lang="ts">
// Tool 注册表数据（与后端 tool.Registry 对应）
const tools = [
  {
    name: 'es_query_logs',
    description: '查询 Elasticsearch 日志，支持关键词搜索、时间范围过滤、命名空间筛选',
    category: '数据采集',
    params: ['keyword', 'namespace', 'time_range', 'level', 'limit'],
    color: 'indigo',
  },
  {
    name: 'trace_analyze',
    description: '分析分布式链路追踪数据，定位慢调用和错误传播路径',
    category: '数据采集',
    params: ['request_uuid', 'service', 'time_range'],
    color: 'indigo',
  },
  {
    name: 'exec_command',
    description: '执行运维恢复命令（dry-run 模式），如重启服务、扩容、清理连接池',
    category: '恢复执行',
    params: ['command', 'target_service', 'dry_run'],
    color: 'amber',
  },
  {
    name: 'send_notification',
    description: '发送通知到企业微信/Webhook，推送诊断结果和告警信息',
    category: '通知推送',
    params: ['channel', 'message', 'severity'],
    color: 'emerald',
  },
]
</script>

<template>
  <div class="space-y-3">
    <div
      v-for="t in tools"
      :key="t.name"
      class="glass-card rounded-xl p-4 transition-all hover:border-indigo-500/30"
    >
      <div class="flex items-center gap-2 mb-2">
        <span
          class="px-2 py-0.5 rounded text-[0.625rem] font-bold uppercase"
          :class="{
            'bg-indigo-500/20 text-indigo-400': t.color === 'indigo',
            'bg-amber-500/20 text-amber-400': t.color === 'amber',
            'bg-emerald-500/20 text-emerald-400': t.color === 'emerald',
          }"
        >
          {{ t.category }}
        </span>
        <code class="text-sm font-mono text-base-content/80">{{ t.name }}</code>
      </div>
      <p class="text-xs text-base-content/60 mb-2">{{ t.description }}</p>
      <div class="flex flex-wrap gap-1">
        <span
          v-for="p in t.params"
          :key="p"
          class="px-1.5 py-0.5 rounded text-[0.5625rem] bg-base-300/50 text-base-content/50 font-mono"
        >
          {{ p }}
        </span>
      </div>
    </div>
  </div>
</template>
