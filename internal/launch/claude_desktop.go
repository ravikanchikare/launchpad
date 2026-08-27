package launch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	claudeDesktopGOOS         = runtime.GOOS
	claudeDesktopIsRunning    = defaultClaudeDesktopIsRunning
	claudeDesktopQuit         = defaultClaudeDesktopQuit
	claudeDesktopOpen         = defaultClaudeDesktopOpen
	claudeDesktopApplyProfile = writeClaudeDesktopProfile
	claudeDesktopUserHome     = os.UserHomeDir
	claudeDesktopPollInterval = 200 * time.Millisecond
)

const (
	claudeDesktopProfileID   = "00000000-0000-4000-8000-000000000115"
	claudeDesktopProfileName = "Launchpad"
)

type ClaudeDesktopProfile struct {
	GatewayURL string
	APIKey     string
	AutoMode   bool
}

// ClaudeDesktopRunning reports whether the macOS Claude Desktop process is open.
func ClaudeDesktopRunning(ctx context.Context) (bool, error) {
	return claudeDesktopIsRunning(ctx)
}

// ConfigureClaudeDesktop writes Launchpad's third-party inference profile and
// starts Claude. A running Claude is quit before the profile is written because
// Claude persists its own settings during shutdown.
func ConfigureClaudeDesktop(ctx context.Context, profile ClaudeDesktopProfile) error {
	if claudeDesktopGOOS != "darwin" {
		return errors.New("Claude Desktop configuration is supported on macOS only")
	}

	running, err := claudeDesktopIsRunning(ctx)
	if err != nil {
		return fmt.Errorf("check whether Claude Desktop is running: %w", err)
	}
	if !running {
		if err := claudeDesktopApplyProfile(profile); err != nil {
			return fmt.Errorf("configure Claude Desktop gateway profile: %w", err)
		}
		if err := claudeDesktopOpen(); err != nil {
			return fmt.Errorf("open Claude Desktop: %w", err)
		}
		return nil
	}

	restartCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := claudeDesktopQuit(restartCtx); err != nil {
		return fmt.Errorf("quit Claude Desktop: %w", err)
	}
	if err := waitForClaudeDesktopExit(restartCtx); err != nil {
		return err
	}
	if err := claudeDesktopApplyProfile(profile); err != nil {
		profileErr := fmt.Errorf("configure Claude Desktop gateway profile: %w", err)
		if openErr := claudeDesktopOpen(); openErr != nil {
			return errors.Join(profileErr, fmt.Errorf("reopen Claude Desktop after profile failure: %w", openErr))
		}
		return profileErr
	}
	if err := claudeDesktopOpen(); err != nil {
		return fmt.Errorf("reopen Claude Desktop: %w", err)
	}
	return nil
}

func waitForClaudeDesktopExit(ctx context.Context) error {
	for {
		running, err := claudeDesktopIsRunning(ctx)
		if err != nil {
			return fmt.Errorf("check whether Claude Desktop is running: %w", err)
		}
		if !running {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("Claude Desktop did not quit; quit it manually and try again")
		case <-time.After(claudeDesktopPollInterval):
		}
	}
}

func defaultClaudeDesktopIsRunning(ctx context.Context) (bool, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/pgrep", "-f", "Claude.app/Contents/MacOS/Claude").Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && ctx.Err() == nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func defaultClaudeDesktopQuit(ctx context.Context) error {
	return exec.CommandContext(ctx, "/usr/bin/osascript", "-e", `tell application "Claude" to quit`).Run()
}

func defaultClaudeDesktopOpen() error {
	home, err := claudeDesktopUserHome()
	if err != nil {
		return err
	}
	for _, path := range []string{
		"/Applications/Claude.app",
		filepath.Join(home, "Applications", "Claude.app"),
	} {
		if _, err := os.Stat(path); err == nil {
			return exec.Command("/usr/bin/open", path).Run()
		}
	}
	return errors.New("Claude Desktop app was not found")
}

