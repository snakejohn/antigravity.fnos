<template>
  <div class="h-screen w-screen flex flex-col overflow-hidden bg-[#f5f7fa] dark:bg-[#0f1117] text-slate-800 dark:text-slate-100 select-none">
    <!-- 顶部全局工作台顶栏 -->
    <header class="h-12 shrink-0 px-3 sm:px-4 bg-white dark:bg-[#161922] border-b border-slate-200/80 dark:border-white/[0.08] flex items-center justify-between gap-2 z-20">
      <div class="flex items-center gap-2.5 min-w-0">
        <!-- 移动端左侧栏抽屉按钮 -->
        <n-button quaternary circle size="small" class="lg:hidden shrink-0" @click="mobileSidebarOpen = true" title="展开会话与工作区">
          <template #icon><n-icon :size="18"><Menu2 /></n-icon></template>
        </n-button>
        <img :src="'./favicon.svg'" alt="Antigravity" class="w-6 h-6 rounded-lg shrink-0" />
        <span class="hidden sm:inline font-bold text-sm tracking-tight whitespace-nowrap">Antigravity</span>
        <n-tag size="tiny" round :bordered="false" type="info" class="font-mono hidden sm:inline-flex">Workbench</n-tag>
        <span class="hidden md:inline text-slate-300 dark:text-slate-700">/</span>
        <span v-if="activeDetail" class="hidden md:inline text-xs font-medium text-slate-600 dark:text-slate-300 truncate max-w-[260px]">
          {{ activeDetail.annotations.title || activeDetail.summary || activeDetail.id }}
        </span>
      </div>

      <!-- 顶栏右侧控制：模型选择、沙箱状态、运行态、辅助面板、主题、返回 Admin -->
      <div class="flex items-center gap-1.5 sm:gap-2 shrink-0">
        <n-select v-model:value="selectedModel" :options="MODEL_OPTIONS" filterable tag size="tiny"
          class="!w-[150px] sm:!w-[220px] font-mono" placeholder="选择或输入模型" @update:value="handleModelChange" />
        <n-tag size="small" round :bordered="false" class="hidden sm:inline-flex font-mono text-[11px] bg-slate-100 dark:bg-white/[0.08]">
          {{ statusData.sandbox_mode || 'workspace-write' }}
        </n-tag>
        <n-tag :type="isRunning ? 'success' : 'warning'" size="small" round :bordered="false" class="hidden sm:inline-flex text-[11px]">
          <span class="inline-block w-1.5 h-1.5 rounded-full mr-1" :class="isRunning ? 'bg-emerald-500' : 'bg-amber-500'" />
          {{ isRunning ? 'Harness 在线' : '已停止' }}
        </n-tag>
        <n-button size="tiny" secondary :type="showAuxPane ? 'primary' : 'default'" @click="showAuxPane = !showAuxPane" title="切换右侧辅助面板 (文件/终端/定制)">
          <template #icon><n-icon :size="15"><LayoutSidebarRight /></n-icon></template>
          <span class="hidden sm:inline">辅助面板</span>
        </n-button>
        <n-button quaternary circle size="tiny" @click="$emit('toggle-theme')" title="切换深浅色主题">
          <template #icon>
            <n-icon :size="16">
              <Sun v-if="themeMode === 'dark'" />
              <Moon v-else />
            </n-icon>
          </template>
        </n-button>
        <n-button size="tiny" secondary @click="goAdminConsole" title="打开管理控制台">
          <template #icon><n-icon :size="14"><Settings /></n-icon></template>
          <span class="hidden sm:inline">管理端</span>
        </n-button>
      </div>
    </header>

    <!-- 运行时未启动提醒横幅 -->
    <div v-if="statusLoaded && !isRunning"
      class="px-4 py-2 bg-amber-500/15 border-b border-amber-500/30 text-amber-700 dark:text-amber-300 text-xs flex items-center justify-between gap-2 shrink-0">
      <span>Antigravity Harness 内部运行时当前处于停止状态，点击右侧按钮立即拉起服务。</span>
      <n-button size="tiny" type="warning" :loading="startingHarness" @click="startHarnessNow">立即启动</n-button>
    </div>

    <!-- 主体三栏工作区 -->
    <div class="flex-1 flex min-h-0 overflow-hidden">
      <!-- 1. 左侧边栏 (Desktop 固定 268px) -->
      <aside class="hidden lg:flex w-[268px] shrink-0 flex-col bg-white dark:bg-[#141720] border-r border-slate-200/80 dark:border-white/[0.08]">
        <!-- 工作区切换器 -->
        <div class="p-3 border-b border-slate-100 dark:border-white/[0.06] space-y-2">
          <div class="flex items-center justify-between text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
            <span>当前工作区</span>
            <span class="font-mono">{{ workspaces.length }} 个</span>
          </div>
          <n-select v-model:value="activeWorkspacePath" :options="workspaceOptions" size="small" filterable
            placeholder="选择工作区目录" @update:value="handleWorkspaceChange" />
          <n-button type="primary" block size="small" @click="startFreshDraft" class="rounded-xl font-medium">
            <template #icon><n-icon><Plus /></n-icon></template>
            新建 Agent 会话
          </n-button>
        </div>

        <!-- 会话搜索 -->
        <div class="px-3 pt-2.5 pb-1.5">
          <n-input v-model:value="convSearch" size="small" placeholder="搜索会话标题或标签..." clearable>
            <template #prefix><n-icon :size="14" class="text-slate-400"><Search /></n-icon></template>
          </n-input>
        </div>

        <!-- 会话列表 -->
        <div class="flex-1 overflow-y-auto px-2 pb-2 space-y-1">
          <div v-for="c in filteredConversations" :key="c.id" @click="selectConversation(c.id)"
            class="group px-2.5 py-2 rounded-xl cursor-pointer transition-all border"
            :class="activeConvId === c.id
              ? 'bg-blue-50/90 dark:bg-blue-500/15 border-blue-200/90 dark:border-blue-500/30'
              : 'bg-transparent border-transparent hover:bg-slate-100/80 dark:hover:bg-white/[0.04]'">
            <div class="flex items-center justify-between gap-1.5">
              <div class="flex items-center gap-1.5 min-w-0 flex-1">
                <span class="w-1.5 h-1.5 rounded-full shrink-0"
                  :class="c.runState === 'RUN_STATE_RUNNING' ? 'bg-amber-400 animate-pulse' : 'bg-emerald-500/70'" />
                <span class="text-xs font-semibold truncate text-slate-800 dark:text-slate-100">
                  {{ c.annotations.title || c.summary || c.id }}
                </span>
              </div>
              <div class="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0" @click.stop>
                <n-button size="tiny" quaternary class="!px-1" @click="toggleStar(c)">
                  <n-icon :size="13" :class="c.annotations.starred ? 'text-amber-500' : 'text-slate-400'"><Star /></n-icon>
                </n-button>
                <n-button size="tiny" quaternary class="!px-1" @click="togglePin(c)">
                  <n-icon :size="13" :class="c.annotations.pinned ? 'text-fnos-blue' : 'text-slate-400'"><Pin /></n-icon>
                </n-button>
                <n-popconfirm @positive-click="deleteConversation(c.id)">
                  <template #trigger>
                    <n-button size="tiny" quaternary type="error" class="!px-1"><n-icon :size="13"><Trash /></n-icon></n-button>
                  </template>
                  永久删除此会话？
                </n-popconfirm>
              </div>
            </div>
            <div class="flex items-center justify-between mt-1 text-[10.5px] font-mono text-slate-400">
              <span class="truncate">{{ c.model }}</span>
              <div class="flex items-center gap-1 shrink-0">
                <n-icon v-if="c.annotations.pinned" :size="11" class="text-fnos-blue"><Pin /></n-icon>
                <n-icon v-if="c.annotations.starred" :size="11" class="text-amber-500"><Star /></n-icon>
                <span>{{ c.turnCount }} 轮</span>
              </div>
            </div>
          </div>
          <div v-if="!filteredConversations.length" class="py-10 text-center text-xs text-slate-400">
            暂无匹配会话
          </div>
        </div>

        <!-- 底部快捷状态与返回管理台 -->
        <div class="p-3 border-t border-slate-100 dark:border-white/[0.06] flex items-center justify-between text-[11px] text-slate-400">
          <div class="flex items-center gap-1.5 font-mono">
            <span class="w-2 h-2 rounded-full" :class="wsConnected ? 'bg-emerald-500' : 'bg-rose-500'" />
            <span>{{ wsConnected ? 'WS 实时同步' : '重连中…' }}</span>
          </div>
          <button class="text-fnos-blue dark:text-blue-400 hover:underline font-medium" @click="goAdminConsole">
            管理控制台 ↗
          </button>
        </div>
      </aside>

      <!-- 2. 中间主对话与任务画布 (Agent Chat & Canvas) -->
      <main class="flex-1 flex flex-col min-w-0 min-h-0 bg-[#f8fafc] dark:bg-[#0f1117]">
        <!-- 消息流滚动区 -->
        <div ref="chatScrollRef" class="flex-1 overflow-y-auto px-3 sm:px-6 py-4 sm:py-6">
          <div class="max-w-3xl mx-auto flex flex-col gap-3.5">
            <!-- 空状态 / 新会话欢迎画布 -->
            <div v-if="!activeDetail || !activeDetail.transcript.length" class="py-8 sm:py-14 flex flex-col items-center text-center gap-5">
              <div class="w-14 h-14 rounded-2xl bg-gradient-to-br from-blue-500 via-indigo-500 to-cyan-500 p-3 shadow-lg shadow-blue-500/20 flex items-center justify-center">
                <img :src="'./favicon.svg'" alt="Antigravity" class="w-full h-full" />
              </div>
              <div class="space-y-1.5 max-w-lg">
                <h1 class="text-xl sm:text-2xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">
                  Antigravity Agent 工作窗口
                </h1>
                <p class="text-xs sm:text-sm text-slate-500 dark:text-slate-400 leading-relaxed">
                  当前工作区：<span class="font-mono text-fnos-blue dark:text-blue-400">{{ activeWorkspacePath || '默认数据区' }}</span><br />
                  直接发送任务指令或使用斜杠命令，右侧辅助面板可实时浏览代码、运行终端与管理 Skills / MCP。
                </p>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 w-full max-w-xl text-left">
                <div v-for="card in starterPrompts" :key="card.title" @click="triggerQuickPrompt(card.prompt)"
                  class="p-3.5 rounded-2xl bg-white dark:bg-[#161922] border border-slate-200/80 dark:border-white/[0.08] hover:border-fnos-blue/60 dark:hover:border-blue-500/50 cursor-pointer transition-all shadow-xs hover:shadow-md">
                  <div class="text-xs font-bold text-slate-800 dark:text-slate-100 mb-1 flex items-center gap-1.5">
                    <span class="text-fnos-blue font-mono">{{ card.badge }}</span>
                    <span>{{ card.title }}</span>
                  </div>
                  <div class="text-[11.5px] text-slate-500 dark:text-slate-400">{{ card.desc }}</div>
                </div>
              </div>
            </div>

            <!-- 消息时间线 -->
            <template v-else>
              <template v-for="entry in activeDetail.transcript" :key="entry.id">
                <!-- User Prompt -->
                <div v-if="entry.role === 'user'" class="flex justify-end">
                  <div class="max-w-[88%] sm:max-w-[80%] rounded-2xl rounded-tr-sm px-4 py-3 bg-fnos-blue text-white shadow-sm select-text">
                    <div class="text-xs sm:text-[13.5px] leading-relaxed whitespace-pre-wrap break-words">{{ entry.content }}</div>
                    <div class="text-[10px] opacity-75 text-right mt-1 font-mono">{{ formatTime(entry.timestamp) }}</div>
                  </div>
                </div>

                <!-- Agent Thinking (可折叠思考卡片) -->
                <div v-else-if="entry.role === 'thought'"
                  class="rounded-xl bg-amber-50/70 dark:bg-amber-950/20 border border-amber-200/70 dark:border-amber-800/40 overflow-hidden">
                  <button @click="toggleCollapse(entry.id)"
                    class="w-full px-3.5 py-2 flex items-center justify-between text-xs font-medium text-amber-800 dark:text-amber-300 hover:bg-amber-500/5 transition-colors">
                    <div class="flex items-center gap-1.5">
                      <n-icon :size="15"><Bulb /></n-icon>
                      <span>Thinking · 思考过程</span>
                    </div>
                    <span class="text-[11px] opacity-75">{{ collapsedMap[entry.id] ? '展开' : '收起' }}</span>
                  </button>
                  <div v-show="!collapsedMap[entry.id]"
                    class="px-3.5 pb-3 pt-1 text-xs text-amber-900/90 dark:text-amber-200/90 leading-relaxed whitespace-pre-wrap break-words select-text border-t border-amber-200/40 dark:border-amber-800/30">
                    {{ entry.content }}
                  </div>
                </div>

                <!-- Tool Call / Tool Result 轨迹卡片 -->
                <div v-else-if="entry.role === 'tool_call' || entry.role === 'tool_result'"
                  class="rounded-xl bg-slate-100/90 dark:bg-[#161922] border border-slate-200/80 dark:border-white/[0.08] overflow-hidden">
                  <button @click="toggleCollapse(entry.id)"
                    class="w-full px-3.5 py-2 flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-300 hover:bg-black/5 dark:hover:bg-white/[0.03]">
                    <div class="flex items-center gap-2 min-w-0">
                      <n-tag size="tiny" :bordered="false" :type="entry.role === 'tool_call' ? 'info' : 'success'">
                        {{ entry.role === 'tool_call' ? 'TOOL CALL' : 'RESULT' }}
                      </n-tag>
                      <span class="font-semibold truncate">{{ entry.tool_name || 'tool' }}</span>
                    </div>
                    <span class="text-[10.5px] opacity-60 shrink-0">{{ collapsedMap[entry.id] ? '展开详情' : '收起' }}</span>
                  </button>
                  <div v-show="!collapsedMap[entry.id]"
                    class="px-3.5 pb-2.5 pt-1.5 text-[11.5px] font-mono text-slate-600 dark:text-slate-300 space-y-1.5 border-t border-slate-200/60 dark:border-white/[0.06] select-text">
                    <div v-if="entry.tool_args" class="p-2 rounded-lg bg-slate-200/60 dark:bg-black/30 break-all">
                      {{ entry.tool_args }}
                    </div>
                    <div class="whitespace-pre-wrap break-words">{{ entry.content }}</div>
                  </div>
                </div>

                <!-- Assistant Markdown / Code 回复 -->
                <div v-else class="rounded-2xl bg-white dark:bg-[#161922] border border-slate-200/80 dark:border-white/[0.08] px-4 py-3.5 shadow-xs select-text">
                  <div class="flex items-center justify-between gap-2 pb-2 mb-2 border-b border-slate-100 dark:border-white/[0.06]">
                    <div class="flex items-center gap-2">
                      <img :src="'./favicon.svg'" alt="AGY" class="w-4 h-4 rounded" />
                      <span class="text-xs font-bold text-slate-800 dark:text-slate-100">Antigravity</span>
                      <n-tag v-if="entry.model" size="tiny" :bordered="false" class="font-mono text-[10px]">{{ entry.model }}</n-tag>
                    </div>
                    <span class="text-[10px] font-mono text-slate-400">{{ formatTime(entry.timestamp) }}</span>
                  </div>
                  <div class="agy-markdown text-xs sm:text-[13.5px] leading-relaxed text-slate-700 dark:text-slate-200 break-words"
                    v-html="renderMarkdown(entry.content)" />
                </div>
              </template>

              <!-- 正在运行思考动效 -->
              <div v-if="activeDetail.runState === 'RUN_STATE_RUNNING'"
                class="flex items-center gap-2.5 px-4 py-3 rounded-xl bg-blue-50/70 dark:bg-blue-950/25 border border-blue-200/60 dark:border-blue-800/40 text-xs text-fnos-blue dark:text-blue-400">
                <n-spin :size="14" />
                <span>Antigravity Agent 正在推理与执行工具…</span>
              </div>
            </template>
          </div>
        </div>

        <!-- 底部斜杠命令 + 多行输入区 -->
        <div class="shrink-0 px-3 sm:px-6 pb-3 pt-2 bg-white/95 dark:bg-[#141720]/95 backdrop-blur-md border-t border-slate-200/80 dark:border-white/[0.08]">
          <div class="max-w-3xl mx-auto space-y-2">
            <!-- 快捷斜杠命令栏 -->
            <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar">
              <span class="text-[11px] text-slate-400 shrink-0">斜杠指令:</span>
              <n-button v-for="sc in slashCommands" :key="sc.cmd" size="tiny" secondary round @click="handleSlashClick(sc)"
                class="!text-[11px] font-mono shrink-0">
                {{ sc.cmd }}
              </n-button>
            </div>
            <!-- 输入框与发送按钮 -->
            <div class="flex items-end gap-2">
              <n-input v-model:value="promptInput" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }"
                placeholder="向 Antigravity Agent 下发任务指令 (Enter 发送，Shift+Enter 换行)..."
                class="flex-1 text-xs sm:text-sm" @keydown="handleComposerKeydown" />
              <n-button type="primary" :loading="sending" :disabled="!promptInput.trim()" @click="submitPrompt"
                class="!h-11 px-4 sm:px-5 rounded-xl font-medium shrink-0">
                <template #icon><n-icon><Send /></n-icon></template>
                发送
              </n-button>
            </div>
          </div>
        </div>
      </main>

      <!-- 3. 右侧辅助面板 (Auxiliary Pane: Files / Terminal / Skills & MCP) -->
      <aside v-show="showAuxPane"
        class="w-full sm:w-[390px] xl:w-[430px] shrink-0 flex flex-col bg-white dark:bg-[#141720] border-l border-slate-200/80 dark:border-white/[0.08] max-sm:fixed max-sm:inset-0 max-sm:top-12 max-sm:z-30">
        <!-- 辅助面板 Tab 切换栏 -->
        <div class="px-3 pt-2.5 pb-2 border-b border-slate-100 dark:border-white/[0.06] flex items-center justify-between gap-2">
          <n-radio-group v-model:value="auxTab" size="small" class="flex-1 flex">
            <n-radio-button value="files" class="flex-1 text-center">文件</n-radio-button>
            <n-radio-button value="terminal" class="flex-1 text-center">终端</n-radio-button>
            <n-radio-button value="custom" class="flex-1 text-center">Skills &amp; MCP</n-radio-button>
          </n-radio-group>
          <n-button quaternary circle size="tiny" class="sm:hidden" @click="showAuxPane = false">
            <template #icon><n-icon><X /></n-icon></template>
          </n-button>
        </div>

        <!-- Tab 1: 文件树与代码查看/编辑器 -->
        <div v-show="auxTab === 'files'" class="flex-1 flex flex-col min-h-0">
          <!-- 当前目录操作栏 -->
          <div class="px-3 py-2 border-b border-slate-100 dark:border-white/[0.06] flex items-center justify-between gap-2 bg-slate-50/70 dark:bg-white/[0.02]">
            <div class="font-mono text-[11px] text-slate-500 dark:text-slate-400 truncate flex-1" :title="fileBrowserCwd">
              {{ fileBrowserCwd || activeWorkspacePath }}
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <n-button size="tiny" secondary :disabled="!fileBrowserParent" @click="browseDir(fileBrowserParent)" title="返回上级目录">
                上级
              </n-button>
              <n-button size="tiny" secondary :loading="loadingFiles" @click="browseDir(fileBrowserCwd)">
                <template #icon><n-icon :size="13"><Refresh /></n-icon></template>
              </n-button>
            </div>
          </div>

          <!-- 打开文件编辑器模式 vs 目录树列表模式 -->
          <div v-if="openedFile" class="flex-1 flex flex-col min-h-0">
            <div class="px-3 py-2 border-b border-slate-100 dark:border-white/[0.06] flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5 min-w-0">
                <n-button size="tiny" quaternary @click="openedFile = null">← 返回</n-button>
                <span class="font-mono text-xs font-bold truncate">{{ openedFile.name }}</span>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <n-button size="tiny" secondary @click="insertIntoChat(`请检查并分析文件 \`${openedFile.path}\`：`)">
                  引用
                </n-button>
                <n-button v-if="!openedFile.is_binary" size="tiny" type="primary" :loading="savingFile" @click="saveOpenedFile">
                  保存
                </n-button>
              </div>
            </div>
            <div class="flex-1 p-2.5 min-h-0 flex flex-col">
              <n-input v-model:value="openedFileContent" type="textarea" :disabled="openedFile.is_binary"
                class="flex-1 font-mono text-xs h-full"
                :input-props="{ style: 'height: 100%; min-height: 360px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;' }" />
              <div class="text-[10.5px] font-mono text-slate-400 pt-1.5 flex justify-between">
                <span class="truncate">{{ openedFile.path }}</span>
                <span>{{ formatBytes(openedFile.size) }}</span>
              </div>
            </div>
          </div>

          <div v-else class="flex-1 overflow-y-auto p-2 space-y-0.5">
            <div v-if="fileError" class="p-3 text-xs text-rose-500">{{ fileError }}</div>
            <div v-for="entry in fileEntries" :key="entry.path"
              @click="entry.is_dir ? browseDir(entry.path) : openFile(entry.path)"
              class="px-2.5 py-2 rounded-lg flex items-center justify-between gap-2 cursor-pointer hover:bg-slate-100 dark:hover:bg-white/[0.05] transition-colors">
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <n-icon :size="16" :class="entry.is_dir ? 'text-amber-500' : 'text-slate-400'">
                  <Folder v-if="entry.is_dir" />
                  <FileCode v-else />
                </n-icon>
                <span class="text-xs font-mono truncate text-slate-700 dark:text-slate-200">{{ entry.name }}</span>
              </div>
              <span class="text-[10.5px] font-mono text-slate-400 shrink-0">
                {{ entry.is_dir ? 'DIR' : formatBytes(entry.size) }}
              </span>
            </div>
            <n-empty v-if="!loadingFiles && !fileEntries.length && !fileError" description="当前目录为空" class="py-12" />
          </div>
        </div>

        <!-- Tab 2: 内置交互式终端 -->
        <div v-show="auxTab === 'terminal'" class="flex-1 flex flex-col min-h-0 p-3 gap-2.5">
          <div class="flex items-center justify-between text-[11px] font-mono text-slate-400">
            <span class="truncate">CWD: {{ fileBrowserCwd || activeWorkspacePath }}</span>
            <n-button size="tiny" quaternary @click="termHistory = []">清屏</n-button>
          </div>
          <div class="flex flex-wrap gap-1">
            <n-button v-for="qc in quickTermCmds" :key="qc" size="tiny" secondary class="font-mono !text-[10.5px]"
              @click="runTermCmd(qc)">
              {{ qc }}
            </n-button>
          </div>
          <div class="flex-1 rounded-xl bg-[#0b0f19] text-slate-200 p-3 font-mono text-[11.5px] overflow-y-auto select-text min-h-0">
            <div v-if="!termHistory.length" class="text-slate-500">
              $ Antigravity Workbench Terminal ready.
            </div>
            <div v-for="(item, idx) in termHistory" :key="idx" class="mb-3 last:mb-0">
              <div class="flex items-center justify-between text-emerald-400 mb-0.5">
                <span class="truncate">$ {{ item.command }}</span>
                <span class="text-[10px] text-slate-400 shrink-0 ml-2">exit {{ item.exit_code }} · {{ item.duration_ms }}ms</span>
              </div>
              <pre v-if="item.stdout" class="whitespace-pre-wrap break-words text-slate-100 m-0">{{ item.stdout }}</pre>
              <pre v-if="item.stderr" class="whitespace-pre-wrap break-words text-rose-400 m-0 mt-1">{{ item.stderr }}</pre>
            </div>
          </div>
          <div class="flex gap-1.5 shrink-0">
            <n-input v-model:value="termInput" size="small" placeholder="输入 Shell / agy 命令..." class="font-mono text-xs"
              :disabled="termRunning" @keydown.enter="runTermCmd()" />
            <n-button size="small" type="primary" :loading="termRunning" :disabled="!termInput.trim()" @click="runTermCmd()">
              执行
            </n-button>
          </div>
        </div>

        <!-- Tab 3: Skills & MCP 定制面板 -->
        <div v-show="auxTab === 'custom'" class="flex-1 overflow-y-auto p-3 space-y-4">
          <!-- Rules 预算 -->
          <div class="p-3 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100 dark:border-white/[0.06] space-y-1.5">
            <div class="flex items-center justify-between text-xs">
              <span class="font-semibold">全局 Rules 预算</span>
              <span class="font-mono text-fnos-blue">{{ ruleTokens }} / 20,000 tok</span>
            </div>
            <n-progress type="line" :percentage="Math.min(100, Math.round((ruleTokens / 20000) * 100))"
              :show-indicator="false" :height="5" />
          </div>

          <!-- Skills 列表 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-slate-700 dark:text-slate-200">已激活 Skills ({{ skills.length }})</span>
              <n-button size="tiny" text type="primary" @click="goAdminConsole">管理定制 ↗</n-button>
            </div>
            <div v-for="s in skills" :key="s.path"
              class="p-2.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100 dark:border-white/[0.06] space-y-1">
              <div class="flex items-center justify-between gap-2">
                <span class="font-mono text-xs font-bold truncate">{{ s.name }}</span>
                <n-button size="tiny" secondary type="primary" @click="insertIntoChat(`请使用技能 \`${s.name}\` 来完成以下任务：`)">
                  + 插入对话
                </n-button>
              </div>
              <div class="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-2">{{ s.description }}</div>
            </div>
          </div>

          <!-- MCP Servers 列表 -->
          <div class="space-y-2">
            <div class="text-xs font-bold text-slate-700 dark:text-slate-200">MCP Servers ({{ mcps.length }})</div>
            <div v-for="m in mcps" :key="m.id"
              class="p-2.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-100 dark:border-white/[0.06] space-y-1">
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-1.5 min-w-0">
                  <span class="font-mono text-xs font-bold truncate">{{ m.id }}</span>
                  <n-tag size="tiny" :bordered="false" :type="m.transport === 'http' ? 'info' : 'success'">{{ m.transport }}</n-tag>
                </div>
                <n-button size="tiny" secondary :loading="testingMcpId === m.id" @click="testMcp(m.id)">测试</n-button>
              </div>
              <div class="text-[10.5px] font-mono text-slate-400 truncate">
                {{ m.transport === 'http' ? m.url : `${m.command} ${(m.args || []).join(' ')}` }}
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>

    <!-- 移动端左侧抽屉 -->
    <n-drawer v-model:show="mobileSidebarOpen" placement="left" :width="280">
      <n-drawer-content title="工作区与会话" :native-scrollbar="false">
        <div class="space-y-3">
          <n-select v-model:value="activeWorkspacePath" :options="workspaceOptions" size="small" @update:value="handleWorkspaceChange" />
          <n-button type="primary" block size="small" @click="startFreshDraft(); mobileSidebarOpen = false">
            + 新建 Agent 会话
          </n-button>
          <div class="space-y-1.5">
            <div v-for="c in filteredConversations" :key="c.id"
              @click="selectConversation(c.id); mobileSidebarOpen = false"
              class="p-2.5 rounded-xl border cursor-pointer"
              :class="activeConvId === c.id ? 'bg-blue-500/10 border-blue-500/40' : 'border-slate-200/70 dark:border-white/[0.08]'">
              <div class="text-xs font-semibold truncate">{{ c.annotations.title || c.summary || c.id }}</div>
              <div class="text-[10.5px] font-mono text-slate-400 mt-0.5">{{ c.model }} · {{ c.turnCount }} 轮</div>
            </div>
          </div>
        </div>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NButton, NTag, NIcon, NSelect, NInput, NSpin, NEmpty, NProgress,
  NRadioGroup, NRadioButton, NPopconfirm, NDrawer, NDrawerContent, useMessage
} from 'naive-ui'
import {
  Menu2, Plus, Search, Star, Pin, Trash, Send, Sun, Moon, Settings,
  Folder, FileCode, Refresh, X, Bulb, LayoutSidebarRight
} from '@vicons/tabler'
import { useAppStore } from '../stores/app'
import { useSystemStore } from '../stores/system'
import { conversationsApi, workspaceApi, customizationsApi } from '../api'
import { renderMarkdown } from '../utils/markdown'
import { DEFAULT_MODEL, MODEL_OPTIONS } from '../constants/models'
import type {
  ConversationMeta, ConversationDetail, WorkspaceFileEntry, WorkspaceFileContent,
  TerminalExecResult, SkillItem, MCPServerView
} from '../types/api'

