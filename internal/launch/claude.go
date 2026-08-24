package launch

import (
	"encoding/json"
)

func ClaudeLaunchArgs(args []string, model string) []string {
	args = EnsureModelArg(args, model)
	return append(args, "--setting-sources", "project,local")
}

func BuildOpenCodeConfig(baseURL, token, model string) (string, error) {
	config := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]any{
			"harnezpad": map[string]any{
				"npm":     "@ai-sdk/openai-compatible",
				"name":    "HarnezPad",
				"options": map[string]any{"baseURL": baseURL, "apiKey": token},
				"models":  map[string]any{model: map[string]any{"name": model}},
			},
		},
		"model": "harnezpad/" + model,
	}
	data, err := json.Marshal(config)
	return string(data), err
}
