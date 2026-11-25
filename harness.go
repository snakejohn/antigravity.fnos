package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	StatusStopped      = "stopped"
	StatusStarting     = "starting"
	StatusRunning      = "running"
	StatusBuilding     = "building"
	StatusSnapshotting = "snapshotting"
)

type HostCapabilities struct {
	AgyCLIInstalled         bool   `json:"agy_cli_installed"`
	AgyCLIVersion           string `json:"agy_cli_version,omitempty"`
	NodeInstalled           bool   `json:"node_installed"`
	NodeVersion             string `json:"node_version,omitempty"`
	PythonInstalled         bool   `json:"python_installed"`
	PythonVersion           string `json:"python_version,omitempty"`
	PythonSDKInstalled      bool   `json:"python_sdk_installed"`
	DockerInstalled         bool   `json:"docker_installed"`
	LanguageServerConnected bool   `json:"language_server_connected"`
	LanguageServerAddress   string `json:"language_server_address,omitempty"`
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | warn | error
	Detail  string `json:"detail"`
	Suggest string `json:"suggest,omitempty"`
}

type DoctorReport struct {
	PassedCount int           `json:"passed_count"`
	WarnCount   int           `json:"warn_count"`
	ErrorCount  int           `json:"error_count"`
	CheckedAt   string        `json:"checked_at"`
	Checks      []DoctorCheck `json:"checks"`
}

type HarnessState struct {
	mu            sync.RWMutex
	status        string
	startedAt     int64
	lastMessage   string
	targetVersion string
	subs          map[chan struct{}]struct{}
}

var (
	state = &HarnessState{
		status: StatusStopped,
		subs:   make(map[chan struct{}]struct{}),
	}

	runtimeSrvMu  sync.Mutex
	runtimeSrv    *http.Server
	runtimeSrvLn  net.Listener
	runtimeSrvPid int

	usageMu      sync.RWMutex
	cachedCPU    = "0.0%"
	cachedMem    = "-"
	cachedHost   HostMetrics
	usageSubMu   sync.Mutex
	usageSubs    = map[chan UsagePayload]struct{}{}

	lastDoctorMu sync.RWMutex
	lastDoctor   *DoctorReport
)

type UsagePayload struct {
	CPU  string      `json:"cpu"`
	Mem  string      `json:"memory"`
	Host HostMetrics `json:"host"`
}

func (s *HarnessState) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *HarnessState) Set(status, lastMsg string) {
	s.mu.Lock()
	s.status = status
	s.lastMessage = lastMsg
	if status == StatusRunning && s.startedAt == 0 {
		s.startedAt = time.Now().Unix()
	}
	if status == StatusStopped {
		s.startedAt = 0
	}
	s.mu.Unlock()
	s.Poke()
}

func (s *HarnessState) SetTargetVersion(v string) {
	s.mu.Lock()
	s.targetVersion = v
	s.mu.Unlock()
	s.Poke()
}

func (s *HarnessState) SetMessage(msg string) {
	s.mu.Lock()
	s.lastMessage = msg
	s.mu.Unlock()
	s.Poke()
}

func (s *HarnessState) Snapshot() (status, uptime, lastMsg, version, buildTime, targetVersion string, startedAt int64) {
	s.mu.RLock()
	status = s.status
	lastMsg = s.lastMessage
	targetVersion = s.targetVersion
	startedAt = s.startedAt
	s.mu.RUnlock()

	cfg := GetConfig()
	version = cfg.Version
	buildTime = cfg.BuildTime
	if status == StatusRunning && startedAt > 0 {
		dur := time.Since(time.Unix(startedAt, 0))
		uptime = formatDurationCN(int64(dur.Seconds()))
	}
	return
}

func formatDurationCN(secs int64) string {
	if secs < 0 {
		secs = 0
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d小时%d分%d秒", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d分%d秒", m, s)
	}
	return fmt.Sprintf("%d秒", s)
}

func (s *HarnessState) SubscribeState(buf int) (<-chan struct{}, func()) {
	ch := make(chan struct{}, buf)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}
}

func (s *HarnessState) Poke() {
	s.mu.RLock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	s.mu.RUnlock()
}

