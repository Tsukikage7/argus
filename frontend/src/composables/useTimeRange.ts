import { ref, computed } from 'vue'

export type TimeRangePreset = '15m' | '1h' | '6h' | '24h' | '7d'

export interface TimeRange {
  preset: TimeRangePreset | 'custom'
  from: Date
  to: Date
}

const PRESET_MS: Record<TimeRangePreset, number> = {
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
}

const PRESET_LABELS: Record<TimeRangePreset, string> = {
  '15m': '15 分钟',
  '1h': '1 小时',
  '6h': '6 小时',
  '24h': '24 小时',
  '7d': '7 天',
}

// 全局单例状态
const activePreset = ref<TimeRangePreset>('1h')
const customFrom = ref<Date | null>(null)
const customTo = ref<Date | null>(null)

export function useTimeRange() {
  const isCustom = computed(() => customFrom.value !== null && customTo.value !== null)

  const currentRange = computed<TimeRange>(() => {
    if (isCustom.value && customFrom.value && customTo.value) {
      return { preset: 'custom', from: customFrom.value, to: customTo.value }
    }
    const now = new Date()
    const ms = PRESET_MS[activePreset.value]
    return { preset: activePreset.value, from: new Date(now.getTime() - ms), to: now }
  })

  const label = computed(() => {
    if (isCustom.value) return '自定义'
    return PRESET_LABELS[activePreset.value]
  })

  /** API 查询参数格式 */
  const queryParam = computed(() => {
    if (isCustom.value) return `custom`
    return activePreset.value
  })

  function setPreset(preset: TimeRangePreset) {
    activePreset.value = preset
    customFrom.value = null
    customTo.value = null
  }

  function setCustomRange(from: Date, to: Date) {
    customFrom.value = from
    customTo.value = to
  }

  return {
    activePreset,
    presets: Object.keys(PRESET_MS) as TimeRangePreset[],
    presetLabels: PRESET_LABELS,
    isCustom,
    currentRange,
    label,
    queryParam,
    setPreset,
    setCustomRange,
  }
}
