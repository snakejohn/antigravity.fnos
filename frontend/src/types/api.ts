export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
  timestamp?: number
}

export interface RequestResult<T = unknown> {
  success: boolean
  data?: T
  code?: number
  message: string
  timestamp?: number
}

export interface HostMetrics {
  hostname: string
  os: string
  arch: string
  cpu_cores: number
  cpu_percent: number
  load_avg: string
  mem_total_bytes: number
  mem_used_bytes: number
  mem_percent: number
  disk_mount: string
  disk_total_bytes: number
  disk_used_bytes: number
  disk_avail_bytes: number
  disk_percent: number
}

export interface HostCapabilities {
  agy_cli_installed: boolean
  agy_cli_version?: string
  node_installed: boolean
  node_version?: string
  python_installed: boolean
  python_version?: string
  python_sdk_installed: boolean
  docker_installed: boolean
  language_server_connected: boolean
  language_server_address?: string
}

export interface CustomizationsSummary {
  skills_count: number
  mcp_count: number
  plugins_count: number
  enabled_plugins: number
  rules_count: number
  hooks_count: number
  total_rule_kb: number
  est_rule_tokens: number
}

export interface StatusData {
  name: string
  version: string
  app_version?: string
  target_version?: string
  status: 'stopped' | 'starting' | 'running' | 'building' | 'snapshotting' | string
  uptime?: string
  started_at: number
  server_port: number
  proxy_port: number
  default_model: string
  sandbox_mode: string
  approval_policy: string
  server_time?: number
  build_time?: string
  pid?: number
  cpu: string
  memory: string
  host?: HostMetrics
  conversations_count?: number
  active_runs?: number
  customizations?: CustomizationsSummary
  capabilities?: HostCapabilities
  last_message?: string
}

export interface DoctorCheck {
  name: string
  status: 'ok' | 'warn' | 'error'
  detail: string
  suggest?: string
}

export interface DoctorReport {
  passed_count: number
  warn_count: number
  error_count: number
  checked_at: string
  checks: DoctorCheck[]
}

export interface SkillItem {
  name: string
  description: string
  scope: string
  path: string
  size_bytes: number
  sub_dirs: string[]
  updated_at: number
  content?: string
}

export interface MCPServerView {
  id: string
  transport: 'stdio' | 'http'
  command?: string
  args?: string[]
  env?: Record<string, string>
  url?: string
  headers?: Record<string, string>
}

export interface MCPTestResult {
  id: string
  ok: boolean
  transport: string
  latency_ms: number
  message: string
  resolved_to?: string
}

export interface PluginItem {
  dir_name: string
  name: string
  description: string
  version: string
  author: string
  homepage: string
  enabled: boolean
  path: string
  skills_count: number
  rules_count: number
  has_mcp: boolean
  has_hooks: boolean
}

export interface RuleFileItem {
  id: string
  name: string
  scope: string
  path: string
  size_bytes: number
  limit_bytes: number
  size_percent: number
  est_tokens: number
  updated_at: number
  content: string
}

export interface HookRule {
  id: string
  name: string
  event: string
  matcher: string
  command: string
  timeout_sec: number
  enabled: boolean
  description?: string
}

export interface ConversationMeta {
  id: string
  model: string
  workspace: string
  createTime: string
  updateTime: string
  runState: 'RUN_STATE_IDLE' | 'RUN_STATE_RUNNING' | 'RUN_STATE_CANCELLING' | string
  summary: string
  turnCount: number
  annotations: {
    title: string
    tags: string[]
    archived: boolean
    pinned: boolean
    starred: boolean
  }
}

export interface TranscriptEntry {
  id: string
  role: 'user' | 'thought' | 'tool_call' | 'tool_result' | 'assistant' | 'system'
  content: string
  tool_name?: string
  tool_args?: string
  model?: string
  timestamp: string
}

export interface ConversationDetail extends ConversationMeta {
  transcript: TranscriptEntry[]
  artifacts: string[]
}

export interface WorkspaceItem {
  workspaceId: string
  title: string
  path: string
  exists: boolean
  has_agents_dir: boolean
  skills_count: number
  has_gemini_rule: boolean
  git_branch?: string
  session_count: number
  createdAt: number
  updatedAt: number
}

export interface TerminalExecResult {
  command: string
  cwd: string
  stdout: string
  stderr: string
  exit_code: number
  duration_ms: number
  timed_out: boolean
}

export interface AccountProfile {
  id: string
  name: string
  provider: string
  email?: string
  api_key?: string
  masked_key?: string
  base_url?: string
  project_id?: string
  region?: string
  default_model?: string
  is_active: boolean
  created_at: number
}

export interface AccountStatusResponse {
  authorized: boolean
  active_account?: AccountProfile
  pool_count: number
  cli_detected: boolean
  sdk_detected: boolean
  ls_connected: boolean
  env_has_gemini: boolean
  console_url: string
  docs_url: string
}

export interface EnvVarItem {
  key: string
  value: string
}

export interface SnapshotMeta {
  id: string
  name: string
  created_at: number
  size_bytes: number
  harness_version: string
  app_version: string
  custom_count: number
  conv_count: number
  compression_level: number
  file_name: string
}

export interface SnapshotSummary {
  items: SnapshotMeta[]
  total_size_bytes: number
}

export interface SnapshotProgressTask {
  active: boolean
  action: 'create' | 'restore' | string
  title: string
  percent: number
}

export interface AppConfig {
  server_port: number
  proxy_port: number
  default_model: string
  sandbox_mode: string
  approval_policy: string
  heap_memory_limit?: number
  network_proxy: string
  proxy_runtime_enabled: boolean
  npm_registry: string
  pypi_index: string
  update_channel: string
  access_mode: string
  reverse_proxy_url?: string
  access_password?: string
  data_library_path?: string
  gemini_home?: string
  version?: string
  build_time?: string
}
