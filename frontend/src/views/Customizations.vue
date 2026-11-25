<template>
  <div class="w-full flex-1 flex flex-col gap-4 sm:gap-5">
    <!-- 吸顶页头 -->
    <div
      class="sticky -top-[14px] sm:-top-6 z-20 -mt-3.5 sm:-mt-4 pt-5 sm:pt-7 pb-2.5 bg-[#f5f7fa]/90 dark:bg-[#12141a]/90 backdrop-blur-md flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-baseline gap-2.5">
        <h1 class="text-xl font-bold text-slate-800 dark:text-slate-100 tracking-tight">定制中心</h1>
        <span class="text-xs text-slate-400">Antigravity 五大定制体系 (Skills · MCP · Plugins · Rules · Hooks)</span>
      </div>
      <n-button size="small" secondary :loading="loading" @click="reloadActive">
        <template #icon><n-icon><Refresh /></n-icon></template>
        刷新
      </n-button>
    </div>

    <!-- 分段标签页 -->
    <n-tabs v-model:value="activeSubTab" type="segment" animated class="w-full">
      <!-- 1. SKILLS -->
      <n-tab-pane name="skills" :tab="`Skills 技能 (${skills.length})`">
        <div class="flex flex-col gap-4 mt-2">
          <div class="flex items-center justify-between gap-3">
            <n-input v-model:value="skillSearch" placeholder="搜索技能名称或描述..." size="small" clearable class="max-w-sm">
              <template #prefix><n-icon><Search /></n-icon></template>
            </n-input>
            <n-button type="primary" size="small" @click="openNewSkillModal">
              <template #icon><n-icon><Plus /></n-icon></template>
              新建技能 (SKILL.md)
            </n-button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
            <n-card v-for="s in filteredSkills" :key="s.path" hoverable :bordered="false" class="rounded-2xl shadow-sm">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1 space-y-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-mono font-bold text-sm text-slate-800 dark:text-slate-100">{{ s.name }}</span>
                    <n-tag size="tiny" round :bordered="false" :type="s.scope === 'global' ? 'info' : s.scope === 'workspace' ? 'success' : 'warning'">
                      {{ s.scope }}
                    </n-tag>
                    <span class="text-[11px] font-mono text-slate-400">{{ (s.size_bytes / 1024).toFixed(1) }} KB</span>
                  </div>
                  <p class="text-xs text-slate-500 dark:text-slate-400 line-clamp-2">{{ s.description }}</p>
                  <div class="text-[11px] font-mono text-slate-400 truncate">{{ s.path }}</div>
                </div>
                <div class="flex items-center gap-1.5 shrink-0">
                  <n-button size="tiny" secondary type="primary" @click="editSkill(s)">编辑</n-button>
                  <n-popconfirm v-if="s.scope === 'global'" @positive-click="deleteSkillItem(s)">
                    <template #trigger>
                      <n-button size="tiny" quaternary type="error"><n-icon><Trash /></n-icon></n-button>
                    </template>
                    确认永久删除技能「{{ s.name }}」？
                  </n-popconfirm>
                </div>
              </div>
            </n-card>
          </div>
          <n-empty v-if="!filteredSkills.length" description="暂无匹配技能" class="py-10" />
        </div>
      </n-tab-pane>

      <!-- 2. MCP SERVERS -->
      <n-tab-pane name="mcp" :tab="`MCP 服务 (${mcpList.length})`">
        <div class="flex flex-col gap-4 mt-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <n-radio-group v-model:value="mcpViewMode" size="small">
                <n-radio-button value="cards">可视化卡片</n-radio-button>
                <n-radio-button value="json">原始 mcp_config.json</n-radio-button>
              </n-radio-group>
              <span class="text-xs font-mono text-slate-400 hidden sm:inline">{{ mcpPath }}</span>
            </div>
            <n-button type="primary" size="small" @click="openNewMCPModal">
              <template #icon><n-icon><Plus /></n-icon></template>
              添加 MCP Server
            </n-button>
          </div>

          <template v-if="mcpViewMode === 'cards'">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
              <n-card v-for="m in mcpList" :key="m.id" hoverable :bordered="false" class="rounded-2xl shadow-sm">
                <div class="flex flex-col gap-2.5">
                  <div class="flex items-center justify-between gap-2">
                    <div class="flex items-center gap-2 min-w-0">
                      <n-icon :size="18" class="text-fnos-blue"><Plug /></n-icon>
                      <span class="font-mono font-bold text-sm text-slate-800 dark:text-slate-100 truncate">{{ m.id }}</span>
                      <n-tag size="tiny" round :bordered="false" :type="m.transport === 'http' ? 'info' : 'success'">
                        {{ m.transport === 'http' ? 'Streamable HTTP' : 'Stdio' }}
                      </n-tag>
                    </div>
                    <div class="flex items-center gap-1.5 shrink-0">
                      <n-button size="tiny" secondary type="info" :loading="testingMcpId === m.id" @click="testMCP(m.id)">
                        测试连通
                      </n-button>
                      <n-popconfirm @positive-click="removeMCP(m.id)">
                        <template #trigger>
                          <n-button size="tiny" quaternary type="error"><n-icon><Trash /></n-icon></n-button>
                        </template>
                        确认移除 MCP Server「{{ m.id }}」？
                      </n-popconfirm>
                    </div>
                  </div>
                  <div class="p-2.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] font-mono text-xs text-slate-600 dark:text-slate-300 break-all">
                    <template v-if="m.transport === 'http'">URL: {{ m.url }}</template>
                    <template v-else>{{ m.command }} {{ (m.args || []).join(' ') }}</template>
                  </div>
                  <div v-if="mcpTestResults[m.id]" class="text-xs px-2.5 py-1.5 rounded-lg"
                    :class="mcpTestResults[m.id].ok ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400' : 'bg-rose-500/10 text-rose-600 dark:text-rose-400'">
                    {{ mcpTestResults[m.id].ok ? '✓' : '✗' }} {{ mcpTestResults[m.id].message }}
                  </div>
                </div>
              </n-card>
            </div>
            <n-empty v-if="!mcpList.length" description="暂未配置 MCP Server" class="py-10" />
          </template>

          <n-card v-else :bordered="false" class="rounded-2xl shadow-sm">
            <div class="space-y-3">
              <n-input v-model:value="mcpRawJson" type="textarea" :rows="14" class="font-mono text-xs" />
              <div class="flex justify-end">
                <n-button type="primary" size="small" :loading="saving" @click="saveRawMCP">保存 mcp_config.json</n-button>
              </div>
            </div>
          </n-card>
        </div>
      </n-tab-pane>

      <!-- 3. PLUGINS -->
      <n-tab-pane name="plugins" :tab="`Plugins 插件 (${plugins.length})`">
        <div class="flex flex-col gap-4 mt-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-slate-500">启停状态持久化至 `~/.gemini/config/config.json`，优先级高于插件默认声明。</span>
            <n-button type="primary" size="small" @click="showInstallPluginModal = true">
              <template #icon><n-icon><Plus /></n-icon></template>
              创建 / 安装插件
            </n-button>
          </div>
          <div class="flex flex-col gap-3">
            <n-card v-for="p in plugins" :key="p.dir_name" :bordered="false" class="rounded-2xl shadow-sm">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div class="space-y-1 min-w-0 flex-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-bold text-sm text-slate-800 dark:text-slate-100">{{ p.name }}</span>
                    <span class="font-mono text-xs text-slate-400">({{ p.dir_name }})</span>
                    <n-tag v-if="p.version" size="tiny" round :bordered="false">v{{ p.version }}</n-tag>
                    <n-tag size="tiny" round :bordered="false" :type="p.enabled ? 'success' : 'default'">
                      {{ p.enabled ? '已启用' : '已停用' }}
                    </n-tag>
                  </div>
                  <div class="text-xs text-slate-500 dark:text-slate-400">{{ p.description || 'Antigravity 插件包' }}</div>
                  <div class="flex items-center gap-2 pt-0.5 flex-wrap">
                    <n-tag size="tiny" :bordered="false" type="info">Skills: {{ p.skills_count }}</n-tag>
                    <n-tag size="tiny" :bordered="false" type="info">Rules: {{ p.rules_count }}</n-tag>
                    <n-tag v-if="p.has_mcp" size="tiny" :bordered="false" type="success">含 MCP</n-tag>
                    <n-tag v-if="p.has_hooks" size="tiny" :bordered="false" type="warning">含 Hooks</n-tag>
                  </div>
                </div>
                <div class="flex items-center gap-3 shrink-0">
                  <n-switch :value="p.enabled" @update:value="(v) => togglePluginItem(p.dir_name, v)" />
                  <n-popconfirm @positive-click="removePluginItem(p.dir_name)">
                    <template #trigger>
                      <n-button size="small" secondary type="error">卸载</n-button>
                    </template>
                    确认永久卸载插件「{{ p.name }}」？
                  </n-popconfirm>
                </div>
              </div>
            </n-card>
          </div>
          <n-empty v-if="!plugins.length" description="暂无插件包" class="py-10" />
        </div>
      </n-tab-pane>

      <!-- 4. RULES (GEMINI.md / AGENTS.md) -->
      <n-tab-pane name="rules" :tab="`Rules 规则 (${rules.length})`">
        <div class="flex flex-col gap-4 mt-2">
          <!-- 全局 Token 预算进度条 -->
          <n-card :bordered="false" class="rounded-2xl shadow-sm">
            <div class="flex items-center justify-between text-xs mb-2">
              <span class="font-medium text-slate-700 dark:text-slate-200">全局常驻规则上下文预算 (20,000 Tokens 上限)</span>
              <span class="font-mono font-bold text-fnos-blue">{{ totalRuleTokens }} / {{ ruleTokenBudget }} tokens</span>
            </div>
            <n-progress type="line" :percentage="Math.min(100, Math.round((totalRuleTokens / ruleTokenBudget) * 100))"
              :status="totalRuleTokens > ruleTokenBudget * 0.85 ? 'warning' : 'info'" :height="7" border-radius="4px" :show-indicator="false" />
          </n-card>

          <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-slate-500">规则文件列表</span>
                <n-button size="tiny" secondary type="primary" @click="createNewRule">+ 新建规则</n-button>
              </div>
              <div v-for="r in rules" :key="r.id" @click="selectRule(r)"
                class="p-3 rounded-xl border cursor-pointer transition-all"
                :class="selectedRule?.id === r.id
                  ? 'bg-blue-50/70 dark:bg-blue-950/30 border-blue-300 dark:border-blue-800'
                  : 'bg-white dark:bg-[#181b22] border-slate-100 dark:border-white/[0.06] hover:border-slate-300'">
                <div class="flex items-center justify-between gap-2">
                  <span class="font-mono text-xs font-bold text-slate-800 dark:text-slate-100 truncate">{{ r.name }}</span>
                  <n-tag size="tiny" round :bordered="false">{{ r.scope }}</n-tag>
                </div>
                <div class="mt-2 space-y-1">
                  <div class="flex justify-between text-[11px] font-mono text-slate-400">
                    <span>{{ (r.size_bytes / 1024).toFixed(1) }} KB / 24 KB</span>
                    <span>~{{ r.est_tokens }} tok</span>
                  </div>
                  <n-progress type="line" :percentage="Math.round(r.size_percent)" :show-indicator="false" :height="4" />
                </div>
              </div>
            </div>

            <n-card :bordered="false" class="lg:col-span-2 rounded-2xl shadow-sm">
              <div class="space-y-3">
                <div class="flex items-center justify-between gap-2">
                  <n-input v-model:value="editingRuleName" size="small" placeholder="文件名，如 GEMINI.md 或 AGENTS.md" class="max-w-xs font-mono" />
                  <n-button type="primary" size="small" :loading="saving" @click="saveCurrentRule">保存规则</n-button>
                </div>
                <n-input v-model:value="editingRuleContent" type="textarea" :rows="15" placeholder="# Agent Rules..." class="font-mono text-xs" />
              </div>
            </n-card>
          </div>
        </div>
      </n-tab-pane>

      <!-- 5. HOOKS -->
      <n-tab-pane name="hooks" :tab="`Hooks 钩子 (${hooks.length})`">
        <div class="flex flex-col gap-4 mt-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-slate-500">支持 SessionStart / PreToolUse / PostToolUse / PreInvocation / PostInvocation / Stop 六大生命周期事件。</span>
            <div class="flex gap-2">
              <n-button size="small" secondary @click="addHookRow"><template #icon><n-icon><Plus /></n-icon></template>添加钩子</n-button>
              <n-button size="small" type="primary" :loading="saving" @click="saveAllHooks">保存 hooks.json</n-button>
            </div>
          </div>
          <div class="space-y-3">
            <n-card v-for="(h, idx) in hooks" :key="h.id || idx" :bordered="false" class="rounded-2xl shadow-sm">
              <div class="grid grid-cols-1 sm:grid-cols-12 gap-3 items-center">
                <div class="sm:col-span-3">
                  <div class="text-[11px] text-slate-400 mb-1">名称</div>
                  <n-input v-model:value="h.name" size="small" placeholder="钩子名称" />
                </div>
                <div class="sm:col-span-2">
                  <div class="text-[11px] text-slate-400 mb-1">生命周期事件</div>
                  <n-select v-model:value="h.event" size="small" :options="hookEventOptions" />
                </div>
                <div class="sm:col-span-2">
                  <div class="text-[11px] text-slate-400 mb-1">正则 Matcher</div>
                  <n-input v-model:value="h.matcher" size="small" class="font-mono" placeholder=".*" />
                </div>
                <div class="sm:col-span-3">
                  <div class="text-[11px] text-slate-400 mb-1">执行命令</div>
                  <n-input v-model:value="h.command" size="small" class="font-mono" placeholder="Shell 命令" />
                </div>
                <div class="sm:col-span-2 flex items-end justify-end gap-2 pt-4">
                  <n-switch v-model:value="h.enabled" size="small" />
                  <n-button size="small" quaternary type="error" @click="hooks.splice(idx, 1)">
                    <template #icon><n-icon><Trash /></n-icon></template>
                  </n-button>
                </div>
              </div>
            </n-card>
          </div>
        </div>
      </n-tab-pane>
    </n-tabs>

    <!-- 新建/编辑 Skill 弹窗 -->
    <n-modal v-model:show="showSkillModal" preset="card" :title="skillForm.isEdit ? `编辑技能: ${skillForm.name}` : '新建 Antigravity 技能'"
      class="w-[92vw] max-w-2xl rounded-2xl">
      <div class="space-y-3">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div class="text-xs text-slate-500 mb-1">技能标识名 (小写连字符)</div>
            <n-input v-model:value="skillForm.name" :disabled="skillForm.isEdit" placeholder="如 docker-audit" class="font-mono" />
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">简短描述 (Frontmatter description)</div>
            <n-input v-model:value="skillForm.description" placeholder="一句话说明何时激活此技能" />
          </div>
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">SKILL.md 完整内容</div>
          <n-input v-model:value="skillForm.content" type="textarea" :rows="14" class="font-mono text-xs" />
        </div>
        <div class="flex justify-end gap-2">
          <n-button @click="showSkillModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="submitSkill">保存技能</n-button>
        </div>
      </div>
    </n-modal>

    <!-- 新增 MCP 弹窗 -->
    <n-modal v-model:show="showMCPModal" preset="card" title="添加 MCP Server" class="w-[92vw] max-w-lg rounded-2xl">
      <div class="space-y-3">
        <div class="flex flex-wrap gap-1.5">
          <span class="text-xs text-slate-400 mr-1">快捷预设:</span>
          <n-button size="tiny" secondary @click="applyMcpPreset('filesystem')">Filesystem</n-button>
          <n-button size="tiny" secondary @click="applyMcpPreset('sqlite')">SQLite</n-button>
          <n-button size="tiny" secondary @click="applyMcpPreset('fetch')">Fetch HTTP</n-button>
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">Server ID</div>
          <n-input v-model:value="mcpForm.id" placeholder="如 nas-filesystem" class="font-mono" />
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">传输协议</div>
          <n-radio-group v-model:value="mcpForm.transport" size="small">
            <n-radio-button value="stdio">Stdio 本地进程</n-radio-button>
            <n-radio-button value="http">Streamable HTTP 远程端点</n-radio-button>
          </n-radio-group>
        </div>
        <template v-if="mcpForm.transport === 'stdio'">
          <div>
            <div class="text-xs text-slate-500 mb-1">启动命令 (command)</div>
            <n-input v-model:value="mcpForm.command" placeholder="如 npx / node / python3" class="font-mono" />
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">参数 (空格分隔)</div>
            <n-input v-model:value="mcpForm.argsText" placeholder="-y @modelcontextprotocol/server-filesystem /vol1" class="font-mono" />
          </div>
        </template>
        <div v-else>
          <div class="text-xs text-slate-500 mb-1">Streamable HTTP URL</div>
          <n-input v-model:value="mcpForm.url" placeholder="https://example.com/mcp" class="font-mono" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <n-button @click="showMCPModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="submitMCP">保存</n-button>
        </div>
      </div>
    </n-modal>

    <!-- 创建/安装插件弹窗 -->
    <n-modal v-model:show="showInstallPluginModal" preset="card" title="创建或从 Git 安装插件包" class="w-[92vw] max-w-lg rounded-2xl">
      <div class="space-y-3">
        <div>
          <div class="text-xs text-slate-500 mb-1">插件目录标识 (dir_name)</div>
          <n-input v-model:value="pluginForm.dir_name" placeholder="如 nas-backup-kit" class="font-mono" />
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">显示名称</div>
          <n-input v-model:value="pluginForm.name" placeholder="如 NAS Backup Kit" />
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">描述</div>
          <n-input v-model:value="pluginForm.description" placeholder="插件包说明" />
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">Git 仓库地址 (可选，留空则创建本地插件脚手架)</div>
          <n-input v-model:value="pluginForm.git_url" placeholder="https://github.com/org/agy-plugin.git" class="font-mono" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <n-button @click="showInstallPluginModal = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="submitInstallPlugin">安装并启用</n-button>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NTabs, NTabPane, NCard, NButton, NInput, NTag, NIcon, NEmpty, NModal, NSwitch,
  NRadioGroup, NRadioButton, NProgress, NSelect, NPopconfirm, useMessage
} from 'naive-ui'
import { Refresh, Plus, Trash, Search, Plug } from '@vicons/tabler'
import { customizationsApi } from '../api'
import type { SkillItem, MCPServerView, MCPTestResult, PluginItem, RuleFileItem, HookRule } from '../types/api'

