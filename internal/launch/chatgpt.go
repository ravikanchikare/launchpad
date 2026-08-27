package launch

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	chatGPTManagedBegin = "# LAUNCHPAD_MANAGED_CHATGPT_BEGIN"
	chatGPTManagedEnd   = "# LAUNCHPAD_MANAGED_CHATGPT_END"
)

type savedRootValue struct {
	Present bool   `json:"present"`
	Line    string `json:"line,omitempty"`
}

type chatGPTRestoreState struct {
	Model            savedRootValue `json:"model"`
	ModelProvider    savedRootValue `json:"modelProvider"`
	ModelCatalogJSON savedRootValue `json:"modelCatalogJson"`
}

func ConfigureChatGPT(providerURL, model, executable string) error {
	configPath, statePath, catalogPath, err := chatGPTPaths()
	if err != nil {
		return err
	}
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	current := string(content)
	if !strings.Contains(current, chatGPTManagedBegin) {
		state := chatGPTRestoreState{
			Model:            captureRootValue(current, "model"),
			ModelProvider:    captureRootValue(current, "model_provider"),
			ModelCatalogJSON: captureRootValue(current, "model_catalog_json"),
		}
		if err := writeJSONAtomic(statePath, state); err != nil {
			return err
		}
	}
	current = removeManagedChatGPTBlock(current)
	current = setRootValue(current, "model", strconv.Quote(model))
	current = setRootValue(current, "model_provider", strconv.Quote("launchpad"))
	current = setRootValue(current, "model_catalog_json", strconv.Quote(catalogPath))
	if err := writeJSONAtomic(catalogPath, map[string]any{
		"launchpad_managed": true,
		"models": []map[string]any{{
			"slug": model, "display_name": model, "description": "Launchpad provider model",
			"default_reasoning_level": nil, "supported_reasoning_levels": []any{},
			"shell_type": "default", "visibility": "list", "supported_in_api": true,
			"priority": 0, "additional_speed_tiers": []any{}, "availability_nux": nil,
			"upgrade": nil, "base_instructions": "You are a coding agent collaborating with the user in their workspace.",
			"model_messages": nil, "supports_reasoning_summaries": false,
			"default_reasoning_summary": "auto", "support_verbosity": false,
			"default_verbosity": nil, "apply_patch_tool_type": nil,
			"web_search_tool_type":         "text",
			"truncation_policy":            map[string]any{"mode": "bytes", "limit": 10000},
			"supports_parallel_tool_calls": false, "supports_image_detail_original": false,
			"context_window": 128000, "max_context_window": 128000,
			"auto_compact_token_limit": nil, "effective_context_window_percent": 95,
			"experimental_supported_tools": []any{}, "input_modalities": []string{"text"},
			"supports_search_tool": false,
		}},
	}); err != nil {
		return err
	}
	if executable == "" {
		executable, err = os.Executable()
		if err != nil {
			return err
		}
	}
	block := chatGPTManagedBegin + "\n" +
		"[model_providers.launchpad]\n" +
		"name = \"Launchpad\"\n" +
		"base_url = " + strconv.Quote(strings.TrimRight(providerURL, "/")+"/v1") + "\n" +
		"wire_api = \"responses\"\n\n" +
		"[model_providers.launchpad.auth]\n" +
		"command = " + strconv.Quote(executable) + "\n" +
		"args = [\"_chatgpt-token\"]\n" +
		"timeout_ms = 5000\n" +
		"refresh_interval_ms = 300000\n" +
		chatGPTManagedEnd + "\n"
	current = strings.TrimRight(current, "\n") + "\n\n" + block
	return writeFileAtomic(configPath, []byte(current), 0o600)
}

