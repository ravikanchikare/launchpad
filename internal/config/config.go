package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"launchpad/internal/provider"
)

const DefaultProviderURL = "http://localhost:4000"

var DefaultCLIName = "launchpad"

type Settings struct {
	ProviderKind provider.Kind `json:"providerKind"`
	ProviderURL  string        `json:"providerUrl"`
	ModelsURL    string        `json:"modelsUrl,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Launchpad", "settings.json"), nil
}

func Load() (Settings, error) {
	settings := Settings{ProviderKind: provider.KindLiteLLM, ProviderURL: DefaultProviderURL}
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
	if value := strings.TrimSpace(os.Getenv("LAUNCHPAD_PROVIDER_URL")); value != "" {
		settings.ProviderURL = value
	}
	if value := strings.TrimSpace(os.Getenv("LAUNCHPAD_PROVIDER_KIND")); value != "" {
		settings.ProviderKind = provider.Kind(value)
	}
	if value := strings.TrimSpace(os.Getenv("LAUNCHPAD_MODELS_URL")); value != "" {
		settings.ModelsURL = value
	}
	if settings.ProviderURL == "" {
		settings.ProviderURL = DefaultProviderURL
	}
	profile, err := settings.Profile()
	if err != nil {
		return Settings{}, err
	}
	settings.ProviderKind = profile.Kind
	settings.ProviderURL = profile.BaseURL
	settings.ModelsURL = profile.ModelsURL
	return settings, nil
}

func Save(settings Settings) error {
	profile, err := settings.Profile()
	if err != nil {
		return err
	}
	settings.ProviderKind = profile.Kind
	settings.ProviderURL = profile.BaseURL
	settings.ModelsURL = profile.ModelsURL
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
	profile, err := provider.NewProfile(provider.KindLiteLLM, value, "")
	return profile.BaseURL, err
}

func (s Settings) Profile() (provider.Profile, error) {
	return provider.NewProfile(s.ProviderKind, s.ProviderURL, s.ModelsURL)
}

func CLIName() string {
	return DefaultCLIName
}
