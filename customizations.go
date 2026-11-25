package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// =========================================================================
// Antigravity Five-Pillar Customizations System
// 1. Skills (SKILL.md)
// 2. MCP Servers (mcp_config.json)
// 3. Plugins (plugins/<name>/plugin.json + config.json)
// 4. Rules (GEMINI.md / AGENTS.md / .agents/rules/*.md)
// 5. Hooks (hooks.json)
// =========================================================================

var customMu sync.Mutex

type CustomizationsSummary struct {
	SkillsCount  int `json:"skills_count"`
	MCPCount     int `json:"mcp_count"`
	PluginsCount int `json:"plugins_count"`
	EnabledPlugs int `json:"enabled_plugins"`
	RulesCount   int `json:"rules_count"`
	HooksCount   int `json:"hooks_count"`
	TotalRuleKB  float64 `json:"total_rule_kb"`
	EstRuleTokens int `json:"est_rule_tokens"`
}

func EnsureDefaultCustomizations() {
	customMu.Lock()
	defer customMu.Unlock()

	// 1. Seed a default starter Skill if skills dir is empty
	skillsDir := filepath.Join(globalConfigDir, "skills")
	_ = os.MkdirAll(skillsDir, 0755)
	if entries, err := os.ReadDir(skillsDir); err == nil && len(entries) == 0 {
		starterDir := filepath.Join(skillsDir, "fnos-nas-ops")
		_ = os.MkdirAll(starterDir, 0755)
		starterSkill := `---
name: fnos-nas-ops
description: Manage and diagnose fnOS NAS volumes, Docker containers, systemd services, and disk health safely.
---

# fnOS NAS Operations Runbook

Use this skill when inspecting or maintaining the fnOS NAS environment.

## Checklist
1. Check storage pools with ` + "`df -h /vol1 /vol2 /vol3`" + `.
2. Inspect running Docker stacks with ` + "`docker ps --format \"table {{.Names}}\\t{{.Status}}\"`" + `.
3. Verify native fnOS apps via ` + "`appcenter-cli list`" + `.
4. Always create a snapshot before modifying production configs.
`
		_ = os.WriteFile(filepath.Join(starterDir, "SKILL.md"), []byte(starterSkill), 0644)

		codeReviewDir := filepath.Join(skillsDir, "code-audit-guardian")
		_ = os.MkdirAll(codeReviewDir, 0755)
		reviewSkill := `---
name: code-audit-guardian
description: Perform structured security, performance, and regression audits on code changes before deployment.
---

# Code Audit Guardian

Follow these steps when auditing a repository:
1. Scan for hardcoded credentials, API keys, or unsafe shell interpolation.
2. Check error handling and resource cleanup (file descriptors, sockets, goroutines).
3. Summarize findings by severity: Critical, High, Medium, Low.
`
		_ = os.WriteFile(filepath.Join(codeReviewDir, "SKILL.md"), []byte(reviewSkill), 0644)
	}

	// 2. Seed default mcp_config.json if missing
	mcpPath := filepath.Join(globalConfigDir, "mcp_config.json")
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		defaultMCP := map[string]any{
			"mcpServers": map[string]any{
				"filesystem-nas": map[string]any{
					"command": "npx",
					"args":    []string{"-y", "@modelcontextprotocol/server-filesystem", "/vol1"},
					"env":     map[string]string{},
				},
				"remote-context-http": map[string]any{
					"url": "https://mcp.antigravity.google/mcp",
				},
			},
		}
		if b, err := json.MarshalIndent(defaultMCP, "", "  "); err == nil {
			_ = os.WriteFile(mcpPath, b, 0644)
		}
	}

	// 3. Seed default Plugin if plugins dir is empty
	pluginsDir := filepath.Join(globalConfigDir, "plugins")
	_ = os.MkdirAll(pluginsDir, 0755)
	if entries, err := os.ReadDir(pluginsDir); err == nil && len(entries) == 0 {
		plugDir := filepath.Join(pluginsDir, "fnos-devkit")
		_ = os.MkdirAll(filepath.Join(plugDir, "rules"), 0755)
		_ = os.MkdirAll(filepath.Join(plugDir, "skills", "docker-compose-helper"), 0755)
		plugJSON := `{
  "name": "fnOS Developer Kit",
  "description": "Bundled fnOS NAS rules, Docker Compose workflow skill, and pre-tool safety hook for Antigravity.",
  "version": "1.0.0",
  "author": "Antigravity Harness",
  "homepage": "https://antigravity.google/docs/plugins"
}
`
		_ = os.WriteFile(filepath.Join(plugDir, "plugin.json"), []byte(plugJSON), 0644)
		_ = os.WriteFile(filepath.Join(plugDir, "rules", "AGENTS.md"), []byte("# fnOS DevKit Rules\n- Prefer `/vol1` or `/vol3` for persistent app data.\n- Never write secrets into git-tracked files.\n"), 0644)
		_ = os.WriteFile(filepath.Join(plugDir, "skills", "docker-compose-helper", "SKILL.md"), []byte("---\nname: docker-compose-helper\ndescription: Compose and validate Docker services on fnOS NAS.\n---\n# Docker Compose Helper\nValidate compose YAML with `docker compose config` before deployment.\n"), 0644)
	}

	// 4. Seed global GEMINI.md rule if missing
	globalRulePath := filepath.Join(globalConfigDir, "GEMINI.md")
	if _, err := os.Stat(globalRulePath); os.IsNotExist(err) {
		defaultRule := `# Antigravity Global Agent Rules (fnOS NAS)

## Environment Guidelines
- Target host is a fnOS (飞牛NAS) x86_64 Linux server.
- Keep logs clean, avoid unbounded disk writes on the root OS partition (` + "`/`" + `).
- Prefer structured JSON/YAML configs and atomic file writes when modifying persistent settings.

## Safety & Sandbox
- Never commit API keys, tokens, or passwords into repositories.
- Verify port availability before binding new network listeners.
`
		_ = os.WriteFile(globalRulePath, []byte(defaultRule), 0644)
	}

	// 5. Seed hooks.json if missing
	hooksPath := filepath.Join(globalConfigDir, "hooks.json")
	if _, err := os.Stat(hooksPath); os.IsNotExist(err) {
		defaultHooks := HooksDocument{
			Hooks: []HookRule{
				{
					ID:          "hook_secret_guard",
					Name:        "敏感凭据提交拦截",
					Event:       "PreToolUse",
					Matcher:     "run_command|edit_file",
					Command:     "echo '{\"decision\":\"allow\"}'",
					TimeoutSec:  10,
					Enabled:     true,
					Description: "在工具调用前校验命令与文件变更，防止敏感密钥泄露",
				},
				{
					ID:          "hook_session_audit",
					Name:        "会话启动环境审计",
					Event:       "SessionStart",
					Matcher:     ".*",
					Command:     "uname -a && date",
					TimeoutSec:  5,
					Enabled:     true,
					Description: "每次新建 Agent 会话时记录宿主机内核与时间快照",
				},
			},
		}
		if b, err := json.MarshalIndent(defaultHooks, "", "  "); err == nil {
			_ = os.WriteFile(hooksPath, b, 0644)
		}
	}
}

