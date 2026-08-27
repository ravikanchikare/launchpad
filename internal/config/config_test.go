package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPrecedenceAndNormalization(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LITELLM_BASE_URL", "https://gateway.example.test/")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.GatewayURL != "https://gateway.example.test" {
		t.Fatalf("GatewayURL = %q", got.GatewayURL)
	}
}

func TestSaveDoesNotStoreSecrets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LITELLM_API_KEY", "do-not-write")
	if err := Save(Settings{GatewayURL: "https://gateway.example.test"}); err != nil {
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