defineProps<{ themeMode: 'light' | 'dark' }>()
defineEmits<{ (e: 'toggle-theme'): void }>()

const appStore = useAppStore()
const systemStore = useSystemStore()
const message = useMessage()

const { statusData, statusLoaded, wsConnected, isRunning } = storeToRefs(systemStore)
const workspaces = computed(() => appStore.workspaces)
const conversations = computed(() => appStore.conversations)

const mobileSidebarOpen = ref(false)
const showAuxPane = ref(typeof window !== 'undefined' ? window.innerWidth >= 1024 : true)
const auxTab = ref<'files' | 'terminal' | 'custom'>('files')
const startingHarness = ref(false)

// Left sidebar state
const activeWorkspacePath = ref('')
const convSearch = ref('')
const activeConvId = ref('')
const activeDetail = ref<ConversationDetail | null>(null)
const selectedModel = ref(DEFAULT_MODEL)

const workspaceOptions = computed(() =>
  workspaces.value.map((w) => ({ label: `${w.title} (${w.path})`, value: w.path }))
)

const filteredConversations = computed(() => {
  const kw = convSearch.value.trim().toLowerCase()
  if (!kw) return conversations.value
  return conversations.value.filter(
    (c) =>
      (c.annotations.title || '').toLowerCase().includes(kw) ||
      (c.summary || '').toLowerCase().includes(kw) ||
      (c.annotations.tags || []).some((t) => t.toLowerCase().includes(kw))
  )
})