// ---------- 1. SKILLS ----------

type SkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type SkillItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"` // global | workspace | plugin
	Path        string   `json:"path"`
	SizeBytes   int64    `json:"size_bytes"`
	SubDirs     []string `json:"sub_dirs"`
	UpdatedAt   int64    `json:"updated_at"`
	Content     string   `json:"content,omitempty"`
}

var validSkillNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func parseSkillFrontmatter(raw string) (SkillFrontmatter, string) {
	var fm SkillFrontmatter
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return fm, raw
	}
	rest := trimmed[3:]
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, raw
	}
	header := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])
	_ = yaml.Unmarshal([]byte(header), &fm)
	return fm, body
}

func scanSkillsDir(baseDir, scope string) []SkillItem {
	out := make([]SkillItem, 0)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(baseDir, e.Name(), "SKILL.md")
		fi, err := os.Stat(skillFile)
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		fm, _ := parseSkillFrontmatter(string(raw))
		name := fm.Name
		if name == "" {
			name = e.Name()
		}
		desc := fm.Description
		if desc == "" {
			desc = "Antigravity custom skill"
		}
		subDirs := make([]string, 0)
		for _, sub := range []string{"scripts", "references", "examples", "resources"} {
			if st, err := os.Stat(filepath.Join(baseDir, e.Name(), sub)); err == nil && st.IsDir() {
				subDirs = append(subDirs, sub)
			}
		}
		out = append(out, SkillItem{
			Name:        name,
			Description: desc,
			Scope:       scope,
			Path:        skillFile,
			SizeBytes:   fi.Size(),
			SubDirs:     subDirs,
			UpdatedAt:   fi.ModTime().Unix(),
		})
	}
	return out
}

