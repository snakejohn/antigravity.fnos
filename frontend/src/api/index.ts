import { http, resolveApiPrefix } from '../utils/http'
import type {
  StatusData,
  DoctorReport,
  SkillItem,
  MCPServerView,
  MCPTestResult,
  PluginItem,
  RuleFileItem,
  HookRule,
  ConversationMeta,
  ConversationDetail,
  WorkspaceItem,
  WorkspaceFilesResponse,
  WorkspaceFileContent,
  TerminalExecResult,
  AccountStatusResponse,
  AccountProfile,
  EnvVarItem,
  SnapshotSummary,
  AppConfig
} from '../types/api'

export const systemApi = {
  getStatus: () => http.get<StatusData>('status'),
  sendAction: (action: string, payload?: Record<string, unknown>) =>
    http.post<StatusData | DoctorReport>('action', { action, ...payload }),
  getDoctor: () => http.get<DoctorReport>('doctor'),
  checkUpdate: () =>
    http.get<{
      current_version: string
      remote_version: string
      has_update: boolean
      channel: string
      message: string
    }>('check-update'),
  getVersions: () =>
    http.get<{ current_version: string; recommended_version: string; versions: string[] }>('deploy/versions')
}

export const customizationsApi = {
  listSkills: () => http.get<{ items: SkillItem[] }>('customizations/skills'),
  getSkill: (scope: string, name: string) =>
    http.get<SkillItem>(`customizations/skills/${encodeURIComponent(scope)}/${encodeURIComponent(name)}`),
  saveSkill: (payload: { scope: string; name: string; description: string; content: string }) =>
    http.post<SkillItem>('customizations/skills', payload),
  deleteSkill: (scope: string, name: string) =>
    http.delete(`customizations/skills/${encodeURIComponent(scope)}/${encodeURIComponent(name)}`),

  getMCP: () => http.get<{ items: MCPServerView[]; raw_json: string; path: string }>('customizations/mcp'),
  saveMCP: (payload: {
    id: string
    command?: string
    args?: string[]
    env?: Record<string, string>
    url?: string
  }) => http.post('customizations/mcp', payload),
  saveMCPRaw: (raw_json: string) => http.post('customizations/mcp/raw', { raw_json }),
  testMCP: (id: string) => http.post<MCPTestResult>('customizations/mcp/test', { id }),
  deleteMCP: (id: string) => http.delete(`customizations/mcp/${encodeURIComponent(id)}`),

  listPlugins: () => http.get<{ items: PluginItem[] }>('customizations/plugins'),
  togglePlugin: (name: string, enabled: boolean) =>
    http.post('customizations/plugins/toggle', { name, enabled }),
  installPlugin: (payload: { dir_name?: string; name?: string; description?: string; author?: string; git_url?: string }) =>
    http.post<PluginItem>('customizations/plugins/install', payload),
  deletePlugin: (name: string) => http.delete(`customizations/plugins/${encodeURIComponent(name)}`),

  listRules: () => http.get<{ items: RuleFileItem[]; total_tokens: number; token_budget: number }>('customizations/rules'),
  saveRule: (name: string, content: string) => http.post<RuleFileItem>('customizations/rules', { name, content }),

  listHooks: () => http.get<{ items: HookRule[]; path: string }>('customizations/hooks'),
  saveHooks: (items: HookRule[]) => http.post('customizations/hooks', { items })
}

export const conversationsApi = {
  list: (includeArchived = false) =>
    http.get<{ items: ConversationMeta[] }>(`conversations?include_archived=${includeArchived}`),
  create: (payload: { title?: string; model?: string; workspace?: string; tags?: string[]; prompt?: string }) =>
    http.post<ConversationMeta>('conversations', payload),
  getDetail: (id: string) => http.get<ConversationDetail>(`conversations/${encodeURIComponent(id)}`),
  sendMessage: (id: string, prompt: string) =>
    http.post(`conversations/${encodeURIComponent(id)}/message`, { prompt }),
  update: (id: string, patch: Record<string, unknown>) =>
    http.patch<ConversationMeta>(`conversations/${encodeURIComponent(id)}`, patch),
  remove: (id: string) => http.delete(`conversations/${encodeURIComponent(id)}`)
}

export const workspaceApi = {
  list: () => http.get<{ items: WorkspaceItem[] }>('workspace/list'),
  add: (title: string, path: string) => http.post<WorkspaceItem>('workspace/add', { title, path }),
  remove: (id: string) => http.delete(`workspace/${encodeURIComponent(id)}`),
  listFiles: (path?: string) =>
    http.get<WorkspaceFilesResponse>(`workspaces/files${path ? `?path=${encodeURIComponent(path)}` : ''}`),
  readFile: (path: string) =>
    http.get<WorkspaceFileContent>(`workspaces/file?path=${encodeURIComponent(path)}`),
  writeFile: (path: string, content: string) =>
    http.post<WorkspaceFileContent>('workspaces/file', { path, content }),
  execTerminal: (command: string, cwd?: string, timeout_sec = 30) =>
    http.post<TerminalExecResult>('terminal/exec', { command, cwd, timeout_sec }, { timeout: 125000 })
}

export const authApi = {
  getStatus: () => http.get<AccountStatusResponse>('account/status'),
  getPool: () => http.get<{ items: AccountProfile[] }>('account/pool'),
  saveAccount: (payload: Partial<AccountProfile>) => http.post<AccountProfile>('account/save', payload),
  switchAccount: (id: string) => http.post<AccountProfile>('account/switch', { id }),
  logout: () => http.post('account/logout'),
  deleteAccount: (id: string) => http.delete(`account/pool/${encodeURIComponent(id)}`),
  getEnvVars: () => http.get<{ items: EnvVarItem[]; path: string }>('env-vars'),
  saveEnvVars: (items: EnvVarItem[]) => http.post('env-vars', { items })
}

export const snapshotApi = {
  list: () => http.get<SnapshotSummary>('snapshots'),
  create: (name: string, compression_level = 1) =>
    http.post('snapshots', { name, compression_level }),
  cancel: () => http.post('snapshots/cancel'),
  restore: (id: string) => http.post(`snapshots/${encodeURIComponent(id)}/restore`),
  remove: (id: string) => http.delete(`snapshots/${encodeURIComponent(id)}`)
}

export const logApi = {
  getLogs: (limit = 400) => http.get<{ lines: string[]; content: string }>(`logs?limit=${limit}`),
  clearLogs: () => http.delete('logs'),
  downloadUrl: () => `${resolveApiPrefix()}/logs/download`
}

export const configApi = {
  getConfig: () => http.get<AppConfig>('config'),
  saveConfig: (cfg: AppConfig) => http.post<AppConfig>('config', cfg)
}
