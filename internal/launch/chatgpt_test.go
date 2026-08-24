package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harnezpad/internal/gateway"
)

func withChatGPTTestLaunch(t *testing.T) {
	t.Helper()
	oldActive, oldFind := chatGPTProcessActive, chatGPTFindBundle
	chatGPTProcessActive = func() bool { return false }
	chatGPTFindBundle = func() (string, error) { return "/Applications/ChatGPT.app", nil }
	t.Cleanup(func() {
		chatGPTProcessActive, chatGPTFindBundle = oldActive, oldFind
	})
}

func TestChatGPTSelectiveRestorePreservesUnrelatedEdits(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	withChatGPTTestLaunch(t)
	configPath, _ := chatGPTConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}
	original := "# user comment\nmodel = \"usual\"\nmodel_provider = \"openai\"\n\n[features]\nmodel = \"table-value\"\nflag = true\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	settings := testGateway{url: "https://gateway.example", token: "secret"}
	if err := ConfigureChatGPT(settings, "model-a", []gateway.Model{{ID: "model-a"}}); err != nil {
		t.Fatal(err)
	}
	configured, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configured), "[model_providers.harnezpad.auth]") || strings.Contains(string(configured), `env_key = "OPENAI_API_KEY"`) {
		t.Fatalf("ChatGPT provider must use command-backed auth:\n%s", configured)
	}
	updated := strings.Replace(string(configured), "flag = true", "flag = false\nnew_setting = 42", 1)
	if err := os.WriteFile(configPath, []byte(updated), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ResetChatGPT(settings); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(restored)
	for _, want := range []string{`model = "usual"`, `model_provider = "openai"`, `model = "table-value"`, "flag = false", "new_setting = 42"} {
		if !strings.Contains(text, want) {
			t.Fatalf("restored config missing %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{ChatGPTConfigMarker, "[model_providers.harnezpad]", "model-a"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("restored config retained %q:\n%s", unwanted, text)
		}
	}
}

func TestConfigureChatGPTDoesNotTouchCodexCLIProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	rootPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(rootPath, []byte("model = \"usual\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := testGateway{url: "https://gateway.example", token: "secret"}
	if err := ConfigureChatGPT(settings, "model-a", []gateway.Model{{ID: "model-a"}}); err != nil {
		t.Fatal(err)
	}
	profilePath, _ := ProfilePath()
	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		t.Fatalf("Codex CLI profile should not be created for ChatGPT launch: %v", err)
	}
	configured, err := os.ReadFile(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configured), ChatGPTConfigMarker) || !strings.Contains(string(configured), `model = "model-a"`) {
		t.Fatalf("root config was not configured for ChatGPT:\n%s", configured)
	}
}

func TestSetRootStringsDoesNotDeleteTableModel(t *testing.T) {
	content := "model = \"root\"\n\n[features]\nmodel = \"nested\"\n"
	updated := setRootStrings(content, map[string]*string{"model": stringPointer("new")})
	if !strings.Contains(updated, `model = "new"`) || !strings.Contains(updated, `model = "nested"`) {
		t.Fatalf("unexpected config:\n%s", updated)
	}
}

func TestNormalizeLegacyChatGPTConfigRemovesAppendedManagedBlock(t *testing.T) {
	content := "model = \"usual\"\n\n[features]\nflag = true\n\n" + ChatGPTConfigMarker + "\nmodel = \"gateway\"\n[model_providers.harnezpad]\nbase_url = \"x\"\n"
	got := normalizeLegacyChatGPTConfig(content)
	if strings.Contains(got, ChatGPTConfigMarker) || strings.Contains(got, "gateway") || !strings.Contains(got, "flag = true") {
		t.Fatalf("legacy config was not normalized:\n%s", got)
	}
}

func TestDefaultChatGPTProcessActiveUsesBundleID(t *testing.T) {
	oldCommand := chatGPTCommand
	chatGPTCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("/bin/sh", "-c", "printf true")
	}
	t.Cleanup(func() { chatGPTCommand = oldCommand })
	if !defaultChatGPTProcessActive() {
		t.Fatal("expected running application response")
	}
}

func TestRemoveManagedProviderSectionPreservesFollowingTables(t *testing.T) {
	content := "[model_providers.harnezpad]\nbase_url = \"x\"\n\n[features]\nflag = true\n"
	updated := removeManagedProviderSection(content)
	if strings.Contains(updated, "model_providers.harnezpad") || !strings.Contains(updated, "[features]") {
		t.Fatalf("unexpected config:\n%s", updated)
	}
}

func TestStopChatGPTTimesOut(t *testing.T) {
	oldActive, oldQuit, oldTerminate, oldSleep, oldTimeout := chatGPTProcessActive, chatGPTQuit, chatGPTTerminate, chatGPTSleep, chatGPTExitTimeout
	chatGPTProcessActive = func() bool { return true }
	chatGPTQuit = func() error { return nil }
	chatGPTTerminate = func() error { return nil }
	chatGPTSleep = func(time.Duration) {}
	chatGPTExitTimeout = time.Millisecond
	t.Cleanup(func() {
		chatGPTProcessActive, chatGPTQuit, chatGPTTerminate, chatGPTSleep, chatGPTExitTimeout = oldActive, oldQuit, oldTerminate, oldSleep, oldTimeout
	})
	if err := stopChatGPT(); err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestStopChatGPTTerminatesAfterQuitEventTimeout(t *testing.T) {
	oldActive, oldQuit, oldTerminate, oldSleep, oldTimeout := chatGPTProcessActive, chatGPTQuit, chatGPTTerminate, chatGPTSleep, chatGPTExitTimeout
	active := true
	forced := false
	chatGPTProcessActive = func() bool { return active }
	chatGPTQuit = func() error { return nil }
	chatGPTTerminate = func() error { forced = true; active = false; return nil }
	chatGPTSleep = func(time.Duration) {}
	chatGPTExitTimeout = time.Millisecond
	t.Cleanup(func() {
		chatGPTProcessActive, chatGPTQuit, chatGPTTerminate, chatGPTSleep, chatGPTExitTimeout = oldActive, oldQuit, oldTerminate, oldSleep, oldTimeout
	})
	if err := stopChatGPT(); err != nil || !forced {
		t.Fatalf("force quit result: forced=%v err=%v", forced, err)
	}
}
