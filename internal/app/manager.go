package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"harnezpad/internal/gateway"
	"harnezpad/internal/keys"
	"harnezpad/internal/launch"
	"harnezpad/internal/platform"
	"harnezpad/internal/update"
)

const DefaultGateway = "https://gateway.example.com"

type Settings = gateway.Settings
type Model = gateway.Model

type Integration struct {
	ID, Name string
	Running  bool
	PID      int
	Error    string
}

type Manager struct {
	mu             sync.Mutex
	Settings       Settings
	procs          map[string]*exec.Cmd
	errors         map[string]string
	Updater        *update.AppUpdater
	childOut       io.Writer
	childErr       io.Writer
	saveNamedKey   func(string, string) error
	deleteNamedKey func(string) error
}

func (m *Manager) setChildOutput(stdout, stderr io.Writer) {
	m.childOut = stdout
	m.childErr = stderr
}

func (m *Manager) saveStoredNamedKey(slug, token string) error {
	if m.saveNamedKey != nil {
		return m.saveNamedKey(slug, token)
	}
	return platform.SaveNamedKey(slug, token)
}

func (m *Manager) deleteStoredNamedKey(slug string) error {
	if m.deleteNamedKey != nil {
		return m.deleteNamedKey(slug)
	}
	return platform.DeleteNamedKey(slug)
}

var Active *Manager

var configPathOverride string

func (m *Manager) GatewayURL() string { return m.Settings.GatewayURL }

func (m *Manager) Token() string { return m.Settings.Token }

func configPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	d, _ := os.UserConfigDir()
	return filepath.Join(d, "HarnezPad", "settings.json")
}

func (m *Manager) Load() {
	b, err := os.ReadFile(configPath())
	if err == nil {
		_ = json.Unmarshal(b, &m.Settings)
	}
	m.Settings.GatewayURL = DefaultGateway
	if strings.TrimSpace(m.Settings.DefaultKeySlug) == "" {
		m.Settings.DefaultKeySlug = keys.ManagementSlug
	}
	m.Settings.Token = platform.LoadGatewayToken()
}

func (m *Manager) DefaultKeySlug() string {
	if strings.TrimSpace(m.Settings.DefaultKeySlug) == "" {
		return keys.ManagementSlug
	}
	return strings.TrimSpace(m.Settings.DefaultKeySlug)
}

func (m *Manager) tokenForSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	token := platform.LoadNamedKey(slug)
	if token == "" && slug == keys.ManagementSlug {
		token = strings.TrimSpace(m.Settings.Token)
	}
	return token
}

func (m *Manager) ResolveLaunchToken(slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	usingDefault := slug == ""
	if usingDefault {
		slug = m.DefaultKeySlug()
	}
	token := m.tokenForSlug(slug)
	if token == "" && slug != keys.ManagementSlug && (usingDefault || slug == m.DefaultKeySlug()) {
		token = m.tokenForSlug(keys.ManagementSlug)
	}
	if token == "" {
		return "", fmt.Errorf("key %q is not configured; save it in HarnezPad Settings or create it on the Keys page", slug)
	}
	return token, nil
}

func (m *Manager) ensureDefaultKeyAvailable() error {
	slug := m.DefaultKeySlug()
	if m.tokenForSlug(slug) != "" {
		return nil
	}
	if slug == keys.ManagementSlug {
		return nil
	}
	m.Settings.DefaultKeySlug = keys.ManagementSlug
	return m.Save()
}

func (m *Manager) ListModelsForKey(ctx context.Context, keySlug string) ([]Model, error) {
	token, err := m.ResolveLaunchToken(keySlug)
	if err != nil {
		return nil, err
	}
	return m.listModelsWithToken(ctx, token)
}

func (m *Manager) listModelsWithToken(ctx context.Context, token string) ([]Model, error) {
	entries, err := gateway.NewClient(m.Settings.GatewayURL, token).ListModelGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(entries))
	for _, entry := range entries {
		model := Model{ID: entry.ID}
		if len(entry.Providers) > 0 {
			model.OwnedBy = entry.Providers[0]
		}
		out = append(out, model)
	}
	return out, nil
}

func (m *Manager) Save() error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m.Settings, "", "  ")
	return os.WriteFile(p, b, 0600)
}

func (m *Manager) ListModels(ctx context.Context) ([]Model, error) {
	return m.ListModelsForKey(ctx, keys.ManagementSlug)
}

