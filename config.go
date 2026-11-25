package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	DefaultServerPort  = 2398
	DefaultProxyPort   = 2399
	DefaultNpmRegistry = "https://registry.npmmirror.com"
	DefaultPyPIIndex   = "https://pypi.tuna.tsinghua.edu.cn/simple"
)

type Config struct {
	ServerPort          int    `json:"server_port"`
	ProxyPort           int    `json:"proxy_port"`
	DefaultModel        string `json:"default_model"`
	SandboxMode         string `json:"sandbox_mode"`
	ApprovalPolicy      string `json:"approval_policy"`
	HeapMemoryLimit     int    `json:"heap_memory_limit,omitempty"`
	NetworkProxy        string `json:"network_proxy"`
	ProxyRuntimeEnabled *bool  `json:"proxy_runtime_enabled,omitempty"`
	NpmRegistry         string `json:"npm_registry,omitempty"`
	PyPIIndex           string `json:"pypi_index,omitempty"`
	UpdateChannel       string `json:"update_channel,omitempty"`
	AccessMode          string `json:"access_mode,omitempty"`
	ReverseProxyURL     string `json:"reverse_proxy_url,omitempty"`
	AccessPassword      string `json:"access_password,omitempty"`
	DataLibraryPath     string `json:"data_library_path,omitempty"`
	GeminiHome          string `json:"gemini_home,omitempty"`
	Version             string `json:"version,omitempty"`
	BuildTime           string `json:"build_time,omitempty"`
	LastRunState        string `json:"last_run_state,omitempty"`
}

func (c Config) IsProxyRuntimeEnabled() bool {
	if c.ProxyRuntimeEnabled == nil {
		return false
	}
	return *c.ProxyRuntimeEnabled
}

func (c Config) GetServerPort() int {
	if c.ServerPort <= 0 {
		return DefaultServerPort
	}
	return c.ServerPort
}

func (c Config) GetProxyPort() int {
	if c.ProxyPort <= 0 {
		return DefaultProxyPort
	}
	return c.ProxyPort
}

func (c Config) GetDefaultModel() string {
	m := strings.TrimSpace(c.DefaultModel)
	if m == "" {
		return "gemini-2.5-pro"
	}
	return m
}

func (c Config) GetSandboxMode() string {
	s := strings.TrimSpace(c.SandboxMode)
	switch s {
	case "read-only", "workspace-write", "full-access":
		return s
	default:
		return "workspace-write"
	}
}

func (c Config) GetApprovalPolicy() string {
	p := strings.TrimSpace(c.ApprovalPolicy)
	switch p {
	case "untrusted", "on-request", "never":
		return p
	default:
		return "on-request"
	}
}

func (c Config) GetUpdateChannel() string {
	switch strings.ToLower(strings.TrimSpace(c.UpdateChannel)) {
	case "latest", "beta", "canary":
		return strings.ToLower(strings.TrimSpace(c.UpdateChannel))
	default:
		return "latest"
	}
}

func (c Config) GetNpmRegistry() string {
	if t := strings.TrimSpace(c.NpmRegistry); t != "" {
		return t
	}
	return DefaultNpmRegistry
}

func (c Config) GetPyPIIndex() string {
	if t := strings.TrimSpace(c.PyPIIndex); t != "" {
		return t
	}
	return DefaultPyPIIndex
}

var (
	globalConfig   Config
	configMu       sync.RWMutex
	configFilePath string
)

func InitConfig() {
	configFilePath = filepath.Join(globalPkgVar, "config.json")
	defaultProxy := false
	globalConfig = Config{
		ServerPort:          DefaultServerPort,
		ProxyPort:           DefaultProxyPort,
		DefaultModel:        "gemini-2.5-pro",
		SandboxMode:         "workspace-write",
		ApprovalPolicy:      "on-request",
		AccessMode:          "fngateway",
		DataLibraryPath:     globalPkgVar,
		GeminiHome:          globalGeminiHome,
		ProxyRuntimeEnabled: &defaultProxy,
		NpmRegistry:         DefaultNpmRegistry,
		PyPIIndex:           DefaultPyPIIndex,
		UpdateChannel:       "latest",
		Version:             "2.0.4",
		BuildTime:           time.Now().Format("2006-01-02 15:04"),
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		persistConfig()
		syncCliSettings()
		return
	}
	_ = json.Unmarshal(data, &globalConfig)
	if globalConfig.ProxyRuntimeEnabled == nil {
		globalConfig.ProxyRuntimeEnabled = &defaultProxy
	}
	if globalConfig.DefaultModel == "" {
		globalConfig.DefaultModel = "gemini-2.5-pro"
	}
	if globalConfig.SandboxMode == "" {
		globalConfig.SandboxMode = "workspace-write"
	}
	if globalConfig.ApprovalPolicy == "" {
		globalConfig.ApprovalPolicy = "on-request"
	}
	if globalConfig.NpmRegistry == "" {
		globalConfig.NpmRegistry = DefaultNpmRegistry
	}
	if globalConfig.PyPIIndex == "" {
		globalConfig.PyPIIndex = DefaultPyPIIndex
	}
	if globalConfig.AccessMode == "" {
		globalConfig.AccessMode = "fngateway"
	}
	if globalConfig.Version == "" {
		globalConfig.Version = "2.0.4"
	}
	if globalConfig.BuildTime == "" {
		globalConfig.BuildTime = time.Now().Format("2006-01-02 15:04")
	}
	globalConfig.DataLibraryPath = globalPkgVar
	globalConfig.GeminiHome = globalGeminiHome
	syncCliSettings()
}

