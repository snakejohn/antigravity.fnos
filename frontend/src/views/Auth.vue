<template>
  <div v-auto-animate class="w-full flex-1 flex flex-col gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">授权管理</h1>
      <n-button size="small" secondary :loading="refreshing" @click="refreshAll">
        <template #icon><n-icon><Refresh /></n-icon></template>
        刷新
      </n-button>
    </div>

    <!-- 1. Google Gemini / Antigravity / Vertex AI 授权状态卡片 -->
    <n-card :bordered="false" class="shadow-sm rounded-2xl">
      <div class="flex flex-col gap-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-start sm:items-center gap-3 min-w-0 flex-1">
            <div class="w-11 h-11 rounded-xl bg-blue-50 dark:bg-blue-950/40 text-fnos-blue dark:text-blue-400 flex items-center justify-center shrink-0">
              <n-icon :size="22"><Key /></n-icon>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="text-base font-bold text-slate-800 dark:text-slate-100">Google Gemini / Antigravity 授权</span>
                <n-tag :type="status.authorized ? 'success' : 'warning'" size="small" round :bordered="false">
                  {{ status.authorized ? '已授权' : '待配置' }}
                </n-tag>
              </div>
              <p class="text-xs text-slate-400 truncate">
                <template v-if="status.active_account">
                  当前激活：{{ status.active_account.name }} ({{ providerLabel(status.active_account.provider) }} · {{ status.active_account.masked_key || '已配置' }})
                </template>
                <template v-else-if="status.env_has_gemini">
                  已通过 .env 环境变量检测到 GEMINI_API_KEY
                </template>
                <template v-else>
                  添加 Google AI Studio Gemini API Key、Vertex AI 或兼容端点，驱动 Antigravity 云端推理
                </template>
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <n-button type="primary" size="small" @click="openAddAccountModal">
              <template #icon><n-icon><Plus /></n-icon></template>
              添加 / 配置密钥
            </n-button>
            <n-button v-if="status.active_account" size="small" quaternary type="error" @click="logoutAccount">
              停用当前
            </n-button>
          </div>
        </div>

        <!-- 多账号池列表 -->
        <div v-if="pool.length" class="pt-3 border-t border-slate-100 dark:border-white/[0.06] space-y-2">
          <div class="text-xs font-medium text-slate-500">已保存的凭据账号池 ({{ pool.length }})</div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div v-for="acc in pool" :key="acc.id"
              class="p-3 rounded-xl border flex items-center justify-between gap-2 transition-all"
              :class="acc.is_active
                ? 'bg-blue-50/70 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800'
                : 'bg-slate-50 dark:bg-white/[0.03] border-slate-100 dark:border-white/[0.06]'">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="text-xs font-bold text-slate-800 dark:text-slate-100 truncate">{{ acc.name }}</span>
                  <n-tag v-if="acc.is_active" type="success" size="tiny" round :bordered="false">当前使用</n-tag>
                </div>
                <div class="text-[11px] font-mono text-slate-400 truncate mt-0.5">
                  {{ providerLabel(acc.provider) }} · {{ acc.masked_key || acc.project_id || '-' }}
                </div>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <n-button v-if="!acc.is_active" size="tiny" secondary type="primary" @click="switchAcc(acc.id)">切换</n-button>
                <n-popconfirm @positive-click="deleteAcc(acc.id)">
                  <template #trigger>
                    <n-button size="tiny" quaternary type="error"><n-icon><Trash /></n-icon></n-button>
                  </template>
                  移除此账号凭据？
                </n-popconfirm>
              </div>
            </div>
          </div>
        </div>
      </div>
    </n-card>

    <!-- 2. .env 自定义环境变量表格 -->
    <n-card :bordered="false" class="shadow-sm rounded-2xl">
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <span class="text-base font-bold text-slate-800 dark:text-slate-100">环境变量 (.env)</span>
            <n-tag size="small" round :bordered="false">{{ envRows.length }} 项</n-tag>
          </div>
          <div class="flex items-center gap-2">
            <n-button size="small" secondary @click="addEnvRow">
              <template #icon><n-icon><Plus /></n-icon></template>
              添加
            </n-button>
            <n-button size="small" type="primary" :loading="savingEnv" @click="saveEnv">保存</n-button>
          </div>
        </div>
      </template>

      <div class="space-y-2.5">
        <div v-for="(row, idx) in envRows" :key="idx" class="grid grid-cols-1 sm:grid-cols-[1fr_1.4fr_auto] gap-2 items-center">
          <n-input v-model:value="row.key" size="small" placeholder="变量名，如 GEMINI_API_KEY" class="font-mono" />
          <n-input v-model:value="row.value" size="small" placeholder="变量值"
            :type="isSensitiveKey(row.key) ? 'password' : 'text'" show-password-on="click" class="font-mono" />
          <n-button size="small" quaternary type="error" @click="envRows.splice(idx, 1)">
            <template #icon><n-icon><Trash /></n-icon></template>
          </n-button>
        </div>
        <div v-if="!envRows.length" class="py-4 text-center text-xs text-slate-400">暂无环境变量，点击右上角添加。</div>
        <div class="text-xs text-slate-400 pt-2">
          文件路径：<span class="font-mono">{{ envPath || 'data/.env' }}</span>（包含 KEY / TOKEN / SECRET 的变量自动掩码保护）
        </div>
      </div>
    </n-card>

    <!-- 添加/保存凭据弹窗 -->
    <n-modal v-model:show="showAddModal" preset="card" title="配置模型授权凭据" class="w-[92vw] max-w-lg rounded-2xl">
      <div class="space-y-3">
        <div>
          <div class="text-xs text-slate-500 mb-1">凭据别名</div>
          <n-input v-model:value="accForm.name" placeholder="例如：主账号 Gemini 4 Neon" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div class="text-xs text-slate-500 mb-1">服务提供商</div>
            <n-select v-model:value="accForm.provider" :options="providerOptions" />
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">默认首选模型</div>
            <n-select v-model:value="accForm.default_model" :options="MODEL_OPTIONS" filterable tag />
          </div>
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">API Key</div>
          <n-input v-model:value="accForm.api_key" type="password" show-password-on="click" placeholder="输入 Google Gemini / Vertex AI API Key" class="font-mono" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div class="text-xs text-slate-500 mb-1">自定义 API Base URL (可选)</div>
            <n-input v-model:value="accForm.base_url" placeholder="https://generativelanguage.googleapis.com" class="font-mono" />
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">GCP Project ID (Vertex AI 可选)</div>
            <n-input v-model:value="accForm.project_id" placeholder="my-gcp-project" class="font-mono" />
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <n-button @click="showAddModal = false">取消</n-button>
          <n-button type="primary" :loading="savingAcc" @click="submitAccount">保存并激活</n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NButton, NTag, NIcon, NInput, NSelect, NModal, NPopconfirm, useMessage } from 'naive-ui'