// Chat composer & transcript state
const chatScrollRef = ref<HTMLElement | null>(null)
const promptInput = ref('')
const sending = ref(false)
const collapsedMap = ref<Record<string, boolean>>({})

const starterPrompts = [
  { badge: '/doctor', title: '运行全面健康巡检', desc: '检查 Harness 运行时、Node/Python、MCP 与存储卷状态', prompt: '/doctor' },
  { badge: '/skills', title: '查看已激活定制项', desc: '列出当前挂载的所有 Antigravity Skills 与 MCP 服务', prompt: '/skills' },
  { badge: '/plan', title: '分析当前工作区', desc: '检查所选 NAS 工作区目录结构并给出优化建议', prompt: '/plan 请检查当前工作区状态并生成一份维护清单' },
  { badge: '/goal', title: '创建自动化任务目标', desc: '以目标驱动模式规划多步骤 Agent 执行方案', prompt: '/goal 梳理当前 fnOS 容器与存储健康并输出检查报告' }
]

const slashCommands = [
  { cmd: '/goal', immediate: false },
  { cmd: '/plan', immediate: false },
  { cmd: '/doctor', immediate: true },
  { cmd: '/skills', immediate: true },
  { cmd: '/compact', immediate: true }
]

// Right pane: File explorer & editor
const fileBrowserCwd = ref('')
const fileBrowserParent = ref('')
const fileEntries = ref<WorkspaceFileEntry[]>([])
const loadingFiles = ref(false)
const fileError = ref('')
const openedFile = ref<WorkspaceFileContent | null>(null)
const openedFileContent = ref('')
const savingFile = ref(false)