func syncCliSettings() {
	cfg := GetConfig()
	cliSettingsPath := filepath.Join(globalCliConfigDir, "settings.json")
	settings := map[string]any{
		"defaultModel":   cfg.GetDefaultModel(),
		"sandboxMode":    cfg.GetSandboxMode(),
		"approvalPolicy": cfg.GetApprovalPolicy(),
		"networkProxy":   cfg.NetworkProxy,
		"updatedAt":      time.Now().Format(time.RFC3339),
	}
	if data, err := json.MarshalIndent(settings, "", "  "); err == nil {
		_ = os.WriteFile(cliSettingsPath, data, 0644)
	}
}

func InitAppEnv() {
	if runtime.GOOS != "windows" {
		// Ensure fnOS nodejs_v24 / nodejs_v22 binaries are discoverable in PATH
		extraPaths := []string{
			"/var/apps/nodejs_v24/target/bin",
			"/var/apps/nodejs_v22/target/bin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
		}
		curPath := os.Getenv("PATH")
		parts := make([]string, 0, len(extraPaths)+1)
		for _, p := range extraPaths {
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				parts = append(parts, p)
			}
		}
		if curPath != "" {
			parts = append(parts, curPath)
		}
		_ = os.Setenv("PATH", strings.Join(parts, ":"))
	}

	ApplyProxyEnv()
	ApplyRegistryEnv()
	// Load custom .env into process env for agent execution
	if items, err := ReadEnvVars(); err == nil {
		for _, item := range items {
			if item.Key != "" && item.Value != "" {
				_ = os.Setenv(item.Key, item.Value)
			}
		}
	}
}

func ApplyRegistryEnv() {
	cfg := GetConfig()
	_ = os.Setenv("npm_config_registry", cfg.GetNpmRegistry())
	_ = os.Setenv("NPM_CONFIG_REGISTRY", cfg.GetNpmRegistry())
	_ = os.Setenv("PIP_INDEX_URL", cfg.GetPyPIIndex())
}

func ApplyProxyEnv() {
	cfg := GetConfig()
	if cfg.NetworkProxy != "" && cfg.IsProxyRuntimeEnabled() {
		noProxy := "localhost,127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,*.local"
		_ = os.Setenv("HTTP_PROXY", cfg.NetworkProxy)
		_ = os.Setenv("HTTPS_PROXY", cfg.NetworkProxy)
		_ = os.Setenv("ALL_PROXY", cfg.NetworkProxy)
		_ = os.Setenv("http_proxy", cfg.NetworkProxy)
		_ = os.Setenv("https_proxy", cfg.NetworkProxy)
		_ = os.Setenv("all_proxy", cfg.NetworkProxy)
		_ = os.Setenv("NO_PROXY", noProxy)
		_ = os.Setenv("no_proxy", noProxy)
	} else {
		for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
			_ = os.Unsetenv(k)
		}
	}
}

func GetConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

func persistConfig() {
	configMu.RLock()
	data, err := json.MarshalIndent(globalConfig, "", "  ")
	configMu.RUnlock()
	if err != nil {
		return
	}
	tmpFile := configFilePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmpFile, configFilePath)
}

func SetLastRunState(st string) {
	configMu.Lock()
	globalConfig.LastRunState = st
	configMu.Unlock()
	persistConfig()
}

func SetHarnessVersion(v string) {
	configMu.Lock()
	globalConfig.Version = v
	globalConfig.BuildTime = time.Now().Format("2006-01-02 15:04")
	configMu.Unlock()
	persistConfig()
}

func SaveConfig(cfg Config) error {
	configMu.Lock()
	if cfg.LastRunState == "" && globalConfig.LastRunState != "" {
		cfg.LastRunState = globalConfig.LastRunState
	}
	if cfg.Version == "" {
		cfg.Version = globalConfig.Version
	}
	if cfg.BuildTime == "" {
		cfg.BuildTime = globalConfig.BuildTime
	}
	cfg.DataLibraryPath = globalPkgVar
	cfg.GeminiHome = globalGeminiHome
	globalConfig = cfg
	configMu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := configFilePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, configFilePath); err != nil {
		return err
	}
	ApplyProxyEnv()
	ApplyRegistryEnv()
	syncCliSettings()
	return nil
}