const message = useMessage()
const activeSubTab = ref('skills')
const loading = ref(false)
const saving = ref(false)

// Skills
const skills = ref<SkillItem[]>([])
const skillSearch = ref('')
const showSkillModal = ref(false)
const skillForm = ref({ isEdit: false, scope: 'global', name: '', description: '', content: '' })

const filteredSkills = computed(() => {
  const kw = skillSearch.value.trim().toLowerCase()
  if (!kw) return skills.value
  return skills.value.filter((s) => s.name.toLowerCase().includes(kw) || s.description.toLowerCase().includes(kw))
})

// MCP
const mcpList = ref<MCPServerView[]>([])
const mcpRawJson = ref('')
const mcpPath = ref('')
const mcpViewMode = ref<'cards' | 'json'>('cards')
const testingMcpId = ref('')
const mcpTestResults = ref<Record<string, MCPTestResult>>({})
const showMCPModal = ref(false)
const mcpForm = ref({ id: '', transport: 'stdio' as 'stdio' | 'http', command: 'npx', argsText: '', url: '' })

// Plugins
const plugins = ref<PluginItem[]>([])
const showInstallPluginModal = ref(false)
const pluginForm = ref({ dir_name: '', name: '', description: '', git_url: '' })

// Rules
const rules = ref<RuleFileItem[]>([])
const totalRuleTokens = ref(0)
const ruleTokenBudget = ref(20000)
const selectedRule = ref<RuleFileItem | null>(null)
const editingRuleName = ref('GEMINI.md')
const editingRuleContent = ref('')

