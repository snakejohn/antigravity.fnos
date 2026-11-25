<template>
  <div class="w-full flex-1 flex flex-col gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3">
      <div class="flex items-baseline gap-2.5">
        <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">工作区与终端</h1>
        <span class="text-xs text-slate-400">共 {{ workspaces.length }} 个绑定工作区</span>
      </div>
      <n-button type="primary" size="small" @click="showAddModal = true">
        <template #icon><n-icon><Plus /></n-icon></template>
        绑定 NAS 目录
      </n-button>
    </div>

    <!-- 工作区卡片网格 -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
      <n-card v-for="w in workspaces" :key="w.workspaceId" hoverable :bordered="false" class="rounded-2xl shadow-sm interactive-card">
        <div class="flex items-start justify-between gap-3">
          <div class="flex items-start gap-3 min-w-0 flex-1">
            <div class="w-10 h-10 rounded-xl bg-blue-50 dark:bg-blue-950/40 text-fnos-blue dark:text-blue-400 flex items-center justify-center shrink-0">
              <n-icon :size="22"><Folder /></n-icon>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-sm text-slate-800 dark:text-slate-100 truncate">{{ w.title }}</span>
                <n-tag size="tiny" round :bordered="false" type="info">{{ w.session_count }} 会话</n-tag>
                <n-tag v-if="w.git_branch" size="tiny" round :bordered="false" type="success">{{ w.git_branch }}</n-tag>
              </div>
              <div class="text-xs font-mono text-slate-400 truncate">{{ w.path }}</div>
              <div class="flex items-center gap-2 text-[11px] text-slate-500 pt-0.5">
                <span>.agents: {{ w.has_agents_dir ? `✓ (${w.skills_count} skills)` : '未初始化' }}</span>
                <span>·</span>
                <span>规则文件: {{ w.has_gemini_rule ? '✓' : '无' }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <n-button size="tiny" secondary type="primary" @click="selectForTerminal(w.path)">在此运行终端</n-button>
            <n-popconfirm @positive-click="removeWorkspace(w.workspaceId)">
              <template #trigger>
                <n-button size="tiny" quaternary type="error"><n-icon><Trash /></n-icon></n-button>
              </template>
              移除工作区绑定？(不会删除磁盘文件)
            </n-popconfirm>
          </div>
        </div>
      </n-card>
    </div>

    <!-- Web 命令运行器 / 终端控制台 -->
    <n-card :bordered="false" class="rounded-2xl shadow-sm">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <n-icon :size="18" class="text-fnos-blue"><Terminal2 /></n-icon>
            <span class="font-bold text-base text-slate-800 dark:text-slate-100">命令测试终端 (Command Runner)</span>
          </div>
          <span class="text-xs font-mono text-slate-400">CWD: {{ terminalCwd || '默认数据区' }}</span>
        </div>
      </template>

      <div class="space-y-3">
        <!-- 快捷命令胶囊 -->
        <div class="flex items-center flex-wrap gap-1.5">
          <span class="text-xs text-slate-400 mr-1">快捷命令:</span>
          <n-button v-for="p in quickCommands" :key="p.cmd" size="tiny" secondary @click="runQuick(p.cmd)">
            {{ p.label }}
          </n-button>
        </div>

        <!-- 输入栏 -->
        <div class="flex gap-2">
          <n-input v-model:value="commandInput" placeholder="输入 Shell 或 agy 命令，如 agy doctor / df -h / ls -la ..."
            class="font-mono text-xs" :disabled="running" @keydown.enter="executeCommand" />
          <n-button type="primary" :loading="running" :disabled="!commandInput.trim()" @click="executeCommand">
            执行
          </n-button>
        </div>

        <!-- 终端输出窗口 -->
        <div class="rounded-xl bg-[#0f172a] text-slate-200 p-4 font-mono text-xs min-h-[220px] max-h-[420px] overflow-y-auto select-text">
          <div v-if="!history.length" class="text-slate-500">
            $ Antigravity Harness Command Runner ready. 点击上方快捷命令或输入指令开始执行...
          </div>
          <div v-for="(item, idx) in history" :key="idx" class="mb-4 last:mb-0">
            <div class="flex items-center justify-between text-emerald-400 mb-1">
              <span>$ {{ item.command }}</span>
              <span class="text-[10px] text-slate-400">
                exit {{ item.exit_code }} · {{ item.duration_ms }}ms
              </span>
            </div>
            <pre v-if="item.stdout" class="whitespace-pre-wrap break-words text-slate-100 m-0">{{ item.stdout }}</pre>
            <pre v-if="item.stderr" class="whitespace-pre-wrap break-words text-rose-400 m-0 mt-1">{{ item.stderr }}</pre>
          </div>
        </div>
      </div>
    </n-card>

    <!-- 添加工作区弹窗 -->
    <n-modal v-model:show="showAddModal" preset="card" title="绑定 fnOS 工作区目录" class="w-[92vw] max-w-md rounded-2xl">
      <div class="space-y-3">
        <div>
          <div class="text-xs text-slate-500 mb-1">工作区名称</div>
          <n-input v-model:value="addForm.title" placeholder="例如：我的 AI 项目库" />
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">NAS 绝对路径</div>
          <n-input v-model:value="addForm.path" placeholder="/vol1/1008/projects" class="font-mono" />
        </div>
        <div class="flex justify-end gap-2 pt-1">
          <n-button @click="showAddModal = false">取消</n-button>
          <n-button type="primary" :loading="adding" @click="submitAdd">确认绑定</n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NCard, NButton, NTag, NIcon, NInput, NModal, NPopconfirm, useMessage } from 'naive-ui'
