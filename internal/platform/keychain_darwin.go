//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const namedKeyService = "com.harnezai.launchpad.keys"
const managementKeySlug = "management-key"

func namedAccount(slug string) string {
	return "key:" + slug
}

func LoadGatewayToken() string {
	return LoadNamedKey(managementKeySlug)
}

func SaveGatewayToken(token string) error {
	if token == "" {
		return errors.New("management key cannot be empty")
	}
	return SaveNamedKey(managementKeySlug, token)
}

func LoadNamedKey(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	out, err := exec.Command("/usr/bin/security", "find-generic-password", "-a", namedAccount(slug), "-s", namedKeyService, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func SaveNamedKey(slug, token string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return errors.New("key name is required")
	}
	if token == "" {
		return errors.New("management key cannot be empty")
	}
	cmd := exec.Command("/usr/bin/security", "add-generic-password", "-U", "-a", namedAccount(slug), "-s", namedKeyService, "-w", token)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func DeleteNamedKey(slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil
	}
	cmd := exec.Command("/usr/bin/security", "delete-generic-password", "-a", namedAccount(slug), "-s", namedKeyService)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "could not be found") {
			return nil
		}
		return errors.New(msg)
	}
	return nil
}
