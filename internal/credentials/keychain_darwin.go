//go:build darwin

package credentials

import (
	"os/exec"
	"strings"
)

const keychainService = "com.harnezai.launchpad.keys"

func loadKeychain(slug string) string {
	out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", slug, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func SaveToKeychain(slug, token string) error {
	return exec.Command("security", "add-generic-password", "-U", "-s", keychainService, "-a", slug, "-w", token).Run()
}
