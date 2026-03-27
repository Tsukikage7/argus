<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminApi, type AdminTenant } from '@/composables/useAdminApi'

const router = useRouter()
const { listTenants, createTenant } = useAdminApi()

const tenants = ref<AdminTenant[]>([])
const loading = ref(true)
const searchQuery = ref('')
const statusFilter = ref<string>('all')

// 创建 Modal
const showCreateModal = ref(false)
const createForm = ref({ slug: '', name: '', allowed_origins: '' })
const createLoading = ref(false)
const createError = ref('')

// 删除确认 Modal
const showDeleteModal = ref(false)
const deletingTenant = ref<AdminTenant | null>(null)

const filtered = computed(() => {
  return tenants.value.filter(t => {
    const matchSearch = !searchQuery.value
      || t.name.toLowerCase().includes(searchQuery.value.toLowerCase())
      || t.slug.toLowerCase().includes(searchQuery.value.toLowerCase())
    const matchStatus = statusFilter.value === 'all' || t.status === statusFilter.value
    return matchSearch && matchStatus
  })
})

async function loadTenants() {
  loading.value = true
  try {
    tenants.value = await listTenants()
  } catch {
    // 静默处理
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  createError.value = ''
  if (!createForm.value.slug || !createForm.value.name) {
    createError.value = 'Slug 和名称为必填项'
    return
  }

  createLoading.value = true
  try {
    const origins = createForm.value.allowed_origins
      ? createForm.value.allowed_origins.split(',').map(s => s.trim()).filter(Boolean)
      : []
    await createTenant({
      slug: createForm.value.slug,
      name: createForm.value.name,
      allowed_origins: origins,
    })
    showCreateModal.value = false
    createForm.value = { slug: '', name: '', allowed_origins: '' }
    await loadTenants()
  } catch (e: unknown) {
    createError.value = e instanceof Error ? e.message : '创建失败'
  } finally {
    createLoading.value = false
  }
}

function confirmDelete(t: AdminTenant) {
  deletingTenant.value = t
  showDeleteModal.value = true
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', year: 'numeric' })
}

onMounted(loadTenants)
</script>

<template>
  <div>
    <!-- 页头 -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-xl font-bold text-base-content">租户管理</h1>
        <p class="text-xs text-base-content/40 mt-0.5">管理 SaaS 租户和接入配置</p>
      </div>
      <button
        class="btn btn-sm text-white border-0 text-xs"
        style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
        @click="showCreateModal = true"
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        新建租户
      </button>
    </div>

    <!-- 搜索 + 筛选 -->
    <div class="flex gap-3 mb-4">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索租户名称或 Slug..."
        class="input input-bordered input-sm flex-1 bg-base-200/50 text-xs"
      />
      <select
        v-model="statusFilter"
        class="select select-bordered select-sm bg-base-200/50 text-xs"
      >
        <option value="all">全部状态</option>
        <option value="active">Active</option>
        <option value="suspended">Suspended</option>
      </select>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-md text-indigo-400"></span>
    </div>

    <!-- 租户表格 -->
    <div v-else class="glass-card rounded-xl overflow-hidden">
      <table class="table table-sm w-full">
        <thead>
          <tr class="text-xs text-base-content/40">
            <th>租户</th>
            <th>Slug</th>
            <th>状态</th>
            <th>CORS Origins</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="t in filtered"
            :key="t.id"
            class="hover:bg-base-200/30 transition-colors cursor-pointer"
            @click="router.push(`/admin/tenants/${t.id}`)"
          >
            <td class="font-medium text-sm text-base-content">{{ t.name }}</td>
            <td class="text-xs text-base-content/50 font-mono">{{ t.slug }}</td>
            <td>
              <span
                class="badge badge-sm text-[10px]"
                :class="t.status === 'active' ? 'badge-success badge-outline' : 'badge-warning badge-outline'"
              >
                {{ t.status }}
              </span>
            </td>
            <td class="text-xs text-base-content/40">
              {{ t.allowed_origins.length > 0 ? t.allowed_origins.join(', ') : '-' }}
            </td>
            <td class="text-xs text-base-content/40">{{ formatDate(t.created_at) }}</td>
            <td class="text-right">
              <button
                class="btn btn-ghost btn-xs text-error/50 hover:text-error"
                title="删除"
                @click.stop="confirmDelete(t)"
              >
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="filtered.length === 0">
            <td colspan="6" class="text-center text-xs text-base-content/30 py-8">
              {{ searchQuery || statusFilter !== 'all' ? '没有匹配的租户' : '暂无租户' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- ═══ 创建 Modal ═══ -->
    <dialog :open="showCreateModal" class="modal" :class="{ 'modal-open': showCreateModal }">
      <div class="modal-box bg-base-200 max-w-md">
        <h3 class="font-bold text-base mb-4">新建租户</h3>
        <form @submit.prevent="handleCreate" class="space-y-3">
          <div>
            <label class="block text-xs text-base-content/50 mb-1">Slug（唯一标识）</label>
            <input
              v-model="createForm.slug"
              type="text"
              placeholder="例如 acme-corp"
              class="input input-bordered input-sm w-full text-xs"
              pattern="[a-z0-9][a-z0-9-]{1,30}[a-z0-9]"
            />
            <p class="text-[10px] text-base-content/30 mt-0.5">3-32 位小写字母、数字和连字符</p>
          </div>
          <div>
            <label class="block text-xs text-base-content/50 mb-1">名称</label>
            <input
              v-model="createForm.name"
              type="text"
              placeholder="例如 Acme Corporation"
              class="input input-bordered input-sm w-full text-xs"
            />
          </div>
          <div>
            <label class="block text-xs text-base-content/50 mb-1">Allowed Origins（逗号分隔，可选）</label>
            <input
              v-model="createForm.allowed_origins"
              type="text"
              placeholder="https://example.com, http://localhost:3000"
              class="input input-bordered input-sm w-full text-xs"
            />
          </div>
          <div v-if="createError" class="text-xs text-error">{{ createError }}</div>
          <div class="modal-action">
            <button type="button" class="btn btn-sm btn-ghost" @click="showCreateModal = false">取消</button>
            <button
              type="submit"
              class="btn btn-sm text-white border-0"
              style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
              :disabled="createLoading"
            >
              <span v-if="createLoading" class="loading loading-spinner loading-xs"></span>
              {{ createLoading ? '创建中...' : '创建' }}
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showCreateModal = false">close</button>
      </form>
    </dialog>

    <!-- ═══ 删除确认 Modal ═══ -->
    <dialog :open="showDeleteModal" class="modal" :class="{ 'modal-open': showDeleteModal }">
      <div class="modal-box bg-base-200 max-w-sm">
        <h3 class="font-bold text-base mb-2">确认删除</h3>
        <p class="text-sm text-base-content/60">
          确定要删除租户 <strong>{{ deletingTenant?.name }}</strong>（{{ deletingTenant?.slug }}）吗？此操作不可撤销。
        </p>
        <div class="modal-action">
          <button class="btn btn-sm btn-ghost" @click="showDeleteModal = false">取消</button>
          <button class="btn btn-sm btn-error" @click="showDeleteModal = false">删除</button>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button @click="showDeleteModal = false">close</button>
      </form>
    </dialog>
  </div>
</template>