// =========================================================================
// .env Environment Variables Manager
// =========================================================================

type EnvVarItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func getEnvFilePath() string {
	return filepath.Join(globalDataDir, ".env")
}

func EnsureEnvFile() (string, error) {
	_ = os.MkdirAll(globalDataDir, 0755)
	envPath := getEnvFilePath()
	if _, err := os.Stat(envPath); err == nil {
		return envPath, nil
	}
	defaultContent := "# Antigravity Harness Environment Variables\nANTIGRAVITY_DEFAULT_MODEL=gemini-2.5-pro\n"
	if err := os.WriteFile(envPath, []byte(defaultContent), 0600); err != nil {
		return "", err
	}
	return envPath, nil
}

func ValidateEnvVars(items []EnvVarItem) error {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			return fmt.Errorf("变量名不能为空")
		}
		if !envKeyPattern.MatchString(key) {
			return fmt.Errorf("变量名「%s」不合法，仅允许字母、数字与下划线，且不能以数字开头", key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("变量名「%s」重复", key)
		}
		seen[key] = struct{}{}
		if strings.ContainsAny(item.Value, "\r\n") {
			return fmt.Errorf("变量「%s」的值不能包含换行符", key)
		}
	}
	return nil
}

func ReadEnvVars() ([]EnvVarItem, error) {
	envPath, err := EnsureEnvFile()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(envPath)
	if err != nil {
		return nil, err
	}
	items := make([]EnvVarItem, 0)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok || !envKeyPattern.MatchString(k) {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		items = append(items, EnvVarItem{Key: k, Value: v})
	}
	return items, nil
}