func ListAllSkills() []SkillItem {
	items := scanSkillsDir(filepath.Join(globalConfigDir, "skills"), "global")
	// Scan plugin bundled skills
	plugEntries, _ := os.ReadDir(filepath.Join(globalConfigDir, "plugins"))
	for _, pe := range plugEntries {
		if !pe.IsDir() {
			continue
		}
		pSkills := scanSkillsDir(filepath.Join(globalConfigDir, "plugins", pe.Name(), "skills"), "plugin:"+pe.Name())
		items = append(items, pSkills...)
	}
	// Scan registered workspaces
	for _, ws := range GetWorkspaceList() {
		if ws.Path != "" {
			wsSkills := scanSkillsDir(filepath.Join(ws.Path, ".agents", "skills"), "workspace")
			items = append(items, wsSkills...)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
	return items
}

func GetSkillDetail(scope, name string) (SkillItem, error) {
	for _, s := range ListAllSkills() {
		if s.Name == name && (scope == "" || s.Scope == scope) {
			raw, err := os.ReadFile(s.Path)
			if err != nil {
				return SkillItem{}, err
			}
			s.Content = string(raw)
			return s, nil
		}
	}
	return SkillItem{}, fmt.Errorf("未找到技能: %s", name)
}

func SaveSkill(scope, name, description, content string) (SkillItem, error) {
	customMu.Lock()
	defer customMu.Unlock()

	name = strings.ToLower(strings.TrimSpace(name))
	if !validSkillNameRe.MatchString(name) {
		return SkillItem{}, fmt.Errorf("技能名「%s」不合法，仅允许小写字母、数字、连字符和下划线", name)
	}
	if scope == "" || scope == "global" {
		scope = "global"
	}

	targetDir := filepath.Join(globalConfigDir, "skills", name)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return SkillItem{}, err
	}

	finalContent := content
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		if description == "" {
			description = "Custom Antigravity skill: " + name
		}
		finalContent = fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n", name, description, trimmed)
	}

	skillFile := filepath.Join(targetDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(finalContent), 0644); err != nil {
		return SkillItem{}, err
	}
	LogInfo("[定制中心] 已保存 Skill「%s」(%s)", name, skillFile)
	return GetSkillDetail(scope, name)
}

func DeleteSkill(scope, name string) error {
	customMu.Lock()
	defer customMu.Unlock()
	if strings.HasPrefix(scope, "plugin:") {
		return fmt.Errorf("插件内置技能请通过卸载插件进行移除")
	}
	name = strings.TrimSpace(name)
	if !validSkillNameRe.MatchString(name) {
		return fmt.Errorf("非法技能名")
	}
	targetDir := filepath.Join(globalConfigDir, "skills", name)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("技能不存在: %s", name)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	LogInfo("[定制中心] 已删除 Skill「%s」", name)
	return nil
}