// Hooks
const hooks = ref<HookRule[]>([])
const hookEventOptions = [
  { label: 'SessionStart', value: 'SessionStart' },
  { label: 'PreToolUse', value: 'PreToolUse' },
  { label: 'PostToolUse', value: 'PostToolUse' },
  { label: 'PreInvocation', value: 'PreInvocation' },
  { label: 'PostInvocation', value: 'PostInvocation' },
  { label: 'Stop', value: 'Stop' }
]

async function loadSkills() {
  const res = await customizationsApi.listSkills()
  if (res.success && res.data) skills.value = res.data.items || []
}
async function loadMCP() {
  const res = await customizationsApi.getMCP()
  if (res.success && res.data) {
    mcpList.value = res.data.items || []
    mcpRawJson.value = res.data.raw_json || '{}'
    mcpPath.value = res.data.path || ''
  }
}
async function loadPlugins() {
  const res = await customizationsApi.listPlugins()
  if (res.success && res.data) plugins.value = res.data.items || []
}
async function loadRules() {
  const res = await customizationsApi.listRules()
  if (res.success && res.data) {
    rules.value = res.data.items || []
    totalRuleTokens.value = res.data.total_tokens || 0
    ruleTokenBudget.value = res.data.token_budget || 20000
    if (!selectedRule.value && rules.value.length) {
      selectRule(rules.value[0])
    }
  }
}
async function loadHooks() {
  const res = await customizationsApi.listHooks()
  if (res.success && res.data) hooks.value = res.data.items || []
}