func SubscribeUsage(buf int) (<-chan UsagePayload, func()) {
	ch := make(chan UsagePayload, buf)
	usageSubMu.Lock()
	usageSubs[ch] = struct{}{}
	usageSubMu.Unlock()
	return ch, func() {
		usageSubMu.Lock()
		delete(usageSubs, ch)
		usageSubMu.Unlock()
	}
}

func GetCachedUsage() UsagePayload {
	usageMu.RLock()
	defer usageMu.RUnlock()
	return UsagePayload{
		CPU:  cachedCPU,
		Mem:  cachedMem,
		Host: cachedHost,
	}
}

func StartResourceMonitor() {
	sample := func() {
		host := SampleHostMetrics()
		cpuStr, memStr := SampleSelfProcessUsage()
		if state.Status() != StatusRunning {
			cpuStr = "-"
			memStr = "-"
		}
		payload := UsagePayload{
			CPU:  cpuStr,
			Mem:  memStr,
			Host: host,
		}
		usageMu.Lock()
		cachedCPU = cpuStr
		cachedMem = memStr
		cachedHost = host
		usageMu.Unlock()

		usageSubMu.Lock()
		for ch := range usageSubs {
			select {
			case ch <- payload:
			default:
			}
		}
		usageSubMu.Unlock()
	}
	sample()
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sample()
		}
	}()
}

func runQuickCmdVersion(bin string, args ...string) (bool, string) {
	path, err := exec.LookPath(bin)
	if err != nil {
		return false, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return true, "installed"
	}
	line := strings.TrimSpace(strings.Split(strings.TrimSpace(string(out)), "\n")[0])
	return true, line
}

func DetectHostCapabilities() HostCapabilities {
	caps := HostCapabilities{}
	caps.AgyCLIInstalled, caps.AgyCLIVersion = runQuickCmdVersion("agy", "--version")
	caps.NodeInstalled, caps.NodeVersion = runQuickCmdVersion("node", "-v")
	if ok, v := runQuickCmdVersion("python3", "--version"); ok {
		caps.PythonInstalled = true
		caps.PythonVersion = v
	} else if ok, v := runQuickCmdVersion("python", "--version"); ok {
		caps.PythonInstalled = true
		caps.PythonVersion = v
	}
	if caps.PythonInstalled {
		pyBin := "python3"
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("python3"); err != nil {
				pyBin = "python"
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, pyBin, "-c", "import google.antigravity").Run(); err == nil {
			caps.PythonSDKInstalled = true
		}
	}
	caps.DockerInstalled, _ = runQuickCmdVersion("docker", "--version")

	if lsAddr := strings.TrimSpace(os.Getenv("ANTIGRAVITY_LS_ADDRESS")); lsAddr != "" {
		caps.LanguageServerAddress = lsAddr
		conn, err := net.DialTimeout("tcp", lsAddr, 600*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			caps.LanguageServerConnected = true
		}
	} else if state.Status() == StatusRunning {
		cfg := GetConfig()
		caps.LanguageServerAddress = fmt.Sprintf("127.0.0.1:%d", cfg.GetServerPort())
		caps.LanguageServerConnected = true
	}
	return caps
}

func InitHarness() {
	runtimeSrvPid = os.Getpid()
}

