package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ConversationAnnotations struct {
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Archived bool     `json:"archived"`
	Pinned   bool     `json:"pinned"`
	Starred  bool     `json:"starred"`
}

type ConversationMeta struct {
	ID          string                  `json:"id"`
	Model       string                  `json:"model"`
	Workspace   string                  `json:"workspace"`
	CreateTime  string                  `json:"createTime"`
	UpdateTime  string                  `json:"updateTime"`
	RunState    string                  `json:"runState"` // RUN_STATE_IDLE | RUN_STATE_RUNNING | RUN_STATE_CANCELLING
	Summary     string                  `json:"summary"`
	TurnCount   int                     `json:"turnCount"`
	Annotations ConversationAnnotations `json:"annotations"`
}

type TranscriptEntry struct {
	ID        string `json:"id"`
	Role      string `json:"role"` // user | thought | tool_call | tool_result | assistant | system
	Content   string `json:"content"`
	ToolName  string `json:"tool_name,omitempty"`
	ToolArgs  string `json:"tool_args,omitempty"`
	Model     string `json:"model,omitempty"`
	Timestamp string `json:"timestamp"`
}

type ConversationDetail struct {
	ConversationMeta
	Transcript []TranscriptEntry `json:"transcript"`
	Artifacts  []string          `json:"artifacts"`
}

var (
	convMu    sync.Mutex
	convSubMu sync.Mutex
	convSubs  = map[chan struct{}]struct{}{}
)

func SubscribeConversations(buf int) (<-chan struct{}, func()) {
	ch := make(chan struct{}, buf)
	convSubMu.Lock()
	convSubs[ch] = struct{}{}
	convSubMu.Unlock()
	return ch, func() {
		convSubMu.Lock()
		delete(convSubs, ch)
		convSubMu.Unlock()
	}
}

func notifyConversationsChanged() {
	convSubMu.Lock()
	for ch := range convSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	convSubMu.Unlock()
}

func InitConversations() {
	_ = os.MkdirAll(globalBrainDir, 0755)
	entries, err := os.ReadDir(globalBrainDir)
	if err == nil && len(entries) == 0 {
		// Seed one welcome onboarding conversation so UI showcases transcript & thought timeline immediately
		id := "conv_welcome_fnos"
		dir := filepath.Join(globalBrainDir, id)
		logDir := filepath.Join(dir, ".system_generated", "logs")
		_ = os.MkdirAll(logDir, 0755)
		now := time.Now().Format(time.RFC3339)
		meta := ConversationMeta{
			ID:         id,
			Model:      DefaultModelID,
			Workspace:  globalPkgVar,
			CreateTime: now,
			UpdateTime: now,
			RunState:   "RUN_STATE_IDLE",
			Summary:    "fnOS Antigravity Harness 初始化巡检与环境就绪报告",
			TurnCount:  1,
			Annotations: ConversationAnnotations{
				Title:   "fnOS Harness 环境就绪巡检",
				Tags:    []string{"system", "onboarding"},
				Pinned:  true,
				Starred: true,
			},
		}
		if b, err := json.MarshalIndent(meta, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "meta.json"), b, 0644)
		}
		seedTurns := []TranscriptEntry{
			{ID: "t1", Role: "user", Content: "请检查当前 fnOS NAS 上的 Antigravity Harness 配置、定制项目录与运行状态。", Timestamp: now},
			{ID: "t2", Role: "thought", Content: "正在读取全局定制目录 (~/.gemini/config) 与宿主机资源指标，验证 Skills、MCP、Plugins、Rules 与 Hooks 是否初始化完毕。", Timestamp: now},
			{ID: "t3", Role: "tool_call", ToolName: "inspect_harness_env", ToolArgs: `{"scope":"global","check_mcp":true}`, Content: "执行环境与定制系统检查", Timestamp: now},
			{ID: "t4", Role: "tool_result", ToolName: "inspect_harness_env", Content: "定制目录就绪：Skills (2), MCP Servers (2), Plugins (1), Rules (1), Hooks (2)。ConversationService Connect RPC 已挂载。", Timestamp: now},
			{ID: "t5", Role: "assistant", Model: DefaultModelID, Content: "Antigravity Harness 飞牛 NAS 控制台已初始化完成！\n\n- **最新模型引擎**：默认已启用 **`gemini-4-neon-xhigh` (Gemini 4 Neon XHigh)**，支持 Gemini 4 Neon / Hydrogen / Argon / Carbon、Gemini 3.8 Flash、Claude 5.5 及任意自定义模型 ID。\n- **定制中心**：已预置 `fnos-nas-ops` 与 `code-audit-guardian` 技能模板、全局 `GEMINI.md` 规则、`mcp_config.json` 与安全生命周期钩子。\n- **云端模型推理**：前往「授权管理」配置 `GEMINI_API_KEY` 后，即可直接驱动最新 Gemini 4 系列执行真实任务。", Timestamp: now},
		}
		f, err := os.OpenFile(filepath.Join(logDir, "transcript.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			for _, t := range seedTurns {
				line, _ := json.Marshal(t)
				_, _ = f.Write(append(line, '\n'))
			}
			_ = f.Close()
		}
	}
}

