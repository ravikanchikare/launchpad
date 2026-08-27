package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"launchpad/internal/provider"
)

func TestLoadPrecedenceAndNormalization(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAUNCHPAD_PROVIDER_URL", "https://provider.example.test/v1/")
	t.Setenv("LAUNCHPAD_PROVIDER_KIND", "openai-compatible")
	t.Setenv("LAUNCHPAD_MODELS_URL", "https://catalog.example.test/models/")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderURL != "https://provider.example.test/v1" {
		t.Fatalf("ProviderURL = %q", got.ProviderURL)
	}
	if got.ProviderKind != provider.KindOpenAICompatible {
		t.Fatalf("ProviderKind = %q", got.ProviderKind)
	}
	if got.ModelsURL != "https://catalog.example.test/models" {
		t.Fatalf("ModelsURL = %q", got.ModelsURL)
	}
}

func TestSaveDoesNotStoreSecrets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "do-not-write")
	if err := Save(Settings{ProviderURL: "https://provider.example.test"}); err != nil {
		t.Fatal(err)
	}
	path, _ := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || filepath.Base(path) != "settings.json" {
		t.Fatalf("unexpected settings file %q", data)
	}
	if strings.Contains(string(data), "do-not-write") {
		t.Fatal("settings contain the API key")
	}
}

func TestCLINameUsesBuildDefault(t *testing.T) {
	original := DefaultCLIName
	DefaultCLIName = "team-launcher"
	t.Cleanup(func() {
		DefaultCLIName = original
	})

	if got := CLIName(); got != "team-launcher" {
		t.Fatalf("CLIName() = %q", got)
	}
}