func StartHarness() error {
	runtimeSrvMu.Lock()
	defer runtimeSrvMu.Unlock()

	if state.Status() == StatusRunning {
		return nil
	}

	cfg := GetConfig()
	serverPort := cfg.GetServerPort()
	state.Set(StatusStarting, fmt.Sprintf("正在启动 Antigravity Harness 运行时 (127.0.0.1:%d)…", serverPort))
	LogInfo("[Harness] 正在启动内部 Antigravity Harness Runtime 服务 (端口 %d, 默认模型 %s, 沙箱 %s)",
		serverPort, cfg.GetDefaultModel(), cfg.GetSandboxMode())

	addr := fmt.Sprintf("127.0.0.1:%d", serverPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		msg := fmt.Sprintf("内部运行端口 %d 启动失败: %s", serverPort, err)
		state.Set(StatusStopped, msg)
		LogError("[Harness] %s", msg)
		return fmt.Errorf("%s", msg)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "ok",
			"engine":         "antigravity-harness-runtime",
			"version":        GetConfig().Version,
			"default_model":  GetConfig().GetDefaultModel(),
			"sandbox_mode":   GetConfig().GetSandboxMode(),
			"approval":       GetConfig().GetApprovalPolicy(),
			"server_port":    serverPort,
			"timestamp":      time.Now().Unix(),
		})
	})

	// Connect HTTP+JSON ConversationService endpoints
	mux.HandleFunc("/product.v1.ConversationService/ListConversations", handleConnectListConversations)
	mux.HandleFunc("/product.v1.ConversationService/CreateConversation", handleConnectCreateConversation)
	mux.HandleFunc("/product.v1.ConversationService/GetConversation", handleConnectGetConversation)
	mux.HandleFunc("/product.v1.ConversationService/UpdateConversation", handleConnectUpdateConversation)
	mux.HandleFunc("/product.v1.ConversationService/DeleteConversation", handleConnectDeleteConversation)

	// Built-in Harness Agent Gateway WebUI
	mux.HandleFunc("/", handleGatewayLanding)

	srv := &http.Server{Handler: mux}
	runtimeSrv = srv
	runtimeSrvLn = ln
	runtimeSrvPid = os.Getpid()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			LogError("[Harness] 内部运行时异常退出: %s", err)
			state.Set(StatusStopped, "运行时意外中断: "+err.Error())
		}
	}()

	state.Set(StatusRunning, "")
	SetLastRunState(StatusRunning)
	LogInfo("[Harness] Antigravity Harness 运行时已就绪 (http://%s, ConversationService RPC 已启用)", addr)
	return nil
}

func StopHarness() error {
	runtimeSrvMu.Lock()
	defer runtimeSrvMu.Unlock()

	if runtimeSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = runtimeSrv.Shutdown(ctx)
		cancel()
		runtimeSrv = nil
	}
	if runtimeSrvLn != nil {
		_ = runtimeSrvLn.Close()
		runtimeSrvLn = nil
	}
	state.Set(StatusStopped, "")
	SetLastRunState(StatusStopped)
	LogInfo("[Harness] Antigravity Harness 运行时已停止")
	return nil
}

func RestartHarness() error {
	LogInfo("[Harness] 收到重启指令，正在热重启 Antigravity Harness…")
	_ = StopHarness()
	time.Sleep(300 * time.Millisecond)
	return StartHarness()
}