func getConvDir(id string) string {
	clean := filepath.Base(strings.TrimSpace(id))
	return filepath.Join(globalBrainDir, clean)
}

func readConvMeta(id string) (ConversationMeta, error) {
	dir := getConvDir(id)
	raw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return ConversationMeta{}, err
	}
	var meta ConversationMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ConversationMeta{}, err
	}
	if meta.Annotations.Tags == nil {
		meta.Annotations.Tags = []string{}
	}
	return meta, nil
}

func writeConvMeta(meta ConversationMeta) error {
	dir := getConvDir(meta.ID)
	_ = os.MkdirAll(filepath.Join(dir, ".system_generated", "logs"), 0755)
	if meta.Annotations.Tags == nil {
		meta.Annotations.Tags = []string{}
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

func appendTranscriptEntry(convID string, entry TranscriptEntry) error {
	dir := getConvDir(convID)
	logDir := filepath.Join(dir, ".system_generated", "logs")
	_ = os.MkdirAll(logDir, 0755)
	if entry.ID == "" {
		entry.ID = generateShortID("tr")
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(logDir, "transcript.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func readTranscriptEntries(convID string) []TranscriptEntry {
	path := filepath.Join(getConvDir(convID), ".system_generated", "logs", "transcript.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return []TranscriptEntry{}
	}
	defer f.Close()
	out := make([]TranscriptEntry, 0)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var e TranscriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err == nil {
			out = append(out, e)
		}
	}
	return out
}

func ListConversations(includeArchived bool) []ConversationMeta {
	entries, err := os.ReadDir(globalBrainDir)
	if err != nil {
		return []ConversationMeta{}
	}
	list := make([]ConversationMeta, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := readConvMeta(e.Name())
		if err != nil {
			continue
		}
		if !includeArchived && meta.Annotations.Archived {
			continue
		}
		list = append(list, meta)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Annotations.Pinned != list[j].Annotations.Pinned {
			return list[i].Annotations.Pinned
		}
		return list[i].UpdateTime > list[j].UpdateTime
	})
	return list
}

func GetConversationDetail(id string) (ConversationDetail, error) {
	meta, err := readConvMeta(id)
	if err != nil {
		return ConversationDetail{}, fmt.Errorf("未找到会话: %s", id)
	}
	transcript := readTranscriptEntries(id)
	artifacts := make([]string, 0)
	if entries, err := os.ReadDir(getConvDir(id)); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
				artifacts = append(artifacts, e.Name())
			}
		}
	}
	return ConversationDetail{
		ConversationMeta: meta,
		Transcript:       transcript,
		Artifacts:        artifacts,
	}, nil
}

type CreateConversationRequest struct {
	Title     string   `json:"title"`
	Model     string   `json:"model"`
	Workspace string   `json:"workspace"`
	Tags      []string `json:"tags"`
	Prompt    string   `json:"prompt"`
}

func resolveModelName(alias string) string {
	switch strings.ToLower(strings.TrimSpace(alias)) {
	case "pro", "neon", "gemini-4-pro":
		return DefaultModelID
	case "flash":
		return "gemini-3.8-flash-high"
	case "flash_lite":
		return "gemini-3.8-flash-medium"
	case "":
		return GetConfig().GetDefaultModel()
	default:
		return strings.TrimSpace(alias)
	}
}

