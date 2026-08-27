package credentials

import (
	"errors"
	"os"
	"strings"
)

const ManagementKeySlug = "management-key"

func Resolve() (string, error) {
	if token := strings.TrimSpace(os.Getenv("LAUNCHPAD_PROVIDER_API_KEY")); token != "" {
		return token, nil
	}
	if os.Getenv("LAUNCHPAD_DISABLE_KEYCHAIN") != "1" {
		if token := strings.TrimSpace(loadKeychain(ManagementKeySlug)); token != "" {
			return token, nil
		}
	}
	return "", errors.New("LAUNCHPAD_PROVIDER_API_KEY is not set and no management key is stored in Keychain")
}

func PersistForDesktop(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("cannot store an empty management key")
	}
	if token == strings.TrimSpace(os.Getenv("LAUNCHPAD_PROVIDER_API_KEY")) {
		return SaveToKeychain(ManagementKeySlug, token)
	}
	return nil
}
