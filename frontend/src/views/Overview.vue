<template>
  <div v-auto-animate class="w-full flex-1 flex flex-col gap-4 sm:gap-6">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2 sm:pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3 w-full">
      <div class="flex items-center gap-2.5">
        <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">概览</h1>
        <n-tag size="small" round :bordered="false" type="info" class="font-mono text-xs">
          {{ statusData.default_model || 'gemini-4-neon-xhigh' }}
        </n-tag>
      </div>
      <span v-if="statusData.build_time" class="text-xs font-mono text-slate-400 dark:text-slate-500">
        Build: {{ statusData.build_time }}
      </span>
    </div>

    <!-- 核心状态卡片 -->
    <n-card :bordered="false" class="shadow-sm rounded-2xl">
      <div class="flex flex-col py-1 sm:py-2">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div class="space-y-1 min-w-0 flex-1">
            <div class="flex items-center gap-2.5 flex-wrap">
              <span class="text-lg sm:text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">
                {{ statusData.name || 'Antigravity Harness' }}
              </span>
              <n-tag size="small" round :bordered="false" type="success" v-if="isRunning" class="font-mono text-xs">
                RPC :{{ statusData.server_port }}
              </n-tag>
              <n-tag size="small" round :bordered="false" class="font-mono text-xs bg-slate-100 dark:bg-white/[0.08]">
                沙箱: {{ statusData.sandbox_mode || 'workspace-write' }}
              </n-tag>
            </div>
            <div class="text-xs sm:text-sm text-slate-400 dark:text-slate-500 flex items-center gap-3 flex-wrap">
              <span>运行时版本: v{{ statusData.version || '2.0.4' }}</span>
              <span>·</span>
              <span>定制项: {{ customTotal }} 项 (Skills {{ custom?.skills_count || 0 }} / MCP {{ custom?.mcp_count || 0 }} / Plugins {{ custom?.plugins_count || 0 }})</span>
            </div>
          </div>

          <div class="flex items-center gap-2.5 shrink-0">
            <n-button secondary size="medium" @click="appStore.setTab('conversations')"
              class="!h-10 px-4 rounded-xl font-medium">
              <template #icon><n-icon><MessageCircle /></n-icon></template>
              发起 Agent 任务
            </n-button>
            <n-button type="primary" size="medium" @click="systemStore.openGateway"
              class="!h-10 px-5 shadow-sm shadow-fnos-blue/20 font-medium rounded-xl">
              <template #icon><n-icon><ExternalLink /></n-icon></template>
              进入工作窗口
            </n-button>
          </div>
        </div>

        <n-divider class="!my-4 sm:!my-5" />

        <!-- 5 列核心指标网格 -->
        <div class="grid grid-cols-2 sm:grid-cols-5 gap-2.5 sm:gap-3 text-center">
          <div class="py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col items-center gap-1">
            <span class="text-xs text-slate-400">运行状态</span>
            <n-tag :type="statusTagType" size="small" round :bordered="false" class="font-medium mt-0.5">
              <template #icon v-if="isStarting || isBuilding"><n-spin :size="10" /></template>
              {{ statusLabel }}
            </n-tag>
          </div>

          <div class="py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col items-center gap-1">
            <span class="text-xs text-slate-400">运行时间</span>
            <div class="text-sm sm:text-base font-bold text-slate-700 dark:text-slate-200 font-mono mt-0.5">
              {{ uptimeText }}
            </div>
          </div>

          <div class="py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col items-center gap-1">
            <span class="text-xs text-slate-400">进程 PID</span>
            <div class="text-sm sm:text-base font-bold text-slate-700 dark:text-slate-200 font-mono mt-0.5">
              {{ isRunning && statusData.pid ? statusData.pid : '-' }}
            </div>
          </div>

          <div class="py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col items-center gap-1">
            <span class="text-xs text-slate-400">Harness CPU / 内存</span>
            <div class="text-xs sm:text-sm font-bold text-slate-700 dark:text-slate-200 font-mono mt-1">
              {{ statusData.cpu || '-' }} / {{ statusData.memory || '-' }}
            </div>
          </div>

          <div class="col-span-2 sm:col-span-1 py-3 px-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] flex flex-col items-center gap-1">
            <span class="text-xs text-slate-400">会话 / 活跃任务</span>
            <div class="text-sm sm:text-base font-bold text-slate-700 dark:text-slate-200 font-mono mt-0.5">
              {{ statusData.conversations_count ?? 0 }} / <span class="text-emerald-500">{{ statusData.active_runs ?? 0 }}</span>
            </div>
          </div>
        </div>
      </div>
    </n-card>

    <!-- fnOS 宿主机系统资源监控 -->
    <n-card :bordered="false" class="shadow-sm rounded-2xl">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 min-w-0">
          <div class="flex flex-wrap items-center gap-2 min-w-0">
            <n-icon :size="18" class="text-fnos-blue dark:text-blue-400 shrink-0"><Server /></n-icon>
            <span class="text-base font-bold text-slate-800 dark:text-slate-100 whitespace-nowrap shrink-0">fnOS 宿主机资源</span>
            <n-tag v-if="host" size="tiny" round :bordered="false" class="font-mono bg-slate-100 dark:bg-white/[0.08] shrink-0">
              {{ host.hostname }} · {{ host.os }} ({{ host.arch }})
            </n-tag>
          </div>
          <span v-if="host?.load_avg" class="text-xs font-mono text-slate-400 whitespace-nowrap shrink-0">Load: {{ host.load_avg }}</span>
        </div>
      </template>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-slate-600 dark:text-slate-300 flex items-center gap-1.5">
              <n-icon :size="15" class="text-blue-500"><Cpu /></n-icon>
              CPU 负载 ({{ host?.cpu_cores || 4 }} 核)
            </span>
            <span class="font-mono font-bold text-slate-800 dark:text-slate-100">{{ (host?.cpu_percent || 0).toFixed(1) }}%</span>
          </div>
          <n-progress type="line" :percentage="Math.min(100, Math.round(host?.cpu_percent || 0))" :show-indicator="false"
            :status="(host?.cpu_percent || 0) > 85 ? 'error' : 'info'" :height="7" border-radius="4px" />
        </div>

        <div class="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-slate-600 dark:text-slate-300 flex items-center gap-1.5">
              <n-icon :size="15" class="text-emerald-500"><Activity /></n-icon>
              物理内存
            </span>
            <span class="font-mono font-bold text-slate-800 dark:text-slate-100">
              {{ formatGB(host?.mem_used_bytes) }} / {{ formatGB(host?.mem_total_bytes) }} ({{ (host?.mem_percent || 0).toFixed(1) }}%)
            </span>
          </div>
          <n-progress type="line" :percentage="Math.min(100, Math.round(host?.mem_percent || 0))" :show-indicator="false"
            :status="(host?.mem_percent || 0) > 88 ? 'warning' : 'success'" :height="7" border-radius="4px" />
        </div>

        <div class="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100/80 dark:border-white/[0.06] space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-slate-600 dark:text-slate-300 flex items-center gap-1.5">
              <n-icon :size="15" class="text-purple-500"><Database /></n-icon>
              存储卷 ({{ host?.disk_mount || '/vol3' }})
            </span>
            <span class="font-mono font-bold text-slate-800 dark:text-slate-100">
              可用 {{ formatGB(host?.disk_avail_bytes) }} ({{ (host?.disk_percent || 0).toFixed(1) }}%)
            </span>
          </div>
          <n-progress type="line" :percentage="Math.min(100, Math.round(host?.disk_percent || 0))" :show-indicator="false"
            :status="(host?.disk_percent || 0) > 90 ? 'error' : 'default'" :height="7" border-radius="4px" />
        </div>
      </div>

      <!-- 宿主机运行时能力标签 -->
      <div v-if="caps" class="mt-4 pt-3 border-t border-slate-100 dark:border-white/[0.06] flex items-center flex-wrap gap-2 text-xs">
        <span class="text-slate-400 mr-1">宿主机能力探测:</span>
        <n-tag size="small" round :bordered="false" :type="caps.language_server_connected ? 'success' : 'default'">
          ConversationService {{ caps.language_server_connected ? '在线' : '离线' }}
        </n-tag>
        <n-tag size="small" round :bordered="false" :type="caps.node_installed ? 'success' : 'warning'">
          Node {{ caps.node_installed ? caps.node_version : '未检测' }}
        </n-tag>
        <n-tag size="small" round :bordered="false" :type="caps.python_installed ? 'success' : 'warning'">
          {{ caps.python_installed ? caps.python_version : 'Python 未检测' }}
        </n-tag>
        <n-tag size="small" round :bordered="false" :type="caps.docker_installed ? 'info' : 'default'">
          Docker {{ caps.docker_installed ? '可用' : '未安装' }}
        </n-tag>
        <n-tag size="small" round :bordered="false" :type="caps.agy_cli_installed ? 'success' : 'info'">
          {{ caps.agy_cli_installed ? `agy ${caps.agy_cli_version}` : '内置 Harness Engine' }}
        </n-tag>
      </div>
    </n-card>

    <!-- 运行控制卡片矩阵 -->
    <div class="space-y-3">
      <h2 class="text-lg font-bold text-slate-800 dark:text-slate-100 tracking-tight">运行控制</h2>
      <div class="grid grid-cols-2 sm:grid-cols-5 gap-3">
        <n-card v-for="a in actionCards" :key="a.action" hoverable :bordered="false"
          class="cursor-pointer text-center interactive-card select-none shadow-sm group rounded-2xl"
          :class="{ 'opacity-50 !cursor-not-allowed': a.disabled && !a.loading }"
          @click="!a.disabled && !a.loading && handleAction(a.action)">
          <div class="flex flex-col items-center justify-center gap-2 py-2">
            <div class="w-11 h-11 rounded-2xl flex items-center justify-center transition-transform duration-200 group-hover:scale-110"
              :class="a.iconBg">
              <n-spin v-if="a.loading" :size="20" />
              <n-icon v-else :size="22" :class="a.iconColor">
                <component :is="a.icon" />
              </n-icon>
            </div>
            <span class="text-sm font-medium text-slate-700 dark:text-slate-200">{{ a.label }}</span>
            <span class="text-[11px] text-slate-400 leading-tight">{{ a.desc }}</span>
          </div>
        </n-card>
      </div>
    </div>

    <!-- 实时告警 -->
    <div class="space-y-2.5">
      <n-alert v-if="statusData.last_message" type="info" :show-icon="true" class="rounded-2xl shadow-sm">
        {{ statusData.last_message }}
      </n-alert>
      <n-alert v-if="!wsConnected" type="error" :show-icon="true" class="rounded-2xl shadow-sm">
        WebSocket 实时通道已断开，正在自动重连…
      </n-alert>
    </div>

    <!-- 健康巡检弹窗 (agy doctor) -->
    <n-modal v-model:show="showDoctorModal" preset="card" title="Antigravity Harness 健康巡检报告"
      class="w-[92vw] max-w-2xl rounded-2xl shadow-xl">
      <n-spin :show="loadingDoctor">
        <div v-if="doctorReport" class="space-y-3">
          <div class="flex items-center justify-between text-xs text-slate-500 pb-2 border-b border-slate-100 dark:border-white/[0.08]">
            <span>巡检时间: {{ doctorReport.checked_at }}</span>
            <div class="flex gap-2">
              <n-tag type="success" size="small" round :bordered="false">通过 {{ doctorReport.passed_count }}</n-tag>
              <n-tag type="warning" size="small" round :bordered="false">警告 {{ doctorReport.warn_count }}</n-tag>
              <n-tag v-if="doctorReport.error_count" type="error" size="small" round :bordered="false">异常 {{ doctorReport.error_count }}</n-tag>
            </div>
          </div>
          <div class="space-y-2 max-h-[55vh] overflow-y-auto pr-1">
            <div v-for="(item, idx) in doctorReport.checks" :key="idx"
              class="p-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100 dark:border-white/[0.06] flex flex-col gap-1">
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm font-semibold text-slate-800 dark:text-slate-100">{{ item.name }}</span>
                <n-tag :type="item.status === 'ok' ? 'success' : item.status === 'warn' ? 'warning' : 'error'" size="tiny" round :bordered="false">
                  {{ item.status === 'ok' ? '正常' : item.status === 'warn' ? '提醒' : '异常' }}
                </n-tag>
              </div>
              <div class="text-xs text-slate-600 dark:text-slate-300">{{ item.detail }}</div>
              <div v-if="item.suggest" class="text-[11px] text-amber-600 dark:text-amber-400">建议：{{ item.suggest }}</div>
            </div>
          </div>
        </div>
      </n-spin>
    </n-modal>

    <!-- 在线部署弹窗 -->
    <n-modal v-model:show="showDeployModal" preset="dialog" type="info" title="部署 / 切换 Harness 版本"
      positive-text="立即部署" negative-text="取消" @positive-click="confirmDeploy">
      <div class="space-y-3 pt-2 text-xs">
        <div class="flex justify-between">
          <span class="text-slate-500">当前版本</span>
          <span class="font-mono font-semibold">v{{ statusData.version }}</span>
        </div>
        <div class="space-y-1">
          <span class="text-slate-500">目标版本</span>
          <n-select v-model:value="selectedVersion" :options="versionOptions" />
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, type Component } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NCard, NDivider, NButton, NTag, NAlert, NSpin, NIcon, NProgress, NModal, NSelect, useMessage
} from 'naive-ui'
import {
  ExternalLink, PlayerPlay, PlayerStop, Refresh, Download, CloudDownload, ShieldCheck,
  MessageCircle, Server, Cpu, Activity, Database
} from '@vicons/tabler'
import { useSystemStore } from '../stores/system'
import { useAppStore } from '../stores/app'
import { systemApi } from '../api'
import type { DoctorReport } from '../types/api'

