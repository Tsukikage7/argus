<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import VueApexCharts from 'vue3-apexcharts'
import { useAdminApi, type AdminTenant, type AdminAPIKey, type UsageBucket, type UsageOverview } from '@/composables/useAdminApi'

// 局部注册 ApexCharts 组件
const apexchart = VueApexCharts

const props = defineProps<{ id: string }>()
const { getTenant, listKeys, createKey, getUsage } = useAdminApi()

// 数据
const tenant = ref<AdminTenant | null>(null)
const keys = ref<AdminAPIKey[]>([])
const usage = ref<{ overview: UsageOverview; buckets: UsageBucket[] } | null>(null)
const loading = ref(true)

// Tab
const activeTab = ref<'keys' | 'usage'>('keys')

// 创建 Key Modal
const showCreateKeyModal = ref(false)
const keyName = ref('')
const createKeyLoading = ref(false)
const newKeyPlaintext = ref('')
const showNewKeyAlert = ref(false)

// count-up 动画
const animatedDiagnoses = ref(0)
const animatedReplays = ref(0)
const animatedCalls = ref(0)

function animateCount(target: number, setter: (v: number) => void, duration = 800) {
  const start = performance.now()
  function step(now: number) {
    const progress = Math.min((now - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    setter(Math.round(eased * target))
    if (progress < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}

async function loadData() {
  loading.value = true
  try {
    const [t, k, u] = await Promise.all([
      getTenant(props.id),
      listKeys(props.id),
      getUsage(props.id),
    ])
    tenant.value = t
    keys.value = k
    usage.value = u

    // 触发 count-up 动画
    animateCount(u.overview.total_diagnoses, v => animatedDiagnoses.value = v)
    animateCount(u.overview.total_replays, v => animatedReplays.value = v)
    animateCount(u.overview.total_api_calls, v => animatedCalls.value = v)
  } catch {
    // 静默处理
  } finally {
    loading.value = false
  }
}

async function handleCreateKey() {
  if (!keyName.value.trim()) return
  createKeyLoading.value = true
  try {
    const resp = await createKey(props.id, keyName.value.trim())
    newKeyPlaintext.value = resp.key
    showNewKeyAlert.value = true
    showCreateKeyModal.value = false
    keyName.value = ''
    keys.value = await listKeys(props.id)
  } catch {
    // 静默处理
  } finally {
    createKeyLoading.value = false
  }
}

function copyKey() {
  navigator.clipboard.writeText(newKeyPlaintext.value)
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

// ApexCharts 配置
const chartOptions = computed(() => {
  if (!usage.value) return null
  return {
    chart: {
      type: 'area' as const,
      height: 280,
      toolbar: { show: false },
      background: 'transparent',
      foreColor: 'oklch(var(--bc) / 0.4)',
    },
    colors: ['#6366f1', '#a78bfa', '#34d399'],
    stroke: { curve: 'smooth' as const, width: 2 },
    fill: {
      type: 'gradient',
      gradient: { opacityFrom: 0.3, opacityTo: 0.05 },
    },
    dataLabels: { enabled: false },
    xaxis: {
      categories: usage.value.buckets.map(b => b.date.slice(5)),
      labels: { style: { fontSize: '10px' } },
    },
    yaxis: { labels: { style: { fontSize: '10px' } } },
    grid: {
      borderColor: 'oklch(var(--b3) / 0.3)',
      strokeDashArray: 4,
    },
    legend: { position: 'top' as const, fontSize: '11px' },
    tooltip: { theme: 'dark' },
  }
})

const chartSeries = computed(() => {
  if (!usage.value) return []
  return [
    { name: '诊断次数', data: usage.value.buckets.map(b => b.diagnoses) },
    { name: '回放次数', data: usage.value.buckets.map(b => b.replays) },
    { name: 'API 调用', data: usage.value.buckets.map(b => b.api_calls) },
  ]
})

onMounted(loadData)
</script>

<template>
  <div>
    <!-- 加载中 -->
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-md text-indigo-400"></span>
    </div>

    <template v-else-if="tenant">
      <!-- 信息卡片 -->
      <div class="glass-card rounded-xl p-5 mb-5">
        <div class="flex items-start justify-between">
          <div>
            <h1 class="text-lg font-bold text-base-content">{{ tenant.name }}</h1>
            <p class="text-xs text-base-content/40 font-mono mt-0.5">{{ tenant.slug }}</p>
          </div>
          <span
            class="badge badge-sm text-[10px]"
            :class="tenant.status === 'active' ? 'badge-success badge-outline' : 'badge-warning badge-outline'"
          >
            {{ tenant.status }}
          </span>
        </div>
        <div class="grid grid-cols-3 gap-4 mt-4 text-xs">
          <div>
            <span class="text-base-content/40">ID</span>
            <p class="text-base-content/70 font-mono text-[10px] mt-0.5 break-all">{{ tenant.id }}</p>
          </div>
          <div>
            <span class="text-base-content/40">Allowed Origins</span>
            <p class="text-base-content/70 mt-0.5">
              {{ tenant.allowed_origins.length > 0 ? tenant.allowed_origins.join(', ') : '未配置' }}
            </p>
          </div>
          <div>
            <span class="text-base-content/40">创建时间</span>
            <p class="text-base-content/70 mt-0.5">{{ formatDate(tenant.created_at) }}</p>
          </div>
        </div>
      </div>

      <!-- Tab 切换 -->
      <div class="flex border-b border-base-300/50 mb-5">
        <button
          v-for="tab in [{ key: 'keys', label: 'API Keys' }, { key: 'usage', label: '用量统计' }] as const"
          :key="tab.key"
          class="px-4 py-2 text-sm font-medium transition-all border-b-2 -mb-[1px]"
          :class="activeTab === tab.key
            ? 'text-indigo-400 border-indigo-400'
            : 'text-base-content/40 border-transparent hover:text-base-content/60'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- ═══ API Keys Tab ═══ -->
      <div v-if="activeTab === 'keys'">
        <!-- 新 Key 提示 -->
        <div v-if="showNewKeyAlert" class="alert bg-emerald-900/20 border-emerald-500/30 mb-4">
          <div class="flex-1">
            <p class="text-xs text-emerald-400 font-medium mb-1">API Key 已创建，请立即复制保存（仅显示一次）</p>
            <code class="text-xs text-emerald-300 bg-emerald-900/30 px-2 py-1 rounded font-mono break-all">
              {{ newKeyPlaintext }}
            </code>
          </div>
          <div class="flex gap-2">
            <button class="btn btn-ghost btn-xs text-emerald-400" @click="copyKey">复制</button>
            <button class="btn btn-ghost btn-xs text-base-content/30" @click="showNewKeyAlert = false">关闭</button>
          </div>
        </div>

        <div class="flex justify-between items-center mb-3">
          <h2 class="text-sm font-medium text-base-content/70">API Keys</h2>
          <button
            class="btn btn-sm btn-outline btn-primary text-xs"
            @click="showCreateKeyModal = true"
          >
            创建 Key
          </button>
        </div>

        <div class="glass-card rounded-xl overflow-hidden">
          <table class="table table-sm w-full">
            <thead>
              <tr class="text-xs text-base-content/40">
                <th>名称</th>
                <th>前缀</th>
                <th>状态</th>
                <th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="k in keys" :key="k.id" class="hover:bg-base-200/30">
                <td class="text-sm font-medium text-base-content">{{ k.name }}</td>
                <td class="text-xs text-base-content/50 font-mono">{{ k.prefix }}***</td>
                <td>
                  <span
                    class="badge badge-sm text-[10px]"
                    :class="{
                      'badge-success badge-outline': k.status === 'active',
                      'badge-warning badge-outline': k.status === 'rotating',
                      'badge-error badge-outline': k.status === 'revoked',
                    }"
                  >
                    {{ k.status }}
                  </span>
                </td>
                <td class="text-xs text-base-content/40">{{ formatDate(k.created_at) }}</td>
              </tr>
              <tr v-if="keys.length === 0">
                <td colspan="4" class="text-center text-xs text-base-content/30 py-6">暂无 API Key</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- ═══ 用量统计 Tab ═══ -->
      <div v-else-if="activeTab === 'usage' && usage">
        <!-- 概览卡片 -->
        <div class="grid grid-cols-3 gap-4 mb-5">
          <div
            v-for="stat in [
              { label: '诊断次数', value: animatedDiagnoses, color: '#6366f1' },
              { label: '回放次数', value: animatedReplays, color: '#a78bfa' },
              { label: 'API 调用', value: animatedCalls, color: '#34d399' },
            ]"
            :key="stat.label"
            class="glass-card rounded-xl p-4 hover:translate-y-[-2px] transition-transform"
          >
            <p class="text-xs text-base-content/40 mb-1">{{ stat.label }}</p>
            <p class="text-2xl font-bold" :style="{ color: stat.color }">
              {{ stat.value.toLocaleString() }}
            </p>
            <p class="text-[10px] text-base-content/30 mt-0.5">近 {{ usage.overview.period }}</p>
          </div>
        </div>

        <!-- 时间序列图 -->
        <div class="glass-card rounded-xl p-4">
          <h3 class="text-sm font-medium text-base-content/70 mb-3">用量趋势（近 30 天）</h3>
          <apexchart
            v-if="chartOptions"
            type="area"
            :options="chartOptions"
            :series="chartSeries"
            height="280"
          />
        </div>
      </div>
    </template>

    <!-- ═══ 创建 Key Modal ═══ -->
    <dialog :open="showCreateKeyModal" class="modal" :class="{ 'modal-open': showCreateKeyModal }">
      <div class="modal-box bg-base-200 max-w-sm">
        <h3 class="font-bold text-base mb-4">创建 API Key</h3>
        <form @submit.prevent="handleCreateKey" class="space-y-3">
          <div>
            <label class="block text-xs text-base-content/50 mb-1">Key 名称</label>
            <input
              v-model="keyName"
              type="text"
              placeholder="例如 Production、Staging"
              class="input input-bordered input-sm w-full text-xs"
            />
          </div>
          <div class="modal-action">
            <button type="button" class="btn btn-sm btn-ghost" @click="showCreateKeyModal = false">取消</button>
            <button
              type="submit"
              class="btn btn-sm text-white border-0"
              style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
              :disabled="createKeyLoading"
            >
              <span v-if="createKeyLoading" class="loading loading-spinner loading-xs"></span>
              {{ createKeyLoading ? '创建中...' : '创建' }}
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showCreateKeyModal = false">close</button>
      </form>
    </dialog>
  </div>
</template>