type claudeDesktopProfilePaths struct {
	normalConfig     string
	deploymentConfig string
	meta             string
	profile          string
}

func claudeDesktopPaths() (claudeDesktopProfilePaths, error) {
	home, err := claudeDesktopUserHome()
	if err != nil {
		return claudeDesktopProfilePaths{}, err
	}
	support := filepath.Join(home, "Library", "Application Support")
	thirdParty := filepath.Join(support, "Claude-3p")
	return claudeDesktopProfilePaths{
		normalConfig:     filepath.Join(support, "Claude", "claude_desktop_config.json"),
		deploymentConfig: filepath.Join(thirdParty, "claude_desktop_config.json"),
		meta:             filepath.Join(thirdParty, "configLibrary", "_meta.json"),
		profile:          filepath.Join(thirdParty, "configLibrary", claudeDesktopProfileID+".json"),
	}, nil
}

func writeClaudeDesktopProfile(profile ClaudeDesktopProfile) error {
	if strings.TrimSpace(profile.GatewayURL) == "" {
		return errors.New("gateway URL is required")
	}
	if strings.TrimSpace(profile.APIKey) == "" {
		return errors.New("management key is required")
	}
	paths, err := claudeDesktopPaths()
	if err != nil {
		return err
	}

	profileConfig, err := readClaudeDesktopJSON(paths.profile)
	if err != nil {
		return fmt.Errorf("read Claude Desktop profile: %w", err)
	}
	profileConfig["inferenceProvider"] = "gateway"
	profileConfig["inferenceGatewayBaseUrl"] = strings.TrimRight(profile.GatewayURL, "/")
	profileConfig["inferenceGatewayApiKey"] = profile.APIKey
	profileConfig["inferenceGatewayAuthScheme"] = "bearer"
	profileConfig["deploymentDisplayName"] = claudeDesktopProfileName
	profileConfig["chatTabEnabled"] = true
	profileConfig["disableDeploymentModeChooser"] = true
	profileConfig["coworkEgressAllowedHosts"] = []string{"*"}
	profileConfig["disableEssentialTelemetry"] = true
	profileConfig["disableNonessentialTelemetry"] = true
	profileConfig["autoModeEnabled"] = profile.AutoMode
	// Claude discovers the current model catalog from the gateway. Persisting a
	// static list here prevents newly enabled LiteLLM models from appearing.
	delete(profileConfig, "inferenceModels")
	if err := writeClaudeDesktopJSON(paths.profile, profileConfig); err != nil {
		return fmt.Errorf("write Claude Desktop profile: %w", err)
	}

	meta, err := readClaudeDesktopJSON(paths.meta)
	if err != nil {
		return fmt.Errorf("read Claude Desktop profile metadata: %w", err)
	}
	meta["appliedId"] = claudeDesktopProfileID
	entries := make([]any, 0)
	for _, entry := range claudeDesktopSlice(meta["entries"]) {
		entryMap, _ := entry.(map[string]any)
		if entryMap != nil {
			if id, _ := entryMap["id"].(string); id == claudeDesktopProfileID {
				continue
			}
		}
		entries = append(entries, entry)
	}
	meta["entries"] = append(entries, map[string]any{"id": claudeDesktopProfileID, "name": claudeDesktopProfileName})
	if err := writeClaudeDesktopJSON(paths.meta, meta); err != nil {
		return fmt.Errorf("write Claude Desktop profile metadata: %w", err)
	}

	for _, path := range []string{paths.normalConfig, paths.deploymentConfig} {
		config, err := readClaudeDesktopJSON(path)
		if err != nil {
			return fmt.Errorf("read Claude Desktop deployment config: %w", err)
		}
		config["deploymentMode"] = "3p"
		if err := writeClaudeDesktopJSON(path, config); err != nil {
			return fmt.Errorf("write Claude Desktop deployment config: %w", err)
		}
	}
	return nil
}

func readClaudeDesktopJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

func writeClaudeDesktopJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".launchpad-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func claudeDesktopSlice(value any) []any {
	items, _ := value.([]any)
	return items
}
