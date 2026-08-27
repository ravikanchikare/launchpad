package launch

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var testClaudeProfile = ClaudeDesktopProfile{
	ProviderURL: "https://provider.example.test",
	APIKey:      "test-key",
	AutoMode:    true,
}

func stubClaudeDesktopLifecycle(t *testing.T) {
	t.Helper()
	oldGOOS := claudeDesktopGOOS
	oldRunning := claudeDesktopIsRunning
	oldQuit := claudeDesktopQuit
	oldOpen := claudeDesktopOpen
	oldApplyProfile := claudeDesktopApplyProfile
	oldUserHome := claudeDesktopUserHome
	oldPollInterval := claudeDesktopPollInterval
	t.Cleanup(func() {
		claudeDesktopGOOS = oldGOOS
		claudeDesktopIsRunning = oldRunning
		claudeDesktopQuit = oldQuit
		claudeDesktopOpen = oldOpen
		claudeDesktopApplyProfile = oldApplyProfile
		claudeDesktopUserHome = oldUserHome
		claudeDesktopPollInterval = oldPollInterval
	})
	claudeDesktopGOOS = "darwin"
	claudeDesktopPollInterval = time.Millisecond
}

func TestConfigureClaudeDesktopQuitsWaitsAppliesProfileAndReopens(t *testing.T) {
	stubClaudeDesktopLifecycle(t)
	states := []bool{true, true, false}
	var events []string
	claudeDesktopIsRunning = func(context.Context) (bool, error) {
		state := states[0]
		states = states[1:]
		events = append(events, "running")
		return state, nil
	}
	claudeDesktopQuit = func(context.Context) error {
		events = append(events, "quit")
		return nil
	}
	claudeDesktopOpen = func() error {
		events = append(events, "open")
		return nil
	}
	claudeDesktopApplyProfile = func(profile ClaudeDesktopProfile) error {
		events = append(events, "profile")
		return nil
	}

	if err := ConfigureClaudeDesktop(context.Background(), testClaudeProfile); err != nil {
		t.Fatal(err)
	}
	want := []string{"running", "quit", "running", "running", "profile", "open"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestConfigureClaudeDesktopAppliesProfileBeforeOpeningStoppedApp(t *testing.T) {
	stubClaudeDesktopLifecycle(t)
	var events []string
	claudeDesktopIsRunning = func(context.Context) (bool, error) {
		events = append(events, "running")
		return false, nil
	}
	claudeDesktopQuit = func(context.Context) error {
		events = append(events, "quit")
		return nil
	}
	claudeDesktopOpen = func() error {
		events = append(events, "open")
		return nil
	}
	claudeDesktopApplyProfile = func(profile ClaudeDesktopProfile) error {
		events = append(events, "profile")
		return nil
	}

	if err := ConfigureClaudeDesktop(context.Background(), testClaudeProfile); err != nil {
		t.Fatal(err)
	}
	want := []string{"running", "profile", "open"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestConfigureClaudeDesktopDoesNotApplyProfileAfterQuitFailure(t *testing.T) {
	stubClaudeDesktopLifecycle(t)
	claudeDesktopIsRunning = func(context.Context) (bool, error) { return true, nil }
	claudeDesktopQuit = func(context.Context) error { return errors.New("quit failed") }
	claudeDesktopOpen = func() error {
		t.Fatal("open called after quit failure")
		return nil
	}
	claudeDesktopApplyProfile = func(ClaudeDesktopProfile) error {
		t.Fatal("profile applied after quit failure")
		return nil
	}

	if err := ConfigureClaudeDesktop(context.Background(), testClaudeProfile); err == nil {
		t.Fatal("expected configuration error")
	}
}

func TestWriteClaudeDesktopProfileConfiguresProviderAndPreservesUserSettings(t *testing.T) {
	stubClaudeDesktopLifecycle(t)
	home := t.TempDir()
	claudeDesktopUserHome = func() (string, error) { return home, nil }
	paths, err := claudeDesktopPaths()
	if err != nil {
		t.Fatal(err)
	}
	writeTestJSON(t, paths.normalConfig, map[string]any{"mcpServers": map[string]any{"keep": true}})
	writeTestJSON(t, paths.deploymentConfig, map[string]any{"preferences": map[string]any{"keep": true}})
	writeTestJSON(t, paths.meta, map[string]any{
		"entries": []any{map[string]any{"id": "user-profile", "name": "Keep"}},
	})
	writeTestJSON(t, paths.profile, map[string]any{
		"userOwned":       "keep",
		"inferenceModels": []any{"stale-model"},
	})

	if err := writeClaudeDesktopProfile(testClaudeProfile); err != nil {
		t.Fatal(err)
	}

	normal := readTestJSON(t, paths.normalConfig)
	if normal["deploymentMode"] != "3p" || normal["mcpServers"] == nil {
		t.Fatalf("normal config = %#v", normal)
	}
	deployment := readTestJSON(t, paths.deploymentConfig)
	if deployment["deploymentMode"] != "3p" || deployment["preferences"] == nil {
		t.Fatalf("third-party config = %#v", deployment)
	}
	profile := readTestJSON(t, paths.profile)
	for key, want := range map[string]any{
		"inferenceProvider":            "gateway",
		"inferenceGatewayBaseUrl":      testClaudeProfile.ProviderURL,
		"inferenceGatewayApiKey":       testClaudeProfile.APIKey,
		"inferenceGatewayAuthScheme":   "bearer",
		"deploymentDisplayName":        claudeDesktopProfileName,
		"disableDeploymentModeChooser": true,
		"autoModeEnabled":              true,
		"userOwned":                    "keep",
	} {
		if got := profile[key]; got != want {
			t.Fatalf("profile[%q] = %#v, want %#v", key, got, want)
		}
	}
	if _, ok := profile["inferenceModels"]; ok {
		t.Fatalf("inferenceModels should be omitted for provider discovery: %#v", profile)
	}
	info, err := os.Stat(paths.profile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("profile mode = %o, want 600", info.Mode().Perm())
	}

	meta := readTestJSON(t, paths.meta)
	if meta["appliedId"] != claudeDesktopProfileID {
		t.Fatalf("appliedId = %#v", meta["appliedId"])
	}
	entries := claudeDesktopSlice(meta["entries"])
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want preserved user profile and Launchpad", entries)
	}
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readTestJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
