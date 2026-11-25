package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var WebFS embed.FS

const basePath = "/app/antigravity-harness"

type ApiResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

func OK(c *gin.Context, data any) {
	OKMsg(c, "success", data)
}

func OKMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:      0,
		Message:   msg,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}

func Fail(c *gin.Context, status int, msg string) {
	c.JSON(status, ApiResponse{
		Code:      status,
		Message:   msg,
		Timestamp: time.Now().UnixMilli(),
	})
}

func registerAPIGroup(api *gin.RouterGroup) {
	api.GET("/ws", handleWS)
	api.GET("/status", handleGetStatus)
	api.GET("/models", handleListModels)
	api.POST("/action", handleAction)
	api.GET("/doctor", handleGetDoctor)
	api.GET("/check-update", handleCheckUpdate)
	api.GET("/deploy/versions", handleListVersions)

	// Customizations (Skills, MCP, Plugins, Rules, Hooks)
	api.GET("/customizations/summary", handleGetCustomizationsSummary)
	api.GET("/customizations/skills", handleListSkills)
	api.GET("/customizations/skills/:scope/:name", handleGetSkill)
	api.POST("/customizations/skills", handleSaveSkill)
	api.DELETE("/customizations/skills/:scope/:name", handleDeleteSkill)

	api.GET("/customizations/mcp", handleGetMCP)
	api.POST("/customizations/mcp", handleSaveMCP)
	api.POST("/customizations/mcp/raw", handleSaveMCPRaw)
	api.POST("/customizations/mcp/test", handleTestMCP)
	api.DELETE("/customizations/mcp/:id", handleDeleteMCP)

	api.GET("/customizations/plugins", handleListPlugins)
	api.POST("/customizations/plugins/toggle", handleTogglePlugin)
	api.POST("/customizations/plugins/install", handleInstallPlugin)
	api.DELETE("/customizations/plugins/:name", handleDeletePlugin)

	api.GET("/customizations/rules", handleListRules)
	api.POST("/customizations/rules", handleSaveRule)

	api.GET("/customizations/hooks", handleListHooks)
	api.POST("/customizations/hooks", handleSaveHooks)

	// Conversations & Swarm Monitor
	api.GET("/conversations", handleListConversations)
	api.POST("/conversations", handleCreateConversation)
	api.GET("/conversations/:id", handleGetConversation)
	api.POST("/conversations/:id/message", handleSendConversationMessage)
	api.PATCH("/conversations/:id", handleUpdateConversation)
	api.DELETE("/conversations/:id", handleDeleteConversation)

	// Workspaces, Workspace Files & Web Command Runner / Terminal
	api.GET("/workspace/list", handleGetWorkspaces)
	api.POST("/workspace/add", handleAddWorkspace)
	api.DELETE("/workspace/:id", handleDeleteWorkspace)
	api.GET("/workspaces/files", handleListWorkspaceFiles)
	api.GET("/workspaces/file", handleReadWorkspaceFile)
	api.POST("/workspaces/file", handleWriteWorkspaceFile)
	api.POST("/terminal/exec", handleTerminalExec)

	// Account Pool & .env Environment Variables
	api.GET("/account/status", handleAccountStatus)
	api.GET("/account/pool", handleAccountPoolList)
	api.POST("/account/save", handleAccountSave)
	api.POST("/account/switch", handleAccountSwitch)
	api.POST("/account/logout", handleAccountLogout)
	api.DELETE("/account/pool/:id", handleAccountDelete)
	api.GET("/env-vars", handleGetEnvVars)
	api.POST("/env-vars", handleSaveEnvVars)

	// Snapshots
	api.GET("/snapshots", handleListSnapshots)
	api.POST("/snapshots", handleCreateSnapshot)
	api.POST("/snapshots/cancel", handleCancelSnapshot)
	api.POST("/snapshots/:id/restore", handleRestoreSnapshot)
	api.DELETE("/snapshots/:id", handleDeleteSnapshot)

	// Logs & Config
	api.GET("/logs", handleGetLogs)
	api.DELETE("/logs", handleDeleteLogs)
	api.GET("/logs/download", handleDownloadLogs)
	api.GET("/config", handleGetConfig)
	api.POST("/config", handleSaveConfig)
}