// Right pane: Terminal
const termInput = ref('')
const termRunning = ref(false)
const termHistory = ref<TerminalExecResult[]>([])
const quickTermCmds = ['agy doctor', 'ls -la', 'df -h', 'free -h', 'uname -a']

// Right pane: Customizations
const skills = ref<SkillItem[]>([])
const mcps = ref<MCPServerView[]>([])
const ruleTokens = ref(0)
const testingMcpId = ref('')

function toggleCollapse(id: string) {
  collapsedMap.value[id] = !collapsedMap.value[id]
}

function scrollChatBottom() {
  nextTick(() => {
    if (chatScrollRef.value) {
      chatScrollRef.value.scrollTop = chatScrollRef.value.scrollHeight
    }
  })
}

async function selectConversation(id: string) {
  activeConvId.value = id
  const res = await conversationsApi.getDetail(id)
  if (res.success && res.data) {
    activeDetail.value = res.data
    if (res.data.model) selectedModel.value = res.data.model
    scrollChatBottom()
  }
}

function startFreshDraft() {
  activeConvId.value = ''
  activeDetail.value = null
  promptInput.value = ''
  selectedModel.value = statusData.value.default_model || DEFAULT_MODEL
}

async function handleModelChange(newModel: string) {
  selectedModel.value = newModel
  if (!activeConvId.value) return
  const res = await conversationsApi.update(activeConvId.value, { model: newModel })
  if (res.success && res.data) {
    if (activeDetail.value) {
      activeDetail.value.model = res.data.model
    }
    message.success(`当前会话模型已切换为 ${res.data.model}`)
  } else {
    message.error(res.message || '切换模型失败')
  }
}

