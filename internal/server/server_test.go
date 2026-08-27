package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"launchpad/internal/config"
	"launchpad/internal/store"
)

func TestLauncherConfigAndVisibleIntegrations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LAUNCHPAD_PROVIDER_URL", "")
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "test-key")
	t.Setenv("LAUNCHPAD_DISABLE_KEYCHAIN", "1")
	originalCLIName := config.DefaultCLIName
	config.DefaultCLIName = "team-launcher"
	t.Cleanup(func() {
		config.DefaultCLIName = originalCLIName
	})
	srv := &Server{Store: &store.Store{DBPath: home + "/db.sqlite"}}
	handler := srv.Handler()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/launcher/config",
		strings.NewReader(`{
			"providerKind":"openai-compatible",
			"providerUrl":"https://provider.example.test/v1/",
			"modelsUrl":"https://catalog.example.test/models/"
		}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("config status=%d body=%s", response.Code, response.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved["providerKind"] != "openai-compatible" ||
		saved["providerUrl"] != "https://provider.example.test/v1" ||
		saved["modelsUrl"] != "https://catalog.example.test/models" ||
		saved["cliName"] != "team-launcher" ||
		saved["apiKeyConfigured"] != true {
		t.Fatalf("config response = %#v", saved)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("integrations status=%d body=%s", response.Code, response.Body.String())
	}
	var integrations []struct {
		ID      string `json:"id"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &integrations); err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for _, integration := range integrations {
		got[integration.ID] = integration.Command
	}
	for _, id := range []string{"claude-desktop", "chatgpt", "claude", "codex", "opencode", "copilot"} {
		if _, ok := got[id]; !ok {
			t.Fatalf("missing integration %q in %#v", id, got)
		}
	}
	for _, hidden := range []string{"cline", "droid", "hermes", "openclaw", "pi", "qwen"} {
		if _, ok := got[hidden]; ok {
			t.Fatalf("hidden integration %q was returned", hidden)
		}
	}
	if got["chatgpt"] != "team-launcher launch chatgpt" {
		t.Fatalf("ChatGPT command=%q", got["chatgpt"])
	}
	if got["claude"] != "team-launcher launch claude" {
		t.Fatalf("Claude command=%q", got["claude"])
	}
}
