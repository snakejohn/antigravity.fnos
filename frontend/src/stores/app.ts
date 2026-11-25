import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SnapshotMeta, SnapshotSummary, SnapshotProgressTask, ConversationMeta, WorkspaceItem } from '../types/api'
import { wsClient } from '../utils/websocket'
import { useSystemStore } from './system'
import { useLogStore } from './log'
import { snapshotApi, conversationsApi, workspaceApi } from '../api'

export const useAppStore = defineStore('app', () => {
  const currentTab = ref('overview')
  const snapshots = ref<SnapshotMeta[]>([])
  const totalSnapshotBytes = ref(0)
  const snapshotProgress = ref<SnapshotProgressTask>({ active: false, action: '', title: '', percent: 0 })
  const conversations = ref<ConversationMeta[]>([])
  const workspaces = ref<WorkspaceItem[]>([])
  let initialized = false

  function setTab(tab: string) {
    currentTab.value = tab
    try {
      localStorage.setItem('agy_active_tab', tab)
    } catch {
      // ignore
    }
  }

  async function refreshSnapshots() {
    const res = await snapshotApi.list()
    if (res.success && res.data) {
      snapshots.value = res.data.items || []
      totalSnapshotBytes.value = res.data.total_size_bytes || 0
    }
  }

  async function refreshConversations(includeArchived = false) {
    const res = await conversationsApi.list(includeArchived)
    if (res.success && res.data) {
      conversations.value = res.data.items || []
    }
  }

  async function refreshWorkspaces() {
    const res = await workspaceApi.list()
    if (res.success && res.data) {
      workspaces.value = res.data.items || []
    }
  }

  function init() {
    if (initialized) return
    initialized = true

    try {
      const saved = localStorage.getItem('agy_active_tab')
      if (saved) currentTab.value = saved
    } catch {
      // ignore
    }

    const systemStore = useSystemStore()
    const logStore = useLogStore()
    systemStore.startClock()
    void systemStore.fetchInitialStatus()

    wsClient.on('connectionChange', (connected) => systemStore.setWsConnected(connected))
    wsClient.on('status', (data) => systemStore.updateStatus(data))
    wsClient.on('usage', (data) => systemStore.updateUsage(data.cpu, data.memory, data.host))
    wsClient.on('log', (chunk) => logStore.appendChunk(chunk))
    wsClient.on('workspace', (data) => {
      workspaces.value = data.items || []
    })
    wsClient.on('conversations', (items) => {
      conversations.value = items || []
    })
    wsClient.on('snapshot', (summary: SnapshotSummary) => {
      snapshots.value = summary.items || []
      totalSnapshotBytes.value = summary.total_size_bytes || 0
    })
    wsClient.on('snapshot_progress', (prog) => {
      snapshotProgress.value = prog
    })
    wsClient.on('reconnected', () => {
      void systemStore.fetchInitialStatus()
      void refreshWorkspaces()
      void refreshConversations()
      void refreshSnapshots()
    })

    wsClient.connect()
  }

  return {
    currentTab,
    snapshots,
    totalSnapshotBytes,
    snapshotProgress,
    conversations,
    workspaces,
    setTab,
    refreshSnapshots,
    refreshConversations,
    refreshWorkspaces,
    init
  }
})
