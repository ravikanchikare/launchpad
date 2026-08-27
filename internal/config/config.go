package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const DefaultGatewayURL = "http://localhost:4000"

var commandNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type Settings struct {
	GatewayURL string `json:"gatewayUrl"`
	CLIName    string `json:"cliName,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Launchpad", "settings.json"), nil
}

func Load() (Settings, error) {
	settings := Settings{GatewayURL: DefaultGatewayURL}
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
		settings.GatewayURL = value
	}
	if settings.GatewayURL == "" {
		settings.GatewayURL = DefaultGatewayURL
	}
	normalized, err := NormalizeGatewayURL(settings.GatewayURL)
	if err != nil {
		return Settings{}, err
	}
	settings.GatewayURL = normalized
	return settings, nil
}

func Save(settings Settings) error {
	normalized, err := NormalizeGatewayURL(settings.GatewayURL)
	if err != nil {
		return err
	}
	settings.GatewayURL = normalized
	if settings.CLIName != "" && !commandNamePattern.MatchString(settings.CLIName) {
		return errors.New("CLI name must be a command name, not a path")
	}
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

func NormalizeGatewayURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", errors.New("gateway URL must be an absolute URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", errors.New("gateway URL must use http or https")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("gateway URL cannot contain a query or fragment")
	}
	return value, nil
}

func CLIName(settings Settings, executable string) string {
	for _, candidate := range []string{
		strings.TrimSpace(os.Getenv("LAUNCHPAD_CLI_NAME")),
		strings.TrimSpace(settings.CLIName),
		filepath.Base(executable),
		"launchpad",
	} {
		if commandNamePattern.MatchString(candidate) {
			return candidate
		}
	}
	return "launchpad"
}
