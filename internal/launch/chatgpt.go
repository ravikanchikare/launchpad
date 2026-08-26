package launch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"harnezpad/internal/fsutil"
	"harnezpad/internal/gateway"
)

const (
	ChatGPTConfigMarker = "# HARNEZPAD_MANAGED_CHATGPT_CONFIG"
	chatGPTBundleID     = "com.openai.codex"
)

type savedRootString struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

type chatGPTRestoreState struct {
	Model            savedRootString `json:"model"`
	ModelProvider    savedRootString `json:"model_provider"`
	ModelCatalogJSON savedRootString `json:"model_catalog_json"`
}

type legacyChatGPTRestoreState struct {
	Exists  bool   `json:"exists"`
	Content string `json:"content"`
}

var (
	chatGPTCommand       = exec.Command
	chatGPTSleep         = time.Sleep
	chatGPTProcessActive = defaultChatGPTProcessActive
	chatGPTFindBundle    = defaultChatGPTFindBundle
	chatGPTQuit          = defaultChatGPTQuit
	chatGPTTerminate     = defaultChatGPTTerminate
	chatGPTExitTimeout   = 5 * time.Second
)

func chatGPTConfigPath() (string, error) {
	dir, err := codexHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func chatGPTRestorePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "HarnezPad", "chatgpt-restore.json"), nil
}

func chatGPTCatalogPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "harnezpad-chatgpt-models.json")
}

func ConfigureChatGPT(settings GatewaySettings, model string, models []gateway.Model) error {
	configPath, err := chatGPTConfigPath()
	if err != nil {
		return err
	}
	content, readErr := os.ReadFile(configPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	current := normalizeLegacyChatGPTConfig(string(content))
	if err := validateManagedTOML(current); err != nil {
		return err
	}

	statePath, err := chatGPTRestorePath()
	if err != nil {
		return err
	}
	if stateData, stateErr := os.ReadFile(statePath); stateErr != nil {
		if !os.IsNotExist(stateErr) {
			return stateErr
		}
		if err := writeChatGPTRestoreState(captureChatGPTRootState(current)); err != nil {
			return err
		}
	} else {
		var state chatGPTRestoreState
		var legacy legacyChatGPTRestoreState
		if json.Unmarshal(stateData, &legacy) == nil && legacy.Content != "" {
			if err := writeChatGPTRestoreState(captureChatGPTRootState(legacy.Content)); err != nil {
				return err
			}
		} else if json.Unmarshal(stateData, &state) != nil || !chatGPTRootManaged(current) {
			if err := writeChatGPTRestoreState(captureChatGPTRootState(current)); err != nil {
				return err
			}
		}
	}

	if len(models) == 0 {
		models = []gateway.Model{{ID: model}}
	}
	catalogPath := chatGPTCatalogPath(configPath)
	if err := writeModelCatalog(catalogPath, models, "HarnezPad gateway model"); err != nil {
		return err
	}

	updated := removeManagedProviderSection(current)
	updated = setRootStrings(updated, map[string]*string{
		"model":              stringPointer(model),
		"model_provider":     stringPointer("harnezpad"),
		"model_catalog_json": stringPointer(catalogPath),
	})
	provider, err := harnezpadProviderTOML(settings)
	if err != nil {
		return err
	}
	updated = insertBeforeFirstTable(updated, ChatGPTConfigMarker+"\n\n"+provider)
	return fsutil.WriteAtomic(configPath, []byte(updated), 0600)
}

func harnezpadProviderTOML(settings GatewaySettings) (string, error) {
	authCommand, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate HarnezPad token helper: %w", err)
	}
	base := BaseURL(settings)
	return "[model_providers.harnezpad]\n" +
		"name = \"HarnezPad\"\n" +
		"base_url = " + strconv.Quote(base) + "\n" +
		"wire_api = \"responses\"\n\n" +
		"[model_providers.harnezpad.auth]\n" +
		"command = " + strconv.Quote(authCommand) + "\n" +
		"args = [\"_chatgpt-token\"]\n" +
		"timeout_ms = 5000\n" +
		"refresh_interval_ms = 300000\n", nil
}

func writeChatGPTRestoreState(state chatGPTRestoreState) error {
	statePath, err := chatGPTRestorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(statePath, data, 0600)
}

