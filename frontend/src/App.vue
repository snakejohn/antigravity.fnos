<template>
  <n-config-provider :theme="currentTheme" :theme-overrides="currentThemeOverrides">
    <n-global-style />
    <n-loading-bar-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <n-message-provider>
            <!-- 全屏 Antigravity 2.0 工作窗口模式 (/fngateway/ 或 ?mode=workbench) -->
            <WorkbenchView v-if="isWorkbenchMode" :theme-mode="themeMode" @toggle-theme="toggleTheme" />

            <!-- 管理控制台模式 (Admin Console) -->
            <n-layout v-else has-sider position="absolute" class="h-screen w-screen bg-[#f5f7fa] dark:bg-[#12141a]">
              <!-- 桌面端左侧导航栏 -->
              <n-layout-sider bordered :width="244" :native-scrollbar="false" class="hidden sm:block select-none z-10"
                content-style="display: flex; flex-direction: column; justify-content: space-between; height: 100%;">
                <div class="flex flex-col gap-3 p-3">
                  <!-- 品牌卡片 -->
                  <div @click="openDocs"
                    class="flex items-center gap-3 px-3 py-3 rounded-2xl bg-slate-50 dark:bg-white/[0.04] border border-slate-100/80 dark:border-white/[0.06] transition-all duration-200 hover:border-slate-200 dark:hover:border-white/[0.12] hover:bg-slate-100/60 dark:hover:bg-white/[0.07] cursor-pointer group">
                    <img :src="'./favicon.svg'" alt="Antigravity"
                      class="w-9 h-9 rounded-xl object-contain shrink-0 transition-transform duration-200 group-hover:scale-105 shadow-xs" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between gap-1.5">
                        <span class="text-sm font-bold text-slate-800 dark:text-slate-100 leading-tight truncate group-hover:text-fnos-blue dark:group-hover:text-blue-400 transition-colors">
                          Antigravity
                        </span>
                        <span class="w-2 h-2 rounded-full shrink-0"
                          :class="systemStore.isRunning ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]' : 'bg-slate-400'" />
                      </div>
                      <div class="text-[11px] text-slate-400 dark:text-slate-500 font-medium truncate mt-0.5">
                        Harness 管理器 · fnOS
                      </div>
                    </div>
                  </div>

                  <!-- 快捷进入独立全屏工作窗口按钮 -->
                  <n-button type="primary" secondary block size="small" @click="systemStore.openGateway"
                    class="!h-9 rounded-xl font-medium">
                    <template #icon><n-icon><ExternalLink /></n-icon></template>
                    打开工作窗口 (WebUI)
                  </n-button>

                  <!-- 主导航菜单 -->
                  <n-menu :value="tab" :options="menuOptions" @update:value="handleMenuSelect" />
                </div>

                <!-- 底部设置与主题切换 -->
                <div class="p-3">
                  <n-divider class="!my-2" />
                  <n-menu :value="tab" :options="settingsMenuOptions" @update:value="handleMenuSelect" />
                  <div class="mt-2 px-2 flex items-center justify-between text-xs text-slate-400 dark:text-slate-500">
                    <span class="font-mono">v{{ systemStore.statusData.app_version || '1.0.0' }}</span>
                    <n-button quaternary circle size="small" @click="toggleTheme" :title="themeMode === 'dark' ? '切换浅色主题' : '切换深色主题'">
                      <template #icon>
                        <n-icon :size="16">
                          <Sun v-if="themeMode === 'dark'" />
                          <Moon v-else />
                        </n-icon>
                      </template>
                    </n-button>
                  </div>
                </div>
              </n-layout-sider>

              <!-- 右侧内容区 -->
              <n-layout class="h-full flex-1 flex flex-col overflow-hidden bg-[#f5f7fa] dark:bg-[#12141a]"
                content-style="display: flex; flex-direction: column; height: 100%; flex: 1;">
                <n-layout-content :native-scrollbar="false" content-class="app-content-scroll"
                  content-style="min-height: 100%; display: flex; flex-direction: column; align-items: center;"
                  class="flex-1">
                  <div class="w-full max-w-6xl flex-1 flex flex-col min-h-0">
                    <Transition name="view-fade-slide" mode="out-in">
                      <KeepAlive>
                        <component :is="currentView" :key="tab" />
                      </KeepAlive>
                    </Transition>
                  </div>
                  <n-back-top :bottom="74" :right="20" class="sm:!bottom-8 sm:!right-8" />
                </n-layout-content>

                <!-- 移动端底部 Tabbar -->
                <n-layout-footer bordered position="absolute"
                  class="sm:hidden z-50 !bg-white/95 dark:!bg-[#181b22]/95 !backdrop-blur-md px-1 pt-1 shadow-lg mobile-tabbar-footer">
                  <div class="flex items-center justify-between w-full overflow-x-auto no-scrollbar gap-0.5">
                    <n-button v-for="t in mobileTabs" :key="t.key" text :type="tab === t.key ? 'primary' : 'default'"
                      @click="tab = t.key" class="flex-1 min-w-[44px] shrink-0 !py-1 !px-0.5">
                      <div class="flex flex-col items-center gap-0.5 select-none">
                        <n-icon :size="18">
                          <component :is="t.icon" />
                        </n-icon>
                        <span class="text-[10px] leading-tight whitespace-nowrap"
                          :class="tab === t.key ? 'font-bold text-fnos-blue dark:text-blue-400' : 'text-slate-500 dark:text-slate-400'">
                          {{ t.label }}
                        </span>
                      </div>
                    </n-button>
                  </div>
                </n-layout-footer>
              </n-layout>
            </n-layout>
          </n-message-provider>
        </n-notification-provider>
      </n-dialog-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { h, ref, computed, watch, onMounted, type Component } from 'vue'
