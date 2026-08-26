package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"harnezpad/internal/version"
)

type UpdateInfo struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature,omitempty"`
}

type UpdateStatus struct {
	CurrentVersion string      `json:"currentVersion"`
	Channel        string      `json:"channel"`
	CheckedAt      *time.Time  `json:"checkedAt,omitempty"`
	Available      bool        `json:"available"`
	Downloaded     bool        `json:"downloaded"`
	Update         *UpdateInfo `json:"update,omitempty"`
	Error          string      `json:"error,omitempty"`
}

type AppUpdater struct {
	mu             sync.Mutex
	currentVersion string
	endpoint       string
	signingSecret  string
	cacheDir       string
	client         *http.Client
	verifyArchive  func(string) error
	status         UpdateStatus
}

func NewAppUpdater(currentVersion, endpoint string) *AppUpdater {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	if endpoint == "" {
		endpoint = version.DefaultUpdateAPI
	}
	u := &AppUpdater{
		currentVersion: currentVersion,
		endpoint:       endpoint,
		signingSecret:  version.UpdateSigningSecret,
		cacheDir:       filepath.Join(cacheDir, "HarnezPad"),
		client:         http.DefaultClient,
		verifyArchive:  verifyHarnezPadArchive,
		status:         UpdateStatus{CurrentVersion: currentVersion, Channel: version.UpdateChannel},
	}
	return u
}

func (u *AppUpdater) Status() UpdateStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	status := u.status
	if status.Update != nil {
		copy := *status.Update
		status.Update = &copy
	}
	return status
}

func (u *AppUpdater) Check(ctx context.Context) (*UpdateInfo, error) {
	if semverParts(u.currentVersion) == nil || u.currentVersion == "dev" {
		return nil, errors.New("self-update is unavailable for development builds")
	}
	requestURL, err := url.Parse(u.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse update endpoint: %w", err)
	}
	query := requestURL.Query()
	query.Set("os", runtime.GOOS)
	query.Set("arch", runtime.GOARCH)
	query.Set("version", u.currentVersion)
	query.Set("channel", version.UpdateChannel)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create update request: %w", err)
	}
	req.Header.Set("User-Agent", "harnezpad/"+u.currentVersion)
	if err := signClientRequest(req, u.signingSecret); err != nil {
		return nil, err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check for update: %w", err)
	}
	defer resp.Body.Close()

	u.mu.Lock()
	now := time.Now()
	u.status.CheckedAt = &now
	u.status.Error = ""
	u.status.Available = false
	u.status.Downloaded = false
	u.status.Update = nil
	u.mu.Unlock()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("update endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var info UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode update response: %w", err)
	}
	if err := validateUpdateInfo(info); err != nil {
		return nil, err
	}
	if CompareVersions(info.Version, u.currentVersion) <= 0 {
		return nil, nil
	}
	u.mu.Lock()
	u.status.Available = true
	u.status.Update = &info
	u.mu.Unlock()
	return &info, nil
}

func (u *AppUpdater) Download(ctx context.Context, info UpdateInfo) error {
	if err := validateUpdateInfo(info); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(u.stagedPath(info)), 0700); err != nil {
		return fmt.Errorf("create update cache: %w", err)
	}
	finalPath := u.stagedPath(info)
	if fileMatchesSHA256(finalPath, info.SHA256) {
		if err := u.verifyArchive(finalPath); err != nil {
			_ = os.Remove(finalPath)
			return fmt.Errorf("verify staged update: %w", err)
		}
		u.markDownloaded(info)
		return nil
	}

	tmp, err := os.CreateTemp(filepath.Dir(finalPath), ".download-*.zip")
	if err != nil {
		return fmt.Errorf("create temporary update: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	resp, err := u.authorizedGet(ctx, info.URL)
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download update: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		tmp.Close()
		return fmt.Errorf("download endpoint returned %s", resp.Status)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		resp.Body.Close()
		tmp.Close()
		return fmt.Errorf("write update: %w", err)
	}
	resp.Body.Close()
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close update: %w", err)
	}
	if !fileMatchesSHA256(tmpPath, info.SHA256) {
		return errors.New("downloaded update SHA-256 does not match release metadata")
	}
	if err := u.verifyArchive(tmpPath); err != nil {
		return fmt.Errorf("verify downloaded update: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("stage update: %w", err)
	}
	u.markDownloaded(info)
	return nil
}

