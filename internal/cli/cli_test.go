package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"launchpad/internal/config"
	"launchpad/internal/provider"
)

func TestLaunchClaudeEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	home := t.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(home, "child.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LAUNCHPAD_TEST_OUTPUT\"\nprintf '%s\\n' \"$ANTHROPIC_BASE_URL\" >> \"$LAUNCHPAD_TEST_OUTPUT\"\nprintf '%s\\n' \"$ANTHROPIC_AUTH_TOKEN\" >> \"$LAUNCHPAD_TEST_OUTPUT\"\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "test-secret")
	t.Setenv("LAUNCHPAD_TEST_OUTPUT", output)
	var stdout, stderr bytes.Buffer
	handled, code := Run(context.Background(), "/tmp/launchpad", []string{
		"launch", "claude", "--model", "cheap-model", "--provider-url", "https://provider.example.test", "--", "--verbose",
	}, IO{In: strings.NewReader(""), Out: &stdout, Err: &stderr})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d stderr=%s", handled, code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"--model\ncheap-model\n", "--verbose\n", "https://provider.example.test\n", "test-secret\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("child output missing %q:\n%s", want, got)
		}
	}
}

func TestModelDiscoveryToOpenCodeEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model_group/info" || r.Header.Get("Authorization") != "Bearer test-secret" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"model_group":"cheap-model","supports_function_calling":true}]}`))
	}))
	defer server.Close()

	home := t.TempDir()
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(home, "child.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$OPENCODE_CONFIG_CONTENT\" > \"$LAUNCHPAD_TEST_OUTPUT\"\n"
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "test-secret")
	t.Setenv("LAUNCHPAD_PROVIDER_URL", server.URL)
	t.Setenv("LAUNCHPAD_ASSUME_TTY", "1")
	t.Setenv("LAUNCHPAD_TEST_OUTPUT", output)
	var stdout, stderr bytes.Buffer
	_, code := Run(context.Background(), "/tmp/launchpad", []string{"launch", "opencode"},
		IO{In: strings.NewReader("cheap\r"), Out: &stdout, Err: &stderr})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"baseURL":"` + server.URL + `/v1"`, `"model":"launchpad/cheap-model"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("provider config missing %q: %s", want, data)
		}
	}
}

func TestMissingKeyDoesNotLaunch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "")
	t.Setenv("LAUNCHPAD_DISABLE_KEYCHAIN", "1")
	var stderr bytes.Buffer
	_, code := Run(context.Background(), "launchpad", []string{"launch", "claude", "--model", "x"},
		IO{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &stderr})
	if code != 1 || !strings.Contains(stderr.String(), "LAUNCHPAD_PROVIDER_API_KEY") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestSetProviderConfiguresDiscovery(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	_, code := Run(context.Background(), "launchpad", []string{
		"config", "set-provider", "https://provider.example.test/v1/",
		"--kind", "openai-compatible",
		"--models-url", "https://catalog.example.test/models/",
	}, IO{In: strings.NewReader(""), Out: &stdout, Err: &stderr})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	settings, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if settings.ProviderKind != provider.KindOpenAICompatible ||
		settings.ProviderURL != "https://provider.example.test/v1" ||
		settings.ModelsURL != "https://catalog.example.test/models" {
		t.Fatalf("settings = %#v", settings)
	}
}

func TestChatGPTRestoreRequiresConfirmationAndPrintsSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "")
	t.Setenv("LAUNCHPAD_DISABLE_KEYCHAIN", "1")

	originalRestore := restoreChatGPT
	originalRunning := chatGPTIsRunning
	originalLaunch := launchChatGPT
	t.Cleanup(func() {
		restoreChatGPT = originalRestore
		chatGPTIsRunning = originalRunning
		launchChatGPT = originalLaunch
	})
	restored := false
	restoreChatGPT = func() error { restored = true; return nil }
	chatGPTIsRunning = func(context.Context) bool { return false }
	launchChatGPT = func(context.Context) error { t.Fatal("ChatGPT should not launch"); return nil }

	var stderr bytes.Buffer
	_, code := Run(context.Background(), "launchpad", []string{"launch", "chatgpt", "--restore"},
		IO{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &stderr})
	if code != 1 || restored || !strings.Contains(stderr.String(), "re-run with --yes") {
		t.Fatalf("code=%d restored=%v stderr=%q", code, restored, stderr.String())
	}

	stderr.Reset()
	_, code = Run(context.Background(), "launchpad", []string{"launch", "chatgpt", "--restore", "--yes"},
		IO{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &stderr})
	if code != 0 || !restored || !strings.Contains(stderr.String(), "restored to your usual profile") {
		t.Fatalf("code=%d restored=%v stderr=%q", code, restored, stderr.String())
	}
}

func TestChatGPTConfigurePrintsRestoreInstructions(t *testing.T) {
	originalConfigure := configureChatGPT
	originalRunning := chatGPTIsRunning
	originalLaunch := launchChatGPT
	t.Cleanup(func() {
		configureChatGPT = originalConfigure
		chatGPTIsRunning = originalRunning
		launchChatGPT = originalLaunch
	})
	configured, launched := false, false
	configureChatGPT = func(providerURL, model, executable string) error {
		configured = providerURL == "https://provider.example" && model == "model-a"
		return nil
	}
	chatGPTIsRunning = func(context.Context) bool { return true }
	launchChatGPT = func(context.Context) error { launched = true; return nil }

	var stderr bytes.Buffer
	err := runChatGPT(context.Background(), "team-launcher", "/tmp/launchpad",
		"https://provider.example", "model-a", true,
		IO{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &stderr})
	if err != nil {
		t.Fatal(err)
	}
	if !configured || !launched {
		t.Fatalf("configured=%v launched=%v", configured, launched)
	}
	for _, want := range []string{
		"ChatGPT profile changed to Launchpad.",
		"team-launcher launch chatgpt --restore",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q: %s", want, stderr.String())
		}
	}
}