func InitRoutes(r *gin.Engine) {
	// Register APIs at BOTH root /api/* and fnOS gateway subpath /app/antigravity-harness/api/*
	registerAPIGroup(r.Group("/api"))
	base := r.Group(basePath)
	registerAPIGroup(base.Group("/api"))

	// Embedded Vue 3 Frontend static asset serving (shared by Admin Console and /fngateway/ Workbench)
	sub, err := fs.Sub(WebFS, "frontend/dist")
	if err != nil {
		LogFatal("[API] 静态前端资源装载失败: %s", err)
	}
	fileServer := http.FileServer(http.FS(sub))

	serveIndex := func(c *gin.Context) {
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	}

	// Antigravity Workbench WebUI (/fngateway/) + Internal RPC/Health Proxy
	serveGatewayOrWorkbench := func(c *gin.Context) {
		p := c.Request.URL.Path
		// Ensure trailing slash on bare /fngateway so relative ./assets/* resolve cleanly
		if p == "/fngateway" || p == basePath+"/fngateway" {
			c.Redirect(http.StatusMovedPermanently, p+"/")
			return
		}
		subPath := strings.TrimPrefix(p, basePath)
		subPath = strings.TrimPrefix(subPath, "/fngateway")
		if subPath == "" {
			subPath = "/"
		}
		// Keep /health and Connect RPC endpoints proxied to internal runtime (127.0.0.1:2398)
		if subPath == "/health" || strings.HasPrefix(subPath, "/product.v1.") {
			handleFnGatewayProxy(c)
			return
		}
		// Serve embedded frontend static assets under /fngateway/assets/*, /fngateway/favicon.svg, etc.
		if subPath != "/" {
			if f, err := sub.Open(strings.TrimPrefix(subPath, "/")); err == nil {
				_ = f.Close()
				c.Request.URL.Path = subPath
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// Render full Antigravity Workbench SPA
		serveIndex(c)
	}

	r.Any("/fngateway/*proxyPath", serveGatewayOrWorkbench)
	r.Any("/fngateway", serveGatewayOrWorkbench)
	base.Any("/fngateway/*proxyPath", serveGatewayOrWorkbench)
	base.Any("/fngateway", serveGatewayOrWorkbench)

	r.GET("/", serveIndex)
	base.GET("/", serveIndex)
	base.GET("", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, basePath+"/")
	})

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, basePath+"/api/") {
			Fail(c, http.StatusNotFound, "接口不存在: "+p)
			return
		}
		fp := strings.TrimPrefix(p, basePath)
		if fp == "" {
			fp = "/"
		}
		f, err := sub.Open(strings.TrimPrefix(fp, "/"))
		if err == nil {
			_ = f.Close()
			c.Request.URL.Path = fp
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		serveIndex(c)
	})
}

func handleFnGatewayProxy(c *gin.Context) {
	if state.Status() != StatusRunning {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusServiceUnavailable,
			"<html><body style='font-family:sans-serif;text-align:center;padding:60px;background:#12141a;color:#e2e8f0'>"+
				"<h2>Antigravity Harness 运行时尚未启动</h2><p>请返回管理控制台点击「启动服务」。</p>"+
				"<a href='/' style='color:#3b82f6'>返回管理控制台</a></body></html>")
		return
	}
	cfg := GetConfig()
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", cfg.GetServerPort()))
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		p := req.URL.Path
		p = strings.TrimPrefix(p, basePath)
		p = strings.TrimPrefix(p, "/fngateway")
		if p == "" {
			p = "/"
		}
		req.URL.Path = p
		req.Host = target.Host
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func statusPayload() gin.H {
	st, uptime, lastMsg, version, buildTime, targetVersion, startedAt := state.Snapshot()
	cfg := GetConfig()
	usage := GetCachedUsage()
	convs := ListConversations(false)
	activeRuns := 0
	for _, cv := range convs {
		if cv.RunState == "RUN_STATE_RUNNING" {
			activeRuns++
		}
	}
	var pidVal any
	if st == StatusRunning {
		pidVal = runtimeSrvPid
	}

	return gin.H{
		"name":                 "Antigravity Harness",
		"version":              version,
		"app_version":          globalAppVer,
		"target_version":       targetVersion,
		"status":               st,
		"uptime":               uptime,
		"started_at":           startedAt,
		"server_port":          cfg.GetServerPort(),
		"proxy_port":           cfg.GetProxyPort(),
		"default_model":        cfg.GetDefaultModel(),
		"sandbox_mode":         cfg.GetSandboxMode(),
		"approval_policy":      cfg.GetApprovalPolicy(),
		"server_time":          time.Now().Unix(),
		"build_time":           buildTime,
		"pid":                  pidVal,
		"cpu":                  usage.CPU,
		"memory":               usage.Mem,
		"host":                 usage.Host,
		"conversations_count":  len(convs),
		"active_runs":          activeRuns,
		"customizations":       GetCustomizationsSummary(),
		"capabilities":         DetectHostCapabilities(),
		"last_message":         lastMsg,
	}
}

