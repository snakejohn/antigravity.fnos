<template>
  <div class="w-full flex-1 flex flex-col gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">应用设置</h1>
      <div class="flex items-center gap-2">
        <n-button size="small" :disabled="saving" @click="loadConfig">重置表单</n-button>
        <n-button type="primary" size="small" :loading="saving" @click="saveSettings">保存设置</n-button>
      </div>
    </div>

    <div class="flex flex-col gap-5">
      <!-- 1. Harness 模型与安全策略 -->
      <n-card title="Antigravity 模型与沙箱策略" :bordered="false" class="shadow-sm rounded-2xl">
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">默认模型档位</div>
            <n-select v-model:value="form.default_model" :options="modelOptions" />
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">终端沙箱策略 (Sandbox)</div>
            <n-select v-model:value="form.sandbox_mode" :options="sandboxOptions" />
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">工具审批策略 (Approval)</div>
            <n-select v-model:value="form.approval_policy" :options="approvalOptions" />
          </div>
        </div>
        <div class="text-xs text-slate-400 mt-3 font-mono">
          配置会自动同步写入：{{ form.gemini_home || '~/.gemini' }}/antigravity-cli/settings.json
        </div>
      </n-card>

      <!-- 2. 端口与接入方式 -->
      <n-card title="核心服务与网络端口" :bordered="false" class="shadow-sm rounded-2xl">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">内部 Harness / RPC 监听端口</div>
            <n-input-number v-model:value="form.server_port" :min="1" :max="65535" class="w-full" />
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">管理控制台对外 HTTP 端口</div>
            <n-input-number v-model:value="form.proxy_port" :min="1" :max="65535" class="w-full" />
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">访问控制密码 (可选)</div>
            <n-input v-model:value="form.access_password" type="password" show-password-on="click" placeholder="留空不启用密码保护" />
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">更新通道</div>
            <n-select v-model:value="form.update_channel" :options="channelOptions" />
          </div>
        </div>
      </n-card>

      <!-- 3. 网络代理与镜像源 -->
      <n-card title="网络代理与依赖镜像" :bordered="false" class="shadow-sm rounded-2xl">
        <div class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">NPM / MCP 镜像源</div>
              <n-input v-model:value="form.npm_registry" placeholder="https://registry.npmmirror.com" class="font-mono" />
            </div>
            <div>
              <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">PyPI SDK 镜像源</div>
              <n-input v-model:value="form.pypi_index" placeholder="https://pypi.tuna.tsinghua.edu.cn/simple" class="font-mono" />
            </div>
          </div>
          <div>
            <div class="text-xs font-medium text-slate-600 dark:text-slate-300 mb-1.5">HTTP / SOCKS5 代理地址</div>
            <n-input v-model:value="form.network_proxy" placeholder="例如 http://192.168.1.100:7890" clearable class="font-mono" />
          </div>
          <div class="flex items-center justify-between pt-2 border-t border-slate-100 dark:border-white/[0.06]">
            <div>
              <div class="text-sm font-medium text-slate-800 dark:text-slate-100">将代理应用于 Agent 运行时</div>
              <div class="text-xs text-slate-400">自动向 Gemini API 请求与 MCP 进程注入 HTTP_PROXY / HTTPS_PROXY</div>
            </div>
            <n-switch v-model:value="form.proxy_runtime_enabled" :disabled="!form.network_proxy?.trim()" />
          </div>
        </div>
      </n-card>

      <!-- 4. 重置修复 -->
      <n-card title="环境重置与自愈修复" :bordered="false" class="shadow-sm rounded-2xl border border-rose-200/70 dark:border-rose-950/50">
        <div class="flex items-center justify-between gap-4">
          <div class="space-y-1">
            <div class="text-sm font-semibold text-slate-800 dark:text-slate-100">保留会话与凭据并重置修复运行时</div>
            <div class="text-xs text-slate-400">重新校验目录权限、同步 CLI 设置并执行健康巡检，不会删除历史会话和 API 密钥。</div>
          </div>
          <n-popconfirm @positive-click="runRepair">
            <template #trigger>
              <n-button type="error" secondary size="small">执行修复</n-button>
            </template>
            确认执行 Antigravity Harness 重置修复？
          </n-popconfirm>
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NButton, NSelect, NInput, NInputNumber, NSwitch, NPopconfirm, useMessage } from 'naive-ui'
import { configApi } from '../api'
import { useSystemStore } from '../stores/system'
import type { AppConfig } from '../types/api'

const message = useMessage()
const systemStore = useSystemStore()
const saving = ref(false)

const form = ref<AppConfig>({
  server_port: 2398,
  proxy_port: 2399,
  default_model: 'gemini-2.5-pro',
  sandbox_mode: 'workspace-write',
  approval_policy: 'on-request',
  network_proxy: '',
  proxy_runtime_enabled: false,
  npm_registry: 'https://registry.npmmirror.com',
  pypi_index: 'https://pypi.tuna.tsinghua.edu.cn/simple',
  update_channel: 'latest',
  access_mode: 'fngateway',
  access_password: ''
})

const modelOptions = [
  { label: 'Gemini 2.5 Pro (最强推理)', value: 'gemini-2.5-pro' },
  { label: 'Gemini 2.5 Flash (均衡高速)', value: 'gemini-2.5-flash' },
  { label: 'Gemini 2.5 Flash-Lite (轻量低延迟)', value: 'gemini-2.5-flash-lite' }
]

const sandboxOptions = [
  { label: 'workspace-write (推荐：仅工作区可写)', value: 'workspace-write' },
  { label: 'read-only (只读安全模式)', value: 'read-only' },
  { label: 'full-access (宿主机完全访问)', value: 'full-access' }
]

const approvalOptions = [
  { label: 'on-request (按需审批)', value: 'on-request' },
  { label: 'untrusted (高危操作严格审批)', value: 'untrusted' },
  { label: 'never (自动放行)', value: 'never' }
]

const channelOptions = [
  { label: 'latest (稳定版)', value: 'latest' },
  { label: 'beta (测试版)', value: 'beta' },
  { label: 'canary (抢先体验)', value: 'canary' }
]

async function loadConfig() {
  const res = await configApi.getConfig()
  if (res.success && res.data) {
    form.value = { ...form.value, ...res.data }
  }
}

async function saveSettings() {
  saving.value = true
  const res = await configApi.saveConfig(form.value)
  saving.value = false
  if (res.success) {
    message.success(res.message || '设置已保存')
    await systemStore.fetchInitialStatus()
  } else {
    message.error(res.message || '保存失败')
  }
}

async function runRepair() {
  const res = await systemStore.sendAction('repair', { keep_plugins: true })
  if (res.success) message.success(res.message)
  else message.error(res.message)
}

onMounted(() => {
  void loadConfig()
})
</script>
