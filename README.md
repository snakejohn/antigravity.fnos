# Antigravity Harness for fnOS (`antigravity.harness.fnos`)

专为飞牛 NAS (**fnOS**) 打造的 **Google Antigravity (AGY) Harness** 可视化管理控制台与原生应用包 (`.fpk`)。项目参考 [`deepseek.harness.fnos`](https://github.com/yuexps/deepseek.harness.fnos) 的飞牛视觉风格与工程实践，针对 Google Antigravity 架构与五大定制体系深度定制。

## 核心功能

0. **Antigravity 2.0 全屏工作窗口 (`Antigravity WebUI` · `/fngateway/`)**
   - 飞牛桌面独立双入口：**Antigravity Admin**（管理控制台）+ **Antigravity WebUI**（全屏三栏式 Agent 工作台），也可从管理端概览页/侧边栏一键打开。
   - **左侧栏**：NAS 工作区快速切换、新建 Agent 会话、会话搜索、星标/置顶/删除与 WebSocket 实时状态同步。
   - **中间主画布**：支持最新 **Gemini 4 Neon / Hydrogen / Argon / Carbon**、**Gemini 3.8 Flash**、**Orchestrator (Neon)**、**Claude 5.5** 及任意自定义模型 ID 选择器、可折叠 `Thinking` 思考过程卡片、`Tool Call / Result` 执行轨迹卡片、代码高亮 Markdown 渲染、快捷斜杠命令 (`/goal`、`/plan`、`/doctor`、`/skills`、`/compact`)。
   - **右侧辅助面板 (Auxiliary Pane)**：
     - **文件**：在线浏览当前工作区真实目录树，支持代码/文档预览、在线编辑保存与一键引用至对话。
     - **终端**：在当前工作区目录直接执行 `agy doctor` 与 Shell 命令并查看输出。
     - **Skills & MCP**：实时查看激活的定制项、Rules Token 预算，支持一键插入技能到对话与测试 MCP 连通性。
1. **概览与服务控制 (`Overview`)**
   - 实时监控 fnOS 宿主机整机指标（CPU 负载、物理内存、存储卷磁盘空间）与 Harness 进程资源占用。
   - 一键启动 / 停止 / 重启 Antigravity Harness 运行时，支持版本部署切换与 `agy doctor` 全面环境健康巡检。
   - 宿主机能力自动探测（`ConversationService` RPC、Node.js、Python 3、Docker、`agy` CLI）。
2. **Antigravity 五大定制中心 (`Customizations`)**
   - **Skills (`SKILL.md`)**：自动扫描全局、工作区与插件技能，支持搜索、创建与在线编辑 YAML Frontmatter + Markdown。
   - **MCP Servers (`mcp_config.json`)**：支持 `Stdio` 本地进程与 `Streamable HTTP` 远程端点可视化配置、预设模板、一键连通性测试及 Raw JSON 编辑。
   - **Plugins (`plugin.json` & `config.json`)**：插件启停开关持久化至 `config.json`（严格遵循 Antigravity 优先级规则），支持脚手架创建与 Git 仓库安装。
   - **Rules (`GEMINI.md` / `AGENTS.md`)**：支持单文件 24KB 容量条与全局 20,000 Tokens 上下文预算可视化监控与在线编辑。
   - **Hooks (`hooks.json`)**：管理 `SessionStart` / `PreToolUse` / `PostToolUse` / `PreInvocation` / `PostInvocation` / `Stop` 六大生命周期钩子。
3. **会话与任务监控 (`Conversations`)**
   - 内置标准 Connect HTTP+JSON `product.v1.ConversationService` 端点及本地 `brain/<id>/transcript.jsonl` 持久化引擎。
   - 支持发起新 Agent 任务、选择模型档位 (`Gemini 2.5 Pro / Flash / Flash-Lite`)、实时查看 Thinking / ToolCall / Result 时间线轨迹、星标/置顶/归档与多轮对话。
4. **工作区与命令终端 (`Workspace & Terminal`)**
   - 绑定 fnOS 存储卷路径，自动探测 `.agents/` 定制项、规则文件与 Git 分支。
   - 内置 Web Command Runner，支持 `agy doctor`、系统运维快捷指令与自定义命令安全执行。
5. **授权管理与环境变量 (`Auth & .env`)**
   - 支持 Google Gemini API、Google Cloud Vertex AI 与自定义代理多账号池保存与一键切换。
   - 可视化 `.env` 环境变量表格编辑器，自动对敏感 Key/Token 掩码保护。
6. **快照备份与实时日志 (`Snapshots & Logs`)**
   - 一键将配置、凭据、五大定制文件与历史会话打包为 `tar.gz` 快照（支持 Lv1 / Lv6 / Lv9 压缩及 WebSocket 实时进度），支持一键还原。
   - WebSocket 实时推送语法高亮日志流，支持级别过滤、自动滚动、下载与清空。
7. **应用设置 (`Settings`)**
   - 配置默认模型档位、沙箱策略 (`workspace-write` / `read-only` / `full-access`)、工具审批策略 (`on-request` / `untrusted` / `never`)、网络代理与镜像源，并自动同步 `~/.gemini/antigravity-cli/settings.json`。

## 技术架构

- **后端**：Go (`package main` 单二进制) + Gin + Gorilla WebSocket + `//go:embed` 内嵌前端静态资源。
- **前端**：Vue 3 + TypeScript + Naive UI + Pinia + Tailwind CSS + `@vicons/tabler` + `highlight.js` + `@trimjs/web-app`。
- **双模监听**：
  - TCP HTTP 监听：`0.0.0.0:2399`（可通过 `AGY_ADMIN_PORT` 覆盖），同时兼容根路径 `/` 与 `/app/antigravity-harness/`。
  - 内部 Harness Runtime / `ConversationService` RPC：`127.0.0.1:2398`。
  - fnOS 原生 Unix Socket：当检测到 `TRIM_APPDEST` 时自动挂载 `${TRIM_APPDEST}/web.sock` 接入飞牛统一桌面网关。

## 快速构建与部署

### 1. 本地编译（生成 Linux `x86_64` 单二进制）

```bash
# Linux / macOS
./build.sh

# Windows
build.cmd
```

构建产物位于 `fnpack/app/bin/antigravity.harness`。

### 2. 飞牛 fnOS 原生安装 (`.fpk` / `appcenter-cli`)

在 `fnpack/` 目录使用飞牛官方 `fnpack build` 打包生成 `antigravity.harness.fpk`，然后在飞牛 NAS 上安装：

```bash
# 通过 .fpk 安装包安装
sudo appcenter-cli install-fpk antigravity.harness.fpk -v 3

# 或直接通过 Systemd 服务部署
sudo bash deploy/install_fnos.sh
```

安装完成后即可通过飞牛桌面图标或浏览器访问 `http://<NAS-IP>:2399/`。

## License

MIT