func handleGetStatus(c *gin.Context) {
	OK(c, statusPayload())
}

func handleListModels(c *gin.Context) {
	models := []gin.H{
		{"id": "gemini-4-neon-xhigh", "label": "Gemini 4 Neon (XHigh) - 旗舰深度思考", "group": "Gemini 4 Neon 系列"},
		{"id": "gemini-4-neon-high", "label": "Gemini 4 Neon (High)", "group": "Gemini 4 Neon 系列"},
		{"id": "gemini-4-neon-medium", "label": "Gemini 4 Neon (Medium)", "group": "Gemini 4 Neon 系列"},
		{"id": "gemini-4-neon-low", "label": "Gemini 4 Neon (Low)", "group": "Gemini 4 Neon 系列"},
		{"id": "gemini-4-hydrogen-xhigh", "label": "Gemini 4 Hydrogen (XHigh)", "group": "Gemini 4 Hydrogen / Argon / Carbon 系列"},
		{"id": "gemini-4-hydrogen-high", "label": "Gemini 4 Hydrogen (High)", "group": "Gemini 4 Hydrogen / Argon / Carbon 系列"},
		{"id": "gemini-4-argon-high", "label": "Gemini 4 Argon (High)", "group": "Gemini 4 Hydrogen / Argon / Carbon 系列"},
		{"id": "gemini-4-argon-medium", "label": "Gemini 4 Argon (Medium)", "group": "Gemini 4 Hydrogen / Argon / Carbon 系列"},
		{"id": "gemini-4-carbon-high", "label": "Gemini 4 Carbon (High)", "group": "Gemini 4 Hydrogen / Argon / Carbon 系列"},
		{"id": "gemini-3.8-flash-high", "label": "Gemini 3.8 Flash (High) - 极速响应", "group": "Gemini 3.8 Flash & Antigravity 编排模型"},
		{"id": "gemini-3.8-flash-medium", "label": "Gemini 3.8 Flash (Medium)", "group": "Gemini 3.8 Flash & Antigravity 编排模型"},
		{"id": "gemini-pro-orchestrator-neon", "label": "Orchestrator (Neon)", "group": "Gemini 3.8 Flash & Antigravity 编排模型"},
		{"id": "claude-opus-5-5-high", "label": "Claude Opus 5.5 (High)", "group": "Gemini 3.8 Flash & Antigravity 编排模型"},
		{"id": "claude-sonnet-5-5-high", "label": "Claude Sonnet 5.5 (High)", "group": "Gemini 3.8 Flash & Antigravity 编排模型"},
		{"id": "gemini-2.5-pro", "label": "Gemini 2.5 Pro (Legacy)", "group": "经典兼容模型"},
		{"id": "gemini-2.5-flash", "label": "Gemini 2.5 Flash (Legacy)", "group": "经典兼容模型"},
		{"id": "gemini-2.5-flash-lite", "label": "Gemini 2.5 Flash-Lite (Legacy)", "group": "经典兼容模型"},
	}
	OK(c, gin.H{"default_model": GetConfig().GetDefaultModel(), "items": models})
}

