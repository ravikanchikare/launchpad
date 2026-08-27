package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"launchpad/internal/config"
	"launchpad/internal/credentials"
	"launchpad/internal/gateway"
	"launchpad/internal/launch"
	"launchpad/internal/picker"
)

const Version = "0.3.0"

var (
	configureChatGPT = launch.ConfigureChatGPT
	restoreChatGPT   = launch.RestoreChatGPT
	chatGPTIsRunning = launch.ChatGPTIsRunning
	launchChatGPT    = launch.LaunchChatGPT
)

type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

func Run(ctx context.Context, executable string, args []string, streams IO) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}
	settings, err := config.Load()
	if err != nil {
		fmt.Fprintln(streams.Err, "launchpad:", err)
		return true, 1
	}
	name := config.CLIName(settings, executable)
	switch args[0] {
	case "launch":
		err = runLaunch(ctx, name, executable, settings, args[1:], streams)
	case "config":
		err = runConfig(name, settings, args[1:], streams)
	case "_chatgpt-token":
		var token string
		token, err = credentials.Resolve()
		if err == nil {
			fmt.Fprintln(streams.Out, token)
		}
	case "version", "--version", "-v":
		fmt.Fprintf(streams.Out, "%s %s\n", name, Version)
	case "help", "--help", "-h":
		printHelp(streams.Out, name)
	default:
		fmt.Fprintf(streams.Err, "%s: unknown command %q\n", name, args[0])
		return true, 2
	}
	if err != nil {
		fmt.Fprintf(streams.Err, "%s: %v\n", name, err)
		return true, 1
	}
	return true, 0
}

func runLaunch(ctx context.Context, name, executable string, settings config.Settings, args []string, streams IO) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s launch <claude|codex|chatgpt|opencode|copilot> [--model MODEL]", name)
	}
	integration := strings.ToLower(args[0])
	switch integration {
	case "claude", "codex", "chatgpt", "opencode", "copilot":
	default:
		return fmt.Errorf("unknown launcher %q", integration)
	}

	model, gatewayURL, restore, yes, passthrough, err := parseLaunchArgs(args[1:])
	if err != nil {
		return err
	}
	if gatewayURL != "" {
		settings.GatewayURL, err = config.NormalizeGatewayURL(gatewayURL)
		if err != nil {
			return err
		}
	}
	if restore {
		if integration != "chatgpt" {
			return errors.New("--restore is only supported for ChatGPT")
		}
		if model != "" || len(passthrough) != 0 {
			return errors.New("--restore cannot be combined with --model or application arguments")
		}
		confirmed, err := confirm(
			"Restore your usual ChatGPT profile?",
			picker.ConfirmOptions{YesLabel: "Restore", NoLabel: "Cancel", Default: picker.ConfirmDefaultNo},
			yes, streams,
		)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
		wasRunning := chatGPTIsRunning(ctx)
		if err := restoreChatGPT(); err != nil {
			return err
		}
		fmt.Fprintln(streams.Err, "ChatGPT restored to your usual profile.")
		if wasRunning {
			fmt.Fprintln(streams.Err, "Restarting ChatGPT with the restored profile…")
			if err := launchChatGPT(ctx); err != nil {
				return fmt.Errorf("profile restored, but ChatGPT could not restart: %w", err)
			}
			fmt.Fprintln(streams.Err, "ChatGPT restarted.")
		}
		return nil
	}
	token, err := credentials.Resolve()
	if err != nil {
		return err
	}
	if model == "" {
		client := gateway.NewClient(settings.GatewayURL, token)
		models, discoverErr := client.ListModels(ctx)
		if discoverErr != nil {
			return discoverErr
		}
		model, err = selectModel(streams, integration, models)
		if err != nil {
			return err
		}
	}
	if integration == "chatgpt" {
		if err := credentials.PersistForDesktop(token); err != nil {
			return fmt.Errorf("store the management key for ChatGPT's token helper: %w", err)
		}
		return runChatGPT(ctx, name, executable, settings.GatewayURL, model, yes, streams)
	}
	runner := launch.Runner{
		GatewayURL: settings.GatewayURL,
		APIKey:     token,
		Executable: executable,
		Stdin:      streams.In,
		Stdout:     streams.Out,
		Stderr:     streams.Err,
	}
	return runner.Run(ctx, integration, model, passthrough)
}

