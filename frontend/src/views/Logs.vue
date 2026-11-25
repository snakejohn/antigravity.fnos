<template>
  <div class="w-full h-[calc(100dvh-82px)] sm:h-[calc(100dvh-48px)] flex flex-col gap-3 sm:gap-4 min-h-0 overflow-hidden select-none">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3 shrink-0">
      <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">运行日志</h1>
      <div class="flex items-center gap-2 shrink-0">
        <n-select v-model:value="levelFilter" :options="levelOptions" size="small" class="!w-[92px]" />
        <n-button size="small" secondary @click="autoScroll = !autoScroll" class="!h-8 rounded-lg text-xs">
          <div class="flex items-center gap-1.5">
            <span>自滚</span>
            <n-switch :value="autoScroll" size="small" @click.stop="autoScroll = !autoScroll" />
          </div>
        </n-button>
        <n-button size="small" secondary @click="logStore.downloadLogs" class="!h-8 rounded-lg text-xs">
          <template #icon><n-icon><Download /></n-icon></template>
          <span class="hidden sm:inline">下载</span>
        </n-button>
        <n-popconfirm @positive-click="clearAll">
          <template #trigger>
            <n-button size="small" secondary type="error" class="!h-8 rounded-lg text-xs">
              <template #icon><n-icon><Trash /></n-icon></template>
              <span class="hidden sm:inline">清空</span>
            </n-button>
          </template>
          确认清空全部运行日志？
        </n-popconfirm>
      </div>
    </div>

    <!-- 实时日志终端 -->
    <div class="relative flex-1 min-h-0 flex flex-col w-full bg-slate-50 dark:bg-[#12141a] rounded-2xl border border-slate-200/80 dark:border-white/[0.08] shadow-sm overflow-hidden">
      <n-log ref="logInstRef" :log="filteredLogText" :hljs="hljs" language="agy-log" :font-size="12"
        :line-height="1.55" trim class="flex-1 min-h-0 w-full p-3 sm:p-4 select-text cursor-text" style="height: 100%;" />
      <div v-if="fetching" class="absolute inset-0 z-20 flex items-center justify-center bg-slate-50/80 dark:bg-[#12141a]/80">
        <n-spin size="medium" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onActivated, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { NSelect, NButton, NSwitch, NIcon, NLog, NSpin, NPopconfirm, useMessage, type LogInst } from 'naive-ui'
import { Download, Trash } from '@vicons/tabler'
import hljs from 'highlight.js/lib/core'
import { useLogStore } from '../stores/log'

hljs.registerLanguage('agy-log', () => ({
  contains: [
    { className: 'type', begin: /\[FATAL\]|\[ERROR\]/ },
    { className: 'keyword', begin: /\[WARN\]|\[WARNING\]/ },
    { className: 'meta', begin: /\[INFO\]/ },
    { className: 'comment', begin: /\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/ },
    { className: 'link', begin: /https?:\/\/[^\s]+/ },
    { className: 'string', begin: /(?:\/[\w.-]+)+\/?/ },
    { className: 'number', begin: /\b\d+\b/ }
  ]
}))

const logStore = useLogStore()
const message = useMessage()
const { displayedText, logAutoScroll: autoScroll, fetching } = storeToRefs(logStore)

const levelFilter = ref('ALL')
const levelOptions = [
  { label: 'ALL', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'FATAL', value: 'FATAL' }
]

const filteredLogText = computed(() => {
  if (levelFilter.value === 'ALL') return displayedText.value
  return displayedText.value
    .split('\n')
    .filter((l) => l.includes(`[${levelFilter.value}]`))
    .join('\n')
})

const logInstRef = ref<LogInst | null>(null)
let offFlush: (() => void) | null = null

function scrollToBottom() {
  if (!autoScroll.value) return
  nextTick(() => {
    logInstRef.value?.scrollTo({ position: 'bottom', silent: true })
  })
}

async function clearAll() {
  const res = await logStore.clearLogs()
  if (res.success) message.success('运行日志已清空')
  else message.error(res.message)
}

onMounted(() => {
  if (!logStore.hasLoadedSnapshot) {
    void logStore.fetchLogs().then(scrollToBottom)
  } else {
    scrollToBottom()
  }
  offFlush = logStore.onFlush(scrollToBottom)
})

onActivated(scrollToBottom)
onUnmounted(() => offFlush?.())
</script>

<style scoped>
:deep(.hljs-type) { color: #ef4444; font-weight: 600; }
:deep(.hljs-keyword) { color: #f59e0b; font-weight: 600; }
:deep(.hljs-meta) { color: #2563eb; font-weight: 600; }
:deep(.hljs-comment) { color: #94a3b8; }
:deep(.hljs-number) { color: #0891b2; }
:deep(.hljs-string) { color: #059669; }
html.dark :deep(.hljs-meta) { color: #60a5fa; }
html.dark :deep(.hljs-string) { color: #34d399; }
</style>