func (u *AppUpdater) markDownloaded(info UpdateInfo) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status.Available = true
	u.status.Downloaded = true
	u.status.Update = &info
	u.status.Error = ""
}

func (u *AppUpdater) stagedPath(info UpdateInfo) string {
	name := stagedArtifactName(info.URL)
	sum := sha256.Sum256([]byte(strings.ToLower(info.SHA256)))
	return filepath.Join(u.cacheDir, "updates", hex.EncodeToString(sum[:]), name)
}

func stagedArtifactName(rawURL string) string {
	if parsed, err := url.Parse(rawURL); err == nil {
		if file := strings.TrimSpace(parsed.Query().Get("file")); file != "" {
			return filepath.Base(file)
		}
		if base := strings.TrimSpace(parsed.Path); base != "" {
			if name := filepath.Base(base); name != "." && name != "/" {
				return name
			}
		}
	}
	name := filepath.Base(rawURL)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "HarnezPad-update.zip"
	}
	return name
}

func (u *AppUpdater) Install(info UpdateInfo) error {
	if runtime.GOOS != "darwin" {
		return errors.New("self-update is currently supported on macOS only")
	}
	if err := validateUpdateInfo(info); err != nil {
		return err
	}
	bundlePath, err := currentBundlePath()
	if err != nil {
		return err
	}
	staged := u.stagedPath(info)
	if !fileMatchesSHA256(staged, info.SHA256) {
		return errors.New("verified staged update was not found")
	}
	if err := u.verifyArchive(staged); err != nil {
		return fmt.Errorf("reverify update: %w", err)
	}

	backupDir := filepath.Join(u.cacheDir, "backup")
	backupPath := filepath.Join(backupDir, "HarnezPad.app")
	if _, err := os.Stat(backupPath); err == nil {
		return errors.New("a previous update is awaiting cleanup; restart HarnezPad before updating again")
	}
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("create update backup: %w", err)
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(bundlePath), ".harnezpad-update-*")
	if err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	if err := extractHarnezPadBundle(staged, tempDir); err != nil {
		return err
	}
	newBundle := filepath.Join(tempDir, "HarnezPad.app")
	if err := u.verifyArchive(newBundle); err != nil {
		return fmt.Errorf("verify extracted app: %w", err)
	}
	if err := os.Rename(bundlePath, backupPath); err != nil {
		return fmt.Errorf("backup current app: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(bundlePath)
			_ = os.Rename(backupPath, bundlePath)
		}
	}()
	if err := os.Rename(newBundle, bundlePath); err != nil {
		return fmt.Errorf("install new app: %w", err)
	}
	if err := os.MkdirAll(u.cacheDir, 0700); err != nil {
		return fmt.Errorf("create upgrade marker directory: %w", err)
	}
	marker := filepath.Join(u.cacheDir, "upgraded")
	if err := os.WriteFile(marker, []byte(info.Version+"\n"), 0600); err != nil {
		return fmt.Errorf("write upgrade marker: %w", err)
	}
	_ = os.Remove(staged)
	failed = false
	return nil
}

func (u *AppUpdater) CleanupAfterUpgrade() (bool, error) {
	marker := filepath.Join(u.cacheDir, "upgraded")
	backupDir := filepath.Join(u.cacheDir, "backup")
	_, markerErr := os.Stat(marker)
	markerExists := markerErr == nil
	if markerErr != nil && !errors.Is(markerErr, os.ErrNotExist) {
		return false, markerErr
	}
	_, backupErr := os.Stat(filepath.Join(backupDir, "HarnezPad.app"))
	backupExists := backupErr == nil
	if backupErr != nil && !errors.Is(backupErr, os.ErrNotExist) {
		return false, backupErr
	}
	if !markerExists && !backupExists {
		return false, nil
	}
	if err := os.RemoveAll(backupDir); err != nil {
		return true, err
	}
	if markerExists {
		if err := os.Remove(marker); err != nil {
			return true, err
		}
	}
	return true, nil
}