async function reloadActive() {
  loading.value = true
  await Promise.all([loadSkills(), loadMCP(), loadPlugins(), loadRules(), loadHooks()])
  loading.value = false
}

function openNewSkillModal() {
  skillForm.value = {
    isEdit: false,
    scope: 'global',
    name: '',
    description: '',
    content: '---\nname: my-custom-skill\ndescription: Describe when the agent should load this skill.\n---\n\n# My Custom Skill\n\n## Steps\n1. Inspect project files.\n2. Execute validation.\n'
  }
  showSkillModal.value = true
}

async function editSkill(s: SkillItem) {
  const res = await customizationsApi.getSkill(s.scope, s.name)
  if (res.success && res.data) {
    skillForm.value = {
      isEdit: true,
      scope: res.data.scope,
      name: res.data.name,
      description: res.data.description,
      content: res.data.content || ''
    }
    showSkillModal.value = true
  }
}

async function submitSkill() {
  saving.value = true
  const res = await customizationsApi.saveSkill(skillForm.value)
  saving.value = false
  if (res.success) {
    message.success('技能已保存')
    showSkillModal.value = false
    await loadSkills()
  } else {
    message.error(res.message)
  }
}

async function deleteSkillItem(s: SkillItem) {
  const res = await customizationsApi.deleteSkill(s.scope, s.name)
  if (res.success) {
    message.success('技能已删除')
    await loadSkills()
  } else message.error(res.message)
}

