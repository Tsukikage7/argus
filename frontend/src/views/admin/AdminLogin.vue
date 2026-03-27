<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAdminApi } from '@/composables/useAdminApi'

const router = useRouter()
const route = useRoute()
const { verifyAdminKey, init } = useAdminApi()

init()

const key = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!key.value.trim()) {
    error.value = '请输入 Admin Key'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const valid = await verifyAdminKey(key.value.trim())
    if (valid) {
      localStorage.setItem('argus-admin-key', key.value.trim())
      const redirect = (route.query.redirect as string) || '/admin/tenants'
      router.push(redirect)
    } else {
      error.value = 'Admin Key 无效，请检查后重试'
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '验证失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center">
    <div class="glass-card rounded-2xl p-8 w-full max-w-sm">
      <!-- Logo -->
      <div class="flex flex-col items-center mb-8">
        <div
          class="w-14 h-14 rounded-xl flex items-center justify-center text-white font-bold text-2xl mb-3"
          style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
        >
          A
        </div>
        <h1 class="text-lg font-bold text-base-content">Argus Admin</h1>
        <p class="text-xs text-base-content/40 mt-1">管理控制台登录</p>
      </div>

      <!-- 表单 -->
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <label class="block text-xs text-base-content/50 mb-1.5">Admin Key</label>
          <input
            v-model="key"
            type="password"
            placeholder="输入管理员密钥..."
            class="input input-bordered w-full bg-base-200/50 text-sm"
            :class="{ 'input-error': error }"
            autocomplete="off"
          />
        </div>

        <!-- 错误提示 -->
        <div v-if="error" class="text-xs text-error px-1">
          {{ error }}
        </div>

        <button
          type="submit"
          class="btn w-full text-sm text-white border-0"
          style="background: linear-gradient(135deg, #6366f1, #a78bfa)"
          :disabled="loading"
        >
          <span v-if="loading" class="loading loading-spinner loading-xs"></span>
          {{ loading ? '验证中...' : '登录' }}
        </button>
      </form>

      <!-- 返回 -->
      <div class="mt-6 text-center">
        <router-link to="/" class="text-xs text-base-content/30 hover:text-base-content/50 transition-colors">
          &larr; 返回诊断面板
        </router-link>
      </div>
    </div>
  </div>
</template>
