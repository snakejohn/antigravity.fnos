<template>
  <div class="w-full flex-1 flex flex-col gap-4 sm:gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3">
      <div class="flex items-baseline gap-2.5">
        <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">会话与任务</h1>
        <span class="text-xs text-slate-400">ConversationService &amp; Transcript 轨迹监控</span>
      </div>
      <div class="flex items-center gap-2">
        <n-button size="small" secondary @click="loadList">
          <template #icon><n-icon><Refresh /></n-icon></template>
          刷新
        </n-button>
        <n-button type="primary" size="small" @click="showCreateModal = true">
          <template #icon><n-icon><Plus /></n-icon></template>
          发起新任务
        </n-button>
      </div>
    </div>

    <!-- 左右分栏布局 -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 min-h-[540px]">
      <!-- 左侧会话列表 -->
      <div class="lg:col-span-4 flex flex-col gap-2.5">
        <n-card v-for="c in list" :key="c.id" hoverable :bordered="false" @click="selectConv(c.id)"
          class="cursor-pointer rounded-2xl shadow-sm transition-all"
          :class="activeId === c.id ? 'ring-2 ring-fnos-blue/70' : ''">
          <div class="flex flex-col gap-1.5">
            <div class="flex items-center justify-between gap-2">
              <span class="font-semibold text-sm text-slate-800 dark:text-slate-100 truncate">
                {{ c.annotations.title || c.summary || c.id }}
              </span>
              <n-tag size="tiny" round :bordered="false" :type="c.runState === 'RUN_STATE_RUNNING' ? 'warning' : 'success'">
                {{ c.runState === 'RUN_STATE_RUNNING' ? '运行中' : '空闲' }}
              </n-tag>
            </div>
            <div class="flex items-center justify-between text-[11px] text-slate-400 font-mono">
              <span>{{ c.model }}</span>
              <span>{{ c.turnCount }} 轮</span>
            </div>
            <div class="flex items-center justify-between pt-1 border-t border-slate-100 dark:border-white/[0.06]">
              <div class="flex gap-1 flex-wrap">
                <n-tag v-if="c.annotations.pinned" size="tiny" type="info" :bordered="false">置顶</n-tag>
                <n-tag v-if="c.annotations.starred" size="tiny" type="warning" :bordered="false">星标</n-tag>
                <n-tag v-for="t in (c.annotations.tags || []).slice(0, 2)" :key="t" size="tiny" :bordered="false">{{ t }}</n-tag>
              </div>
              <div class="flex items-center gap-1" @click.stop>
                <n-button size="tiny" quaternary @click="toggleStar(c)">
                  <n-icon :class="c.annotations.starred ? 'text-amber-500' : 'text-slate-400'"><Star /></n-icon>
                </n-button>
                <n-button size="tiny" quaternary @click="togglePin(c)">
                  <n-icon :class="c.annotations.pinned ? 'text-fnos-blue' : 'text-slate-400'"><Pin /></n-icon>
                </n-button>
                <n-popconfirm @positive-click="removeConv(c.id)">
                  <template #trigger>
                    <n-button size="tiny" quaternary type="error"><n-icon><Trash /></n-icon></n-button>
                  </template>
                  永久删除此会话及其 transcript？
                </n-popconfirm>
              </div>
            </div>
          </div>
        </n-card>
        <n-empty v-if="!list.length" description="暂无会话，点击右上角发起新任务" class="py-12" />
      </div>

      <!-- 右侧 Transcript 轨迹与交互区 -->
      <n-card :bordered="false" class="lg:col-span-8 rounded-2xl shadow-sm flex flex-col"
        content-style="display: flex; flex-direction: column; height: 100%; gap: 14px;">
        <template v-if="detail">
          <div class="flex items-center justify-between gap-2 pb-2 border-b border-slate-100 dark:border-white/[0.08]">
            <div class="min-w-0">
              <div class="font-bold text-base text-slate-800 dark:text-slate-100 truncate">
                {{ detail.annotations.title || detail.id }}
              </div>
              <div class="text-xs font-mono text-slate-400 truncate">
                ID: {{ detail.id }} · 模型: {{ detail.model }} · 工作区: {{ detail.workspace }}
              </div>
            </div>
            <n-tag :type="detail.runState === 'RUN_STATE_RUNNING' ? 'warning' : 'success'" size="small" round :bordered="false">
              <template #icon v-if="detail.runState === 'RUN_STATE_RUNNING'"><n-spin :size="10" /></template>
              {{ detail.runState }}
            </n-tag>
          </div>

          <!-- Transcript 时间线 -->
          <div class="flex-1 overflow-y-auto space-y-3 max-h-[52vh] pr-1">
            <div v-for="entry in detail.transcript" :key="entry.id" class="rounded-xl p-3 text-xs leading-relaxed"
              :class="roleCardClass(entry.role)">
              <div class="flex items-center justify-between gap-2 mb-1">
                <span class="font-semibold uppercase tracking-wide text-[10px]">
                  {{ roleLabel(entry) }}
                </span>
                <span class="font-mono text-[10px] opacity-60">{{ formatTs(entry.timestamp) }}</span>
              </div>
              <div v-if="entry.tool_args" class="font-mono text-[11px] bg-black/5 dark:bg-black/30 rounded-lg px-2 py-1 mb-1.5 break-all">
                {{ entry.tool_name }}({{ entry.tool_args }})
              </div>
              <div class="whitespace-pre-wrap break-words select-text">{{ entry.content }}</div>
            </div>
          </div>

          <!-- 追加指令输入框 -->
          <div class="flex gap-2 pt-2 border-t border-slate-100 dark:border-white/[0.08]">
            <n-input v-model:value="followUpPrompt" placeholder="向当前 Agent 会话发送后续指令 (Enter 发送)..."
              :disabled="sending" @keydown.enter="sendFollowUp" />
            <n-button type="primary" :loading="sending" :disabled="!followUpPrompt.trim()" @click="sendFollowUp">
              <template #icon><n-icon><Send /></n-icon></template>
              发送
            </n-button>
          </div>
        </template>
        <div v-else class="flex-1 flex items-center justify-center py-20">
          <n-empty description="从左侧选择一个会话查看 Transcript 执行轨迹" />
        </div>
      </n-card>
    </div>

    <!-- 新建任务弹窗 -->
    <n-modal v-model:show="showCreateModal" preset="card" title="发起新 Antigravity Agent 任务" class="w-[92vw] max-w-lg rounded-2xl">
      <div class="space-y-3">
        <div>
          <div class="text-xs text-slate-500 mb-1">任务标题 (可选)</div>
          <n-input v-model:value="createForm.title" placeholder="例如：fnOS 磁盘与容器健康诊断" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div class="text-xs text-slate-500 mb-1">模型档位</div>
            <n-select v-model:value="createForm.model" :options="modelOptions" />
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">绑定工作区</div>
            <n-select v-model:value="createForm.workspace" :options="workspaceOptions" filterable tag />
          </div>
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">初始 Prompt 指令</div>
          <n-input v-model:value="createForm.prompt" type="textarea" :rows="4"
            placeholder="输入你希望 Antigravity Agent 执行的任务，例如：检查当前工作区并汇总可用定制项..." />
        </div>
        <div class="flex justify-end gap-2 pt-1">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" :loading="creating" :disabled="!createForm.prompt.trim()" @click="submitCreate">
            创建并运行
          </n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import {
  NCard, NButton, NTag, NIcon, NInput, NSelect, NModal, NEmpty, NSpin, NPopconfirm, useMessage
} from 'naive-ui'
import { Refresh, Plus, Star, Pin, Trash, Send } from '@vicons/tabler'
import { conversationsApi } from '../api'
import { useAppStore } from '../stores/app'
import type { ConversationMeta, ConversationDetail, TranscriptEntry } from '../types/api'