// ---------- 2. MCP SERVERS (mcp_config.json) ----------

type MCPServerEntry struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type MCPConfigDoc struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

type MCPServerView struct {
	ID        string            `json:"id"`
	Transport string            `json:"transport"` // stdio | http
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func getMCPConfigPath() string {
	return filepath.Join(globalConfigDir, "mcp_config.json")
}

func ReadMCPConfig() ([]MCPServerView, string, error) {
	path := getMCPConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []MCPServerView{}, "{\n  \"mcpServers\": {}\n}", nil
		}
		return nil, "", err
	}
	var doc MCPConfigDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, string(raw), fmt.Errorf("mcp_config.json 解析失败: %w", err)
	}
	list := make([]MCPServerView, 0, len(doc.MCPServers))
	for id, srv := range doc.MCPServers {
		transport := "stdio"
		if srv.URL != "" {
			transport = "http"
		}
		list = append(list, MCPServerView{
			ID:        id,
			Transport: transport,
			Command:   srv.Command,
			Args:      srv.Args,
			Env:       srv.Env,
			URL:       srv.URL,
			Headers:   srv.Headers,
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list, string(raw), nil
}

func SaveMCPServer(id string, entry MCPServerEntry) error {
	customMu.Lock()
	defer customMu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("MCP Server ID 不能为空")
	}
	if entry.Command == "" && entry.URL == "" {
		return fmt.Errorf("请填写 Stdio 启动命令 (command) 或 Streamable HTTP 地址 (url)")
	}
	var doc MCPConfigDoc
	if raw, err := os.ReadFile(getMCPConfigPath()); err == nil {
		_ = json.Unmarshal(raw, &doc)
	}
	if doc.MCPServers == nil {
		doc.MCPServers = make(map[string]MCPServerEntry)
	}
	doc.MCPServers[id] = entry
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(getMCPConfigPath(), out, 0644); err != nil {
		return err
	}
	LogInfo("[定制中心] 已保存 MCP Server「%s」", id)
	return nil
}

func SaveMCPRawJSON(rawJSON string) error {
	customMu.Lock()
	defer customMu.Unlock()
	var doc MCPConfigDoc
	if err := json.Unmarshal([]byte(rawJSON), &doc); err != nil {
		return fmt.Errorf("JSON 语法错误: %w", err)
	}
	if doc.MCPServers == nil {
		doc.MCPServers = make(map[string]MCPServerEntry)
	}
	pretty, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(getMCPConfigPath(), pretty, 0644); err != nil {
		return err
	}
	LogInfo("[定制中心] 已更新 mcp_config.json (%d 个 MCP 服务)", len(doc.MCPServers))
	return nil
}

func DeleteMCPServer(id string) error {
	customMu.Lock()
	defer customMu.Unlock()
	var doc MCPConfigDoc
	if raw, err := os.ReadFile(getMCPConfigPath()); err == nil {
		_ = json.Unmarshal(raw, &doc)
	}
	if doc.MCPServers == nil {
		return fmt.Errorf("MCP 服务不存在: %s", id)
	}
	if _, ok := doc.MCPServers[id]; !ok {
		return fmt.Errorf("MCP 服务不存在: %s", id)
	}
	delete(doc.MCPServers, id)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(getMCPConfigPath(), out, 0644); err != nil {
		return err
	}
	LogInfo("[定制中心] 已删除 MCP Server「%s」", id)
	return nil
}

type MCPTestResult struct {
	ID         string `json:"id"`
	OK         bool   `json:"ok"`
	Transport  string `json:"transport"`
	LatencyMs  int64  `json:"latency_ms"`
	Message    string `json:"message"`
	ResolvedTo string `json:"resolved_to,omitempty"`
}

func TestMCPServer(id string) MCPTestResult {
	list, _, err := ReadMCPConfig()
	if err != nil {
		return MCPTestResult{ID: id, OK: false, Message: err.Error()}
	}
	var target *MCPServerView
	for i := range list {
		if list[i].ID == id {
			target = &list[i]
			break
		}
	}
	if target == nil {
		return MCPTestResult{ID: id, OK: false, Message: "未找到 MCP Server: " + id}
	}
	start := time.Now()
	if target.Transport == "http" {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest(http.MethodGet, target.URL, nil)
		if err != nil {
			return MCPTestResult{ID: id, OK: false, Transport: "http", Message: "无效 URL: " + err.Error()}
		}
		for k, v := range target.Headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return MCPTestResult{ID: id, OK: false, Transport: "http", LatencyMs: elapsed, Message: "连接失败: " + err.Error()}
		}
		_ = resp.Body.Close()
		ok := resp.StatusCode < 500
		return MCPTestResult{
			ID:        id,
			OK:        ok,
			Transport: "http",
			LatencyMs: elapsed,
			Message:   fmt.Sprintf("HTTP 端点响应状态码 %d (%dms)", resp.StatusCode, elapsed),
		}
	}

	// Stdio transport test: verify command binary exists on host
	resolved, err := exec.LookPath(target.Command)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return MCPTestResult{
			ID:        id,
			OK:        false,
			Transport: "stdio",
			LatencyMs: elapsed,
			Message:   fmt.Sprintf("未在系统 PATH 中找到可执行文件 `%s`", target.Command),
		}
	}
	return MCPTestResult{
		ID:         id,
		OK:         true,
		Transport:  "stdio",
		LatencyMs:  elapsed,
		ResolvedTo: resolved,
		Message:    fmt.Sprintf("可执行文件就绪: %s (参数: %s)", resolved, strings.Join(target.Args, " ")),
	}
}