// =========================================================================
// WebSocket Hub (/api/ws)
// =========================================================================

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsMsg struct {
	Type      string `json:"type"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func handleWS(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		LogWarning("[API] WebSocket 升级失败: %s", err)
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	sendMsg := func(msgType string, data any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteJSON(wsMsg{
			Type:      msgType,
			Data:      data,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	// Send initial snapshot immediately upon connection
	sendMsg("status", statusPayload())
	sendMsg("usage", GetCachedUsage())
	sendMsg("workspace", WorkspacePayload{Items: GetWorkspaceList()})
	sendMsg("conversations", ListConversations(false))
	if snapSummary, err := ListSnapshots(); err == nil {
		sendMsg("snapshot", snapSummary)
	}
	if prog := GetCurrentSnapshotProgress(); prog.Active {
		sendMsg("snapshot_progress", prog)
	}

	stateCh, unsubState := state.SubscribeState(16)
	defer unsubState()
	logCh, unsubLog := SubscribeLog(256)
	defer unsubLog()
	usageCh, unsubUsage := SubscribeUsage(16)
	defer unsubUsage()
	wsCh, unsubWs := SubscribeWorkspace(16)
	defer unsubWs()
	convCh, unsubConv := SubscribeConversations(16)
	defer unsubConv()
	snapCh, unsubSnap := SubscribeSnapshot(16)
	defer unsubSnap()
	snapProgCh, unsubSnapProg := SubscribeSnapshotProgress(32)
	defer unsubSnapProg()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var inMsg struct {
				Type string `json:"type"`
			}
			if err := conn.ReadJSON(&inMsg); err != nil {
				return
			}
			if inMsg.Type == "ping" {
				sendMsg("pong", gin.H{"server_time": time.Now().Unix()})
			}
		}
	}()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-done:
			return
		case <-stateCh:
			sendMsg("status", statusPayload())
		case chunk := <-logCh:
			sendMsg("log", chunk)
		case u := <-usageCh:
			sendMsg("usage", u)
		case <-wsCh:
			sendMsg("workspace", WorkspacePayload{Items: GetWorkspaceList()})
		case <-convCh:
			sendMsg("conversations", ListConversations(false))
			sendMsg("status", statusPayload())
		case <-snapCh:
			if snapSummary, err := ListSnapshots(); err == nil {
				sendMsg("snapshot", snapSummary)
			}
		case p := <-snapProgCh:
			sendMsg("snapshot_progress", p)
		case <-heartbeat.C:
			writeMu.Lock()
			_ = conn.WriteMessage(websocket.PingMessage, nil)
			writeMu.Unlock()
		}
	}
}

// =========================================================================
// Lifecycle Actions, Doctor, Versions
// =========================================================================

func handleAction(c *gin.Context) {
	var req struct {
		Action             string `json:"action"`
		TargetVersion      string `json:"target_version"`
		KeepCustomizations bool   `json:"keep_plugins"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	switch req.Action {
	case "start":
		if err := StartHarness(); err != nil {
			Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		OKMsg(c, "Antigravity Harness 服务已启动", statusPayload())
	case "stop":
		_ = StopHarness()
		OKMsg(c, "Antigravity Harness 服务已停止", statusPayload())
	case "restart":
		if err := RestartHarness(); err != nil {
			Fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		OKMsg(c, "Antigravity Harness 服务已重启", statusPayload())
	case "doctor":
		report := RunDoctorDiagnostics()
		OKMsg(c, fmt.Sprintf("巡检完成：通过 %d 项，警告 %d 项", report.PassedCount, report.WarnCount), report)
	case "deploy":
		DeployHarnessVersion(req.TargetVersion)
		OKMsg(c, "开始部署 Antigravity Harness v"+req.TargetVersion, statusPayload())
	case "upgrade":
		res := CheckHarnessUpdate()
		DeployHarnessVersion(res.RemoteVersion)
		OKMsg(c, "开始在线升级至 v"+res.RemoteVersion, statusPayload())
	case "repair":
		RepairHarness(req.KeepCustomizations)
		OKMsg(c, "开始重置修复 Antigravity Harness 环境…", statusPayload())
	default:
		Fail(c, http.StatusBadRequest, "未知操作: "+req.Action)
	}
}

func handleGetDoctor(c *gin.Context) {
	if report := GetLastDoctorReport(); report != nil {
		OK(c, report)
		return
	}
	OK(c, RunDoctorDiagnostics())
}

func handleCheckUpdate(c *gin.Context) {
	res := CheckHarnessUpdate()
	OKMsg(c, res.Message, res)
}

func handleListVersions(c *gin.Context) {
	OK(c, ListHarnessVersions())
}

// =========================================================================
// Customizations Endpoints
// =========================================================================

func handleGetCustomizationsSummary(c *gin.Context) {
	OK(c, GetCustomizationsSummary())
}

func handleListSkills(c *gin.Context) {
	OK(c, gin.H{"items": ListAllSkills()})
}

func handleGetSkill(c *gin.Context) {
	item, err := GetSkillDetail(c.Param("scope"), c.Param("name"))
	if err != nil {
		Fail(c, http.StatusNotFound, err.Error())
		return
	}
	OK(c, item)
}

func handleSaveSkill(c *gin.Context) {
	var req struct {
		Scope       string `json:"scope"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	item, err := SaveSkill(req.Scope, req.Name, req.Description, req.Content)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "技能保存成功", item)
}

func handleDeleteSkill(c *gin.Context) {
	if err := DeleteSkill(c.Param("scope"), c.Param("name")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "技能已删除", nil)
}

func handleGetMCP(c *gin.Context) {
	items, raw, err := ReadMCPConfig()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "raw_json": raw, "path": getMCPConfigPath()})
}

func handleSaveMCP(c *gin.Context) {
	var req struct {
		ID      string            `json:"id"`
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	err := SaveMCPServer(req.ID, MCPServerEntry{
		Command: strings.TrimSpace(req.Command),
		Args:    req.Args,
		Env:     req.Env,
		URL:     strings.TrimSpace(req.URL),
		Headers: req.Headers,
	})
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "MCP Server 已保存", nil)
}

func handleSaveMCPRaw(c *gin.Context) {
	var req struct {
		RawJSON string `json:"raw_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := SaveMCPRawJSON(req.RawJSON); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "mcp_config.json 已保存", nil)
}

func handleTestMCP(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		Fail(c, http.StatusBadRequest, "请指定要测试的 MCP Server ID")
		return
	}
	res := TestMCPServer(req.ID)
	OK(c, res)
}

func handleDeleteMCP(c *gin.Context) {
	if err := DeleteMCPServer(c.Param("id")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "MCP Server 已移除", nil)
}

func handleListPlugins(c *gin.Context) {
	OK(c, gin.H{"items": ListAllPlugins()})
}

func handleTogglePlugin(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := TogglePlugin(req.Name, req.Enabled); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "插件状态已更新", nil)
}

func handleInstallPlugin(c *gin.Context) {
	var req struct {
		DirName     string `json:"dir_name"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Author      string `json:"author"`
		GitURL      string `json:"git_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := InstallPlugin(req.DirName, req.Name, req.Description, req.Author, req.GitURL)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "插件安装成功", item)
}

func handleDeletePlugin(c *gin.Context) {
	if err := DeletePlugin(c.Param("name")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "插件已卸载", nil)
}

func handleListRules(c *gin.Context) {
	rules := ListAllRules()
	totalTokens := 0
	for _, r := range rules {
		totalTokens += r.EstTokens
	}
	OK(c, gin.H{
		"items":        rules,
		"total_tokens": totalTokens,
		"token_budget": 20000,
	})
}

func handleSaveRule(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := SaveRuleFile(req.Name, req.Content)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "规则文件已保存", item)
}

func handleListHooks(c *gin.Context) {
	items, err := ListHooks()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "path": getHooksFilePath()})
}

