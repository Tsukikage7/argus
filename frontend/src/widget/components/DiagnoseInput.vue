<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  diagnose: [input: string]
}>()

const input = ref('')

function submit() {
  const text = input.value.trim()
  if (!text) return
  emit('diagnose', text)
}
</script>

<template>
  <div class="px-4 pb-4">
    <div class="relative">
      <input
        v-model="input"
        type="text"
        placeholder="描述你遇到的问题..."
        class="w-full rounded-xl border border-white/20 bg-white/5 px-4 py-3 pr-12 text-sm text-white placeholder-white/30 outline-none backdrop-blur-sm transition focus:border-indigo-400/50 focus:ring-1 focus:ring-indigo-400/30"
        @keydown.enter="submit"
      />
      <button
        class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg bg-gradient-to-r from-indigo-500 to-purple-500 px-3 py-1.5 text-xs font-medium text-white transition hover:opacity-90 active:scale-95 disabled:opacity-40"
        :disabled="!input.trim()"
        @click="submit"
      >
        诊断
      </button>
    </div>
  </div>
</template>
