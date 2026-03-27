<script setup lang="ts">
// Agent 配置参数（与后端 agent.Config 对应）
const configs = [
  { key: 'max_steps', label: '最大推理步数', value: '15', desc: 'Agent 单次诊断最多执行的 Think-Act-Observe 循环次数' },
  { key: 'auto_recover_threshold', label: '自动恢复阈值', value: '0.8', desc: '置信度 >= 0.8 时自动执行恢复操作' },
  { key: 'confirm_threshold', label: '确认阈值', value: '0.5', desc: '置信度 >= 0.5 时需要人工确认后执行恢复' },
  { key: 'timeout', label: '诊断超时', value: '5min', desc: '单次诊断任务的最大执行时间' },
  { key: 'model', label: 'LLM 模型', value: 'qwen-plus', desc: '推理使用的大语言模型' },
]

const tools = [
  { name: 'es_query_logs', desc: '查询 ES 日志' },
  { name: 'trace_analyze', desc: '分析链路追踪' },
  { name: 'exec_command', desc: '执行恢复命令（dry-run）' },
  { name: 'send_notification', desc: '发送通知' },
]
</script>

<template>
  <div class="glass-card rounded-xl p-4">
    <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-3">Agent 配置</div>

    <!-- 参数列表 -->
    <div class="space-y-2 mb-4">
      <div
        v-for="cfg in configs"
        :key="cfg.key"
        class="flex items-center justify-between p-3 rounded-lg bg-base-200/50"
      >
        <div>
          <div class="text-sm text-base-content/80">{{ cfg.label }}</div>
          <div class="text-[0.625rem] text-base-content/40">{{ cfg.desc }}</div>
        </div>
        <code class="text-sm font-mono text-indigo-400 bg-indigo-500/10 px-2 py-0.5 rounded">
          {{ cfg.value }}
        </code>
      </div>
    </div>

    <!-- 已注册 Tools -->
    <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-2">已注册 Tools</div>
    <div class="flex flex-wrap gap-2">
      <span
        v-for="t in tools"
        :key="t.name"
        class="px-2 py-1 rounded-lg text-xs bg-base-200/50 text-base-content/60"
        :title="t.desc"
      >
        <code class="font-mono text-amber-400">{{ t.name }}</code>
        <span class="text-base-content/40 ml-1">{{ t.desc }}</span>
      </span>
    </div>
  </div>
</template>