func handleSaveHooks(c *gin.Context) {
	var req struct {
		Items []HookRule `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if err := SaveHooks(req.Items); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	state.Poke()
	OKMsg(c, "Hooks 生命周期配置已保存", nil)
}

// =========================================================================
// Conversations Endpoints
// =========================================================================

func handleListConversations(c *gin.Context) {
	includeArchived := c.Query("include_archived") == "true"
	OK(c, gin.H{"items": ListConversations(includeArchived)})
}

func handleCreateConversation(c *gin.Context) {
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	conv, err := CreateConversation(req)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "会话已创建", conv)
}

func handleGetConversation(c *gin.Context) {
	detail, err := GetConversationDetail(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusNotFound, err.Error())
		return
	}
	OK(c, detail)
}

func handleSendConversationMessage(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Prompt) == "" {
		Fail(c, http.StatusBadRequest, "请输入指令内容")
		return
	}
	if err := DispatchConversationPrompt(c.Param("id"), req.Prompt); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "指令已派发至 Agent 执行", nil)
}

func handleUpdateConversation(c *gin.Context) {
	var patch map[string]any
	if err := c.ShouldBindJSON(&patch); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	conv, err := UpdateConversationAnnotations(c.Param("id"), patch)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "会话信息已更新", conv)
}

func handleDeleteConversation(c *gin.Context) {
	if err := DeleteConversation(c.Param("id")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "会话已删除", nil)
}

// =========================================================================
// Workspace & Terminal Endpoints
// =========================================================================

func handleGetWorkspaces(c *gin.Context) {
	OK(c, WorkspacePayload{Items: GetWorkspaceList()})
}

func handleAddWorkspace(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
		Path  string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	item, err := AddWorkspace(req.Title, req.Path)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "工作区已绑定", item)
}

func handleDeleteWorkspace(c *gin.Context) {
	if err := DeleteWorkspace(c.Param("id")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "工作区已移除", nil)
}

func handleTerminalExec(c *gin.Context) {
	var req TerminalExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	res := ExecuteTerminalCommand(req)
	OK(c, res)
}

func handleListWorkspaceFiles(c *gin.Context) {
	res, err := ListWorkspaceFiles(c.Query("path"))
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OK(c, res)
}

func handleReadWorkspaceFile(c *gin.Context) {
	res, err := ReadWorkspaceFile(c.Query("path"))
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OK(c, res)
}

func handleWriteWorkspaceFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误")
		return
	}
	res, err := WriteWorkspaceFile(req.Path, req.Content)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "文件已保存", res)
}

// =========================================================================
// Account & Env Vars Endpoints
// =========================================================================

func handleAccountStatus(c *gin.Context) {
	OK(c, GetAccountStatus())
}

func handleAccountPoolList(c *gin.Context) {
	OK(c, gin.H{"items": GetAccountPoolSanitized()})
}

func handleAccountSave(c *gin.Context) {
	var req AccountProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	acc, err := SaveOrAddAccount(req)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "账号凭据已保存并激活", acc)
}

func handleAccountSwitch(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		Fail(c, http.StatusBadRequest, "请选择目标账号")
		return
	}
	acc, err := SwitchAccount(req.ID)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "账号切换成功", acc)
}

func handleAccountLogout(c *gin.Context) {
	if err := LogoutActiveAccount(); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OKMsg(c, "已退出当前激活账号", nil)
}

func handleAccountDelete(c *gin.Context) {
	if err := DeleteAccount(c.Param("id")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "账号凭据已移除", nil)
}

func handleGetEnvVars(c *gin.Context) {
	items, err := ReadEnvVars()
	if err != nil {
		Fail(c, http.StatusInternalServerError, "读取环境变量失败: "+err.Error())
		return
	}
	OK(c, gin.H{"items": items, "path": getEnvFilePath()})
}

func handleSaveEnvVars(c *gin.Context) {
	var req struct {
		Items []EnvVarItem `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if err := ValidateEnvVars(req.Items); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := WriteEnvVars(req.Items); err != nil {
		Fail(c, http.StatusInternalServerError, "保存环境变量失败: "+err.Error())
		return
	}
	LogInfo("[环境变量] 已保存 %d 项自定义环境变量", len(req.Items))
	OKMsg(c, fmt.Sprintf("已保存 %d 项环境变量", len(req.Items)), gin.H{"count": len(req.Items)})
}

// =========================================================================
// Snapshots Endpoints
// =========================================================================

func handleListSnapshots(c *gin.Context) {
	summary, err := ListSnapshots()
	if err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, summary)
}