function openNewMCPModal() {
  mcpForm.value = { id: '', transport: 'stdio', command: 'npx', argsText: '', url: '' }
  showMCPModal.value = true
}

function applyMcpPreset(kind: string) {
  if (kind === 'filesystem') {
    mcpForm.value = { id: 'nas-filesystem', transport: 'stdio', command: 'npx', argsText: '-y @modelcontextprotocol/server-filesystem /vol1', url: '' }
  } else if (kind === 'sqlite') {
    mcpForm.value = { id: 'sqlite-db', transport: 'stdio', command: 'uvx', argsText: 'mcp-server-sqlite --db-path /vol3/data.db', url: '' }
  } else {
    mcpForm.value = { id: 'remote-mcp', transport: 'http', command: '', argsText: '', url: 'https://mcp.antigravity.google/mcp' }
  }
}

async function submitMCP() {
  saving.value = true
  const payload =
    mcpForm.value.transport === 'http'
      ? { id: mcpForm.value.id, url: mcpForm.value.url }
      : {
          id: mcpForm.value.id,
          command: mcpForm.value.command,
          args: mcpForm.value.argsText.trim() ? mcpForm.value.argsText.trim().split(/\s+/) : []
        }
  const res = await customizationsApi.saveMCP(payload)
  saving.value = false
  if (res.success) {
    message.success('MCP Server 已保存')
    showMCPModal.value = false
    await loadMCP()
  } else message.error(res.message)
}