func ResetChatGPT(settings GatewaySettings) error {
	configPath, err := chatGPTConfigPath()
	if err != nil {
		return err
	}
	content, readErr := os.ReadFile(configPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	current := normalizeLegacyChatGPTConfig(string(content))
	if err := validateManagedTOML(current); err != nil {
		return err
	}

	statePath, err := chatGPTRestorePath()
	if err != nil {
		return err
	}
	stateData, stateErr := os.ReadFile(statePath)
	if stateErr == nil {
		var state chatGPTRestoreState
		if err := json.Unmarshal(stateData, &state); err != nil {
			return fmt.Errorf("read ChatGPT restore state: %w", err)
		}
		updated := removeManagedProviderSection(current)
		if chatGPTRootManaged(current) {
			updated = restoreChatGPTRootState(updated, state)
		}
		updated = removeMarker(updated)
		if strings.TrimSpace(updated) == "" && readErr != nil {
			if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else if err := fsutil.WriteAtomic(configPath, []byte(updated), 0600); err != nil {
			return err
		}
		if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if !os.IsNotExist(stateErr) {
		return stateErr
	} else if strings.Contains(current, ChatGPTConfigMarker) {
		updated := removeMarker(removeManagedProviderSection(current))
		updated = setRootStrings(updated, map[string]*string{"model": nil, "model_provider": nil, "model_catalog_json": nil})
		if err := fsutil.WriteAtomic(configPath, []byte(updated), 0600); err != nil {
			return err
		}
	}

	if err := removeOwnedModelCatalog(chatGPTCatalogPath(configPath)); err != nil {
		return err
	}
	if err := relaunchChatGPT(settings); err != nil {
		return fmt.Errorf("ChatGPT configuration was restored, but automatic restart failed: %w; quit and reopen ChatGPT manually", err)
	}
	return nil
}

func ChatGPTLaunchCommand() (*exec.Cmd, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("ChatGPT launching is currently supported on macOS only")
	}
	if err := stopChatGPT(); err != nil {
		return nil, err
	}
	bundle, err := chatGPTFindBundle()
	if err != nil {
		return nil, err
	}
	return chatGPTCommand("open", "-a", bundle), nil
}

func relaunchChatGPT(settings GatewaySettings) error {
	cmd, err := ChatGPTLaunchCommand()
	if err != nil {
		return err
	}
	cmd.Env = IsolatedChildEnvironment(nil, []string{"HARNEZPAD_MODEL", "OPENAI_API_KEY", "OPENAI_BASE_URL", "OPENAI_MODEL"})
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Start()
}

func stopChatGPT() error {
	if !chatGPTProcessActive() {
		return nil
	}
	if err := chatGPTQuit(); err != nil {
		return fmt.Errorf("quit ChatGPT: %w", err)
	}
	if waitForChatGPTExit(chatGPTExitTimeout) {
		return nil
	}
	if err := chatGPTTerminate(); err != nil {
		return fmt.Errorf("terminate ChatGPT gracefully: %w", err)
	}
	if waitForChatGPTExit(chatGPTExitTimeout) {
		return nil
	}
	return errors.New("ChatGPT did not quit before the restart timeout")
}

func waitForChatGPTExit(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !chatGPTProcessActive() {
			return true
		}
		chatGPTSleep(100 * time.Millisecond)
	}
	return !chatGPTProcessActive()
}

func defaultChatGPTProcessActive() bool {
	out, err := chatGPTCommand("osascript", "-e", `application id "`+chatGPTBundleID+`" is running`).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func defaultChatGPTQuit() error {
	return chatGPTCommand("osascript", "-e", `tell application id "`+chatGPTBundleID+`" to quit`).Run()
}

func defaultChatGPTTerminate() error {
	out, err := chatGPTCommand("ps", "ax", "-o", "pid=,command=").Output()
	if err != nil {
		return err
	}
	found := false
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		command := strings.Join(fields[1:], " ")
		if !strings.HasSuffix(command, "/Applications/ChatGPT.app/Contents/MacOS/ChatGPT") &&
			!strings.HasSuffix(command, "/Applications/Codex.app/Contents/MacOS/Codex") {
			continue
		}
		pid, parseErr := strconv.Atoi(fields[0])
		if parseErr != nil {
			continue
		}
		process, findErr := os.FindProcess(pid)
		if findErr != nil {
			return findErr
		}
		if terminateErr := process.Signal(syscall.SIGTERM); terminateErr != nil {
			return terminateErr
		}
		found = true
	}
	if !found {
		return errors.New("running ChatGPT process was not found")
	}
	return nil
}

func defaultChatGPTFindBundle() (string, error) {
	home, _ := os.UserHomeDir()
	for _, path := range []string{
		"/Applications/ChatGPT.app", "/Applications/Codex.app",
		filepath.Join(home, "Applications", "ChatGPT.app"), filepath.Join(home, "Applications", "Codex.app"),
	} {
		if info, err := os.Stat(filepath.Join(path, "Contents", "Info.plist")); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("ChatGPT application bundle was not found")
}

func captureChatGPTRootState(content string) chatGPTRestoreState {
	return chatGPTRestoreState{
		Model:            rootString(content, "model"),
		ModelProvider:    rootString(content, "model_provider"),
		ModelCatalogJSON: rootString(content, "model_catalog_json"),
	}
}

func normalizeLegacyChatGPTConfig(content string) string {
	marker := strings.Index(content, ChatGPTConfigMarker)
	if marker < 0 {
		return content
	}
	firstTable := strings.Index(content, "[")
	if firstTable >= 0 && marker > firstTable {
		return strings.TrimRight(content[:marker], "\n") + "\n"
	}
	return content
}

func restoreChatGPTRootState(content string, state chatGPTRestoreState) string {
	values := map[string]*string{}
	for key, saved := range map[string]savedRootString{
		"model": state.Model, "model_provider": state.ModelProvider, "model_catalog_json": state.ModelCatalogJSON,
	} {
		if saved.Present {
			value := saved.Value
			values[key] = &value
		} else {
			values[key] = nil
		}
	}
	return setRootStrings(content, values)
}

func chatGPTRootManaged(content string) bool {
	provider := rootString(content, "model_provider")
	return provider.Present && provider.Value == "harnezpad"
}

func validateManagedTOML(content string) error {
	for _, key := range []string{"model", "model_provider", "model_catalog_json"} {
		if value := rootString(content, key); value.Present && strings.ContainsAny(value.Value, "\r\n") {
			return fmt.Errorf("invalid %s value in Codex config", key)
		}
	}
	return nil
}

func rootString(content, key string) savedRootString {
	for _, line := range rootLines(content) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, key) {
			continue
		}
		left, right, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(left) != key {
			continue
		}
		value := strings.TrimSpace(right)
		if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		if decoded, err := strconv.Unquote(value); err == nil {
			return savedRootString{Present: true, Value: decoded}
		}
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			return savedRootString{Present: true, Value: value[1 : len(value)-1]}
		}
	}
	return savedRootString{}
}

