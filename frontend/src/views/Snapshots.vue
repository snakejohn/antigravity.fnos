<template>
  <div class="w-full flex-1 flex flex-col gap-4 sm:gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex items-center justify-between gap-3">
      <div class="flex items-baseline gap-2.5">
        <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">快照管理</h1>
        <span v-if="snapshots.length" class="text-xs text-slate-400 font-medium">
          共 {{ snapshots.length }} 个 · {{ formatSize(totalBytes) }}
        </span>
      </div>
      <n-button type="primary" size="small" :disabled="progress.active" @click="openCreateModal">
        <template #icon><n-icon><Plus /></n-icon></template>
        创建快照
      </n-button>
    </div>

    <!-- 快照进度条 -->
    <n-card v-if="progress.active" :bordered="false" class="rounded-2xl shadow-sm border border-blue-200 dark:border-blue-900/50">
      <div class="space-y-2">
        <div class="flex items-center justify-between text-xs">
          <div class="flex items-center gap-2 font-medium text-slate-700 dark:text-slate-200">
            <n-spin :size="14" />
            <span>{{ progress.title }}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="font-mono font-bold text-fnos-blue">{{ Math.round(progress.percent) }}%</span>
            <n-button v-if="progress.action === 'create'" size="tiny" secondary type="error" @click="cancelSnap">取消</n-button>
          </div>
        </div>
        <n-progress type="line" :percentage="Math.round(progress.percent)" :show-indicator="false" processing status="info" :height="6" />
      </div>
    </n-card>

    <!-- 快照列表 -->
    <div v-if="!snapshots.length" class="w-full">
      <n-card :bordered="false" class="py-12 text-center shadow-sm rounded-2xl">
        <n-empty description="暂无快照备份（可一键打包 config、Skills、MCP、Rules 与历史会话）">
          <template #icon><n-icon :size="48" class="text-slate-300 dark:text-slate-600"><History /></n-icon></template>
        </n-empty>
      </n-card>
    </div>

    <div v-else class="flex flex-col gap-3">
      <n-card v-for="item in snapshots" :key="item.id" hoverable :bordered="false" class="rounded-2xl shadow-sm interactive-card">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-10 h-10 rounded-xl bg-slate-100 dark:bg-white/[0.06] text-fnos-blue dark:text-blue-400 flex items-center justify-center shrink-0">
              <n-icon :size="20"><History /></n-icon>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-semibold text-sm text-slate-800 dark:text-slate-100 truncate">{{ item.name }}</span>
                <n-tag size="tiny" :bordered="false" class="font-mono">v{{ item.harness_version }}</n-tag>
                <n-tag size="tiny" :bordered="false" type="info">Lv{{ item.compression_level }}</n-tag>
              </div>
              <div class="text-xs text-slate-400 flex items-center flex-wrap gap-x-2.5">
                <span class="flex items-center gap-1"><n-icon :size="12"><Clock /></n-icon>{{ formatTime(item.created_at) }}</span>
                <span>·</span>
                <span class="font-mono">{{ formatSize(item.size_bytes) }}</span>
                <span>·</span>
                <span>{{ item.custom_count }} 定制项 / {{ item.conv_count }} 会话</span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <n-popconfirm @positive-click="restoreSnap(item.id)">
              <template #trigger>
                <n-button size="small" secondary type="primary" :disabled="progress.active">
                  <template #icon><n-icon><Rotate /></n-icon></template>
                  还原
                </n-button>
              </template>
              确定还原快照「{{ item.name }}」？当前配置与定制将被覆盖。
            </n-popconfirm>
            <n-popconfirm @positive-click="deleteSnap(item.id)">
              <template #trigger>
                <n-button size="small" secondary type="error" :disabled="progress.active">
                  <template #icon><n-icon><Trash /></n-icon></template>
                  删除
                </n-button>
              </template>
              永久删除快照「{{ item.name }}」？
            </n-popconfirm>
          </div>
        </div>
      </n-card>
    </div>

    <!-- 创建快照弹窗 -->
    <n-modal v-model:show="showCreateModal" preset="dialog" title="创建 Antigravity 快照"
      positive-text="立即打包" negative-text="取消" @positive-click="submitCreate">
      <div class="space-y-3 pt-2 text-xs">
        <div class="p-2.5 rounded-xl bg-slate-50 dark:bg-white/[0.04] text-slate-500">
          自动将 config.json、.env、多账号池、Skills/MCP/Plugins/Rules/Hooks 与全部会话 transcript 打包为 tar.gz 归档。
        </div>
        <div>
          <div class="mb-1 font-medium">快照名称</div>
          <n-input v-model:value="snapName" placeholder="快照名称" />
        </div>
        <div>
          <div class="mb-1 font-medium">压缩级别</div>
          <n-select v-model:value="compressionLevel" :options="compressionOptions" />
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NCard, NButton, NTag, NIcon, NEmpty, NModal, NInput, NSelect, NProgress, NSpin, NPopconfirm, useMessage
} from 'naive-ui'
import { History, Plus, Clock, Rotate, Trash } from '@vicons/tabler'
import { useAppStore } from '../stores/app'
import { snapshotApi } from '../api'

const appStore = useAppStore()
const message = useMessage()

const snapshots = computed(() => appStore.snapshots)
const totalBytes = computed(() => appStore.totalSnapshotBytes)
const progress = computed(() => appStore.snapshotProgress)

const showCreateModal = ref(false)
const snapName = ref('')
const compressionLevel = ref(1)
const compressionOptions = [
  { label: '快速 (Lv 1)', value: 1 },
  { label: '标准 (Lv 6)', value: 6 },
  { label: '高压缩 (Lv 9)', value: 9 }
]

function openCreateModal() {
  const d = new Date()
  const pad = (n: number) => (n < 10 ? '0' + n : String(n))
  snapName.value = `AGY快照_${pad(d.getMonth() + 1)}${pad(d.getDate())}_${pad(d.getHours())}${pad(d.getMinutes())}`
  showCreateModal.value = true
}

async function submitCreate() {
  showCreateModal.value = false
  const res = await snapshotApi.create(snapName.value.trim(), compressionLevel.value)
  if (res.success) message.info(res.message)
  else message.error(res.message)
}

async function cancelSnap() {
  const res = await snapshotApi.cancel()
  if (res.success) message.warning(res.message)
}

async function restoreSnap(id: string) {
  const res = await snapshotApi.restore(id)
  if (res.success) message.info(res.message)
  else message.error(res.message)
}

async function deleteSnap(id: string) {
  const res = await snapshotApi.remove(id)
  if (res.success) {
    message.success('快照已删除')
    await appStore.refreshSnapshots()
  } else message.error(res.message)
}

function formatSize(bytes: number) {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatTime(ts: number) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

onMounted(() => {
  void appStore.refreshSnapshots()
})
</script>