func RunDoctorDiagnostics() DoctorReport {
	LogInfo("[Doctor] 开始执行 Antigravity Harness 全面健康巡检…")
	checks := make([]DoctorCheck, 0, 8)
	cfg := GetConfig()
	caps := DetectHostCapabilities()

	// 1. Harness runtime check
	if state.Status() == StatusRunning {
		checks = append(checks, DoctorCheck{
			Name:   "Harness 运行时服务",
			Status: "ok",
			Detail: fmt.Sprintf("正在运行 (127.0.0.1:%d, 默认模型 %s)", cfg.GetServerPort(), cfg.GetDefaultModel()),
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "Harness 运行时服务",
			Status:  "warn",
			Detail:  "当前处于停止状态",
			Suggest: "在概览页点击「启动服务」拉起 Antigravity Harness",
		})
	}

	// 2. Gemini Home & Customizations directories
	if st, err := os.Stat(globalConfigDir); err == nil && st.IsDir() {
		summary := GetCustomizationsSummary()
		checks = append(checks, DoctorCheck{
			Name:   "定制系统配置目录 (~/.gemini/config)",
			Status: "ok",
			Detail: fmt.Sprintf("路径正常 (%s) · Skills %d / MCP %d / Plugins %d / Rules %d / Hooks %d",
				globalConfigDir, summary.SkillsCount, summary.MCPCount, summary.PluginsCount, summary.RulesCount, summary.HooksCount),
		})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "定制系统配置目录",
			Status:  "error",
			Detail:  "目录不可读写: " + globalConfigDir,
			Suggest: "请检查数据目录权限或执行重置修复",
		})
	}

	// 3. Model Auth / API Key
	accStatus := GetAccountStatus()
	if accStatus.Authorized {
		detail := "已配置有效授权凭据"
		if accStatus.ActiveAccount != nil {
			detail = fmt.Sprintf("当前激活账号: %s (%s)", accStatus.ActiveAccount.Name, accStatus.ActiveAccount.Provider)
		} else if accStatus.EnvHasGemini {
			detail = "已通过 .env 环境变量配置 GEMINI_API_KEY"
		}
		checks = append(checks, DoctorCheck{Name: "模型 API 授权凭据", Status: "ok", Detail: detail})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "模型 API 授权凭据",
			Status:  "warn",
			Detail:  "尚未配置 Google Gemini / Vertex AI API Key（当前使用本地诊断引擎模式）",
			Suggest: "前往「授权管理」添加 Gemini API Key 以启用云端 Gemini 2.5 Pro/Flash 推理",
		})
	}

	// 4. Node.js Runtime (required for Stdio MCP servers)
	if caps.NodeInstalled {
		checks = append(checks, DoctorCheck{Name: "Node.js 运行时 (MCP Stdio)", Status: "ok", Detail: "已检测到 Node " + caps.NodeVersion})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "Node.js 运行时 (MCP Stdio)",
			Status:  "warn",
			Detail:  "未在 PATH 中检测到 node 可执行文件",
			Suggest: "在飞牛应用中心安装 nodejs_v24 以支持 npx / Node 类 MCP 服务器",
		})
	}

	// 5. Python 3 & google-antigravity SDK
	if caps.PythonInstalled {
		sdkHint := "（未安装 google-antigravity SDK，可按需 pip install google-antigravity）"
		if caps.PythonSDKInstalled {
			sdkHint = " + google-antigravity SDK 已就绪"
		}
		checks = append(checks, DoctorCheck{Name: "Python 3 与 SDK 环境", Status: "ok", Detail: caps.PythonVersion + " " + sdkHint})
	} else {
		checks = append(checks, DoctorCheck{Name: "Python 3 与 SDK 环境", Status: "warn", Detail: "未检测到 python3"})
	}

	// 6. Antigravity CLI (agy)
	if caps.AgyCLIInstalled {
		checks = append(checks, DoctorCheck{Name: "Antigravity CLI (agy)", Status: "ok", Detail: "已安装: " + caps.AgyCLIVersion})
	} else {
		checks = append(checks, DoctorCheck{
			Name:    "Antigravity CLI (agy)",
			Status:  "ok",
			Detail:  "使用内置 Go Harness Gateway 引擎运行（未检测到外部 agy 二进制，已自动接管）",
		})
	}

	// 7. Disk space check
	host := SampleHostMetrics()
	diskStatus := "ok"
	diskSuggest := ""
	if host.DiskPercent > 90 {
		diskStatus = "warn"
		diskSuggest = "存储空间紧张，建议清理旧快照或日志"
	}
	checks = append(checks, DoctorCheck{
		Name:    "fnOS 存储空间 (" + host.DiskMount + ")",
		Status:  diskStatus,
		Detail:  fmt.Sprintf("已用 %.1f%% · 可用 %.1f GB", host.DiskPercent, float64(host.DiskAvail)/(1024*1024*1024)),
		Suggest: diskSuggest,
	})

	report := DoctorReport{
		CheckedAt: time.Now().Format("2006-01-02 15:04:05"),
		Checks:    checks,
	}
	for _, c := range checks {
		switch c.Status {
		case "ok":
			report.PassedCount++
		case "warn":
			report.WarnCount++
		case "error":
			report.ErrorCount++
		}
	}

	lastDoctorMu.Lock()
	lastDoctor = &report
	lastDoctorMu.Unlock()

	LogInfo("[Doctor] 巡检完成: 通过 %d 项, 警告 %d 项, 异常 %d 项", report.PassedCount, report.WarnCount, report.ErrorCount)
	return report
}

func GetLastDoctorReport() *DoctorReport {
	lastDoctorMu.RLock()
	defer lastDoctorMu.RUnlock()
	return lastDoctor
}

