package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"harnezpad/internal/version"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func TestHelpListsLaunchCommands(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"harnezpad", "help"}
	out := captureStdout(t, func() {
		if !Run() {
			t.Fatal("help should be handled as a CLI command")
		}
	})
	if strings.TrimSpace(out) != HelpText {
		t.Fatalf("help output mismatch:\n%s", out)
	}
	for _, cmd := range []string{
		"harnezpad launch claude",
		"harnezpad launch codex",
		"harnezpad launch chatgpt",
		"harnezpad launch opencode",
	} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("help missing %q", cmd)
		}
	}
}

func TestVersionPrintsPackageVersion(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"harnezpad", "version"}
	out := captureStdout(t, func() {
		if !Run() {
			t.Fatal("version should be handled as a CLI command")
		}
	})
	want := "harnezpad " + version.Version
	if strings.TrimSpace(out) != want {
		t.Fatalf("version: got %q want %q", strings.TrimSpace(out), want)
	}
}

func TestReadmeDocumentsConnectAndLaunch(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	gif := "assets/screenshots/harnezpad.gif"
	if !strings.Contains(text, "]("+gif+")") && !strings.Contains(text, "](./"+gif+")") {
		t.Fatalf("README.md must image-embed %s", gif)
	}
	gifBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(gif)))
	if err != nil {
		t.Fatal(err)
	}
	if len(gifBytes) < 4 || string(gifBytes[:4]) != "GIF8" {
		t.Fatal("committed GIF is missing GIF8 magic")
	}
	for _, cmd := range []string{
		"harnezpad launch claude",
		"harnezpad launch codex",
		"harnezpad launch chatgpt",
		"harnezpad launch opencode",
	} {
		if !strings.Contains(text, cmd) {
			t.Fatalf("README.md must document %q", cmd)
		}
	}
	if !strings.Contains(text, "Settings") || !strings.Contains(strings.ToLower(text), "onboarding") {
		t.Fatal("README.md must explain connecting via Settings/onboarding")
	}
}
