package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const DefaultProviderURL = "http://localhost:4000"

var DefaultCLIName = "launchpad"

type Settings struct {
	ProviderURL string `json:"providerUrl"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Launchpad", "settings.json"), nil
}

func Load() (Settings, error) {
	settings := Settings{ProviderURL: DefaultProviderURL}
	path, err := Path()
	if err != nil {
		return Settings{}, err
	}
	if data, readErr := os.ReadFile(path); readErr == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return Settings{}, fmt.Errorf("read settings: %w", err)
		}
	} else if !os.IsNotExist(readErr) {
		return Settings{}, readErr
	}
	if value := strings.TrimSpace(os.Getenv("LITELLM_BASE_URL")); value != "" {
		settings.ProviderURL = value
	}
	if settings.ProviderURL == "" {
		settings.ProviderURL = DefaultProviderURL
	}
	normalized, err := NormalizeProviderURL(settings.ProviderURL)
	if err != nil {
		return Settings{}, err
	}
	settings.ProviderURL = normalized
	return settings, nil
}

func Save(settings Settings) error {
	normalized, err := NormalizeProviderURL(settings.ProviderURL)
	if err != nil {
		return err
	}
	settings.ProviderURL = normalized
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func NormalizeProviderURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", errors.New("provider URL must be an absolute URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", errors.New("provider URL must use http or https")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("provider URL cannot contain a query or fragment")
	}
	return value, nil
}

func CLIName() string {
	return DefaultCLIName
}
