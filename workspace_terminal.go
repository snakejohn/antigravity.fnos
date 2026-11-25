package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type WorkspaceItem struct {
	ID            string `json:"workspaceId"`
	Title         string `json:"title"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	HasAgentsDir  bool   `json:"has_agents_dir"`
	SkillsCount   int    `json:"skills_count"`
	HasGeminiRule bool   `json:"has_gemini_rule"`
	GitBranch     string `json:"git_branch,omitempty"`
	SessionCount  int    `json:"session_count"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type WorkspacePayload struct {
	Items []WorkspaceItem `json:"items"`
}

var (
	wsMu    sync.Mutex
	wsSubMu sync.Mutex
	wsSubs  = map[chan struct{}]struct{}{}
)

func getWorkspacesStorePath() string {
	return filepath.Join(globalDataDir, "workspaces.json")
}

func SubscribeWorkspace(buf int) (<-chan struct{}, func()) {
	ch := make(chan struct{}, buf)
	wsSubMu.Lock()
	wsSubs[ch] = struct{}{}
	wsSubMu.Unlock()
	return ch, func() {
		wsSubMu.Lock()
		delete(wsSubs, ch)
		wsSubMu.Unlock()
	}
}

func notifyWorkspaceChanged() {
	wsSubMu.Lock()
	for ch := range wsSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	wsSubMu.Unlock()
}

func InitWorkspaces() {
	wsMu.Lock()
	defer wsMu.Unlock()
	path := getWorkspacesStorePath()
	if _, err := os.Stat(path); err == nil {
		return
	}
	now := time.Now().UnixMilli()
	defaults := []WorkspaceItem{
		{
			ID:        "ws_default_data",
			Title:     "Antigravity Harness 数据区",
			Path:      globalPkgVar,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	// On fnOS NAS, auto-register existing volume roots if present
	for _, vol := range []struct {
		id    string
		title string
		path  string
	}{
		{"ws_vol1", "fnOS 存储卷 1 (/vol1)", "/vol1"},
		{"ws_vol3", "fnOS 应用卷 3 (/vol3)", "/vol3"},
	} {
		if st, err := os.Stat(vol.path); err == nil && st.IsDir() {
			defaults = append(defaults, WorkspaceItem{
				ID:        vol.id,
				Title:     vol.title,
				Path:      vol.path,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}
	if b, err := json.MarshalIndent(defaults, "", "  "); err == nil {
		_ = os.WriteFile(path, b, 0644)
	}
}

func loadWorkspacesRaw() []WorkspaceItem {
	raw, err := os.ReadFile(getWorkspacesStorePath())
	if err != nil {
		return []WorkspaceItem{}
	}
	var list []WorkspaceItem
	_ = json.Unmarshal(raw, &list)
	return list
}

func saveWorkspacesRaw(list []WorkspaceItem) error {
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getWorkspacesStorePath(), b, 0644)
}

func enrichWorkspace(w WorkspaceItem, convs []ConversationMeta) WorkspaceItem {
	if st, err := os.Stat(w.Path); err == nil && st.IsDir() {
		w.Exists = true
		w.UpdatedAt = st.ModTime().UnixMilli()
		if ast, err := os.Stat(filepath.Join(w.Path, ".agents")); err == nil && ast.IsDir() {
			w.HasAgentsDir = true
			w.SkillsCount = len(scanSkillsDir(filepath.Join(w.Path, ".agents", "skills"), "workspace"))
		}
		if _, err := os.Stat(filepath.Join(w.Path, "GEMINI.md")); err == nil {
			w.HasGeminiRule = true
		} else if _, err := os.Stat(filepath.Join(w.Path, "AGENTS.md")); err == nil {
			w.HasGeminiRule = true
		}
		// Quick check for .git/HEAD
		if head, err := os.ReadFile(filepath.Join(w.Path, ".git", "HEAD")); err == nil {
			line := strings.TrimSpace(string(head))
			if strings.HasPrefix(line, "ref: refs/heads/") {
				w.GitBranch = strings.TrimPrefix(line, "ref: refs/heads/")
			}
		}
	}
	cnt := 0
	for _, c := range convs {
		if c.Workspace == w.Path {
			cnt++
		}
	}
	w.SessionCount = cnt
	return w
}

func GetWorkspaceList() []WorkspaceItem {
	wsMu.Lock()
	raw := loadWorkspacesRaw()
	wsMu.Unlock()
	convs := ListConversations(true)
	out := make([]WorkspaceItem, 0, len(raw))
	for _, w := range raw {
		out = append(out, enrichWorkspace(w, convs))
	}
	return out
}

func AddWorkspace(title, dirPath string) (WorkspaceItem, error) {
	wsMu.Lock()
	defer wsMu.Unlock()
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return WorkspaceItem{}, fmt.Errorf("工作区目录路径不能为空")
	}
	if st, err := os.Stat(dirPath); err != nil || !st.IsDir() {
		return WorkspaceItem{}, fmt.Errorf("目录不存在或不可访问: %s", dirPath)
	}
	if title == "" {
		title = filepath.Base(dirPath)
	}
	list := loadWorkspacesRaw()
	for _, existing := range list {
		if existing.Path == dirPath {
			return WorkspaceItem{}, fmt.Errorf("该路径已存在于工作区列表中")
		}
	}
	now := time.Now().UnixMilli()
	item := WorkspaceItem{
		ID:        generateShortID("ws"),
		Title:     title,
		Path:      dirPath,
		CreatedAt: now,
		UpdatedAt: now,
	}
	list = append([]WorkspaceItem{item}, list...)
	if err := saveWorkspacesRaw(list); err != nil {
		return WorkspaceItem{}, err
	}
	LogInfo("[工作区] 已绑定工作区「%s」(%s)", title, dirPath)
	notifyWorkspaceChanged()
	return enrichWorkspace(item, ListConversations(true)), nil
}

func DeleteWorkspace(id string) error {
	wsMu.Lock()
	defer wsMu.Unlock()
	list := loadWorkspacesRaw()
	next := make([]WorkspaceItem, 0, len(list))
	found := false
	for _, w := range list {
		if w.ID == id {
			found = true
			continue
		}
		next = append(next, w)
	}
	if !found {
		return fmt.Errorf("工作区不存在: %s", id)
	}
	if err := saveWorkspacesRaw(next); err != nil {
		return err
	}
	LogInfo("[工作区] 已移除工作区绑定: %s", id)
	notifyWorkspaceChanged()
	return nil
}

// =========================================================================
// Web Command Runner / Terminal Execution Engine
// =========================================================================

type TerminalExecRequest struct {
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	TimeoutSec int    `json:"timeout_sec"`
}

type TerminalExecResult struct {
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	TimedOut   bool   `json:"timed_out"`
}

func ExecuteTerminalCommand(req TerminalExecRequest) TerminalExecResult {
	cmdText := strings.TrimSpace(req.Command)
	cwd := strings.TrimSpace(req.Cwd)
	if cwd == "" {
		cwd = globalPkgVar
	}
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		cwd = globalPkgVar
	}
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120
	}

	if cmdText == "" {
		return TerminalExecResult{
			Cwd:      cwd,
			Stderr:   "命令不能为空",
			ExitCode: 1,
		}
	}

	// Built-in virtual commands for agy doctor / harness info when agy CLI isn't installed
	if cmdText == "agy doctor" || cmdText == "harness doctor" {
		start := time.Now()
		report := RunDoctorDiagnostics()
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Antigravity Harness Doctor (%s)\n", report.CheckedAt))
		sb.WriteString("------------------------------------------------------------\n")
		for _, ch := range report.Checks {
			badge := "[PASS]"
			if ch.Status == "warn" {
				badge = "[WARN]"
			} else if ch.Status == "error" {
				badge = "[FAIL]"
			}
			sb.WriteString(fmt.Sprintf("%s %s: %s\n", badge, ch.Name, ch.Detail))
			if ch.Suggest != "" {
				sb.WriteString(fmt.Sprintf("       -> 建议: %s\n", ch.Suggest))
			}
		}
		sb.WriteString("------------------------------------------------------------\n")
		sb.WriteString(fmt.Sprintf("Summary: %d passed, %d warnings, %d errors\n", report.PassedCount, report.WarnCount, report.ErrorCount))
		return TerminalExecResult{
			Command:    cmdText,
			Cwd:        cwd,
			Stdout:     sb.String(),
			ExitCode:   0,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", cmdText)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", cmdText)
	}
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"GEMINI_HOME="+globalGeminiHome,
		"AGY_CONFIG_DIR="+globalConfigDir,
	)
	setupCommandProcessGroup(cmd)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	LogInfo("[终端] 执行命令 (cwd=%s): %s", cwd, cmdText)
	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	res := TerminalExecResult{
		Command:    cmdText,
		Cwd:        cwd,
		Stdout:     cleanLogLine(stdoutBuf.Bytes()),
		Stderr:     cleanLogLine(stderrBuf.Bytes()),
		ExitCode:   0,
		DurationMs: elapsed,
	}

	if ctx.Err() == context.DeadlineExceeded {
		killCommandProcessGroup(cmd)
		res.TimedOut = true
		res.ExitCode = 124
		res.Stderr = strings.TrimSpace(res.Stderr + "\n命令执行超时并已被终止")
	} else if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = 1
			if res.Stderr == "" {
				res.Stderr = err.Error()
			}
		}
	}

	if res.ExitCode != 0 {
		LogWarning("[终端] 命令退出码 %d (%dms): %s", res.ExitCode, elapsed, cmdText)
	}
	return res
}