async function submitPrompt() {
  const text = promptInput.value.trim()
  if (!text || sending.value) return
  sending.value = true
  promptInput.value = ''
  try {
    if (!activeConvId.value) {
      const res = await conversationsApi.create({
        model: selectedModel.value,
        workspace: activeWorkspacePath.value || workspaces.value[0]?.path,
        tags: ['workbench'],
        prompt: text
      })
      if (res.success && res.data) {
        await appStore.refreshConversations()
        await selectConversation(res.data.id)
      } else {
        message.error(res.message || '创建会话失败')
      }
    } else {
      const res = await conversationsApi.sendMessage(activeConvId.value, text, selectedModel.value)
      if (res.success) {
        await selectConversation(activeConvId.value)
      } else {
        message.error(res.message || '发送失败')
      }
    }
  } finally {
    sending.value = false
  }
}

function handleComposerKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    void submitPrompt()
  }
}

function handleSlashClick(sc: { cmd: string; immediate: boolean }) {
  if (sc.immediate) {
    promptInput.value = sc.cmd
    void submitPrompt()
  } else {
    promptInput.value = `${sc.cmd} ${promptInput.value}`.trimStart()
  }
}

function triggerQuickPrompt(text: string) {
  promptInput.value = text
  void submitPrompt()
}

