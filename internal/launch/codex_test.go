package launch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harnezpad/internal/gateway"
)

func TestConfigureAndRestoreCodexOwnedProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	rootPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(rootPath, []byte("model = \"usual\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := testGateway{url: "https://gateway.example", token: "secret"}
	if err := Configure(settings, "model-a", []gateway.Model{{ID: "model-a"}, {ID: "model-b"}}); err != nil {
		t.Fatal(err)
	}
	profilePath, _ := ProfilePath()
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{CodexProfileMarker, `model = "model-a"`, `model_provider = "harnezpad-launch"`, `[model_providers.harnezpad-launch]`, `env_key = "OPENAI_API_KEY"`, `wire_api = "responses"`} {
		if !strings.Contains(string(profile), want) {
			t.Fatalf("profile missing %q:\n%s", want, profile)
		}
	}
	if err := RestoreCodex(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		t.Fatalf("profile still exists: %v", err)
	}
	root, err := os.ReadFile(rootPath)
	if err != nil || string(root) != "model = \"usual\"\n" {
		t.Fatalf("root config changed: %q, %v", root, err)
	}
}

func TestConfigureCodexRefusesUserOwnedProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	profilePath, _ := ProfilePath()
	if err := os.WriteFile(profilePath, []byte("model = \"mine\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := Configure(testGateway{url: "https://gateway.example"}, "model-a", []gateway.Model{{ID: "model-a"}})
	if err == nil || !strings.Contains(err.Error(), "user-owned") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestCodexLaunchArgsOwnProfileAndModel(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	settings := testGateway{url: "https://gateway.example"}
	got, err := LaunchArgs(settings, []string{"--sandbox", "workspace-write"}, "model-a")
	if err != nil {
		t.Fatal(err)
	}
	catalogPath, _ := CatalogPath()
	want := []string{
		"--profile", CodexProfileName,
		"-c", `model_provider="harnezpad-launch"`,
		"-c", `model_providers.harnezpad-launch.name="HarnezPad"`,
		"-c", `model_providers.harnezpad-launch.base_url="https://gateway.example/v1"`,
		"-c", `model_providers.harnezpad-launch.env_key="OPENAI_API_KEY"`,
		"-c", `model_providers.harnezpad-launch.wire_api="responses"`,
		"-c", `model_catalog_json="` + catalogPath + `"`,
		"-m", "model-a", "--sandbox", "workspace-write",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("args = %v, want %v", got, want)
	}
}

func TestModelCatalogIncludesDesktopRequiredFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	path := filepath.Join(dir, "catalog.json")
	if err := writeModelCatalog(path, []gateway.Model{{ID: "model-a"}}, "HarnezPad gateway model"); err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Models []map[string]any `json:"models"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Models) != 1 {
		t.Fatalf("models = %d", len(catalog.Models))
	}
	for _, key := range []string{"supported_reasoning_levels", "base_instructions", "truncation_policy", "effective_context_window_percent", "experimental_supported_tools"} {
		if _, ok := catalog.Models[0][key]; !ok {
			t.Errorf("catalog entry missing %q", key)
		}
	}
}

type testGateway struct {
	url   string
	token string
}

func (g testGateway) GatewayURL() string { return g.url }
func (g testGateway) Token() string      { return g.token }
