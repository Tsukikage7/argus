<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { listScenarios, type Scenario } from '@/composables/useApi'

const scenarios = ref<Scenario[]>([])
const loading = ref(true)
const filter = ref<'all' | 'preset' | 'captured'>('all')

onMounted(async () => {
  try {
    scenarios.value = await listScenarios()
  } catch (e) {
    console.error('加载场景失败:', e)
  } finally {
    loading.value = false
  }
})

const filtered = computed(() => {
  if (filter.value === 'all') return scenarios.value
  return scenarios.value.filter(s => s.type === filter.value)
})
</script>

<template>
  <div>
    <!-- 过滤 -->
    <div class="flex gap-1 mb-4 p-1 rounded-lg bg-base-200/50 w-fit">
      <button
        v-for="f in [
          { key: 'all', label: '全部' },
          { key: 'preset', label: '预设' },
          { key: 'captured', label: '沉淀' },
        ]"
        :key="f.key"
        class="px-2.5 py-1 rounded-md text-[0.6875rem] font-medium transition-all"
        :class="filter === f.key
          ? 'bg-indigo-500/20 text-indigo-400'
          : 'text-base-content/50 hover:text-base-content/70'"
        @click="filter = f.key as any"
      >
        {{ f.label }}
      </button>
    </div>

    <div v-if="loading" class="text-center text-sm py-8 text-base-content/30">加载中…</div>

    <div v-else-if="filtered.length === 0" class="text-center text-sm py-8 text-base-content/30">
      暂无场景
    </div>

    <div v-else class="space-y-2">
      <div
        v-for="s in filtered"
        :key="s.name"
        class="glass-card rounded-xl p-4 transition-all hover:border-indigo-500/30"
      >
        <div class="flex items-center gap-2 mb-1">
          <span
            class="px-1.5 py-0.5 rounded text-[0.5625rem] font-bold uppercase"
            :class="s.type === 'captured'
              ? 'bg-emerald-500/20 text-emerald-400'
              : 'bg-indigo-500/20 text-indigo-400'"
          >
            {{ s.type === 'captured' ? '沉淀' : '预设' }}
          </span>
          <span class="text-sm font-medium text-base-content/80">{{ s.name }}</span>
          <span
            v-if="s.confidence"
            class="ml-auto text-[0.625rem] text-base-content/40"
          >
            置信度 {{ Math.round(s.confidence * 100) }}%
          </span>
        </div>
        <p class="text-xs text-base-content/60">{{ s.description }}</p>
      </div>
    </div>
  </div>
</template>