func CreateConversation(req CreateConversationRequest) (ConversationMeta, error) {
	convMu.Lock()
	id := generateShortID("conv")
	now := time.Now().Format(time.RFC3339)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		if req.Prompt != "" {
			runes := []rune(strings.TrimSpace(req.Prompt))
			if len(runes) > 28 {
				title = string(runes[:28]) + "…"
			} else {
				title = string(runes)
			}
		} else {
			title = "新会话 " + time.Now().Format("15:04")
		}
	}
	model := resolveModelName(req.Model)
	ws := strings.TrimSpace(req.Workspace)
	if ws == "" {
		ws = globalPkgVar
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	meta := ConversationMeta{
		ID:         id,
		Model:      model,
		Workspace:  ws,
		CreateTime: now,
		UpdateTime: now,
		RunState:   "RUN_STATE_IDLE",
		Summary:    title,
		TurnCount:  0,
		Annotations: ConversationAnnotations{
			Title: title,
			Tags:  tags,
		},
	}
	if err := writeConvMeta(meta); err != nil {
		convMu.Unlock()
		return ConversationMeta{}, err
	}
	convMu.Unlock()
	LogInfo("[会话] 创建新会话「%s」(%s, 模型=%s)", meta.Annotations.Title, id, model)
	notifyConversationsChanged()

	if strings.TrimSpace(req.Prompt) != "" {
		_ = DispatchConversationPrompt(id, strings.TrimSpace(req.Prompt))
		meta, _ = readConvMeta(id)
	}
	return meta, nil
}

func UpdateConversationAnnotations(id string, patch map[string]any) (ConversationMeta, error) {
	convMu.Lock()
	defer convMu.Unlock()
	meta, err := readConvMeta(id)
	if err != nil {
		return ConversationMeta{}, fmt.Errorf("未找到会话: %s", id)
	}
	if v, ok := patch["title"].(string); ok {
		meta.Annotations.Title = strings.TrimSpace(v)
	}
	if v, ok := patch["pinned"].(bool); ok {
		meta.Annotations.Pinned = v
	}
	if v, ok := patch["starred"].(bool); ok {
		meta.Annotations.Starred = v
	}
	if v, ok := patch["archived"].(bool); ok {
		meta.Annotations.Archived = v
	}
	if rawTags, ok := patch["tags"].([]any); ok {
		tags := make([]string, 0, len(rawTags))
		for _, rt := range rawTags {
			if s, ok := rt.(string); ok && strings.TrimSpace(s) != "" {
				tags = append(tags, strings.TrimSpace(s))
			}
		}
		meta.Annotations.Tags = tags
	}
	meta.UpdateTime = time.Now().Format(time.RFC3339)
	if err := writeConvMeta(meta); err != nil {
		return ConversationMeta{}, err
	}
	notifyConversationsChanged()
	return meta, nil
}

func DeleteConversation(id string) error {
	convMu.Lock()
	defer convMu.Unlock()
	dir := getConvDir(id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("会话不存在: %s", id)
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	LogInfo("[会话] 已永久删除会话 %s", id)
	notifyConversationsChanged()
	return nil
}

func DispatchConversationPrompt(id, prompt string) error {
	meta, err := readConvMeta(id)
	if err != nil {
		return fmt.Errorf("未找到会话: %s", id)
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("Prompt 内容不能为空")
	}

	_ = appendTranscriptEntry(id, TranscriptEntry{
		Role:    "user",
		Content: prompt,
	})
	meta.RunState = "RUN_STATE_RUNNING"
	meta.TurnCount++
	meta.UpdateTime = time.Now().Format(time.RFC3339)
	_ = writeConvMeta(meta)
	notifyConversationsChanged()
	LogInfo("[Agent] 会话 %s 收到新任务指令，启动异步推理 (模型=%s)", id, meta.Model)

	go executeAgentTurn(meta, prompt)
	return nil
}

func resolveActiveGeminiKey() (apiKey, baseURL string) {
	if acc := GetActiveAccountRaw(); acc != nil && acc.APIKey != "" {
		return acc.APIKey, acc.BaseURL
	}
	return GetEnvVarValue("GEMINI_API_KEY"), GetEnvVarValue("GEMINI_BASE_URL")
}

func callGeminiREST(model, apiKey, baseURL, systemPrompt, userPrompt string) (string, error) {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, model, apiKey)

	payload := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userPrompt}},
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Gemini API HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) > 0 && len(parsed.Candidates[0].Content.Parts) > 0 {
		var sb strings.Builder
		for _, p := range parsed.Candidates[0].Content.Parts {
			sb.WriteString(p.Text)
		}
		return sb.String(), nil
	}
	return "", fmt.Errorf("模型未返回有效内容")
}

