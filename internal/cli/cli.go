package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"launchpad/internal/config"
	"launchpad/internal/credentials"
	"launchpad/internal/launch"
	"launchpad/internal/picker"
	"launchpad/internal/provider"
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
	name := config.CLIName()
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

	model, providerURL, restore, yes, passthrough, err := parseLaunchArgs(args[1:])
	if err != nil {
		return err
	}
	if providerURL != "" {
		settings.ProviderURL, err = config.NormalizeProviderURL(providerURL)
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
		client := provider.NewClient(settings.ProviderURL, token)
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
		return runChatGPT(ctx, name, executable, settings.ProviderURL, model, yes, streams)
	}
	runner := launch.Runner{
		ProviderURL: settings.ProviderURL,
		APIKey:      token,
		Executable:  executable,
		Stdin:       streams.In,
		Stdout:      streams.Out,
		Stderr:      streams.Err,
	}
	return runner.Run(ctx, integration, model, passthrough)
}

func parseLaunchArgs(args []string) (model, providerURL string, restore, yes bool, passthrough []string, err error) {
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
		case arg == "--provider-url":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				err = errors.New("--provider-url requires a value")
				return
			}
			i++
			providerURL = args[i]
		case strings.HasPrefix(arg, "--provider-url="):
			providerURL = strings.TrimPrefix(arg, "--provider-url=")
			if providerURL == "" {
				err = errors.New("--provider-url requires a value")
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

func selectModel(streams IO, integration string, models []provider.Model) (string, error) {
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

func runChatGPT(ctx context.Context, cliName, executable, providerURL, model string, yes bool, streams IO) error {
	if err := configureChatGPT(providerURL, model, executable); err != nil {
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
		fmt.Fprintf(streams.Out, "providerUrl: %s\ncliName: %s\napiKeyConfigured: %t\n",
			settings.ProviderURL, config.CLIName(), keyErr == nil)
		return nil
	}
	switch args[0] {
	case "set-provider":
		if len(args) != 2 {
			return fmt.Errorf("usage: %s config set-provider <URL>", name)
		}
		settings.ProviderURL = args[1]
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
	fmt.Fprintf(w, `%s — launch coding agents through a LiteLLM provider

Usage:
  %s launch <claude|codex|chatgpt|opencode|copilot> [--model MODEL] [--provider-url URL] [--yes] [-- args]
  %s launch chatgpt --restore [--yes]
  %s config show
  %s config set-provider <URL>

Environment:
  LITELLM_API_KEY  Provider key (preferred over Keychain)
  LITELLM_BASE_URL Provider URL override
`, name, name, name, name, name)
}