func rootLines(content string) []string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			return lines[:i]
		}
	}
	return lines
}

func setRootStrings(content string, values map[string]*string) string {
	lines := strings.Split(content, "\n")
	firstTable := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstTable = i
			break
		}
	}
	root := make([]string, 0, firstTable+len(values))
	for _, line := range lines[:firstTable] {
		trimmed := strings.TrimSpace(line)
		remove := false
		for key := range values {
			left, _, ok := strings.Cut(trimmed, "=")
			if ok && strings.TrimSpace(left) == key {
				remove = true
				break
			}
		}
		if !remove {
			root = append(root, line)
		}
	}
	rootText := strings.TrimRight(strings.Join(root, "\n"), "\n")
	for _, key := range []string{"model", "model_provider", "model_catalog_json"} {
		if value, ok := values[key]; ok && value != nil {
			if rootText != "" {
				rootText += "\n"
			}
			rootText += key + " = " + strconv.Quote(*value)
		}
	}
	tail := strings.Join(lines[firstTable:], "\n")
	if rootText != "" && tail != "" {
		return rootText + "\n\n" + strings.TrimLeft(tail, "\n")
	}
	return rootText + tail
}

func removeManagedProviderSection(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skipping := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[model_providers.harnezpad]" {
			skipping = true
			continue
		}
		if skipping && strings.HasPrefix(trimmed, "[") {
			skipping = false
		}
		if !skipping {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func removeMarker(content string) string {
	lines := strings.Split(content, "\n")
	out := lines[:0]
	for _, line := range lines {
		if strings.TrimSpace(line) != ChatGPTConfigMarker {
			out = append(out, line)
		}
	}
	return strings.TrimLeft(strings.Join(out, "\n"), "\n")
}

func insertBeforeFirstTable(content, block string) string {
	content = removeMarker(content)
	lines := strings.Split(content, "\n")
	firstTable := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			firstTable = i
			break
		}
	}
	root := strings.TrimRight(strings.Join(lines[:firstTable], "\n"), "\n")
	tail := strings.TrimLeft(strings.Join(lines[firstTable:], "\n"), "\n")
	parts := []string{}
	if root != "" {
		parts = append(parts, root)
	}
	parts = append(parts, strings.TrimSpace(block))
	if tail != "" {
		parts = append(parts, tail)
	}
	return strings.Join(parts, "\n\n") + "\n"
}

func stringPointer(value string) *string { return &value }