func handleCreateSnapshot(c *gin.Context) {
	var params CreateSnapshotParams
	_ = c.ShouldBindJSON(&params)
	if strings.TrimSpace(params.Name) == "" {
		params.Name = "手动快照_" + time.Now().Format("0102_1504")
	}
	go func() {
		_, _ = CreateSnapshot(params)
	}()
	OKMsg(c, "快照创建任务已启动", gin.H{"name": params.Name})
}

func handleCancelSnapshot(c *gin.Context) {
	if err := CancelSnapshotCreation(); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "已发送取消指令", nil)
}

func handleRestoreSnapshot(c *gin.Context) {
	id := c.Param("id")
	go func() {
		_ = RestoreSnapshot(id)
	}()
	OKMsg(c, "快照还原任务已启动", gin.H{"id": id})
}

func handleDeleteSnapshot(c *gin.Context) {
	if err := DeleteSnapshot(c.Param("id")); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OKMsg(c, "快照已删除", nil)
}

// =========================================================================
// Logs & Config Endpoints
// =========================================================================

func readLastNLines(path string, maxLines int) ([]string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, "", err
	}
	size := stat.Size()
	if size == 0 {
		return []string{}, "", nil
	}
	readSize := int64(256 * 1024)
	if size <= 512*1024 {
		readSize = size
	}
	buf := make([]byte, readSize)
	_, err = file.ReadAt(buf, size-readSize)
	if err != nil && err != io.EOF {
		return nil, "", err
	}
	raw := string(buf)
	if readSize < size {
		if idx := strings.IndexByte(raw, '\n'); idx >= 0 {
			raw = raw[idx+1:]
		}
	}
	var lines []string
	for _, l := range strings.Split(raw, "\n") {
		if l != "" {
			lines = append(lines, l+"\n")
		}
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, strings.Join(lines, ""), nil
}