async function saveRawMCP() {
  saving.value = true
  const res = await customizationsApi.saveMCPRaw(mcpRawJson.value)
  saving.value = false
  if (res.success) {
    message.success('mcp_config.json 已保存')
    await loadMCP()
  } else message.error(res.message)
}

async function testMCP(id: string) {
  testingMcpId.value = id
  const res = await customizationsApi.testMCP(id)
  testingMcpId.value = ''
  if (res.success && res.data) {
    mcpTestResults.value[id] = res.data
    if (res.data.ok) message.success(`${id}: ${res.data.message}`)
    else message.warning(`${id}: ${res.data.message}`)
  }
}

async function removeMCP(id: string) {
  const res = await customizationsApi.deleteMCP(id)
  if (res.success) {
    message.success('已移除')
    await loadMCP()
  } else message.error(res.message)
}

async function togglePluginItem(name: string, enabled: boolean) {
  const res = await customizationsApi.togglePlugin(name, enabled)
  if (res.success) {
    message.success(enabled ? `已启用插件 ${name}` : `已停用插件 ${name}`)
    await loadPlugins()
  } else message.error(res.message)
}

async function submitInstallPlugin() {
  saving.value = true
  const res = await customizationsApi.installPlugin(pluginForm.value)
  saving.value = false
  if (res.success) {
    message.success('插件已安装并启用')
    showInstallPluginModal.value = false
    pluginForm.value = { dir_name: '', name: '', description: '', git_url: '' }
    await loadPlugins()
  } else message.error(res.message)
}

