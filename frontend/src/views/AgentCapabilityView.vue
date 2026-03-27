<script setup lang="ts">
import ToolRegistryPanel from '@/components/Capability/ToolRegistryPanel.vue'
import ScenarioLibraryPanel from '@/components/Capability/ScenarioLibraryPanel.vue'
import AgentArchDiagram from '@/components/Capability/AgentArchDiagram.vue'

const activeTab = ref<'arch' | 'tools' | 'scenarios'>('arch')

import { ref } from 'vue'
</script>

<template>
  <div class="p-6">
    <h2 class="text-lg font-semibold text-base-content/80 mb-4">Agent 能力展示</h2>

    <!-- Tab 切换 -->
    <div class="flex gap-1 mb-6 p-1 rounded-lg bg-base-200/50 w-fit">
      <button
        v-for="tab in [
          { key: 'arch', label: 'ReAct 架构' },
          { key: 'tools', label: 'Tool 注册表' },
          { key: 'scenarios', label: '场景库' },
        ]"
        :key="tab.key"
        class="px-3 py-1.5 rounded-md text-xs font-medium transition-all"
        :class="activeTab === tab.key
          ? 'bg-indigo-500/20 text-indigo-400'
          : 'text-base-content/50 hover:text-base-content/70'"
        @click="activeTab = tab.key as any"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- 内容区 -->
    <AgentArchDiagram v-if="activeTab === 'arch'" />
    <ToolRegistryPanel v-else-if="activeTab === 'tools'" />
    <ScenarioLibraryPanel v-else-if="activeTab === 'scenarios'" />
  </div>
</template>
