package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupAfterUpgrade(t *testing.T) {
	updater := NewAppUpdater("1.0.0", "https://example.com/update")
	updater.cacheDir = t.TempDir()
	backupDir := filepath.Join(updater.cacheDir, "backup")
	if err := os.MkdirAll(filepath.Join(backupDir, "HarnezPad.app"), 0755); err != nil {
		t.Fatal(err)
	}

	cleaned, err := updater.CleanupAfterUpgrade()
	if err != nil {
		t.Fatal(err)
	}
	if !cleaned {
		t.Fatal("expected orphaned backup to be cleaned")
	}
	if _, err := os.Stat(backupDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected backup dir removed, got %v", err)
	}

	marker := filepath.Join(updater.cacheDir, "upgraded")
	if err := os.WriteFile(marker, []byte("1.1.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(backupDir, "HarnezPad.app"), 0755); err != nil {
		t.Fatal(err)
	}
	cleaned, err = updater.CleanupAfterUpgrade()
	if err != nil {
		t.Fatal(err)
	}
	if !cleaned {
		t.Fatal("expected marked upgrade backup to be cleaned")
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected marker removed, got %v", err)
	}
}

func TestBundlePathFromLegacyHostAndNativeHelper(t *testing.T) {
	for name, executable := range map[string]string{
		"legacy host":   "/Applications/HarnezPad.app/Contents/MacOS/HarnezPad",
		"native helper": "/Applications/HarnezPad.app/Contents/Resources/harnezpad",
	} {
		t.Run(name, func(t *testing.T) {
			path, err := bundlePathFromExecutable(executable)
			if err != nil {
				t.Fatal(err)
			}
			if path != "/Applications/HarnezPad.app" {
				t.Fatalf("bundle path = %q", path)
			}
		})
	}
	for _, executable := range []string{
		"/tmp/harnezpad",
		"/Applications/HarnezPad.app/Resources/harnezpad",
		"/Applications/HarnezPad.app/Contents/Helpers/harnezpad",
		"/tmp/Resources/harnezpad",
	} {
		if _, err := bundlePathFromExecutable(executable); err == nil {
			t.Fatalf("expected unpackaged executable %q to be rejected", executable)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	if CompareVersions("1.2.0", "1.1.9") != 1 {
		t.Fatal("expected newer version")
	}
	if CompareVersions("1.2.0", "1.2.0") != 0 {
		t.Fatal("expected equal version")
	}
	if CompareVersions("1.1.9", "1.2.0") != -1 {
		t.Fatal("expected older version")
	}
}

func TestUpdaterCheckAndDownload(t *testing.T) {
	payload := []byte("harnezpad update payload")
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/update" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.1.0","url":"https://example.com/harnezpad.zip","sha256":"` + checksum + `"}`))
			return
		}
		if r.URL.Path == "/artifact" {
			_, _ = w.Write(payload)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	updater := NewAppUpdater("1.0.0", server.URL+"/update")
	updater.cacheDir = t.TempDir()
	updater.signingSecret = "test-secret"
	updater.verifyArchive = func(string) error { return nil }
	info, err := updater.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	info.URL = server.URL + "/artifact"
	if err := validateUpdateInfo(*info); err == nil {
		t.Fatal("expected non-HTTPS test URL to be rejected")
	}
	info.URL = "https://example.com/harnezpad.zip"
	updater.client = &http.Client{Transport: rewriteTransport{target: server.URL + "/artifact"}}
	if err := updater.Download(context.Background(), *info); err != nil {
		t.Fatal(err)
	}
	if !updater.Status().Downloaded {
		t.Fatal("expected update to be marked downloaded")
	}
	if _, err := os.Stat(updater.stagedPath(*info)); err != nil {
		t.Fatal(err)
	}
}

func TestStagedArtifactNameUsesSignedReleaseFile(t *testing.T) {
	rawURL := "https://update-api.example.com/api/release?expires=123&file=HarnezPad-darwin-universal.zip&sig=abc"
	if name := stagedArtifactName(rawURL); name != "HarnezPad-darwin-universal.zip" {
		t.Fatalf("expected stable zip filename, got %q", name)
	}
}

func TestSkipArchiveEntry(t *testing.T) {
	for _, name := range []string{
		"__MACOSX/",
		"__MACOSX/._HarnezPad.app",
		".DS_Store",
		"HarnezPad.app/._Contents",
	} {
		if !skipArchiveEntry(name) {
			t.Fatalf("expected %q to be skipped", name)
		}
	}
	if skipArchiveEntry("HarnezPad.app/Contents/Info.plist") {
		t.Fatal("expected bundle entries to be kept")
	}
}

type rewriteTransport struct{ target string }

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL, _ = req.URL.Parse(t.target)
	return http.DefaultTransport.RoundTrip(clone)
}