const appStore = useAppStore()
const message = useMessage()

const list = computed(() => appStore.conversations)
const activeId = ref('')
const detail = ref<ConversationDetail | null>(null)
const followUpPrompt = ref('')
const sending = ref(false)
const showCreateModal = ref(false)
const creating = ref(false)

const createForm = ref({
  title: '',
  model: 'gemini-2.5-pro',
  workspace: '',
  prompt: ''
})

const modelOptions = [
  { label: 'Gemini 2.5 Pro (pro)', value: 'gemini-2.5-pro' },
  { label: 'Gemini 2.5 Flash (flash)', value: 'gemini-2.5-flash' },
  { label: 'Gemini 2.5 Flash-Lite (flash_lite)', value: 'gemini-2.5-flash-lite' }
]

const workspaceOptions = computed(() =>
  appStore.workspaces.map((w) => ({ label: `${w.title} (${w.path})`, value: w.path }))
)

async function loadList() {
  await appStore.refreshConversations()
  if (!activeId.value && list.value.length) {
    await selectConv(list.value[0].id)
  } else if (activeId.value) {
    await selectConv(activeId.value)
  }
}

async function selectConv(id: string) {
  activeId.value = id
  const res = await conversationsApi.getDetail(id)
  if (res.success && res.data) {
    detail.value = res.data
  }
}

