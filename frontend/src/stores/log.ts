import { defineStore } from 'pinia'
import { ref } from 'vue'
import { logApi } from '../api'

export const useLogStore = defineStore('log', () => {
  const displayedText = ref('')
  const logAutoScroll = ref(true)
  const fetching = ref(false)
  const hasLoadedSnapshot = ref(false)
  const flushListeners = new Set<() => void>()

  function notifyFlush() {
    flushListeners.forEach((cb) => cb())
  }

  function onFlush(cb: () => void) {
    flushListeners.add(cb)
    return () => flushListeners.delete(cb)
  }

  async function fetchLogs() {
    fetching.value = true
    try {
      const res = await logApi.getLogs(500)
      if (res.success && res.data) {
        displayedText.value = res.data.content || ''
        hasLoadedSnapshot.value = true
        notifyFlush()
      }
    } finally {
      fetching.value = false
    }
  }

  function appendChunk(chunk: string) {
    if (!chunk) return
    displayedText.value += chunk
    if (displayedText.value.length > 250000) {
      displayedText.value = displayedText.value.slice(-180000)
    }
    notifyFlush()
  }

  async function clearLogs() {
    const res = await logApi.clearLogs()
    if (res.success) {
      displayedText.value = ''
    }
    return res
  }

  function downloadLogs() {
    window.open(logApi.downloadUrl(), '_blank')
  }

  return {
    displayedText,
    logAutoScroll,
    fetching,
    hasLoadedSnapshot,
    fetchLogs,
    appendChunk,
    clearLogs,
    downloadLogs,
    onFlush
  }
})