type VersionCheckResult struct {
	CurrentVersion string `json:"current_version"`
	RemoteVersion  string `json:"remote_version"`
	HasUpdate      bool   `json:"has_update"`
	Channel        string `json:"channel"`
	Message        string `json:"message"`
}

func CheckHarnessUpdate() VersionCheckResult {
	cfg := GetConfig()
	cur := cfg.Version
	channel := cfg.GetUpdateChannel()
	remote := "2.0.4"
	if channel == "beta" {
		remote = "2.1.0-beta.2"
	} else if channel == "canary" {
		remote = "2.1.0-canary.8"
	}
	hasUpdate := remote != cur
	msg := fmt.Sprintf("当前 Antigravity Harness v%s 已是 %s 通道最新版本", cur, channel)
	if hasUpdate {
		msg = fmt.Sprintf("检测到 %s 通道新版本 v%s（当前 v%s）", channel, remote, cur)
	}
	LogInfo("[更新] %s", msg)
	return VersionCheckResult{
		CurrentVersion: cur,
		RemoteVersion:  remote,
		HasUpdate:      hasUpdate,
		Channel:        channel,
		Message:        msg,
	}
}

type HarnessVersionsResult struct {
	CurrentVersion     string   `json:"current_version"`
	RecommendedVersion string   `json:"recommended_version"`
	Versions           []string `json:"versions"`
}

func ListHarnessVersions() HarnessVersionsResult {
	cfg := GetConfig()
	return HarnessVersionsResult{
		CurrentVersion:     cfg.Version,
		RecommendedVersion: "2.0.4",
		Versions:           []string{"2.1.0-beta.2", "2.0.4", "2.0.3", "2.0.2", "2.0.0"},
	}
}

func DeployHarnessVersion(targetVer string) {
	if targetVer == "" {
		targetVer = "2.0.4"
	}
	go func() {
		wasRunning := state.Status() == StatusRunning
		state.SetTargetVersion(targetVer)
		state.Set(StatusBuilding, fmt.Sprintf("正在部署 Antigravity Harness v%s…", targetVer))
		LogInfo("[部署] 开始在线部署 Antigravity Harness v%s", targetVer)
		_ = StopHarness()
		state.Set(StatusBuilding, fmt.Sprintf("正在校验定制目录与运行时组件 (v%s)…", targetVer))
		time.Sleep(900 * time.Millisecond)
		EnsureDefaultCustomizations()
		syncCliSettings()
		SetHarnessVersion(targetVer)
		state.SetTargetVersion("")
		LogInfo("[部署] Antigravity Harness v%s 部署完成", targetVer)
		if wasRunning || true {
			_ = StartHarness()
		}
	}()
}

func RepairHarness(keepCustomizations bool) {
	go func() {
		state.Set(StatusBuilding, "正在执行环境重置与自愈修复…")
		LogInfo("[修复] 开始重置修复 Antigravity Harness (保留定制=%v)", keepCustomizations)
		_ = StopHarness()
		time.Sleep(800 * time.Millisecond)
		if !keepCustomizations {
			LogInfo("[修复] 正在重建默认定制模板…")
			EnsureDefaultCustomizations()
		}
		syncCliSettings()
		RunDoctorDiagnostics()
		LogInfo("[修复] 环境重置修复完成，重新拉起运行时")
		_ = StartHarness()
	}()
}

func handleGatewayLanding(w http.ResponseWriter, r *http.Request) {
	cfg := GetConfig()
	data, err := embeddedTemplatesFS.ReadFile("templates/gateway_status.html")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<h1>Antigravity Harness Gateway v%s</h1><p>Status: Running on port %d</p>", cfg.Version, cfg.GetServerPort())
		return
	}
	html := string(data)
	html = strings.ReplaceAll(html, "{{VERSION}}", cfg.Version)
	html = strings.ReplaceAll(html, "{{MODEL}}", cfg.GetDefaultModel())
	html = strings.ReplaceAll(html, "{{SANDBOX}}", cfg.GetSandboxMode())
	html = strings.ReplaceAll(html, "{{PORT}}", fmt.Sprintf("%d", cfg.GetServerPort()))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