func RestoreChatGPT() error {
	configPath, statePath, catalogPath, err := chatGPTPaths()
	if err != nil {
		return err
	}
	stateData, err := os.ReadFile(statePath)
	if os.IsNotExist(err) {
		return errors.New("no Launchpad ChatGPT restore state was found")
	}
	if err != nil {
		return err
	}
	var state chatGPTRestoreState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return err
	}
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	current := removeManagedChatGPTBlock(string(content))
	current = restoreRootValue(current, "model", state.Model)
	current = restoreRootValue(current, "model_provider", state.ModelProvider)
	current = restoreRootValue(current, "model_catalog_json", state.ModelCatalogJSON)
	if strings.TrimSpace(current) == "" {
		_ = os.Remove(configPath)
	} else if err := writeFileAtomic(configPath, []byte(strings.TrimLeft(current, "\n")), 0o600); err != nil {
		return err
	}
	_ = os.Remove(catalogPath)
	return os.Remove(statePath)
}

func ChatGPTLaunchCommand(ctx context.Context) (*exec.Cmd, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("ChatGPT launching is currently supported on macOS only")
	}
	if ChatGPTIsRunning(ctx) {
		if err := exec.CommandContext(ctx, "osascript", "-e", `tell application id "com.openai.codex" to quit`).Run(); err != nil {
			return nil, fmt.Errorf("quit ChatGPT before applying the provider profile: %w", err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			active, _ := exec.CommandContext(ctx, "osascript", "-e", `application id "com.openai.codex" is running`).Output()
			if strings.TrimSpace(string(active)) != "true" {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	home, _ := os.UserHomeDir()
	for _, app := range []string{
		"/Applications/ChatGPT.app",
		"/Applications/Codex.app",
		filepath.Join(home, "Applications", "ChatGPT.app"),
		filepath.Join(home, "Applications", "Codex.app"),
	} {
		if info, err := os.Stat(filepath.Join(app, "Contents", "Info.plist")); err == nil && !info.IsDir() {
			return exec.CommandContext(ctx, "open", "-a", app), nil
		}
	}
	return nil, errors.New("ChatGPT application bundle was not found")
}

func ChatGPTIsRunning(ctx context.Context) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	running, err := exec.CommandContext(ctx, "osascript", "-e", `application id "com.openai.codex" is running`).Output()
	return err == nil && strings.TrimSpace(string(running)) == "true"
}

func LaunchChatGPT(ctx context.Context) error {
	command, err := ChatGPTLaunchCommand(ctx)
	if err != nil {
		return err
	}
	return command.Start()
}

func chatGPTPaths() (string, string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", "", err
	}
	codexDir := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexDir == "" {
		codexDir = filepath.Join(home, ".codex")
	}
	return filepath.Join(codexDir, "config.toml"),
		filepath.Join(configDir, "Launchpad", "chatgpt-restore.json"),
		filepath.Join(codexDir, "launchpad-chatgpt-models.json"), nil
}

func captureRootValue(content, key string) savedRootValue {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			break
		}
		if rootLineKey(line) == key {
			return savedRootValue{Present: true, Line: line}
		}
	}
	return savedRootValue{}
}

func setRootValue(content, key, value string) string {
	replacement := key + " = " + value
	lines := strings.Split(content, "\n")
	table := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			table = i
			break
		}
		if rootLineKey(line) == key {
			lines[i] = replacement
			return strings.Join(lines, "\n")
		}
	}
	lines = append(lines[:table], append([]string{replacement}, lines[table:]...)...)
	return strings.Join(lines, "\n")
}

func restoreRootValue(content, key string, saved savedRootValue) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			break
		}
		if rootLineKey(line) == key {
			if saved.Present {
				lines[i] = saved.Line
			} else {
				lines = append(lines[:i], lines[i+1:]...)
			}
			return strings.Join(lines, "\n")
		}
	}
	if saved.Present {
		return saved.Line + "\n" + content
	}
	return content
}

func rootLineKey(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return ""
	}
	key, _, ok := strings.Cut(trimmed, "=")
	if !ok {
		return ""
	}
	return strings.TrimSpace(key)
}

func removeManagedChatGPTBlock(content string) string {
	start := strings.Index(content, chatGPTManagedBegin)
	if start < 0 {
		return content
	}
	endOffset := strings.Index(content[start:], chatGPTManagedEnd)
	if endOffset < 0 {
		return content
	}
	end := start + endOffset + len(chatGPTManagedEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return strings.TrimRight(content[:start], "\n") + "\n" + strings.TrimLeft(content[end:], "\n")
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o600)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