import {
  NConfigProvider,
  NGlobalStyle,
  NMessageProvider,
  NDialogProvider,
  NNotificationProvider,
  NLoadingBarProvider,
  NLayout,
  NLayoutSider,
  NLayoutContent,
  NLayoutFooter,
  NMenu,
  NDivider,
  NButton,
  NBackTop,
  NIcon,
  darkTheme,
  useOsTheme,
  type MenuOption
} from 'naive-ui'
import {
  Dashboard,
  Puzzle,
  MessageCircle,
  Folder,
  Key,
  History,
  FileText,
  Settings,
  Sun,
  Moon,
  ExternalLink
} from '@vicons/tabler'
import { getThemeOverrides } from './theme'
import WorkbenchView from './views/Workbench.vue'
import Overview from './views/Overview.vue'
import Customizations from './views/Customizations.vue'
import Conversations from './views/Conversations.vue'
import Workspace from './views/Workspace.vue'
import AuthView from './views/Auth.vue'
import Snapshots from './views/Snapshots.vue'
import Logs from './views/Logs.vue'
import SettingsView from './views/Settings.vue'
import { useAppStore } from './stores/app'
import { useSystemStore } from './stores/system'
import { trimSdk } from './utils/trimSdk'

const appStore = useAppStore()
const systemStore = useSystemStore()
const osTheme = useOsTheme()

const savedTheme = typeof localStorage !== 'undefined' ? localStorage.getItem('agy_theme') : null
const themeMode = ref<'light' | 'dark'>(
  savedTheme === 'dark' || savedTheme === 'light'
    ? savedTheme
    : osTheme.value === 'dark'
      ? 'dark'
      : 'light'
)

watch(
  themeMode,
  (mode) => {
    document.documentElement.dataset.theme = mode
    document.documentElement.classList.toggle('dark', mode === 'dark')
    try {
      localStorage.setItem('agy_theme', mode)
    } catch {
      // ignore
    }
  },
  { immediate: true }
)

function toggleTheme() {
  themeMode.value = themeMode.value === 'dark' ? 'light' : 'dark'
}