func (m *Manager) RequireGatewayToken() error {
	if strings.TrimSpace(m.Settings.Token) == "" {
		return errors.New("management key is not configured; save it in HarnezPad Settings before launching")
	}
	return nil
}

type SettingsResponse struct {
	GatewayURL      string `json:"gatewayUrl"`
	TokenConfigured bool   `json:"tokenConfigured"`
	TokenValid      bool   `json:"tokenValid"`
	SetupReason     string `json:"setupReason,omitempty"`
	DefaultKeySlug  string `json:"defaultKeySlug"`
}

func (m *Manager) validateManagementKeyBeforeSave(ctx context.Context, token string) error {
	if keys.IsTestInvalidManagementKey(token) {
		return nil
	}
	return m.ValidateManagementToken(ctx, token)
}

func (m *Manager) ValidateManagementToken(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("management key is not configured")
	}
	if keys.IsTestInvalidManagementKey(token) {
		return fmt.Errorf("management key is invalid")
	}
	return gateway.NewClient(m.Settings.GatewayURL, token).ValidateManagementKey(ctx)
}

func (m *Manager) validateStoredManagementKey(ctx context.Context) error {
	if keys.IsTestInvalidManagementKey(m.Settings.Token) {
		return fmt.Errorf("management key is invalid")
	}
	return m.gatewayClient().ValidateManagementKey(ctx)
}

func (m *Manager) SaveManagementToken(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("management key cannot be empty")
	}
	if err := m.validateManagementKeyBeforeSave(ctx, token); err != nil {
		return err
	}
	if err := platform.SaveGatewayToken(token); err != nil {
		return fmt.Errorf("save management key: %w", err)
	}
	m.Settings.Token = token
	if strings.TrimSpace(m.Settings.DefaultKeySlug) == "" {
		m.Settings.DefaultKeySlug = keys.ManagementSlug
	}
	return m.Save()
}

func (m *Manager) SettingsResponse(ctx context.Context) SettingsResponse {
	configured := strings.TrimSpace(m.Settings.Token) != ""
	resp := SettingsResponse{
		GatewayURL:      m.Settings.GatewayURL,
		TokenConfigured: configured,
		TokenValid:      configured,
		DefaultKeySlug:  m.DefaultKeySlug(),
	}
	if !configured {
		resp.TokenValid = false
		resp.SetupReason = "missing"
		return resp
	}
	if err := m.validateStoredManagementKey(ctx); err != nil {
		if gateway.IsManagementKeyAuthError(err) {
			resp.TokenValid = false
			if strings.Contains(strings.ToLower(err.Error()), "expired") {
				resp.SetupReason = "expired"
			} else {
				resp.SetupReason = "invalid"
			}
		}
	}
	return resp
}

func (m *Manager) gatewayClient() *gateway.Client {
	return gateway.NewClient(m.Settings.GatewayURL, m.Settings.Token)
}

