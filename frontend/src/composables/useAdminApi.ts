import { ref } from 'vue'

// ═══ 类型定义 ═══

export interface AdminTenant {
  id: string
  slug: string
  name: string
  status: string
  allowed_origins: string[]
  created_at: string
  updated_at: string
}

export interface AdminAPIKey {
  id: string
  tenant_id: string
  prefix: string
  name: string
  status: string
  expires_at?: string
  created_at: string
}

export interface CreateKeyResponse {
  key: string
  api_key: AdminAPIKey
}

export interface UsageBucket {
  date: string
  diagnoses: number
  replays: number
  api_calls: number
}

export interface UsageOverview {
  total_diagnoses: number
  total_replays: number
  total_api_calls: number
  period: string
}

// ═══ Mock 数据 ═══

const MOCK_TENANTS: AdminTenant[] = [
  {
    id: 'a1b2c3d4-1111-2222-3333-444455556666',
    slug: 'acme-corp',
    name: 'Acme Corporation',
    status: 'active',
    allowed_origins: ['https://acme.example.com'],
    created_at: '2026-03-01T10:00:00Z',
    updated_at: '2026-03-15T14:30:00Z',
  },
  {
    id: 'b2c3d4e5-2222-3333-4444-555566667777',
    slug: 'globex-inc',
    name: 'Globex Inc',
    status: 'active',
    allowed_origins: ['https://globex.example.com', 'http://localhost:3000'],
    created_at: '2026-03-05T08:00:00Z',
    updated_at: '2026-03-18T09:15:00Z',
  },
  {
    id: 'c3d4e5f6-3333-4444-5555-666677778888',
    slug: 'initech',
    name: 'Initech Systems',
    status: 'suspended',
    allowed_origins: [],
    created_at: '2026-02-20T12:00:00Z',
    updated_at: '2026-03-10T16:00:00Z',
  },
]

const MOCK_KEYS: Record<string, AdminAPIKey[]> = {
  'a1b2c3d4-1111-2222-3333-444455556666': [
    { id: 'k1', tenant_id: 'a1b2c3d4-1111-2222-3333-444455556666', prefix: 'arg_acme-corp_', name: 'Production', status: 'active', created_at: '2026-03-01T10:05:00Z' },
    { id: 'k2', tenant_id: 'a1b2c3d4-1111-2222-3333-444455556666', prefix: 'arg_acme-corp_', name: 'Staging', status: 'active', created_at: '2026-03-10T11:00:00Z' },
  ],
  'b2c3d4e5-2222-3333-4444-555566667777': [
    { id: 'k3', tenant_id: 'b2c3d4e5-2222-3333-4444-555566667777', prefix: 'arg_globex-inc_', name: 'Default', status: 'active', created_at: '2026-03-05T08:10:00Z' },
  ],
}

function generateMockUsage(): UsageBucket[] {
  const buckets: UsageBucket[] = []
  const now = new Date()
  for (let i = 29; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    buckets.push({
      date: d.toISOString().slice(0, 10),
      diagnoses: Math.floor(Math.random() * 40) + 5,
      replays: Math.floor(Math.random() * 20) + 2,
      api_calls: Math.floor(Math.random() * 200) + 50,
    })
  }
  return buckets
}

// ═══ API 调用 ═══

function getAdminKey(): string {
  return localStorage.getItem('argus-admin-key') || ''
}