const systemStore = useSystemStore()
const appStore = useAppStore()
const message = useMessage()

const {
  statusData, wsConnected, activeAction, isActionLocked, isRunning, isStarting, isBuilding,
  statusTagType, statusLabel, uptimeText
} = storeToRefs(systemStore)

const host = computed(() => statusData.value.host)
const caps = computed(() => statusData.value.capabilities)
const custom = computed(() => statusData.value.customizations)
const customTotal = computed(() => {
  const c = custom.value
  if (!c) return 0
  return c.skills_count + c.mcp_count + c.plugins_count + c.rules_count + c.hooks_count
})

const showDoctorModal = ref(false)
const loadingDoctor = ref(false)
const doctorReport = ref<DoctorReport | null>(null)

const showDeployModal = ref(false)
const selectedVersion = ref('2.0.4')
const versionOptions = ref([
  { label: 'v2.1.0-beta.2 (Beta)', value: '2.1.0-beta.2' },
  { label: 'v2.0.4 (推荐稳定版)', value: '2.0.4' },
  { label: 'v2.0.3', value: '2.0.3' },
  { label: 'v2.0.0', value: '2.0.0' }
])

function formatGB(bytes?: number) {
  if (!bytes) return '-'
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

interface ActionCard {
  action: string
  icon: Component
  label: string
  desc: string
  iconBg: string
  iconColor: string
  disabled: boolean
  loading: boolean
}

const actionCards = computed<ActionCard[]>(() => [
  isRunning.value
    ? {
        action: 'stop',
        icon: PlayerStop,
        label: '停止服务',
        desc: '优雅停止 Harness 运行时',
        iconBg: 'bg-rose-50 dark:bg-rose-950/30',
        iconColor: 'text-rose-600 dark:text-rose-400',
        disabled: isActionLocked.value,
        loading: activeAction.value === 'stop'
      }
    : {
        action: 'start',
        icon: PlayerPlay,
        label: '启动服务',
        desc: '启动 Harness 与 RPC 网关',
        iconBg: 'bg-emerald-50 dark:bg-emerald-950/30',
        iconColor: 'text-emerald-600 dark:text-emerald-400',
        disabled: isActionLocked.value,
        loading: activeAction.value === 'start' || isStarting.value
      },
  {
    action: 'restart',
    icon: Refresh,
    label: '重启服务',
    desc: '热重启使配置即时生效',
    iconBg: 'bg-amber-50 dark:bg-amber-950/30',
    iconColor: 'text-amber-600 dark:text-amber-400',
    disabled: isActionLocked.value,
    loading: activeAction.value === 'restart'
  },
  {
    action: 'doctor',
    icon: ShieldCheck,
    label: '健康巡检',
    desc: '诊断环境、MCP 与凭据',
    iconBg: 'bg-teal-50 dark:bg-teal-950/30',
    iconColor: 'text-teal-600 dark:text-teal-400',
    disabled: loadingDoctor.value,
    loading: loadingDoctor.value
  },
  {
    action: 'check_update',
    icon: Download,
    label: '检查更新',
    desc: '检测 Antigravity 新版本',
    iconBg: 'bg-blue-50 dark:bg-blue-950/30',
    iconColor: 'text-fnos-blue dark:text-blue-400',
    disabled: isActionLocked.value,
    loading: activeAction.value === 'check_update'
  },
  {
    action: 'deploy',
    icon: CloudDownload,
    label: '版本部署',
    desc: '切换或重装运行时版本',
    iconBg: 'bg-purple-50 dark:bg-purple-950/30',
    iconColor: 'text-purple-600 dark:text-purple-400',
    disabled: isActionLocked.value,
    loading: isBuilding.value
  }
])

async function handleAction(action: string) {
  if (action === 'doctor') {
    showDoctorModal.value = true
    loadingDoctor.value = true
    const res = await systemApi.sendAction('doctor')
    loadingDoctor.value = false
    if (res.success && res.data) {
      doctorReport.value = res.data as DoctorReport
    } else {
      message.error(res.message || '健康巡检失败')
    }
    return
  }
  if (action === 'check_update') {
    const res = await systemApi.checkUpdate()
    if (res.success && res.data) {
      message.info(res.data.message)
    } else {
      message.error(res.message || '检查更新失败')
    }
    return
  }
  if (action === 'deploy') {
    const res = await systemApi.getVersions()
    if (res.success && res.data) {
      versionOptions.value = res.data.versions.map((v) => ({ label: `v${v}`, value: v }))
      selectedVersion.value = res.data.recommended_version
    }
    showDeployModal.value = true
    return
  }
  const res = await systemStore.sendAction(action)
  if (res.success) {
    message.success(res.message || '操作成功')
  } else {
    message.error(res.message || '操作失败')
  }
}

async function confirmDeploy() {
  showDeployModal.value = false
  const res = await systemStore.sendAction('deploy', { target_version: selectedVersion.value })
  if (res.success) message.success(res.message)
  else message.error(res.message)
}
</script>