// Refresh active detail whenever WS broadcasts conversation updates
watch(
  () => appStore.conversations,
  () => {
    if (activeId.value) {
      void selectConv(activeId.value)
    }
  }
)

function roleLabel(entry: TranscriptEntry) {
  switch (entry.role) {
    case 'user': return 'User Prompt'
    case 'thought': return 'Agent Thinking'
    case 'tool_call': return `Tool Call · ${entry.tool_name || ''}`
    case 'tool_result': return `Tool Result · ${entry.tool_name || ''}`
    case 'assistant': return `Antigravity (${entry.model || 'agent'})`
    default: return entry.role
  }
}

function roleCardClass(role: string) {
  switch (role) {
    case 'user':
      return 'bg-blue-50 dark:bg-blue-950/35 border border-blue-200/70 dark:border-blue-800/60 text-slate-800 dark:text-slate-100'
    case 'thought':
      return 'bg-amber-50/70 dark:bg-amber-950/20 border border-amber-200/60 dark:border-amber-900/40 text-amber-900 dark:text-amber-200'
    case 'tool_call':
    case 'tool_result':
      return 'bg-slate-100/80 dark:bg-white/[0.04] border border-slate-200/70 dark:border-white/[0.08] text-slate-600 dark:text-slate-300'
    default:
      return 'bg-emerald-50/60 dark:bg-emerald-950/25 border border-emerald-200/70 dark:border-emerald-900/40 text-slate-800 dark:text-slate-100'
  }
}

function formatTs(ts: string) {
  if (!ts) return ''
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleTimeString()
}

async function submitCreate() {
  creating.value = true
  const res = await conversationsApi.create({
    title: createForm.value.title,
    model: createForm.value.model,
    workspace: createForm.value.workspace || appStore.workspaces[0]?.path,
    prompt: createForm.value.prompt,
    tags: ['console']
  })
  creating.value = false
  if (res.success && res.data) {
    message.success('任务已创建并派发执行')
    showCreateModal.value = false
    createForm.value = { title: '', model: 'gemini-2.5-pro', workspace: '', prompt: '' }
    await appStore.refreshConversations()
    await selectConv(res.data.id)
  } else {
    message.error(res.message)
  }
}

async function sendFollowUp() {
  if (!detail.value || !followUpPrompt.value.trim()) return
  sending.value = true
  const text = followUpPrompt.value.trim()
  followUpPrompt.value = ''
  const res = await conversationsApi.sendMessage(detail.value.id, text)
  sending.value = false
  if (res.success) {
    await selectConv(detail.value.id)
  } else {
    message.error(res.message)
  }
}

async function toggleStar(c: ConversationMeta) {
  await conversationsApi.update(c.id, { starred: !c.annotations.starred })
  await appStore.refreshConversations()
}

async function togglePin(c: ConversationMeta) {
  await conversationsApi.update(c.id, { pinned: !c.annotations.pinned })
  await appStore.refreshConversations()
}

async function removeConv(id: string) {
  const res = await conversationsApi.remove(id)
  if (res.success) {
    message.success('会话已删除')
    if (activeId.value === id) {
      activeId.value = ''
      detail.value = null
    }
    await loadList()
  }
}

onMounted(async () => {
  await Promise.all([appStore.refreshWorkspaces(), loadList()])
})
</script>
