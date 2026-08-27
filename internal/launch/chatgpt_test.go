package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureAndRestoreChatGPTPreservesUserConfig(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, "config")
	codexHome := filepath.Join(home, ".codex")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("CODEX_HOME", codexHome)
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	original := "model = \"user-model\"\napproval_policy = \"on-request\"\n\n[features]\nweb_search = true\n"
	configPath := filepath.Join(codexHome, "config.toml")
	if err := os.WriteFile(configPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureChatGPT("https://provider.example/v1", "cheap-model", "/tmp/launchpad"); err != nil {
		t.Fatal(err)
	}
	managed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`model = "cheap-model"`, `approval_policy = "on-request"`, chatGPTManagedBegin, `base_url = "https://provider.example/v1"`} {
		if !strings.Contains(string(managed), want) {
			t.Fatalf("managed config missing %q:\n%s", want, managed)
		}
	}
	if err := RestoreChatGPT(); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(restored)) != strings.TrimSpace(original) {
		t.Fatalf("restored config differs\nwant:\n%s\ngot:\n%s", original, restored)
	}
}
