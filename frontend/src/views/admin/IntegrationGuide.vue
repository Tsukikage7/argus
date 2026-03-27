<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { createHighlighter, type Highlighter } from 'shiki'

const highlighter = ref<Highlighter | null>(null)
const copiedIndex = ref<number | null>(null)

// 代码示例
const examples = [
  {
    title: 'Widget 嵌入',
    description: '在任何网页中嵌入 Argus 诊断 Widget，只需一行 script 标签',
    lang: 'html',
    code: `<!-- Argus Widget 嵌入 -->
<script
  src="https://your-argus-domain.com/widget.umd.js"
  data-api-key="arg_your-tenant_xxxxxxxxxxxxxxxx"
  data-base-url="https://your-argus-domain.com"
><\/script>`,
  },
  {
    title: 'cURL 调用',
    description: '通过 cURL 触发诊断并查询结果',
    lang: 'bash',
    code: `# 触发诊断
curl -X POST https://your-argus-domain.com/api/v1/diagnose \\
  -H "Authorization: Bearer arg_your-tenant_xxxxxxxxxxxxxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{"input": "prj-ubill 连接池耗尽", "source": "api"}'

# 查询结果（替换 task_id）
curl https://your-argus-domain.com/api/v1/tasks/{task_id} \\
  -H "Authorization: Bearer arg_your-tenant_xxxxxxxxxxxxxxxx"`,
  },
  {
    title: 'JavaScript / Fetch',
    description: '在前端或 Node.js 中通过 Fetch API 调用',
    lang: 'javascript',
    code: `const API_KEY = 'arg_your-tenant_xxxxxxxxxxxxxxxx'
const BASE_URL = 'https://your-argus-domain.com'

// 触发诊断
const { task_id } = await fetch(\`\${BASE_URL}/api/v1/diagnose\`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': \`Bearer \${API_KEY}\`,
  },
  body: JSON.stringify({
    input: 'prj-ubill 连接池耗尽',
    source: 'api',
  }),
}).then(r => r.json())

// SSE 实时推送
const sse = new EventSource(
  \`\${BASE_URL}/api/v1/stream/\${task_id}\`
)
sse.addEventListener('step', (e) => {
  console.log('推理步骤:', JSON.parse(e.data))
})
sse.addEventListener('diagnosis', (e) => {
  console.log('诊断结论:', JSON.parse(e.data))
  sse.close()
})`,
  },
  {
    title: 'Python',
    description: '使用 Python requests 库调用 Argus API',
    lang: 'python',
    code: `import requests
import json

API_KEY = "arg_your-tenant_xxxxxxxxxxxxxxxx"
BASE_URL = "https://your-argus-domain.com"

headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {API_KEY}",
}

# 触发诊断
resp = requests.post(
    f"{BASE_URL}/api/v1/diagnose",
    headers=headers,
    json={"input": "prj-ubill 连接池耗尽", "source": "api"},
)
task_id = resp.json()["task_id"]
print(f"Task ID: {task_id}")

# 轮询结果
import time
while True:
    result = requests.get(
        f"{BASE_URL}/api/v1/tasks/{task_id}",
        headers=headers,
    ).json()
    if result["status"] in ("completed", "failed"):
        print(json.dumps(result["diagnosis"], indent=2, ensure_ascii=False))
        break
    time.sleep(3)`,
  },
]

// 渲染高亮 HTML
const highlightedCode = ref<string[]>([])

onMounted(async () => {
  try {
    highlighter.value = await createHighlighter({
      themes: ['github-dark'],
      langs: ['html', 'bash', 'javascript', 'python'],
    })
    highlightedCode.value = examples.map(ex =>
      highlighter.value!.codeToHtml(ex.code, {
        lang: ex.lang,
        theme: 'github-dark',
      })
    )
  } catch {
    // Shiki 加载失败时使用原始代码
    highlightedCode.value = examples.map(ex => `<pre><code>${escapeHtml(ex.code)}</code></pre>`)
  }
})

function escapeHtml(s: string) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

async function copyCode(index: number) {
  await navigator.clipboard.writeText(examples[index].code)
  copiedIndex.value = index
  setTimeout(() => { copiedIndex.value = null }, 2000)
}

// 步骤引导
const steps = [
  { num: 1, title: '获取 API Key', desc: '在租户管理页面创建 API Key，密钥仅显示一次请妥善保存' },
  { num: 2, title: '选择集成方式', desc: '根据业务场景选择 Widget 嵌入、REST API 或 SSE 实时推送' },
  { num: 3, title: '配置 CORS', desc: '如使用浏览器端调用，需在租户 Allowed Origins 中添加前端域名' },
  { num: 4, title: '开始诊断', desc: '调用 /api/v1/diagnose 发起诊断，通过 SSE 或轮询获取结果' },
]
</script>

<template>
  <div>
    <h1 class="text-xl font-bold text-base-content mb-1">集成指南</h1>
    <p class="text-xs text-base-content/40 mb-6">快速接入 Argus 智能诊断能力</p>

    <!-- 步骤引导 -->
    <div class="grid grid-cols-4 gap-4 mb-8">
      <div
        v-for="step in steps"
        :key="step.num"
        class="glass-card rounded-xl p-4 hover:translate-y-[-2px] transition-transform"
      >
        <div
          class="w-7 h-7 rounded-full flex items-center justify-center text-white text-xs font-bold mb-2.5"
          style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
        >
          {{ step.num }}
        </div>
        <h3 class="text-sm font-medium text-base-content mb-1">{{ step.title }}</h3>
        <p class="text-[11px] text-base-content/40 leading-relaxed">{{ step.desc }}</p>
      </div>
    </div>

    <!-- 代码示例 -->
    <div class="space-y-5">
      <div
        v-for="(ex, i) in examples"
        :key="i"
        class="glass-card rounded-xl overflow-hidden"
      >
        <div class="flex items-center justify-between px-4 py-3 border-b" style="border-color: oklch(var(--b3) / 0.5)">
          <div>
            <h3 class="text-sm font-medium text-base-content">{{ ex.title }}</h3>
            <p class="text-[11px] text-base-content/40 mt-0.5">{{ ex.description }}</p>
          </div>
          <button
            class="btn btn-ghost btn-xs text-base-content/40 hover:text-base-content/70"
            @click="copyCode(i)"
          >
            <template v-if="copiedIndex === i">
              <svg class="w-3.5 h-3.5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              已复制
            </template>
            <template v-else>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9.75a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
              </svg>
              复制
            </template>
          </button>
        </div>
        <div class="p-4 overflow-x-auto text-xs [&_pre]:!bg-transparent [&_pre]:!m-0 [&_pre]:!p-0 [&_code]:!text-xs">
          <div v-if="highlightedCode[i]" v-html="highlightedCode[i]" />
          <pre v-else class="text-base-content/60"><code>{{ ex.code }}</code></pre>
        </div>
      </div>
    </div>
  </div>
</template>