// ---------- 3. PLUGINS (plugins/<name>/plugin.json + config.json) ----------

type PluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Author      any    `json:"author,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Repository  any    `json:"repository,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type PluginItem struct {
	DirName     string `json:"dir_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
	Enabled     bool   `json:"enabled"`
	Path        string `json:"path"`
	SkillsCount int    `json:"skills_count"`
	RulesCount  int    `json:"rules_count"`
	HasMCP      bool   `json:"has_mcp"`
	HasHooks    bool   `json:"has_hooks"`
}

type AntigravityUserConfig struct {
	Plugins map[string]struct {
		Enabled bool `json:"enabled"`
	} `json:"plugins"`
}

func getUserPluginsStatePath() string {
	return filepath.Join(globalConfigDir, "config.json")
}

func loadPluginStates() map[string]bool {
	out := make(map[string]bool)
	raw, err := os.ReadFile(getUserPluginsStatePath())
	if err != nil {
		return out
	}
	var cfg AntigravityUserConfig
	if err := json.Unmarshal(raw, &cfg); err == nil {
		for k, v := range cfg.Plugins {
			out[k] = v.Enabled
		}
	}
	return out
}

func savePluginState(dirName string, enabled bool) error {
	path := getUserPluginsStatePath()
	var full map[string]any
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &full)
	}
	if full == nil {
		full = make(map[string]any)
	}
	plugs, ok := full["plugins"].(map[string]any)
	if !ok || plugs == nil {
		plugs = make(map[string]any)
	}
	plugs[dirName] = map[string]any{"enabled": enabled}
	full["plugins"] = plugs
	data, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func extractAuthorString(a any) string {
	switch v := a.(type) {
	case string:
		return v
	case map[string]any:
		if n, ok := v["name"].(string); ok {
			return n
		}
	}
	return ""
}

func ListAllPlugins() []PluginItem {
	pluginsDir := filepath.Join(globalConfigDir, "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return []PluginItem{}
	}
	states := loadPluginStates()
	out := make([]PluginItem, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(pluginsDir, e.Name())
		manifestPath := filepath.Join(dirPath, "plugin.json")
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m PluginManifest
		_ = json.Unmarshal(raw, &m)
		displayName := m.Name
		if displayName == "" {
			displayName = e.Name()
		}
		enabled := !m.Disabled
		if override, exists := states[e.Name()]; exists {
			enabled = override
		}
		skillsCnt := len(scanSkillsDir(filepath.Join(dirPath, "skills"), "plugin"))
		rulesCnt := 0
		if rEntries, err := os.ReadDir(filepath.Join(dirPath, "rules")); err == nil {
			for _, re := range rEntries {
				if !re.IsDir() && strings.HasSuffix(strings.ToLower(re.Name()), ".md") {
					rulesCnt++
				}
			}
		}
		_, mcpErr := os.Stat(filepath.Join(dirPath, "mcp_config.json"))
		_, hooksErr := os.Stat(filepath.Join(dirPath, "hooks.json"))

		out = append(out, PluginItem{
			DirName:     e.Name(),
			Name:        displayName,
			Description: m.Description,
			Version:     m.Version,
			Author:      extractAuthorString(m.Author),
			Homepage:    m.Homepage,
			Enabled:     enabled,
			Path:        dirPath,
			SkillsCount: skillsCnt,
			RulesCount:  rulesCnt,
			HasMCP:      mcpErr == nil,
			HasHooks:    hooksErr == nil,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DirName < out[j].DirName })
	return out
}

func TogglePlugin(dirName string, enabled bool) error {
	customMu.Lock()
	defer customMu.Unlock()
	dirPath := filepath.Join(globalConfigDir, "plugins", dirName)
	if _, err := os.Stat(filepath.Join(dirPath, "plugin.json")); err != nil {
		return fmt.Errorf("插件不存在: %s", dirName)
	}
	if err := savePluginState(dirName, enabled); err != nil {
		return err
	}
	stateWord := "启用"
	if !enabled {
		stateWord = "停用"
	}
	LogInfo("[定制中心] 插件「%s」已%s (写入 config.json)", dirName, stateWord)
	return nil
}

func InstallPlugin(dirName, displayName, description, author, gitURL string) (PluginItem, error) {
	customMu.Lock()
	defer customMu.Unlock()
	dirName = strings.ToLower(strings.TrimSpace(dirName))
	if dirName == "" && gitURL != "" {
		base := filepath.Base(strings.TrimSuffix(strings.TrimSpace(gitURL), ".git"))
		dirName = strings.ToLower(regexp.MustCompile(`[^a-z0-9_-]`).ReplaceAllString(base, "-"))
	}
	if !validSkillNameRe.MatchString(dirName) {
		return PluginItem{}, fmt.Errorf("插件标识名「%s」不合法，仅允许小写字母、数字和连字符", dirName)
	}
	targetDir := filepath.Join(globalConfigDir, "plugins", dirName)
	if gitURL != "" {
		if _, err := exec.LookPath("git"); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", gitURL, targetDir)
			if out, err := cmd.CombinedOutput(); err != nil {
				return PluginItem{}, fmt.Errorf("Git 克隆插件失败: %s", strings.TrimSpace(string(out)))
			}
			// Ensure plugin.json exists
			if _, err := os.Stat(filepath.Join(targetDir, "plugin.json")); os.IsNotExist(err) {
				m := PluginManifest{Name: dirName, Description: "Imported from " + gitURL, Version: "1.0.0", Author: author}
				if b, err := json.MarshalIndent(m, "", "  "); err == nil {
					_ = os.WriteFile(filepath.Join(targetDir, "plugin.json"), b, 0644)
				}
			}
		} else {
			return PluginItem{}, fmt.Errorf("宿主机未安装 git，无法克隆远程仓库")
		}
	} else {
		_ = os.MkdirAll(filepath.Join(targetDir, "rules"), 0755)
		_ = os.MkdirAll(filepath.Join(targetDir, "skills"), 0755)
		if displayName == "" {
			displayName = dirName
		}
		if description == "" {
			description = "Custom Antigravity plugin bundle"
		}
		if author == "" {
			author = "fnOS Admin"
		}
		m := PluginManifest{
			Name:        displayName,
			Description: description,
			Version:     "1.0.0",
			Author:      author,
		}
		b, _ := json.MarshalIndent(m, "", "  ")
		if err := os.WriteFile(filepath.Join(targetDir, "plugin.json"), b, 0644); err != nil {
			return PluginItem{}, err
		}
		_ = os.WriteFile(filepath.Join(targetDir, "rules", "AGENTS.md"), []byte("# "+displayName+" Rules\n"), 0644)
	}
	_ = savePluginState(dirName, true)
	LogInfo("[定制中心] 已安装并启用插件「%s」", dirName)
	for _, p := range ListAllPlugins() {
		if p.DirName == dirName {
			return p, nil
		}
	}
	return PluginItem{DirName: dirName, Name: displayName, Enabled: true}, nil
}

func DeletePlugin(dirName string) error {
	customMu.Lock()
	defer customMu.Unlock()
	if !validSkillNameRe.MatchString(dirName) {
		return fmt.Errorf("非法插件名")
	}
	targetDir := filepath.Join(globalConfigDir, "plugins", dirName)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("插件不存在: %s", dirName)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	LogInfo("[定制中心] 已卸载插件「%s」", dirName)
	return nil
}

// ---------- 4. RULES (GEMINI.md / AGENTS.md / .agents/rules/*.md) ----------

type RuleFileItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Scope        string  `json:"scope"` // global | workspace
	Path         string  `json:"path"`
	SizeBytes    int64   `json:"size_bytes"`
	LimitBytes   int64   `json:"limit_bytes"`
	SizePercent  float64 `json:"size_percent"`
	EstTokens    int     `json:"est_tokens"`
	UpdatedAt    int64   `json:"updated_at"`
	Content      string  `json:"content"`
}

const MaxRuleBytes = 24 * 1024 // 24KB single-file Antigravity rule limit

func ListAllRules() []RuleFileItem {
	rules := make([]RuleFileItem, 0)
	addIfExists := func(id, name, scope, p string) {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			return
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return
		}
		sz := fi.Size()
		pct := float64(sz) / float64(MaxRuleBytes) * 100
		if pct > 100 {
			pct = 100
		}
		rules = append(rules, RuleFileItem{
			ID:          id,
			Name:        name,
			Scope:       scope,
			Path:        p,
			SizeBytes:   sz,
			LimitBytes:  MaxRuleBytes,
			SizePercent: pct,
			EstTokens:   len(raw)/3 + 1,
			UpdatedAt:   fi.ModTime().Unix(),
			Content:     string(raw),
		})
	}

	addIfExists("global:GEMINI.md", "GEMINI.md (全局规则)", "global", filepath.Join(globalConfigDir, "GEMINI.md"))
	addIfExists("global:AGENTS.md", "AGENTS.md (全局代理准则)", "global", filepath.Join(globalConfigDir, "AGENTS.md"))

	// Also list custom rules inside globalConfigDir/rules/*.md
	if entries, err := os.ReadDir(filepath.Join(globalConfigDir, "rules")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
				addIfExists("global:rules/"+e.Name(), e.Name(), "global", filepath.Join(globalConfigDir, "rules", e.Name()))
			}
		}
	}

	// Scan registered workspaces
	for _, ws := range GetWorkspaceList() {
		if ws.Path == "" {
			continue
		}
		addIfExists("ws:"+ws.ID+":GEMINI.md", ws.Title+"/GEMINI.md", "workspace", filepath.Join(ws.Path, "GEMINI.md"))
		addIfExists("ws:"+ws.ID+":AGENTS.md", ws.Title+"/AGENTS.md", "workspace", filepath.Join(ws.Path, "AGENTS.md"))
	}
	return rules
}

func SaveRuleFile(fileName, content string) (RuleFileItem, error) {
	customMu.Lock()
	defer customMu.Unlock()

	if len([]byte(content)) > MaxRuleBytes {
		return RuleFileItem{}, fmt.Errorf("规则文件超过 Antigravity 单文件 24KB 上限 (当前 %d 字节)", len([]byte(content)))
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "GEMINI.md"
	}

	var targetPath string
	switch fileName {
	case "GEMINI.md", "global:GEMINI.md":
		targetPath = filepath.Join(globalConfigDir, "GEMINI.md")
	case "AGENTS.md", "global:AGENTS.md":
		targetPath = filepath.Join(globalConfigDir, "AGENTS.md")
	default:
		clean := filepath.Base(fileName)
		if !strings.HasSuffix(strings.ToLower(clean), ".md") {
			clean += ".md"
		}
		_ = os.MkdirAll(filepath.Join(globalConfigDir, "rules"), 0755)
		targetPath = filepath.Join(globalConfigDir, "rules", clean)
	}

	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return RuleFileItem{}, err
	}
	LogInfo("[定制中心] 已保存规则文件 %s (%d bytes)", targetPath, len(content))
	for _, r := range ListAllRules() {
		if r.Path == targetPath {
			return r, nil
		}
	}
	return RuleFileItem{Path: targetPath, Content: content}, nil
}

// ---------- 5. HOOKS (hooks.json) ----------

type HookRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Event       string `json:"event"` // SessionStart | PreToolUse | PostToolUse | PreInvocation | PostInvocation | Stop
	Matcher     string `json:"matcher"`
	Command     string `json:"command"`
	TimeoutSec  int    `json:"timeout_sec"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type HooksDocument struct {
	Hooks []HookRule `json:"hooks"`
}