function insertIntoChat(snippet: string) {
  promptInput.value = promptInput.value ? `${promptInput.value}\n${snippet}` : snippet
  message.info('已插入到对话输入框')
}

async function toggleStar(c: ConversationMeta) {
  await conversationsApi.update(c.id, { starred: !c.annotations.starred })
  await appStore.refreshConversations()
}

async function togglePin(c: ConversationMeta) {
  await conversationsApi.update(c.id, { pinned: !c.annotations.pinned })
  await appStore.refreshConversations()
}

async function deleteConversation(id: string) {
  const res = await conversationsApi.remove(id)
  if (res.success) {
    message.success('会话已删除')
    if (activeConvId.value === id) {
      activeConvId.value = ''
      activeDetail.value = null
    }
    await appStore.refreshConversations()
  }
}

// File browser
async function browseDir(path?: string) {
  loadingFiles.value = true
  fileError.value = ''
  openedFile.value = null
  const res = await workspaceApi.listFiles(path)
  loadingFiles.value = false
  if (res.success && res.data) {
    fileBrowserCwd.value = res.data.cwd
    fileBrowserParent.value = res.data.parent
    fileEntries.value = res.data.entries || []
  } else {
    fileError.value = res.message || '读取目录失败'
  }
}

async function openFile(path: string) {
  const res = await workspaceApi.readFile(path)
  if (res.success && res.data) {
    openedFile.value = res.data
    openedFileContent.value = res.data.content
  } else {
    message.error(res.message || '打开文件失败')
  }
}