func (u *AppUpdater) LaunchNewApp() error {
	bundlePath, err := currentBundlePath()
	if err != nil {
		return err
	}
	return exec.Command("open", "-n", bundlePath).Start()
}

func (u *AppUpdater) StartBackground(ctx context.Context, onUpdate func(UpdateInfo)) {
	go func() {
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		check := func() {
			info, err := u.Check(ctx)
			if err != nil {
				u.mu.Lock()
				u.status.Error = err.Error()
				u.mu.Unlock()
				return
			}
			if info != nil && onUpdate != nil {
				onUpdate(*info)
			}
		}
		check()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				check()
			}
		}
	}()
}

func validateUpdateInfo(info UpdateInfo) error {
	if semverParts(info.Version) == nil {
		return errors.New("update response contains an invalid version")
	}
	u, err := url.Parse(info.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("update response contains an unsafe download URL")
	}
	decoded, decodeErr := hex.DecodeString(info.SHA256)
	if decodeErr != nil || len(decoded) != sha256.Size {
		return errors.New("update response contains an invalid SHA-256")
	}
	return nil
}

func semverParts(value string) []int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i, part := range parts {
		if idx := strings.IndexAny(part, "-+"); idx >= 0 {
			part = part[:idx]
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil
		}
		out[i] = n
	}
	return out
}

func CompareVersions(left, right string) int {
	a, b := semverParts(left), semverParts(right)
	if a == nil || b == nil {
		return 0
	}
	for i := range a {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

func fileMatchesSHA256(path, expected string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected)
}

func currentBundlePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return bundlePathFromExecutable(exe)
}

func bundlePathFromExecutable(executable string) (string, error) {
	executableDir := filepath.Dir(executable)
	directoryName := filepath.Base(executableDir)
	contentsDir := filepath.Dir(executableDir)
	if (directoryName != "MacOS" && directoryName != "Resources") || filepath.Base(contentsDir) != "Contents" {
		return "", errors.New("self-update requires a packaged HarnezPad.app")
	}
	return filepath.Dir(contentsDir), nil
}

func verifyHarnezPadArchive(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if isZipArchive(path) {
		dir, err := os.MkdirTemp("", "harnezpad-update-verify-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		if err := extractHarnezPadBundle(path, dir); err != nil {
			return err
		}
		path = filepath.Join(dir, "HarnezPad.app")
	}
	return verifyHarnezPadBundle(path)
}

func isZipArchive(path string) bool {
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		return true
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	var signature [2]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		return false
	}
	return signature[0] == 'P' && signature[1] == 'K'
}

func verifyHarnezPadBundle(path string) error {
	return exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", "--verbose=2", path).Run()
}

func skipArchiveEntry(name string) bool {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == ".." {
		return true
	}
	base := filepath.Base(clean)
	if base == ".DS_Store" || strings.HasPrefix(base, "._") {
		return true
	}
	if clean == "__MACOSX" || strings.HasPrefix(clean, "__MACOSX"+string(filepath.Separator)) {
		return true
	}
	return false
}

func extractHarnezPadBundle(archivePath, destination string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open update archive: %w", err)
	}
	defer r.Close()
	for _, entry := range r.File {
		if skipArchiveEntry(entry.Name) {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(entry.Name))
		if filepath.IsAbs(clean) || (clean != "HarnezPad.app" && !strings.HasPrefix(clean, "HarnezPad.app"+string(filepath.Separator))) {
			return fmt.Errorf("update archive contains unsafe path %q", entry.Name)
		}
		destinationPath := filepath.Join(destination, clean)
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(destinationPath, 0755); err != nil {
				return err
			}
			continue
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			source, err := entry.Open()
			if err != nil {
				return err
			}
			link, readErr := io.ReadAll(source)
			source.Close()
			if readErr != nil || filepath.IsAbs(string(link)) {
				return fmt.Errorf("update archive contains unsafe symlink %q", entry.Name)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(destinationPath), filepath.FromSlash(string(link))))
			if !withinDirectory(destination, resolved) {
				return fmt.Errorf("update archive symlink escapes bundle %q", entry.Name)
			}
			if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
				return err
			}
			if err := os.Symlink(string(link), destinationPath); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		dest, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(dest, source)
		source.Close()
		closeErr := dest.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func withinDirectory(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