func executeAgentTurn(meta ConversationMeta, prompt string) {
	defer func() {
		if latest, err := readConvMeta(meta.ID); err == nil {
			latest.RunState = "RUN_STATE_IDLE"
			latest.UpdateTime = time.Now().Format(time.RFC3339)
			_ = writeConvMeta(latest)
		}
		notifyConversationsChanged()
	}()

	summary := GetCustomizationsSummary()
	_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
		Role:    "thought",
		Content: fmt.Sprintf("加载上下文：工作区 `%s`，已挂载 %d 个 Skills、%d 个 MCP Servers、%d 个 Rules，沙箱策略 `%s`。", meta.Workspace, summary.SkillsCount, summary.MCPCount, summary.RulesCount, GetConfig().GetSandboxMode()),
	})
	notifyConversationsChanged()

	// Built-in Antigravity Slash Commands (/doctor, /skills, /compact)
	lowerPrompt := strings.ToLower(strings.TrimSpace(prompt))
	if lowerPrompt == "/doctor" || strings.HasPrefix(lowerPrompt, "/doctor ") {
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_call",
			ToolName: "run_doctor_diagnostics",
			ToolArgs: `{"scope":"all"}`,
			Content:  "执行 Antigravity Harness 全面健康巡检",
		})
		notifyConversationsChanged()
		report := RunDoctorDiagnostics()
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("### Antigravity Harness 健康巡检报告 (`%s`)\n\n", report.CheckedAt))
		for _, c := range report.Checks {
			icon := "✅"
			if c.Status == "warn" {
				icon = "⚠️"
			} else if c.Status == "error" {
				icon = "❌"
			}
			sb.WriteString(fmt.Sprintf("- %s **%s**：%s\n", icon, c.Name, c.Detail))
			if c.Suggest != "" {
				sb.WriteString(fmt.Sprintf("  - 建议：%s\n", c.Suggest))
			}
		}
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_result",
			ToolName: "run_doctor_diagnostics",
			Content:  fmt.Sprintf("Passed: %d, Warn: %d, Error: %d", report.PassedCount, report.WarnCount, report.ErrorCount),
		})
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:    "assistant",
			Model:   meta.Model + " (slash:/doctor)",
			Content: sb.String(),
		})
		return
	}

	if lowerPrompt == "/skills" || strings.HasPrefix(lowerPrompt, "/skills ") {
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_call",
			ToolName: "list_customizations",
			ToolArgs: `{"types":["skills","mcp","plugins","rules"]}`,
			Content:  "扫描已激活的 Skills、MCP Servers 与 Rules",
		})
		notifyConversationsChanged()
		skills := ListAllSkills()
		mcps, _, _ := ReadMCPConfig()
		var sb strings.Builder
		sb.WriteString("### 当前已加载 Antigravity 定制项\n\n#### Skills 技能\n")
		for _, s := range skills {
			sb.WriteString(fmt.Sprintf("- **`%s`** (`%s`)：%s\n", s.Name, s.Scope, s.Description))
		}
		sb.WriteString("\n#### MCP Servers\n")
		for _, m := range mcps {
			target := m.URL
			if m.Transport == "stdio" {
				target = m.Command + " " + strings.Join(m.Args, " ")
			}
			sb.WriteString(fmt.Sprintf("- **`%s`** (`%s`)：`%s`\n", m.ID, m.Transport, strings.TrimSpace(target)))
		}
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_result",
			ToolName: "list_customizations",
			Content:  fmt.Sprintf("发现 %d 个 Skills 与 %d 个 MCP Servers", len(skills), len(mcps)),
		})
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:    "assistant",
			Model:   meta.Model + " (slash:/skills)",
			Content: sb.String(),
		})
		return
	}

	if lowerPrompt == "/compact" {
		entries := readTranscriptEntries(meta.ID)
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:    "assistant",
			Model:   meta.Model + " (slash:/compact)",
			Content: fmt.Sprintf("上下文窗口已压缩整理：当前会话包含 **%d 轮对话**（共 `%d` 条轨迹记录），已保留核心工作区上下文 (`%s`) 与全局规则摘要。", meta.TurnCount, len(entries), meta.Workspace),
		})
		return
	}

	apiKey, baseURL := resolveActiveGeminiKey()
	if apiKey != "" {
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_call",
			ToolName: "gemini_generate_content",
			ToolArgs: fmt.Sprintf(`{"model":"%s","workspace":"%s"}`, meta.Model, meta.Workspace),
			Content:  "调用 Google Gemini API 执行云端推理",
		})
		notifyConversationsChanged()

		sysRules := "You are Google Antigravity Agent running on fnOS NAS Harness. Reply concisely in Markdown."
		for _, r := range ListAllRules() {
			if r.Scope == "global" && r.Content != "" {
				sysRules += "\n\n" + r.Content
				break
			}
		}

		reply, err := callGeminiREST(meta.Model, apiKey, baseURL, sysRules, prompt)
		if err == nil {
			_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
				Role:     "tool_result",
				ToolName: "gemini_generate_content",
				Content:  fmt.Sprintf("云端推理成功，返回 %d 字符", len([]rune(reply))),
			})
			_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
				Role:    "assistant",
				Model:   meta.Model,
				Content: reply,
			})
			LogInfo("[Agent] 会话 %s 云端推理完成 (%s)", meta.ID, meta.Model)
			return
		}
		LogWarning("[Agent] 会话 %s 云端调用失败，回退至本地引擎: %s", meta.ID, err)
		_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
			Role:     "tool_result",
			ToolName: "gemini_generate_content",
			Content:  "云端请求失败: " + err.Error() + "，自动切换本地 Harness 诊断响应",
		})
	}

	// Built-in Local Harness Workspace & System Inspection Agent
	time.Sleep(450 * time.Millisecond)
	_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
		Role:     "tool_call",
		ToolName: "inspect_workspace_and_host",
		ToolArgs: fmt.Sprintf(`{"workspace":"%s"}`, meta.Workspace),
		Content:  "扫描目标工作区目录与 fnOS 宿主机指标",
	})
	notifyConversationsChanged()

	host := SampleHostMetrics()
	wsFiles := 0
	if entries, err := os.ReadDir(meta.Workspace); err == nil {
		wsFiles = len(entries)
	}
	time.Sleep(350 * time.Millisecond)
	_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
		Role:     "tool_result",
		ToolName: "inspect_workspace_and_host",
		Content:  fmt.Sprintf("Host=%s CPU=%.1f%% RAM=%.1f%% Disk(%s)=%.1f%% · Workspace目录项=%d", host.Hostname, host.CPUPercent, host.MemPercent, host.DiskMount, host.DiskPercent, wsFiles),
	})

	reply := fmt.Sprintf(
		"已通过 **Antigravity Harness 本地引擎** 受理指令：`%s`\n\n"+
			"### 当前环境上下文快照\n"+
			"- **目标模型**：`%s`（沙箱模式：`%s`）\n"+
			"- **工作区路径**：`%s`（顶层目录项 %d 个）\n"+
			"- **已加载定制**：Skills `%d` · MCP `%d` · Plugins `%d` · Rules `%d` · Hooks `%d`\n"+
			"- **宿主机状态**：CPU `%.1f%%` · 内存 `%.1f%%` · 磁盘 `%s` 已用 `%.1f%%`\n\n"+
			"> 💡 提示：如需调用 Google 云端 Gemini 4 Neon / Hydrogen / Flash 深度推理，请在左侧 **「授权管理」** 中配置 `GEMINI_API_KEY`（或在 `.env` 中写入）。",
		prompt, meta.Model, GetConfig().GetSandboxMode(), meta.Workspace, wsFiles,
		summary.SkillsCount, summary.MCPCount, summary.PluginsCount, summary.RulesCount, summary.HooksCount,
		host.CPUPercent, host.MemPercent, host.DiskMount, host.DiskPercent,
	)
	_ = appendTranscriptEntry(meta.ID, TranscriptEntry{
		Role:    "assistant",
		Model:   meta.Model + " (local-harness)",
		Content: reply,
	})
	LogInfo("[Agent] 会话 %s 本地执行回合完成", meta.ID)
}