func parseLaunchArgs(args []string) (model, gatewayURL string, restore, yes bool, passthrough []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			passthrough = append(passthrough, args[i+1:]...)
			return
		}
		switch {
		case arg == "--model":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				err = errors.New("--model requires a value")
				return
			}
			i++
			model = args[i]
		case strings.HasPrefix(arg, "--model="):
			model = strings.TrimPrefix(arg, "--model=")
			if model == "" {
				err = errors.New("--model requires a value")
				return
			}
		case arg == "--gateway-url":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				err = errors.New("--gateway-url requires a value")
				return
			}
			i++
			gatewayURL = args[i]
		case strings.HasPrefix(arg, "--gateway-url="):
			gatewayURL = strings.TrimPrefix(arg, "--gateway-url=")
			if gatewayURL == "" {
				err = errors.New("--gateway-url requires a value")
				return
			}
		case arg == "--restore":
			restore = true
		case arg == "--yes" || arg == "-y":
			yes = true
		default:
			passthrough = append(passthrough, arg)
		}
	}
	return
}

func selectModel(streams IO, integration string, models []gateway.Model) (string, error) {
	items := make([]picker.Item, 0, len(models))
	for _, model := range models {
		description := strings.Join(model.Providers, ", ")
		if model.HealthStatus != "" {
			if description != "" {
				description += " • "
			}
			description += model.HealthStatus
		}
		items = append(items, picker.Item{Name: model.ID, Description: description})
	}
	displayName := map[string]string{
		"claude": "Claude Code", "codex": "Codex", "chatgpt": "ChatGPT",
		"opencode": "OpenCode", "copilot": "Copilot CLI",
	}[integration]
	return picker.Select("Select a model for "+displayName, items, streams.In, streams.Err)
}

func confirm(prompt string, options picker.ConfirmOptions, yes bool, streams IO) (bool, error) {
	if yes {
		return true, nil
	}
	return picker.Confirm(prompt, options, streams.In, streams.Err)
}

func runChatGPT(ctx context.Context, cliName, executable, gatewayURL, model string, yes bool, streams IO) error {
	if err := configureChatGPT(gatewayURL, model, executable); err != nil {
		return err
	}
	fmt.Fprintln(streams.Err, "ChatGPT profile changed to Launchpad.")
	fmt.Fprintln(streams.Err, "To restore your usual ChatGPT profile, run: "+
		cliName+" launch chatgpt --restore")

	running := chatGPTIsRunning(ctx)
	if running {
		restart, err := confirm(
			"Restart ChatGPT to use Launchpad?",
			picker.ConfirmOptions{YesLabel: "Restart now", NoLabel: "Later", Default: picker.ConfirmDefaultYes},
			yes, streams,
		)
		if err != nil {
			return err
		}
		if !restart {
			fmt.Fprintln(streams.Err, "Quit and reopen ChatGPT when you're ready for the profile change to take effect.")
			return nil
		}
	}
	if running {
		fmt.Fprintln(streams.Err, "Restarting ChatGPT…")
	} else {
		fmt.Fprintln(streams.Err, "Opening ChatGPT…")
	}
	if err := launchChatGPT(ctx); err != nil {
		return err
	}
	if running {
		fmt.Fprintln(streams.Err, "ChatGPT restarted with Launchpad.")
	}
	return nil
}

func runConfig(name string, settings config.Settings, args []string, streams IO) error {
	if len(args) == 0 || args[0] == "show" {
		_, keyErr := credentials.Resolve()
		fmt.Fprintf(streams.Out, "gatewayUrl: %s\ncliName: %s\napiKeyConfigured: %t\n",
			settings.GatewayURL, config.CLIName(settings, name), keyErr == nil)
		return nil
	}
	switch args[0] {
	case "set-gateway":
		if len(args) != 2 {
			return fmt.Errorf("usage: %s config set-gateway <URL>", name)
		}
		settings.GatewayURL = args[1]
	case "set-cli-name":
		if len(args) != 2 {
			return fmt.Errorf("usage: %s config set-cli-name <NAME>", name)
		}
		settings.CLIName = args[1]
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
	if err := config.Save(settings); err != nil {
		return err
	}
	fmt.Fprintln(streams.Out, "Configuration saved.")
	return nil
}

func printHelp(w io.Writer, name string) {
	fmt.Fprintf(w, `%s — launch coding agents through a LiteLLM gateway

Usage:
  %s launch <claude|codex|chatgpt|opencode|copilot> [--model MODEL] [--gateway-url URL] [--yes] [-- args]
  %s launch chatgpt --restore [--yes]
  %s config show
  %s config set-gateway <URL>
  %s config set-cli-name <NAME>

Environment:
  LITELLM_API_KEY    Gateway key (preferred over Keychain)
  LITELLM_BASE_URL   Gateway URL override
  LAUNCHPAD_CLI_NAME Display name override
`, name, name, name, name, name, name)
}