function adminHeaders(): HeadersInit {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${getAdminKey()}`,
  }
}

async function adminRequest<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...options,
    headers: { ...adminHeaders(), ...options?.headers },
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`API error ${res.status}: ${body}`)
  }
  return res.json() as Promise<T>
}

/** 模拟网络延迟 */
function delay(ms?: number): Promise<void> {
  const t = ms ?? (800 + Math.random() * 700)
  return new Promise(r => setTimeout(r, t))
}

// ═══ Mock 开关 ═══

const useMock = ref(false)

/** 检测后端是否可用，不可用则启用 Mock */
async function detectBackend(): Promise<boolean> {
  try {
    const res = await fetch('/health', { signal: AbortSignal.timeout(3000) })
    return res.ok
  } catch {
    return false
  }
}

// ═══ 公共 API ═══

/** 验证 AdminKey 是否有效 */
async function verifyAdminKey(key: string): Promise<boolean> {
  if (useMock.value) {
    await delay()
    // Mock 模式下任意非空 key 视为有效
    return key.length > 0
  }
  try {
    const res = await fetch('/admin/v1/tenants', {
      headers: { Authorization: `Bearer ${key}` },
    })
    return res.ok
  } catch {
    return false
  }
}

/** 获取租户列表 */
async function listTenants(): Promise<AdminTenant[]> {
  if (useMock.value) {
    await delay()
    return [...MOCK_TENANTS]
  }
  const data = await adminRequest<{ tenants: AdminTenant[] }>('/admin/v1/tenants')
  return data.tenants
}

/** 获取单个租户 */
async function getTenant(id: string): Promise<AdminTenant> {
  if (useMock.value) {
    await delay()
    const t = MOCK_TENANTS.find(t => t.id === id)
    if (!t) throw new Error('Tenant not found')
    return { ...t }
  }
  return adminRequest<AdminTenant>(`/admin/v1/tenants/${id}`)
}

/** 创建租户 */
async function createTenant(params: { slug: string; name: string; allowed_origins?: string[] }): Promise<AdminTenant> {
  if (useMock.value) {
    await delay()
    const t: AdminTenant = {
      id: crypto.randomUUID(),
      slug: params.slug,
      name: params.name,
      status: 'active',
      allowed_origins: params.allowed_origins || [],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    MOCK_TENANTS.push(t)
    return t
  }
  return adminRequest<AdminTenant>('/admin/v1/tenants', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

/** 获取租户的 API Key 列表 */
async function listKeys(tenantId: string): Promise<AdminAPIKey[]> {
  if (useMock.value) {
    await delay()
    return [...(MOCK_KEYS[tenantId] || [])]
  }
  const data = await adminRequest<{ keys: AdminAPIKey[] }>(`/admin/v1/tenants/${tenantId}/keys`)
  return data.keys
}

/** 创建 API Key */
async function createKey(tenantId: string, name: string): Promise<CreateKeyResponse> {
  if (useMock.value) {
    await delay()
    const tenant = MOCK_TENANTS.find(t => t.id === tenantId)
    const prefix = `arg_${tenant?.slug || 'unknown'}_`
    const key: AdminAPIKey = {
      id: crypto.randomUUID(),
      tenant_id: tenantId,
      prefix,
      name,
      status: 'active',
      created_at: new Date().toISOString(),
    }
    if (!MOCK_KEYS[tenantId]) MOCK_KEYS[tenantId] = []
    MOCK_KEYS[tenantId].push(key)
    return { key: prefix + 'mock_' + Math.random().toString(36).slice(2, 18), api_key: key }
  }
  return adminRequest<CreateKeyResponse>(`/admin/v1/tenants/${tenantId}/keys`, {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

/** 获取用量统计（目前仅 Mock） */
async function getUsage(tenantId: string): Promise<{ overview: UsageOverview; buckets: UsageBucket[] }> {
  await delay()
  const buckets = generateMockUsage()
  const overview: UsageOverview = {
    total_diagnoses: buckets.reduce((s, b) => s + b.diagnoses, 0),
    total_replays: buckets.reduce((s, b) => s + b.replays, 0),
    total_api_calls: buckets.reduce((s, b) => s + b.api_calls, 0),
    period: '30d',
  }
  // tenantId 仅用于未来 API 对接
  void tenantId
  return { overview, buckets }
}

/** 初始化：检测后端可用性 */
async function init() {
  const available = await detectBackend()
  useMock.value = !available
}

export function useAdminApi() {
  return {
    useMock,
    init,
    verifyAdminKey,
    listTenants,
    getTenant,
    createTenant,
    listKeys,
    createKey,
    getUsage,
  }
}
