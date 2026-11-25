import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { StatusData, HostMetrics } from '../types/api'
import { systemApi } from '../api'
import { trimSdk } from '../utils/trimSdk'

export const useSystemStore = defineStore('system', () => {
  const statusData = ref<StatusData>({
    name: 'Antigravity Harness',
    version: '2.0.4',
    app_version: '1.0.0',
    status: 'stopped',
    uptime: '-',
    started_at: 0,
    server_port: 2398,
    proxy_port: 2399,
    default_model: 'gemini-2.5-pro',
    sandbox_mode: 'workspace-write',
    approval_policy: 'on-request',
    cpu: '-',
    memory: '-',
    last_message: ''
  })

  const wsConnected = ref(true)
  const statusLoaded = ref(false)
  const activeAction = ref<string | null>(null)
  const currentTime = ref(Date.now())
  const serverTimeOffset = ref(0)
  let clockTimer: ReturnType<typeof setInterval> | null = null

  function startClock() {
    if (clockTimer) return
    clockTimer = setInterval(() => {
      currentTime.value = Date.now()
    }, 1000)
  }

  function setWsConnected(v: boolean) {
    wsConnected.value = v
  }

  function updateStatus(data: Partial<StatusData>) {
    statusLoaded.value = true
    statusData.value = { ...statusData.value, ...data }
    if (data.server_time) {
      serverTimeOffset.value = data.server_time * 1000 - Date.now()
    }
    if (activeAction.value && (data.status === 'running' || data.status === 'stopped')) {
      activeAction.value = null
    }
  }

  function updateUsage(cpu?: string, memory?: string, host?: HostMetrics) {
    if (cpu !== undefined) statusData.value.cpu = cpu
    if (memory !== undefined) statusData.value.memory = memory
    if (host !== undefined) statusData.value.host = host
  }

  const isRunning = computed(() => statusData.value.status === 'running')
  const isStarting = computed(() => statusData.value.status === 'starting')
  const isBuilding = computed(() => statusData.value.status === 'building')
  const isSnapshotting = computed(() => statusData.value.status === 'snapshotting')
  const isActionLocked = computed(() => Boolean(activeAction.value) || isStarting.value || isBuilding.value)

  const statusTagType = computed<'success' | 'warning' | 'info' | 'default'>(() => {
    if (isRunning.value) return 'success'
    if (isStarting.value) return 'warning'
    if (isBuilding.value || isSnapshotting.value) return 'info'
    return 'default'
  })

  const statusLabel = computed(() => {
    if (isRunning.value) return '运行中'
    if (isStarting.value) return '启动中'
    if (isBuilding.value) return '部署中'
    if (isSnapshotting.value) return '快照中'
    return '已停止'
  })

  const uptimeText = computed(() => {
    const s = statusData.value
    if (s.status !== 'running' || !s.started_at) return '-'
    const serverNow = currentTime.value + serverTimeOffset.value
    const secs = Math.max(0, Math.floor((serverNow - s.started_at * 1000) / 1000))
    const h = Math.floor(secs / 3600)
    const m = Math.floor((secs % 3600) / 60)
    const sec = secs % 60
    if (h > 0) return `${h}小时${m}分${sec}秒`
    if (m > 0) return `${m}分${sec}秒`
    return `${sec}秒`
  })

  async function fetchInitialStatus() {
    const res = await systemApi.getStatus()
    if (res.success && res.data) {
      updateStatus(res.data)
    } else {
      statusLoaded.value = true
    }
  }

  async function sendAction(action: string, payload?: Record<string, unknown>) {
    activeAction.value = action
    try {
      const res = await systemApi.sendAction(action, payload)
      if (res.success && res.data && 'status' in res.data) {
        updateStatus(res.data as StatusData)
      }
      return res
    } finally {
      setTimeout(() => {
        activeAction.value = null
      }, 300)
    }
  }

  async function openGateway() {
    const prefix = window.location.pathname.startsWith('/app/antigravity-harness')
      ? '/app/antigravity-harness/fngateway/'
      : '/fngateway/'
    await trimSdk.openURL(`${window.location.origin}${prefix}`, '_blank')
  }

  return {
    statusData,
    wsConnected,
    statusLoaded,
    activeAction,
    isRunning,
    isStarting,
    isBuilding,
    isSnapshotting,
    isActionLocked,
    statusTagType,
    statusLabel,
    uptimeText,
    startClock,
    setWsConnected,
    updateStatus,
    updateUsage,
    fetchInitialStatus,
    sendAction,
    openGateway
  }
})
