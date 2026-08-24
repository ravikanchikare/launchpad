package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"harnezpad/internal/keys"
)

func TestClaudeEnvironmentDisablesInheritedAWSRouting(t *testing.T) {
	t.Setenv("AWS_PROFILE", "example-sso")
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
	t.Setenv("CLAUDE_CODE_USE_MANTLE", "1")
	t.Setenv("ANTHROPIC_BEDROCK_BASE_URL", "https://bedrock.example")
	t.Setenv("ANTHROPIC_MODEL", "inherited")
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example", Token: "token"}}
	cmd := exec.Command("/usr/bin/true")
	if err := m.ApplyEnvironmentForModel(cmd, "claude", "kimi-k3", "token"); err != nil {
		t.Fatal(err)
	}
	values := envMap(cmd.Env)
	if _, ok := values["AWS_PROFILE"]; ok {
		t.Fatal("AWS_PROFILE leaked into Claude child environment")
	}
	if _, ok := values["ANTHROPIC_BEDROCK_BASE_URL"]; ok {
		t.Fatal("ANTHROPIC_BEDROCK_BASE_URL leaked into Claude child environment")
	}
	if _, ok := values["CLAUDE_CODE_USE_BEDROCK"]; ok {
		t.Fatal("CLAUDE_CODE_USE_BEDROCK should be absent from Claude child environment")
	}
	if values["ANTHROPIC_BASE_URL"] != "https://gateway.example" || values["ANTHROPIC_AUTH_TOKEN"] != "token" || values["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "kimi-k3" {
		t.Fatalf("Claude gateway environment = %v", values)
	}
	if values["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"] != "1" {
		t.Fatalf("CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY = %q", values["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"])
	}
	if got := os.Getenv("AWS_PROFILE"); got != "example-sso" {
		t.Fatalf("parent AWS_PROFILE changed to %q", got)
	}
}

func TestResolveLaunchTokenFallsBackWhenDefaultKeyMissing(t *testing.T) {
	m := &Manager{
		Settings: Settings{
			GatewayURL:     "https://gateway.example",
			Token:          "mgmt-token",
			DefaultKeySlug: "harnezpad-unit-test-missing-launch-key",
		},
	}
	want := m.tokenForSlug(keys.ManagementSlug)
	token, err := m.ResolveLaunchToken("")
	if err != nil {
		t.Fatalf("ResolveLaunchToken(\"\") err = %v", err)
	}
	if token != want {
		t.Fatalf("ResolveLaunchToken(\"\") = %q, want fallback %q", token, want)
	}
	token, err = m.ResolveLaunchToken("harnezpad-unit-test-missing-launch-key")
	if err != nil {
		t.Fatalf("ResolveLaunchToken(default slug) err = %v", err)
	}
	if token != want {
		t.Fatalf("ResolveLaunchToken(default slug) = %q, want fallback %q", token, want)
	}
}

func TestResolveLaunchTokenRejectsMissingExplicitNonDefaultKey(t *testing.T) {
	m := &Manager{
		Settings: Settings{
			GatewayURL:     "https://gateway.example",
			Token:          "mgmt-token",
			DefaultKeySlug: "harnezpad-unit-test-missing-launch-key",
		},
	}
	if _, err := m.ResolveLaunchToken("harnezpad-unit-test-other-missing-key"); err == nil {
		t.Fatal("expected error for missing explicit non-default key")
	}
}

func TestMissingGatewayTokenRejected(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example"}}
	if err := m.RequireGatewayToken(); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected missing token error, got %v", err)
	}
	cmd := exec.Command("/usr/bin/true")
	if err := m.ApplyEnvironmentForModel(cmd, "claude", "model-a", ""); err == nil {
		t.Fatal("expected launch environment to reject missing token")
	}
}

func TestCodexEnvironmentInheritsParentAndSetsLaunchToken(t *testing.T) {
	t.Setenv("HARNEZPAD_PARENT_ENV_TEST", "parent")
	t.Setenv("OPENAI_BASE_URL", "https://inherited.example")
	t.Setenv("OPENAI_MODEL", "inherited-model")
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example", Token: "secret"}}
	cmd := exec.Command("/usr/bin/true")
	if err := m.ApplyEnvironmentForModel(cmd, "codex-cli", "model-a", "launch-token"); err != nil {
		t.Fatal(err)
	}
	values := envMap(cmd.Env)
	if values["OPENAI_API_KEY"] != "launch-token" {
		t.Fatalf("OPENAI_API_KEY = %q, want launch-token", values["OPENAI_API_KEY"])
	}
	if values["HARNEZPAD_PARENT_ENV_TEST"] != "parent" || values["OPENAI_BASE_URL"] != "https://inherited.example" || values["OPENAI_MODEL"] != "inherited-model" {
		t.Fatalf("Codex launch did not inherit parent environment: %v", values)
	}
}

func TestChatGPTEnvironmentUsesLaunchToken(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example", Token: "secret"}}
	cmd := exec.Command("/usr/bin/true")
	if err := m.ApplyEnvironmentForModel(cmd, "codex-desktop", "model-a", "launch-token"); err != nil {
		t.Fatal(err)
	}
	values := envMap(cmd.Env)
	if values["OPENAI_API_KEY"] != "launch-token" {
		t.Fatalf("OPENAI_API_KEY = %q, want launch-token", values["OPENAI_API_KEY"])
	}
}

func TestIntegrationDisplayName(t *testing.T) {
	if got := IntegrationDisplayName("codex-desktop"); got != "ChatGPT" {
		t.Fatalf("display name = %q", got)
	}
}

func TestListModelsForKeyUsesModelGroupInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model_group/info" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"model_group": "gpt-5.5", "providers": []string{"openai"}},
				map[string]any{"model_group": "claude-sonnet-5", "providers": []string{"bedrock"}},
			},
		})
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "token"}}
	m.saveNamedKey = func(slug, token string) error { return nil }
	models, err := m.ListModelsForKey(context.Background(), keys.ManagementSlug)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "claude-sonnet-5" || models[0].OwnedBy != "bedrock" {
		t.Fatalf("models = %#v", models)
	}
	if models[1].ID != "gpt-5.5" || models[1].OwnedBy != "openai" {
		t.Fatalf("models = %#v", models)
	}
}

func envMap(env []string) map[string]string {
	values := make(map[string]string)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values
}
