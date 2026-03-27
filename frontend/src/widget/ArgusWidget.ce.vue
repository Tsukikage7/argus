<script setup lang="ts">
import { computed } from 'vue'
import { useWidgetApi } from './composables/useWidgetApi'
import WidgetHeader from './components/WidgetHeader.vue'
import DiagnoseInput from './components/DiagnoseInput.vue'
import InferenceStream from './components/InferenceStream.vue'
import ResultCard from './components/ResultCard.vue'
import type { Diagnosis } from './composables/types'

const props = defineProps<{
  apiKey: string
  baseUrl: string
}>()

const api = useWidgetApi(props.apiKey, props.baseUrl)

const inputSummary = computed(() => '')

function handleDiagnose(input: string) {
  api.diagnose(input)
}

const defaultDiagnosis: Diagnosis = {
  root_cause: '',
  confidence: 0,
  affected_services: [],
  recovery_suggestion: '',
  summary: '',
}
</script>

<template>
  <div class="argus-widget-root flex max-h-[600px] w-full max-w-[400px] flex-col overflow-hidden rounded-2xl border border-white/15 bg-[#1a1a2e]/90 shadow-2xl backdrop-blur-xl">
    <WidgetHeader :status="api.status.value" :input-summary="inputSummary" />

    <!-- 输入态 -->
    <DiagnoseInput v-if="api.status.value === 'idle'" @diagnose="handleDiagnose" />

    <!-- 推理态 -->
    <InferenceStream
      v-else-if="api.status.value === 'diagnosing'"
      :steps="api.steps.value"
      :is-active="true"
    />

    <!-- 结论态 -->
    <ResultCard
      v-else
      :diagnosis="api.diagnosis.value ?? defaultDiagnosis"
      :error="api.error.value"
      @retry="api.reset()"
    />

    <!-- 缺少 API Key 提示 -->
    <div v-if="!apiKey" class="px-4 py-6 text-center text-xs text-red-300/70">
      Missing API Key configuration
    </div>
  </div>
</template>

<style>
@import "tailwindcss" source("../");

@keyframes fadeSlideIn {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(400%); }
}
</style>