func LookupExecutable(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, ".npm-global", "bin", name),
		filepath.Join("/opt/homebrew/bin", name),
		filepath.Join("/usr/local/bin", name),
	}
	if name == "codex" {
		matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", name))
		for i := len(matches) - 1; i >= 0; i-- {
			candidates = append(candidates, matches[i])
		}
	}
	if name == "opencode" {
		candidates = append(candidates, filepath.Join(home, ".opencode", "bin", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s is not installed", name)
}

func (m *Manager) CommandWithArgs(id string, args []string) (*exec.Cmd, error) {
	switch id {
	case "claude":
		path, err := LookupExecutable("claude")
		if err != nil {
			return nil, err
		}
		return exec.Command(path, args...), nil
	case "codex-cli":
		path, err := LookupExecutable("codex")
		if err != nil {
			return nil, err
		}
		return exec.Command(path, args...), nil
	case "codex-desktop":
		return launch.ChatGPTLaunchCommand()
	case "opencode":
		path, err := LookupExecutable("opencode")
		if err != nil {
			return nil, err
		}
		return exec.Command(path, args...), nil
	}
	return nil, errors.New("unknown integration")
}

func (m *Manager) ApplyEnvironmentForModel(cmd *exec.Cmd, id, model, launchToken string) error {
	if strings.TrimSpace(launchToken) == "" {
		return errors.New("management key is not configured; save it in HarnezPad Settings before launching")
	}
	gatewayURL := strings.TrimRight(m.Settings.GatewayURL, "/")
	base := gatewayURL + "/v1"
	overrides := map[string]string{"HARNEZPAD_MODEL": model}
	var unset []string
	switch id {
	case "claude":
		overrides["ANTHROPIC_BASE_URL"] = gatewayURL
		overrides["ANTHROPIC_API_KEY"] = ""
		overrides["ANTHROPIC_AUTH_TOKEN"] = launchToken
		overrides["ANTHROPIC_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		overrides["CLAUDE_CODE_SUBAGENT_MODEL"] = model
		overrides["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"] = "1"
		unset = append(unset, launch.ClaudeProviderEnvironment...)
	case "codex-cli":
		overrides["OPENAI_API_KEY"] = launchToken
	case "codex-desktop":
		overrides["OPENAI_API_KEY"] = launchToken
		unset = append(unset, "OPENAI_BASE_URL", "OPENAI_MODEL")
	case "opencode":
		if model == "" {
			return errors.New("select a model before launching OpenCode")
		}
		config, err := launch.BuildOpenCodeConfig(base, launchToken, model)
		if err != nil {
			return err
		}
		overrides["OPENAI_BASE_URL"] = base
		overrides["OPENAI_API_KEY"] = launchToken
		overrides["OPENCODE_CONFIG_CONTENT"] = config
	}
	cmd.Env = launch.IsolatedChildEnvironment(overrides, unset)
	return nil
}

func (m *Manager) Start(id, modelOverride string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p := m.procs[id]; p != nil && p.Process != nil {
		return nil
	}
	model := strings.TrimSpace(modelOverride)
	if model == "" {
		return fmt.Errorf("no model selected for %s; use harnezpad launch %s from a terminal", id, id)
	}
	if err := m.RequireGatewayToken(); err != nil {
		return err
	}
	launchToken, err := m.ResolveLaunchToken("")
	if err != nil {
		return err
	}
	var args []string
	if id == "codex-desktop" {
		models, discoverErr := m.ListModels(context.Background())
		if discoverErr != nil {
			models = []Model{{ID: model}}
		}
		if err := launch.ConfigureChatGPT(m, model, models); err != nil {
			return err
		}
	} else if id == "codex-cli" {
		models, discoverErr := m.ListModels(context.Background())
		if discoverErr != nil {
			models = []Model{{ID: model}}
		}
		if err := launch.Configure(m, model, models); err != nil {
			return err
		}
		args, err = launch.LaunchArgs(m, nil, model)
		if err != nil {
			return err
		}
	} else if id == "claude" {
		args = launch.ClaudeLaunchArgs(nil, model)
	}
	c, err := m.CommandWithArgs(id, args)
	if err != nil {
		return err
	}
	if err := m.ApplyEnvironmentForModel(c, id, model, launchToken); err != nil {
		return err
	}
	c.Stdin = os.Stdin
	c.Stdout = m.childOut
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	c.Stderr = m.childErr
	if c.Stderr == nil {
		c.Stderr = os.Stderr
	}
	if err := c.Start(); err != nil {
		return err
	}
	m.procs[id] = c
	go func() {
		err := c.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.procs, id)
		if err != nil {
			m.errors[id] = err.Error()
		}
	}()
	return nil
}

func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.procs[id]
	if p == nil || p.Process == nil {
		return nil
	}
	return p.Process.Signal(os.Interrupt)
}

func (m *Manager) Integrations() []Integration {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := []string{"claude", "codex-cli", "codex-desktop", "opencode"}
	names := []string{"Claude Code", "Codex", "ChatGPT", "OpenCode"}
	out := make([]Integration, 0, len(ids))
	for i, id := range ids {
		v := Integration{ID: id, Name: names[i]}
		if p := m.procs[id]; p != nil && p.Process != nil {
			v.Running = true
			v.PID = p.Process.Pid
		}
		v.Error = m.errors[id]
		out = append(out, v)
	}
	return out
}

func (m *Manager) RunForeground(id, model string, args []string, keySlug string) error {
	launchToken, err := m.ResolveLaunchToken(keySlug)
	if err != nil {
		return err
	}
	cmd, err := m.CommandWithArgs(id, args)
	if err != nil {
		return err
	}
	if err := m.ApplyEnvironmentForModel(cmd, id, model, launchToken); err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if id == "codex-desktop" {
		return cmd.Start()
	}
	return cmd.Run()
}

func IntegrationDisplayName(id string) string {
	switch id {
	case "claude":
		return "Claude Code"
	case "codex-cli":
		return "Codex"
	case "codex-desktop":
		return "ChatGPT"
	case "opencode":
		return "OpenCode"
	default:
		return id
	}
}