async function removePluginItem(name: string) {
  const res = await customizationsApi.deletePlugin(name)
  if (res.success) {
    message.success('插件已卸载')
    await loadPlugins()
  } else message.error(res.message)
}

function selectRule(r: RuleFileItem) {
  selectedRule.value = r
  editingRuleName.value = r.id.startsWith('global:') ? r.id.replace('global:', '') : r.name
  editingRuleContent.value = r.content
}

function createNewRule() {
  selectedRule.value = null
  editingRuleName.value = 'AGENTS.md'
  editingRuleContent.value = '# Project Agent Rules\n\n- Follow strict type safety.\n'
}

async function saveCurrentRule() {
  saving.value = true
  const res = await customizationsApi.saveRule(editingRuleName.value, editingRuleContent.value)
  saving.value = false
  if (res.success) {
    message.success('规则文件已保存')
    await loadRules()
  } else message.error(res.message)
}

function addHookRow() {
  hooks.value.push({
    id: '',
    name: '新生命周期钩子',
    event: 'PreToolUse',
    matcher: '.*',
    command: 'echo \'{"decision":"allow"}\'',
    timeout_sec: 10,
    enabled: true
  })
}

async function saveAllHooks() {
  saving.value = true
  const res = await customizationsApi.saveHooks(hooks.value)
  saving.value = false
  if (res.success) {
    message.success('Hooks 配置已保存')
    await loadHooks()
  } else message.error(res.message)
}

onMounted(() => {
  void reloadActive()
})
</script>
