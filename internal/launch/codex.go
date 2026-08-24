package launch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"harnezpad/internal/fsutil"
	"harnezpad/internal/gateway"
)

const (
	CodexProfileName   = "harnezpad-launch"
	CodexProfileMarker = "# HARNEZPAD_MANAGED_CODEX_PROFILE"
	codexProfileName   = CodexProfileName
	codexProviderName  = "HarnezPad"
	codexProfileMarker = CodexProfileMarker
)

func codexHome() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("CODEX_HOME")); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

func codexProfilePath() (string, error) {
	dir, err := codexHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, codexProfileName+".config.toml"), nil
}

func codexCatalogPath() (string, error) {
	dir, err := codexHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "harnezpad-models.json"), nil
}

func Configure(settings GatewaySettings, model string, models []gateway.Model) error {
	profilePath, err := codexProfilePath()
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(profilePath); err == nil {
		if !strings.Contains(string(existing), codexProfileMarker) {
			return fmt.Errorf("refusing to overwrite user-owned Codex profile %s", profilePath)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	catalogPath, err := codexCatalogPath()
	if err != nil {
		return err
	}
	if len(models) == 0 {
		models = []gateway.Model{{ID: model}}
	}
	if err := writeModelCatalog(catalogPath, models, "HarnezPad gateway model"); err != nil {
		return err
	}
	base := BaseURL(settings)
	profile := codexProfileMarker + "\n" +
		"model = " + strconv.Quote(model) + "\n" +
		"model_provider = " + strconv.Quote(codexProfileName) + "\n" +
		"model_catalog_json = " + strconv.Quote(catalogPath) + "\n\n" +
		"[model_providers." + codexProfileName + "]\n" +
		"name = " + strconv.Quote(codexProviderName) + "\n" +
		"base_url = " + strconv.Quote(base) + "\n" +
		"env_key = \"OPENAI_API_KEY\"\n" +
		"wire_api = \"responses\"\n"
	return fsutil.WriteAtomic(profilePath, []byte(profile), 0600)
}

func BaseURL(settings GatewaySettings) string {
	return strings.TrimRight(settings.GatewayURL(), "/") + "/v1"
}

func ManagedConfigOverrides(settings GatewaySettings, modelCatalogPath string) []string {
	overrides := []string{
		fmt.Sprintf("model_provider=%q", codexProfileName),
		fmt.Sprintf("model_providers.%s.name=%q", codexProfileName, codexProviderName),
		fmt.Sprintf("model_providers.%s.base_url=%q", codexProfileName, BaseURL(settings)),
		fmt.Sprintf("model_providers.%s.env_key=%q", codexProfileName, "OPENAI_API_KEY"),
		fmt.Sprintf("model_providers.%s.wire_api=%q", codexProfileName, "responses"),
	}
	if strings.TrimSpace(modelCatalogPath) != "" {
		overrides = append(overrides, fmt.Sprintf("model_catalog_json=%q", modelCatalogPath))
	}
	return overrides
}

func RestoreCodex() error {
	profilePath, err := codexProfilePath()
	if err != nil {
		return err
	}
	if data, readErr := os.ReadFile(profilePath); readErr == nil {
		if !strings.Contains(string(data), codexProfileMarker) {
			return fmt.Errorf("refusing to remove user-owned Codex profile %s", profilePath)
		}
		if err := os.Remove(profilePath); err != nil {
			return err
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	catalogPath, err := codexCatalogPath()
	if err != nil {
		return err
	}
	if err := removeOwnedModelCatalog(catalogPath); err != nil {
		return err
	}
	return nil
}

func LaunchArgs(settings GatewaySettings, args []string, model string) ([]string, error) {
	catalogPath, err := codexCatalogPath()
	if err != nil {
		return nil, err
	}
	launchArgs := []string{"--profile", codexProfileName}
	for _, override := range ManagedConfigOverrides(settings, catalogPath) {
		launchArgs = append(launchArgs, "-c", override)
	}
	if model != "" {
		launchArgs = append(launchArgs, "-m", model)
	}
	return append(launchArgs, args...), nil
}

func ProfilePath() (string, error) {
	return codexProfilePath()
}

func CatalogPath() (string, error) {
	return codexCatalogPath()
}

func writeModelCatalog(path string, models []gateway.Model, description string) error {
	if existing, err := os.ReadFile(path); err == nil {
		if !ownedModelCatalog(existing) {
			return fmt.Errorf("refusing to overwrite user-owned model catalog %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	entries := make([]map[string]any, 0, len(models))
	baseInstructions := codexBaseInstructions()
	seen := map[string]bool{}
	for i, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		entries = append(entries, map[string]any{
			"slug":                             id,
			"display_name":                     id,
			"description":                      description,
			"default_reasoning_level":          nil,
			"supported_reasoning_levels":       []any{},
			"shell_type":                       "default",
			"visibility":                       "list",
			"supported_in_api":                 true,
			"priority":                         i,
			"additional_speed_tiers":           []any{},
			"availability_nux":                 nil,
			"upgrade":                          nil,
			"base_instructions":                baseInstructions,
			"model_messages":                   nil,
			"supports_reasoning_summaries":     false,
			"default_reasoning_summary":        "auto",
			"support_verbosity":                false,
			"default_verbosity":                nil,
			"apply_patch_tool_type":            nil,
			"web_search_tool_type":             "text",
			"truncation_policy":                map[string]any{"mode": "bytes", "limit": 10000},
			"supports_parallel_tool_calls":     false,
			"supports_image_detail_original":   false,
			"context_window":                   128000,
			"max_context_window":               128000,
			"auto_compact_token_limit":         nil,
			"effective_context_window_percent": 95,
			"experimental_supported_tools":     []any{},
			"input_modalities":                 []string{"text"},
			"supports_search_tool":             false,
		})
	}
	if len(entries) == 0 {
		return errors.New("model catalog cannot be empty")
	}
	data, err := json.Marshal(map[string]any{"harnezpad_managed": true, "models": entries})
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(path, data, 0600)
}

func codexBaseInstructions() string {
	dir, err := codexHome()
	if err == nil {
		var cached struct {
			Models []struct {
				BaseInstructions string `json:"base_instructions"`
			} `json:"models"`
		}
		if data, readErr := os.ReadFile(filepath.Join(dir, "models_cache.json")); readErr == nil && json.Unmarshal(data, &cached) == nil {
			for _, model := range cached.Models {
				if instructions := strings.TrimSpace(model.BaseInstructions); instructions != "" {
					return instructions
				}
			}
		}
	}
	return "You are Codex, a coding agent. You and the user share the same workspace and collaborate to achieve the user's goals."
}

func ownedModelCatalog(data []byte) bool {
	var catalog struct {
		HarnezPadManaged bool `json:"harnezpad_managed"`
		Models       []struct {
			Description string `json:"description"`
		} `json:"models"`
	}
	if json.Unmarshal(data, &catalog) != nil {
		return false
	}
	if catalog.HarnezPadManaged {
		return true
	}
	// Accept catalogs written by HarnezPad versions before the ownership field was
	// introduced so the first managed-profile launch can migrate them safely.
	if len(catalog.Models) == 0 {
		return false
	}
	for _, model := range catalog.Models {
		if model.Description != "HarnezPad gateway model" {
			return false
		}
	}
	return true
}

func removeOwnedModelCatalog(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !ownedModelCatalog(data) {
		return fmt.Errorf("refusing to remove user-owned model catalog %s", path)
	}
	return os.Remove(path)
}