import { Folder, Plus, Trash, Terminal2 } from '@vicons/tabler'
import { workspaceApi } from '../api'
import { useAppStore } from '../stores/app'
import type { TerminalExecResult } from '../types/api'

const appStore = useAppStore()
const message = useMessage()
const workspaces = computed(() => appStore.workspaces)

const showAddModal = ref(false)
const adding = ref(false)
const addForm = ref({ title: '', path: '' })

const terminalCwd = ref('')
const commandInput = ref('')
const running = ref(false)
const history = ref<TerminalExecResult[]>([])

const quickCommands = [
  { label: 'agy doctor', cmd: 'agy doctor' },
  { label: '系统与内核', cmd: 'uname -a' },
  { label: '存储卷空间', cmd: 'df -h' },
  { label: '内存状态', cmd: 'free -h' },
  { label: 'Node & Python', cmd: 'node -v; python3 --version' },
  { label: '列出定制目录', cmd: 'ls -la "$AGY_CONFIG_DIR"' }
]

function selectForTerminal(path: string) {
  terminalCwd.value = path
  message.info(`终端工作目录已切换至: ${path}`)
}

async function runQuick(cmd: string) {
  commandInput.value = cmd
  await executeCommand()
}

async function executeCommand() {
  if (!commandInput.value.trim() || running.value) return
  running.value = true
  const cmd = commandInput.value.trim()
  const res = await workspaceApi.execTerminal(cmd, terminalCwd.value || undefined)
  running.value = false
  if (res.success && res.data) {
    history.value.unshift(res.data)
    if (history.value.length > 20) history.value.pop()
  } else {
    message.error(res.message || '执行失败')
  }
}

async function submitAdd() {
  adding.value = true
  const res = await workspaceApi.add(addForm.value.title, addForm.value.path)
  adding.value = false
  if (res.success) {
    message.success('工作区已绑定')
    showAddModal.value = false
    addForm.value = { title: '', path: '' }
    await appStore.refreshWorkspaces()
  } else {
    message.error(res.message)
  }
}

async function removeWorkspace(id: string) {
  const res = await workspaceApi.remove(id)
  if (res.success) {
    message.success('已移除')
    await appStore.refreshWorkspaces()
  } else message.error(res.message)
}

onMounted(async () => {
  await appStore.refreshWorkspaces()
  if (!terminalCwd.value && workspaces.value.length) {
    terminalCwd.value = workspaces.value[0].path
  }
})
</script>