func handleGetLogs(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "400"))
	if err != nil || limit <= 0 {
		limit = 400
	}
	lines, content, err := readLastNLines(GetLogFilePath(), limit)
	if err != nil {
		if os.IsNotExist(err) {
			OK(c, gin.H{"lines": []string{}, "content": ""})
			return
		}
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OK(c, gin.H{"lines": lines, "content": content})
}

func handleDeleteLogs(c *gin.Context) {
	if err := ClearLogs(); err != nil {
		Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	OKMsg(c, "运行日志已清空", true)
}

func handleDownloadLogs(c *gin.Context) {
	c.FileAttachment(GetLogFilePath(), "antigravity-harness.log")
}

func handleGetConfig(c *gin.Context) {
	OK(c, GetConfig())
}

func checkPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	_ = ln.Close()
	return nil
}

func handleSaveConfig(c *gin.Context) {
	var cfg Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		Fail(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if cfg.ServerPort < 1 || cfg.ServerPort > 65535 || cfg.ProxyPort < 1 || cfg.ProxyPort > 65535 {
		Fail(c, http.StatusBadRequest, "端口号必须在 1 ~ 65535 之间")
		return
	}
	if cfg.ServerPort == cfg.ProxyPort {
		Fail(c, http.StatusBadRequest, "内部运行端口与控制台对外端口不能相同")
		return
	}

	oldCfg := GetConfig()
	serverPortChanged := oldCfg.ServerPort != cfg.ServerPort
	if serverPortChanged {
		if err := checkPortAvailable(cfg.ServerPort); err != nil {
			Fail(c, http.StatusBadRequest, fmt.Sprintf("内部运行端口 %d 已被占用", cfg.ServerPort))
			return
		}
	}

	if err := SaveConfig(cfg); err != nil {
		Fail(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	LogInfo("[配置] 应用设置已保存 (默认模型=%s, 沙箱=%s, 内部端口=%d, 控制台端口=%d)",
		cfg.GetDefaultModel(), cfg.GetSandboxMode(), cfg.GetServerPort(), cfg.GetProxyPort())

	if serverPortChanged && state.Status() == StatusRunning {
		go func() {
			_ = RestartHarness()
		}()
	}
	state.Poke()
	OKMsg(c, "应用设置已保存", GetConfig())
}
