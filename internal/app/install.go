package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type CLIStatus struct {
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	Error     string `json:"error,omitempty"`
}

func InstallCLI() CLIStatus {
	if runtime.GOOS != "darwin" {
		return cliFailure("", errors.New("HarnezPad CLI installation is only supported on macOS"))
	}
	exe, err := os.Executable()
	if err != nil {
		return cliFailure("", err)
	}
	bundledCLI, err := packagedCLIPath(exe)
	if err != nil {
		return cliFailure("", err)
	}
	info, err := os.Stat(bundledCLI)
	if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return cliFailure(bundledCLI, errors.New("HarnezPad CLI is missing from the application bundle. Reinstall HarnezPad"))
	}
	target := bundledCLI
	dir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return cliFailure(filepath.Join(dir, "harnezpad"), err)
	}
	link := filepath.Join(dir, "harnezpad")
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return cliFailure(link, fmt.Errorf("refusing to replace existing %s", link))
		}
		existingTarget, readErr := os.Readlink(link)
		if readErr != nil {
			if err := os.Remove(link); err != nil {
				return cliFailure(link, err)
			}
		} else if existingTarget == target || managedCLISymlink(existingTarget) {
			if err := os.Remove(link); err != nil {
				return cliFailure(link, err)
			}
		} else {
			return cliFailure(link, fmt.Errorf("refusing to replace existing symlink %s", link))
		}
	}
	if err := os.Symlink(target, link); err != nil {
		return cliFailure(link, err)
	}
	return CLIStatus{Installed: true, Path: link}
}

func packagedCLIPath(executable string) (string, error) {
	switch filepath.Base(executable) {
	case "HarnezPad":
		return filepath.Join(filepath.Dir(filepath.Dir(executable)), "Resources", "harnezpad"), nil
	case "harnezpad":
		if filepath.Base(filepath.Dir(executable)) == "Resources" && filepath.Base(filepath.Dir(filepath.Dir(executable))) == "Contents" {
			return executable, nil
		}
	}
	return "", errors.New("HarnezPad CLI is only installed by the packaged app")
}

func cliFailure(path string, err error) CLIStatus {
	return CLIStatus{Path: path, Error: err.Error()}
}

func managedCLISymlink(target string) bool {
	return strings.Contains(target, "/HarnezPad.app/Contents/Resources/harnezpad") ||
		strings.Contains(target, "/HarnezPad.app/Contents/MacOS/HarnezPad")
}
