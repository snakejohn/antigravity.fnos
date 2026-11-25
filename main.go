package main

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist/*
var embeddedWebFS embed.FS

//go:embed templates/*
var embeddedTemplatesFS embed.FS

var (
	globalPkgVar       string
	globalAppVer       string
	globalAppDest      string
	globalRunUser      string
	globalGeminiHome   string
	globalConfigDir    string
	globalBrainDir     string
	globalCliConfigDir string
	globalSnapshotsDir string
	globalDataDir      string
)

const AppVersion = "1.0.0"

func main() {
	globalPkgVar = strings.TrimSpace(os.Getenv("DATA_LIBRARY_PATH"))
	if globalPkgVar == "" {
		if trimVar := strings.TrimSpace(os.Getenv("TRIM_PKGVAR")); trimVar != "" {
			globalPkgVar = trimVar
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			globalPkgVar = filepath.Join(cwd, "agy-data")
		}
	}
	if abs, err := filepath.Abs(globalPkgVar); err == nil {
		globalPkgVar = abs
	}

	globalAppDest = strings.TrimSpace(os.Getenv("TRIM_APPDEST"))
	globalAppVer = strings.TrimSpace(os.Getenv("TRIM_APPVER"))
	if globalAppVer == "" {
		globalAppVer = AppVersion
	}

	globalRunUser = strings.TrimSpace(os.Getenv("AGY_RUN_USER"))
	if globalRunUser == "" {
		globalRunUser = "root"
	}

	globalDataDir = filepath.Join(globalPkgVar, "data")
	globalSnapshotsDir = filepath.Join(globalPkgVar, "snapshots")

	// Resolve Gemini/Antigravity home directory
	if envGemini := strings.TrimSpace(os.Getenv("AGY_GEMINI_HOME")); envGemini != "" {
		globalGeminiHome = envGemini
	} else {
		globalGeminiHome = filepath.Join(globalPkgVar, "gemini-home")
	}
	globalConfigDir = filepath.Join(globalGeminiHome, "config")
	globalBrainDir = filepath.Join(globalGeminiHome, "antigravity", "brain")
	globalCliConfigDir = filepath.Join(globalGeminiHome, "antigravity-cli")

	for _, dir := range []string{
		globalPkgVar,
		globalDataDir,
		globalSnapshotsDir,
		globalGeminiHome,
		globalConfigDir,
		filepath.Join(globalConfigDir, "skills"),
		filepath.Join(globalConfigDir, "plugins"),
		globalBrainDir,
		globalCliConfigDir,
	} {
		_ = os.MkdirAll(dir, 0755)
	}

	InitLogger()
	LogInfo("[主程序] Antigravity Harness Console v%s 启动初始化 (OS=%s/%s, DATA=%s, GEMINI_HOME=%s)",
		globalAppVer, runtime.GOOS, runtime.GOARCH, globalPkgVar, globalGeminiHome)

	InitConfig()
	InitAppEnv()
	EnsureDefaultCustomizations()
	InitWorkspaces()
	InitConversations()
	InitHarness()

	// Seed default sample data if first boot
	cfg := GetConfig()

	// Allow env override for admin port
	adminPort := cfg.GetProxyPort()
	if envPort := strings.TrimSpace(os.Getenv("AGY_ADMIN_PORT")); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 && p <= 65535 {
			adminPort = p
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	WebFS = embeddedWebFS
	InitRoutes(r)

	// Start periodic host & process resource sampler
	StartResourceMonitor()

	// Auto-start internal Antigravity Harness runtime unless explicitly stopped last time
	if cfg.LastRunState != StatusStopped {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := StartHarness(); err != nil {
				LogWarning("[主程序] 自动启动 Antigravity Harness 失败: %s", err)
			}
		}()
	}

	var servers []*http.Server

	// 1. Optional fnOS Native Unix Domain Socket listener
	var unixSocketPath string
	if globalAppDest != "" && runtime.GOOS != "windows" {
		_ = os.MkdirAll(globalAppDest, 0755)
		unixSocketPath = filepath.Join(globalAppDest, "web.sock")
		_ = os.Remove(unixSocketPath)
		unixLn, err := net.Listen("unix", unixSocketPath)
		if err != nil {
			LogWarning("[主程序] 飞牛 Unix Socket 监听失败 [%s]: %s", unixSocketPath, err)
		} else {
			_ = os.Chmod(unixSocketPath, 0666)
			unixSrv := &http.Server{Handler: r}
			servers = append(servers, unixSrv)
			go func() {
				LogInfo("[主程序] 飞牛网关 Unix Socket 已就绪: %s", unixSocketPath)
				if err := unixSrv.Serve(unixLn); err != nil && err != http.ErrServerClosed {
					LogError("[主程序] Unix Socket 服务异常退出: %s", err)
				}
			}()
		}
	}

	// 2. Primary TCP HTTP Listener (0.0.0.0:<adminPort>, default 2399)
	tcpAddr := fmt.Sprintf("0.0.0.0:%d", adminPort)
	tcpLn, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		LogFatal("[主程序] HTTP TCP 监听失败 [%s]: %s", tcpAddr, err)
	}
	tcpSrv := &http.Server{Handler: r}
	servers = append(servers, tcpSrv)

	// Graceful shutdown handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		LogInfo("[主程序] 捕获终止信号 %s，执行停机流程...", sig)
		_ = StopHarness()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, srv := range servers {
			_ = srv.Shutdown(ctx)
		}
		if unixSocketPath != "" {
			_ = os.Remove(unixSocketPath)
		}
		os.Exit(0)
	}()

	LogInfo("[主程序] HTTP 管理控制台已就绪，监听地址: http://%s (同时支持 / 与 /app/antigravity-harness/)", tcpAddr)
	if err := tcpSrv.Serve(tcpLn); err != nil && err != http.ErrServerClosed {
		LogFatal("[主程序] HTTP 服务异常退出: %s", err)
	}
}
