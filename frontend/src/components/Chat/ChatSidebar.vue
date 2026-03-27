<script setup lang="ts">
import { ref } from 'vue'
import { useChatStore } from '@/store/useChatStore'

const store = useChatStore()

// 重命名编辑状态
const editingId = ref<string | null>(null)
const editTitle = ref('')

function startRename(id: string, title: string) {
  editingId.value = id
  editTitle.value = title
}

async function confirmRename(id: string) {
  if (editTitle.value.trim()) {
    await store.renameSession(id, editTitle.value.trim())
  }
  editingId.value = null
}

function cancelRename() {
  editingId.value = null
}

// 格式化时间
function formatTime(iso: string): string {
  const d = new Date(iso)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- 新建会话按钮 -->
    <div class="p-3 border-b border-base-300/50">
      <button
        class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm
               border border-dashed border-base-content/20 text-base-content/60
               hover:border-indigo-500/50 hover:text-indigo-400 transition-all cursor-pointer"
        @click="store.currentSessionId = null"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        新建会话
      </button>
    </div>

    <!-- 会话列表 -->
    <div class="flex-1 overflow-y-auto scroller py-2">
      <div v-if="store.sessions.length === 0" class="text-center text-xs text-base-content/30 py-8">
        暂无会话
      </div>
      <div
        v-for="session in store.sessions"
        :key="session.id"
        class="group mx-2 mb-1 px-3 py-2.5 rounded-lg cursor-pointer transition-all"
        :class="store.currentSessionId === session.id
          ? 'bg-indigo-500/15 text-indigo-400'
          : 'text-base-content/60 hover:bg-base-300/30 hover:text-base-content/80'"
        @click="store.switchSession(session.id)"
      >
        <!-- 正常显示 -->
        <template v-if="editingId !== session.id">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium truncate flex-1">
              {{ session.title || '新会话' }}
            </div>
            <!-- 操作按钮 -->
            <div class="hidden group-hover:flex items-center gap-1 ml-2 shrink-0">
              <button
                class="p-1 rounded hover:bg-base-300/50 text-base-content/40 hover:text-base-content/70"
                title="重命名"
                @click.stop="startRename(session.id, session.title)"
              >
                <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
              <button
                class="p-1 rounded hover:bg-red-500/20 text-base-content/40 hover:text-red-400"
                title="删除"
                @click.stop="store.removeSession(session.id)"
              >
                <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
          <div class="flex items-center gap-2 mt-1">
            <span v-if="session.last_intent" class="text-[0.6rem] px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400/70">
              {{ session.last_intent }}
            </span>
            <span class="text-[0.6rem] text-base-content/30">{{ formatTime(session.updated_at) }}</span>
          </div>
        </template>

        <!-- 重命名编辑 -->
        <template v-else>
          <input
            v-model="editTitle"
            class="w-full px-2 py-1 rounded text-sm bg-base-200 border border-indigo-500/50
                   text-base-content focus:outline-none"
            @keyup.enter="confirmRename(session.id)"
            @keyup.escape="cancelRename"
            @click.stop
          />
        </template>
      </div>
    </div>
  </div>
</template>