async function saveOpenedFile() {
  if (!openedFile.value) return
  savingFile.value = true
  const res = await workspaceApi.writeFile(openedFile.value.path, openedFileContent.value)
  savingFile.value = false
  if (res.success && res.data) {
    openedFile.value = res.data
    message.success(`已保存 ${res.data.name}`)
  } else {
    message.error(res.message || '保存失败')
  }
}

function handleWorkspaceChange(path: string) {
  activeWorkspacePath.value = path
  void browseDir(path)
}

// Terminal
async function runTermCmd(preset?: string) {
  const cmd = (preset ?? termInput.value).trim()
  if (!cmd || termRunning.value) return
  if (!preset) termInput.value = ''
  termRunning.value = true
  const res = await workspaceApi.execTerminal(cmd, fileBrowserCwd.value || activeWorkspacePath.value || undefined)
  termRunning.value = false
  if (res.success && res.data) {
    termHistory.value.push(res.data)
    if (termHistory.value.length > 30) termHistory.value.shift()
  } else {
    message.error(res.message || '命令执行失败')
  }
}

// Customizations panel
async function loadCustomizations() {
  const [sk, mp, rl] = await Promise.all([
    customizationsApi.listSkills(),
    customizationsApi.getMCP(),
    customizationsApi.listRules()
  ])
  if (sk.success && sk.data) skills.value = sk.data.items || []
  if (mp.success && mp.data) mcps.value = mp.data.items || []
  if (rl.success && rl.data) ruleTokens.value = rl.data.total_tokens || 0
}

