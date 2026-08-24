package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"harnezpad/internal/app"
	"harnezpad/internal/gateway"
	"harnezpad/internal/launch"
	"harnezpad/internal/picker"
	"harnezpad/internal/version"
)

// HelpText is the `harnezpad help` output.
const HelpText = "HarnezPad\n\nCommands:\n  harnezpad launch claude [--key KEY] [--model MODEL]\n  harnezpad launch codex [--key KEY] [--model MODEL] [--restore]\n  harnezpad launch chatgpt [--key KEY] [--model MODEL] [--restore]\n  harnezpad launch opencode [--key KEY] [--model MODEL]"

func Run() bool {
	if len(os.Args) < 2 {
		return false
	}
	if os.Args[1] == "serve-native" {
		flags := flag.NewFlagSet("serve-native", flag.ContinueOnError)
		flags.SetOutput(os.Stderr)
		parentPID := flags.Int("parent-pid", 0, "PID of the Native SDK host")
		if err := flags.Parse(os.Args[2:]); err != nil || *parentPID <= 1 || flags.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "usage: harnezpad serve-native --parent-pid <pid>")
			os.Exit(2)
		}
		if err := app.RunNativeHelper(*parentPID); err != nil {
			fmt.Fprintln(os.Stderr, "harnezpad:", err)
			os.Exit(1)
		}
		return true
	}
	m := app.NewManager()
	m.Load()
	switch os.Args[1] {
	case "_chatgpt-token":
		token, err := m.ResolveLaunchToken("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "harnezpad:", err)
			os.Exit(1)
		}
		fmt.Println(token)
		return true
	case "_compat-probe":
		return runCompatProbe(m)
	case "launch":
		return runLaunch(m)
	case "version", "--version", "-v":
		fmt.Println("harnezpad " + version.Version)
		return true
	case "help", "--help", "-h":
		fmt.Println(HelpText)
		return true
	default:
		fmt.Fprintf(os.Stderr, "harnezpad: unknown command %q\n", os.Args[1])
		os.Exit(2)
		return true
	}
}

func reportCLIError(err error) {
	fmt.Fprintln(os.Stderr, "harnezpad:", app.UserFacingError(err))
	os.Exit(1)
}

func runLaunch(m *app.Manager) bool {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: harnezpad launch <claude|codex|chatgpt|opencode> [args...]")
		os.Exit(2)
	}
	id := map[string]string{"claude": "claude", "codex": "codex-cli", "chatgpt": "codex-desktop", "opencode": "opencode"}[os.Args[2]]
	if id == "" {
		fmt.Fprintf(os.Stderr, "unknown launcher %q\n", os.Args[2])
		os.Exit(2)
	}
	launchArgs := os.Args[3:]
	model, launchArgs, err := launch.ParseModelFlag(launchArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harnezpad:", err)
		os.Exit(2)
	}
	keySlug, launchArgs, err := launch.ParseKeyFlag(launchArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harnezpad:", err)
		os.Exit(2)
	}
	if launch.HasRestoreFlag(launchArgs) {
		if model != "" || keySlug != "" {
			fmt.Fprintln(os.Stderr, "harnezpad: --restore cannot be combined with --model or --key")
			os.Exit(2)
		}
		if len(launch.StripRestoreFlag(launchArgs)) != 0 {
			fmt.Fprintln(os.Stderr, "harnezpad: --restore cannot be combined with application arguments")
			os.Exit(2)
		}
		if err := RestoreIntegration(m, id); err != nil {
			reportCLIError(err)
		}
		return true
	}
	if _, err := m.ResolveLaunchToken(keySlug); err != nil {
		reportCLIError(err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model, err = selectModel(m, id, keySlug)
		if err != nil {
			reportCLIError(err)
		}
	}
	if model == "" {
		fmt.Fprintln(os.Stderr, "harnezpad: no model selected")
		os.Exit(1)
	}
	launchArgs = launch.StripRestoreFlag(launch.StripKeyFlag(launch.StripModelFlag(launchArgs)))
	if id == "codex-desktop" {
		models, discoverErr := m.ListModelsForKey(context.Background(), keySlug)
		if discoverErr != nil {
			models = []gateway.Model{{ID: model}}
		}
		if err := launch.ConfigureChatGPT(m, model, models); err != nil {
			reportCLIError(err)
		}
		fmt.Fprintln(os.Stderr, "ChatGPT profile changed to HarnezPad.")
		fmt.Fprintln(os.Stderr, "To restore your usual ChatGPT profile, run: harnezpad launch chatgpt --restore")
		restart, confirmErr := picker.ConfirmChatGPTRestart()
		if confirmErr != nil {
			reportCLIError(confirmErr)
		}
		if !restart {
			fmt.Fprintln(os.Stderr, "Quit and reopen ChatGPT when you're ready for the profile change to take effect.")
			return true
		}
	} else if id == "codex-cli" {
		models, discoverErr := m.ListModelsForKey(context.Background(), keySlug)
		if discoverErr != nil {
			models = []gateway.Model{{ID: model}}
		}
		if err := launch.Configure(m, model, models); err != nil {
			reportCLIError(err)
		}
		launchArgs, err = launch.LaunchArgs(m, launchArgs, model)
		if err != nil {
			reportCLIError(err)
		}
	} else if id == "opencode" {
		// args already stripped
	} else if id == "claude" {
		launchArgs = launch.ClaudeLaunchArgs(launchArgs, model)
	} else {
		launchArgs = launch.EnsureModelArg(launchArgs, model)
	}
	if err := m.RunForeground(id, model, launchArgs, keySlug); err != nil {
		reportCLIError(err)
	}
	return true
}

func selectModel(m *app.Manager, id, keySlug string) (string, error) {
	models, err := m.ListModelsForKey(context.Background(), keySlug)
	if err != nil {
		return "", fmt.Errorf("unable to discover models for %s: %w", id, err)
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no models are available for your management key")
	}
	return picker.Run("Select model for "+app.IntegrationDisplayName(id)+":", models)
}
