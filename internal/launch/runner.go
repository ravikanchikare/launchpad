package launch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct {
	GatewayURL string
	APIKey     string
	Executable string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

func (r Runner) Run(ctx context.Context, integration, model string, args []string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("model cannot be empty")
	}
	cmd, err := r.command(ctx, integration, model, args)
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r.Stdin, r.Stdout, r.Stderr
	if integration == "chatgpt" {
		return cmd.Start()
	}
	return cmd.Run()
}

func (r Runner) command(ctx context.Context, integration, model string, args []string) (*exec.Cmd, error) {
	gatewayRoot := strings.TrimRight(r.GatewayURL, "/")
	openAIBase := gatewayRoot + "/v1"
	overrides := map[string]string{"LAUNCHPAD_MODEL": model}
	unset := append([]string{"LITELLM_API_KEY", "LITELLM_BASE_URL", "LAUNCHPAD_CLI_NAME"}, ClaudeProviderEnvironment...)
	var cmd *exec.Cmd

	switch integration {
	case "claude":
		path, err := lookupExecutable("claude")
		if err != nil {
			return nil, err
		}
		launchArgs := ensureModelArg(args, model)
		launchArgs = append(launchArgs, "--setting-sources", "project,local")
		cmd = exec.CommandContext(ctx, path, launchArgs...)
		overrides["ANTHROPIC_BASE_URL"] = gatewayRoot
		overrides["ANTHROPIC_API_KEY"] = ""
		overrides["ANTHROPIC_AUTH_TOKEN"] = r.APIKey
		overrides["ANTHROPIC_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
		overrides["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
		overrides["CLAUDE_CODE_SUBAGENT_MODEL"] = model
		overrides["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"] = "1"
	case "codex":
		path, err := lookupExecutable("codex")
		if err != nil {
			return nil, err
		}
		launchArgs := []string{
			"-c", `model_provider="launchpad"`,
			"-c", `model_providers.launchpad.name="Launchpad"`,
			"-c", fmt.Sprintf(`model_providers.launchpad.base_url=%q`, openAIBase),
			"-c", `model_providers.launchpad.env_key="OPENAI_API_KEY"`,
			"-c", `model_providers.launchpad.wire_api="responses"`,
			"-m", model,
		}
		cmd = exec.CommandContext(ctx, path, append(launchArgs, args...)...)
		overrides["OPENAI_API_KEY"] = r.APIKey
	case "chatgpt":
		if err := ConfigureChatGPT(gatewayRoot, model, r.Executable); err != nil {
			return nil, err
		}
		chatGPTCommand, err := ChatGPTLaunchCommand(ctx)
		if err != nil {
			return nil, err
		}
		cmd = chatGPTCommand
		overrides["OPENAI_API_KEY"] = r.APIKey
		unset = append(unset, "OPENAI_BASE_URL", "OPENAI_MODEL")
	case "opencode":
		path, err := lookupExecutable("opencode")
		if err != nil {
			return nil, err
		}
		config, err := buildOpenCodeConfig(openAIBase, r.APIKey, model)
		if err != nil {
			return nil, err
		}
		cmd = exec.CommandContext(ctx, path, args...)
		overrides["OPENAI_BASE_URL"] = openAIBase
		overrides["OPENAI_API_KEY"] = r.APIKey
		overrides["OPENCODE_CONFIG_CONTENT"] = config
	case "copilot":
		path, err := lookupExecutable("copilot")
		if err != nil {
			return nil, err
		}
		cmd = exec.CommandContext(ctx, path, args...)
		overrides["COPILOT_PROVIDER_BASE_URL"] = openAIBase
		overrides["COPILOT_PROVIDER_TYPE"] = "openai"
		overrides["COPILOT_PROVIDER_API_KEY"] = r.APIKey
		overrides["COPILOT_MODEL"] = model
	default:
		return nil, fmt.Errorf("unsupported launcher %q", integration)
	}
	cmd.Env = IsolatedChildEnvironment(overrides, unset)
	return cmd, nil
}

func lookupExecutable(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, ".opencode", "bin", name),
		filepath.Join(home, ".npm-global", "bin", name),
		filepath.Join("/opt/homebrew/bin", name),
		filepath.Join("/usr/local/bin", name),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s is not installed", name)
}

func ensureModelArg(args []string, model string) []string {
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) || strings.HasPrefix(arg, "--model=") {
			return args
		}
	}
	return append([]string{"--model", model}, args...)
}

func buildOpenCodeConfig(baseURL, token, model string) (string, error) {
	value := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]any{
			"launchpad": map[string]any{
				"npm":     "@ai-sdk/openai-compatible",
				"name":    "Launchpad",
				"options": map[string]any{"baseURL": baseURL, "apiKey": token},
				"models":  map[string]any{model: map[string]any{"name": model}},
			},
		},
		"model": "launchpad/" + model,
	}
	data, err := json.Marshal(value)
	return string(data), err
}