async function testMcp(id: string) {
  testingMcpId.value = id
  const res = await customizationsApi.testMCP(id)
  testingMcpId.value = ''
  if (res.success && res.data) {
    if (res.data.ok) message.success(`${id}: ${res.data.message}`)
    else message.warning(`${id}: ${res.data.message}`)
  }
}

async function startHarnessNow() {
  startingHarness.value = true
  const res = await systemStore.sendAction('start')
  startingHarness.value = false
  if (res.success) message.success('Antigravity Harness 已启动')
  else message.error(res.message)
}

function goAdminConsole() {
  const target = window.location.pathname.startsWith('/app/antigravity-harness')
    ? '/app/antigravity-harness/'
    : '/'
  window.location.href = target
}

function formatTime(ts: string) {
  if (!ts) return ''
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleTimeString()
}

function formatBytes(b: number) {
  if (!b) return '0 B'
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
  return `${(b / (1024 * 1024)).toFixed(1)} MB`
}

// Sync active conversation detail on WebSocket updates
watch(
  () => appStore.conversations,
  () => {
    if (activeConvId.value) {
      void selectConversation(activeConvId.value)
    }
  }
)

onMounted(async () => {
  await Promise.all([
    appStore.refreshWorkspaces(),
    appStore.refreshConversations(),
    loadCustomizations()
  ])
  if (workspaces.value.length) {
    activeWorkspacePath.value = workspaces.value[0].path
    await browseDir(activeWorkspacePath.value)
  } else {
    await browseDir()
  }
  if (conversations.value.length) {
    await selectConversation(conversations.value[0].id)
  }
})
</script>

<style scoped>
:deep(.agy-inline-code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.9em;
  padding: 1px 5px;
  border-radius: 5px;
  background: rgba(148, 163, 184, 0.16);
}
:deep(.agy-code-wrap) {
  margin: 8px 0;
  border-radius: 10px;
  overflow: hidden;
  background: #0f172a;
  color: #e2e8f0;
}
:deep(.agy-code-lang) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  text-transform: uppercase;
  padding: 4px 10px;
  color: #94a3b8;
  background: rgba(255, 255, 255, 0.04);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
:deep(.agy-code-pre) {
  margin: 0;
  padding: 10px 12px;
  overflow-x: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.55;
}
:deep(.hljs-keyword) { color: #c084fc; }
:deep(.hljs-string) { color: #34d399; }
:deep(.hljs-number) { color: #38bdf8; }
:deep(.hljs-comment) { color: #64748b; }
:deep(.hljs-title), :deep(.hljs-attr) { color: #60a5fa; }
</style>