func WriteEnvVars(items []EnvVarItem) error {
	envPath, err := EnsureEnvFile()
	if err != nil {
		return err
	}
	lines := []string{"# Antigravity Harness Environment Variables"}
	for _, item := range items {
		k := strings.TrimSpace(item.Key)
		v := item.Value
		if strings.ContainsAny(v, " \"'#\\") {
			v = `"` + strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`) + `"`
		}
		lines = append(lines, k+"="+v)
		_ = os.Setenv(k, item.Value)
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(envPath, []byte(content), 0600)
}

func GetEnvVarValue(key string) string {
	if items, err := ReadEnvVars(); err == nil {
		for _, item := range items {
			if item.Key == key && strings.TrimSpace(item.Value) != "" {
				return strings.TrimSpace(item.Value)
			}
		}
	}
	return strings.TrimSpace(os.Getenv(key))
}

// =========================================================================
// Antigravity / Google Gemini / Vertex AI Account Pool Manager
// =========================================================================

type AccountProfile struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"` // gemini_api | vertex_ai | antigravity_oauth | openai_compat
	Email         string `json:"email,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	MaskedKey     string `json:"masked_key,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	Region        string `json:"region,omitempty"`
	DefaultModel  string `json:"default_model,omitempty"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     int64  `json:"created_at"`
	LastCheckedAt int64  `json:"last_checked_at,omitempty"`
}

type AccountStatusResponse struct {
	Authorized     bool            `json:"authorized"`
	ActiveAccount  *AccountProfile `json:"active_account,omitempty"`
	PoolCount      int             `json:"pool_count"`
	CliDetected    bool            `json:"cli_detected"`
	SdkDetected    bool            `json:"sdk_detected"`
	LsConnected    bool            `json:"ls_connected"`
	EnvHasGemini   bool            `json:"env_has_gemini"`
	ConsoleURL     string          `json:"console_url"`
	DocsURL        string          `json:"docs_url"`
}

var accountMu sync.Mutex

func getAccountPoolPath() string {
	return filepath.Join(globalDataDir, "account_pool.json")
}

func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + "••••" + s[len(s)-4:]
}

func generateShortID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func loadAccountPoolRaw() []AccountProfile {
	data, err := os.ReadFile(getAccountPoolPath())
	if err != nil {
		return []AccountProfile{}
	}
	var list []AccountProfile
	if err := json.Unmarshal(data, &list); err != nil {
		return []AccountProfile{}
	}
	return list
}

func saveAccountPoolRaw(list []AccountProfile) error {
	_ = os.MkdirAll(globalDataDir, 0755)
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getAccountPoolPath(), data, 0600)
}

func sanitizeAccount(a AccountProfile) AccountProfile {
	out := a
	out.MaskedKey = maskSecret(a.APIKey)
	out.APIKey = ""
	return out
}

func GetAccountPoolSanitized() []AccountProfile {
	accountMu.Lock()
	defer accountMu.Unlock()
	raw := loadAccountPoolRaw()
	out := make([]AccountProfile, 0, len(raw))
	for _, a := range raw {
		out = append(out, sanitizeAccount(a))
	}
	return out
}

func GetActiveAccountRaw() *AccountProfile {
	accountMu.Lock()
	defer accountMu.Unlock()
	raw := loadAccountPoolRaw()
	for i := range raw {
		if raw[i].IsActive {
			cp := raw[i]
			return &cp
		}
	}
	return nil
}

func SaveOrAddAccount(req AccountProfile) (AccountProfile, error) {
	accountMu.Lock()
	defer accountMu.Unlock()

	if strings.TrimSpace(req.Name) == "" {
		return AccountProfile{}, fmt.Errorf("账号名称不能为空")
	}
	if req.Provider == "" {
		req.Provider = "gemini_api"
	}

	list := loadAccountPoolRaw()
	if req.ID == "" {
		req.ID = generateShortID("acc")
		req.CreatedAt = time.Now().Unix()
	}
	req.LastCheckedAt = time.Now().Unix()

	// Deactivate others when saving/activating
	for i := range list {
		list[i].IsActive = false
	}
	req.IsActive = true

	updated := false
	for i := range list {
		if list[i].ID == req.ID {
			if req.APIKey == "" {
				req.APIKey = list[i].APIKey
			}
			req.CreatedAt = list[i].CreatedAt
			list[i] = req
			updated = true
			break
		}
	}
	if !updated {
		list = append([]AccountProfile{req}, list...)
	}

	if err := saveAccountPoolRaw(list); err != nil {
		return AccountProfile{}, err
	}
	applyActiveAccountEnv(req)
	LogInfo("[授权] 已保存并激活账号「%s」(%s)", req.Name, req.Provider)
	return sanitizeAccount(req), nil
}

func SwitchAccount(id string) (*AccountProfile, error) {
	accountMu.Lock()
	defer accountMu.Unlock()
	list := loadAccountPoolRaw()
	var target *AccountProfile
	for i := range list {
		if list[i].ID == id {
			list[i].IsActive = true
			cp := list[i]
			target = &cp
		} else {
			list[i].IsActive = false
		}
	}
	if target == nil {
		return nil, fmt.Errorf("未找到账号凭据: %s", id)
	}
	if err := saveAccountPoolRaw(list); err != nil {
		return nil, err
	}
	applyActiveAccountEnv(*target)
	LogInfo("[授权] 已切换当前激活账号为「%s」", target.Name)
	sanitized := sanitizeAccount(*target)
	return &sanitized, nil
}

func DeleteAccount(id string) error {
	accountMu.Lock()
	defer accountMu.Unlock()
	list := loadAccountPoolRaw()
	next := make([]AccountProfile, 0, len(list))
	found := false
	for _, a := range list {
		if a.ID == id {
			found = true
			continue
		}
		next = append(next, a)
	}
	if !found {
		return fmt.Errorf("未找到账号凭据: %s", id)
	}
	if err := saveAccountPoolRaw(next); err != nil {
		return err
	}
	LogInfo("[授权] 已移除账号凭据: %s", id)
	return nil
}

func LogoutActiveAccount() error {
	accountMu.Lock()
	defer accountMu.Unlock()
	list := loadAccountPoolRaw()
	for i := range list {
		list[i].IsActive = false
	}
	if err := saveAccountPoolRaw(list); err != nil {
		return err
	}
	LogInfo("[授权] 已取消当前激活账号")
	return nil
}

func applyActiveAccountEnv(acc AccountProfile) {
	if acc.APIKey != "" {
		_ = os.Setenv("GEMINI_API_KEY", acc.APIKey)
	}
	if acc.ProjectID != "" {
		_ = os.Setenv("GOOGLE_CLOUD_PROJECT", acc.ProjectID)
	}
}

func GetAccountStatus() AccountStatusResponse {
	pool := GetAccountPoolSanitized()
	var active *AccountProfile
	for i := range pool {
		if pool[i].IsActive {
			cp := pool[i]
			active = &cp
			break
		}
	}
	envHasKey := GetEnvVarValue("GEMINI_API_KEY") != ""
	caps := DetectHostCapabilities()
	return AccountStatusResponse{
		Authorized:    active != nil || envHasKey,
		ActiveAccount: active,
		PoolCount:     len(pool),
		CliDetected:   caps.AgyCLIInstalled,
		SdkDetected:   caps.PythonSDKInstalled,
		LsConnected:   caps.LanguageServerConnected,
		EnvHasGemini:  envHasKey,
		ConsoleURL:    "https://aistudio.google.com/apikey",
		DocsURL:       "https://antigravity.google/docs/getting-started",
	}
}
