<script setup lang="ts">
import IntegrationPanel from '@/components/Settings/IntegrationPanel.vue'
import AgentConfigPanel from '@/components/Settings/AgentConfigPanel.vue'
</script>

<template>
  <div class="p-6 space-y-6">
    <h2 class="text-lg font-semibold text-base-content/80">系统设置</h2>

    <!-- 系统信息 -->
    <div class="glass-card rounded-xl p-4">
      <div class="text-[0.8125rem] font-semibold text-base-content/70 mb-3">系统信息</div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div>
          <div class="text-[0.625rem] text-base-content/40 uppercase">平台</div>
          <div class="text-sm text-base-content/70">Argus v1.0</div>
        </div>
        <div>
          <div class="text-[0.625rem] text-base-content/40 uppercase">运行时</div>
          <div class="text-sm text-base-content/70">Go 1.25+</div>
        </div>
        <div>
          <div class="text-[0.625rem] text-base-content/40 uppercase">架构</div>
          <div class="text-sm text-base-content/70">ReAct Agent + CQRS</div>
        </div>
        <div>
          <div class="text-[0.625rem] text-base-content/40 uppercase">API 版本</div>
          <div class="text-sm text-base-content/70">v1</div>
        </div>
      </div>
    </div>

    <!-- 集成配置 -->
    <IntegrationPanel />

    <!-- Agent 配置 -->
    <AgentConfigPanel />
  </div>
</template>