func getHooksFilePath() string {
	return filepath.Join(globalConfigDir, "hooks.json")
}

func ListHooks() ([]HookRule, error) {
	raw, err := os.ReadFile(getHooksFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []HookRule{}, nil
		}
		return nil, err
	}
	var doc HooksDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Hooks == nil {
		return []HookRule{}, nil
	}
	return doc.Hooks, nil
}

func SaveHooks(hooks []HookRule) error {
	customMu.Lock()
	defer customMu.Unlock()
	for i := range hooks {
		if hooks[i].ID == "" {
			hooks[i].ID = generateShortID("hook")
		}
		if hooks[i].TimeoutSec <= 0 {
			hooks[i].TimeoutSec = 10
		}
	}
	doc := HooksDocument{Hooks: hooks}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(getHooksFilePath(), data, 0644); err != nil {
		return err
	}
	LogInfo("[定制中心] 已保存生命周期 Hooks 配置 (%d 条)", len(hooks))
	return nil
}

func GetCustomizationsSummary() CustomizationsSummary {
	skills := ListAllSkills()
	mcps, _, _ := ReadMCPConfig()
	plugs := ListAllPlugins()
	rules := ListAllRules()
	hooks, _ := ListHooks()

	enabledPlugs := 0
	for _, p := range plugs {
		if p.Enabled {
			enabledPlugs++
		}
	}
	var totalBytes int64
	totalTokens := 0
	for _, r := range rules {
		totalBytes += r.SizeBytes
		totalTokens += r.EstTokens
	}
	return CustomizationsSummary{
		SkillsCount:   len(skills),
		MCPCount:      len(mcps),
		PluginsCount:  len(plugs),
		EnabledPlugs:  enabledPlugs,
		RulesCount:    len(rules),
		HooksCount:    len(hooks),
		TotalRuleKB:   float64(totalBytes) / 1024.0,
		EstRuleTokens: totalTokens,
	}
}