import { Key, Plus, Trash, Refresh } from '@vicons/tabler'
import { authApi } from '../api'
import { DEFAULT_MODEL, MODEL_OPTIONS } from '../constants/models'
import type { AccountStatusResponse, AccountProfile, EnvVarItem } from '../types/api'

const message = useMessage()
const refreshing = ref(false)
const savingAcc = ref(false)
const savingEnv = ref(false)

const status = ref<AccountStatusResponse>({
  authorized: false,
  pool_count: 0,
  cli_detected: false,
  sdk_detected: false,
  ls_connected: false,
  env_has_gemini: false,
  console_url: '',
  docs_url: ''
})
const pool = ref<AccountProfile[]>([])
const envRows = ref<EnvVarItem[]>([])
const envPath = ref('')

const showAddModal = ref(false)
const accForm = ref({
  name: '',
  provider: 'gemini_api',
  api_key: '',
  base_url: '',
  project_id: '',
  default_model: DEFAULT_MODEL
})

const providerOptions = [
  { label: 'Google Gemini API (AI Studio)', value: 'gemini_api' },
  { label: 'Google Cloud Vertex AI', value: 'vertex_ai' },
  { label: 'Antigravity OAuth / Custom Proxy', value: 'antigravity_oauth' }
]

function providerLabel(p: string) {
  return providerOptions.find((x) => x.value === p)?.label || p
}

function isSensitiveKey(k: string) {
  return /KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|AUTH/i.test(k)
}

async function refreshAll() {
  refreshing.value = true
  const [stRes, poolRes, envRes] = await Promise.all([
    authApi.getStatus(),
    authApi.getPool(),
    authApi.getEnvVars()
  ])
  if (stRes.success && stRes.data) status.value = stRes.data
  if (poolRes.success && poolRes.data) pool.value = poolRes.data.items || []
  if (envRes.success && envRes.data) {
    envRows.value = envRes.data.items || []
    envPath.value = envRes.data.path || ''
  }
  refreshing.value = false
}

function openAddAccountModal() {
  accForm.value = { name: '', provider: 'gemini_api', api_key: '', base_url: '', project_id: '', default_model: DEFAULT_MODEL }
  showAddModal.value = true
}

async function submitAccount() {
  savingAcc.value = true
  const res = await authApi.saveAccount(accForm.value)
  savingAcc.value = false
  if (res.success) {
    message.success('账号凭据已保存并激活')
    showAddModal.value = false
    await refreshAll()
  } else message.error(res.message)
}

async function switchAcc(id: string) {
  const res = await authApi.switchAccount(id)
  if (res.success) {
    message.success('已切换激活账号')
    await refreshAll()
  } else message.error(res.message)
}

async function deleteAcc(id: string) {
  const res = await authApi.deleteAccount(id)
  if (res.success) {
    message.success('凭据已移除')
    await refreshAll()
  } else message.error(res.message)
}

async function logoutAccount() {
  const res = await authApi.logout()
  if (res.success) {
    message.success('已停用当前激活账号')
    await refreshAll()
  }
}

function addEnvRow() {
  envRows.value.push({ key: '', value: '' })
}

async function saveEnv() {
  savingEnv.value = true
  const cleaned = envRows.value.filter((r) => r.key.trim() !== '')
  const res = await authApi.saveEnvVars(cleaned)
  savingEnv.value = false
  if (res.success) {
    message.success(res.message || '环境变量已保存')
    await refreshAll()
  } else message.error(res.message)
}

onMounted(() => {
  void refreshAll()
})
</script>