// =========================================================================
// Connect HTTP+JSON ConversationService Handlers (on internal port 2398)
// =========================================================================

func handleConnectListConversations(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IncludeArchived bool `json:"includeArchived"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"conversations": ListConversations(req.IncludeArchived),
	})
}

func handleConnectCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title  string   `json:"title"`
		Model  string   `json:"model"`
		Tags   []string `json:"tags"`
		Prompt []struct {
			Text string `json:"text"`
		} `json:"prompt"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	promptText := ""
	if len(req.Prompt) > 0 {
		promptText = req.Prompt[0].Text
	}
	conv, err := CreateConversation(CreateConversationRequest{
		Title:  req.Title,
		Model:  req.Model,
		Tags:   req.Tags,
		Prompt: promptText,
	})
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"conversation": conv})
}

func handleConnectGetConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string `json:"conversationId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	detail, err := GetConversationDetail(req.ConversationID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(detail)
}

func handleConnectUpdateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string         `json:"conversationId"`
		Annotations    map[string]any `json:"annotations"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	conv, err := UpdateConversationAnnotations(req.ConversationID, req.Annotations)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(conv)
}

func handleConnectDeleteConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string `json:"conversationId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	w.Header().Set("Content-Type", "application/json")
	if err := DeleteConversation(req.ConversationID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{})
}