const isWorkbenchMode = computed(() => {
  if (typeof window === 'undefined') return false
  return (
    window.location.pathname.includes('/fngateway') ||
    new URLSearchParams(window.location.search).get('mode') === 'workbench'
  )
})

const currentTheme = computed(() => (themeMode.value === 'dark' ? darkTheme : null))
const currentThemeOverrides = computed(() => getThemeOverrides(themeMode.value))

type TabKey = 'overview' | 'customizations' | 'conversations' | 'workspace' | 'auth' | 'snapshots' | 'logs' | 'settings'

const tabLabels: Record<TabKey, string> = {
  overview: '概览 · Antigravity Harness',
  customizations: '定制中心 · Antigravity Harness',
  conversations: '会话与任务 · Antigravity Harness',
  workspace: '工作区与终端 · Antigravity Harness',
  auth: '授权管理 · Antigravity Harness',
  snapshots: '快照管理 · Antigravity Harness',
  logs: '运行日志 · Antigravity Harness',
  settings: '应用设置 · Antigravity Harness'
}

const views: Record<TabKey, Component> = {
  overview: Overview,
  customizations: Customizations,
  conversations: Conversations,
  workspace: Workspace,
  auth: AuthView,
  snapshots: Snapshots,
  logs: Logs,
  settings: SettingsView
}

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  { key: 'overview', label: '概览', icon: renderIcon(Dashboard) },
  { key: 'customizations', label: '定制中心', icon: renderIcon(Puzzle) },
  { key: 'conversations', label: '会话与任务', icon: renderIcon(MessageCircle) },
  { key: 'workspace', label: '工作区与终端', icon: renderIcon(Folder) },
  { key: 'auth', label: '授权管理', icon: renderIcon(Key) },
  { key: 'snapshots', label: '快照管理', icon: renderIcon(History) },
  { key: 'logs', label: '运行日志', icon: renderIcon(FileText) }
]

const settingsMenuOptions: MenuOption[] = [
  { key: 'settings', label: '应用设置', icon: renderIcon(Settings) }
]

const mobileTabs = [
  { key: 'overview' as TabKey, label: '概览', icon: Dashboard },
  { key: 'customizations' as TabKey, label: '定制', icon: Puzzle },
  { key: 'conversations' as TabKey, label: '会话', icon: MessageCircle },
  { key: 'workspace' as TabKey, label: '终端', icon: Folder },
  { key: 'auth' as TabKey, label: '授权', icon: Key },
  { key: 'snapshots' as TabKey, label: '快照', icon: History },
  { key: 'logs' as TabKey, label: '日志', icon: FileText },
  { key: 'settings' as TabKey, label: '设置', icon: Settings }
]

const tab = computed<TabKey>({
  get: () => (views[appStore.currentTab as TabKey] ? (appStore.currentTab as TabKey) : 'overview'),
  set: (v) => appStore.setTab(v)
})

watch(
  [tab, isWorkbenchMode],
  ([newTab, workbench]) => {
    void trimSdk.setTitle(workbench ? 'Antigravity Workbench · 工作窗口' : (tabLabels[newTab] || 'Antigravity Harness'))
  },
  { immediate: true }
)

function handleMenuSelect(key: string) {
  tab.value = key as TabKey
}

const currentView = computed(() => views[tab.value])

function openDocs() {
  void trimSdk.openURL('https://antigravity.google/docs/getting-started', '_blank')
}

onMounted(() => {
  appStore.init()
  void trimSdk.initPlatformTheme((theme) => {
    if (!savedTheme) themeMode.value = theme
  })
})
</script>

<style scoped>
:deep(.app-content-scroll) {
  padding: 14px 14px calc(72px + env(safe-area-inset-bottom, 0px)) 14px;
}
.mobile-tabbar-footer {
  padding-bottom: calc(8px + env(safe-area-inset-bottom, 0px));
}
@media (min-width: 640px) {
  :deep(.app-content-scroll) {
    padding: 24px;
  }
}
</style>
