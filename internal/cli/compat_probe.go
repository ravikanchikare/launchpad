package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"harnezpad/internal/app"
	"harnezpad/internal/gateway"
	"harnezpad/internal/launch"
)

type compatProbeResult struct {
	Agent      string `json:"agent"`
	Model      string `json:"model"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

func runCompatProbe(m *app.Manager) bool {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: harnezpad _compat-probe <claude|codex|opencode> --model <id> [--prompt \"...\"] [--key KEY]")
		os.Exit(2)
	}
	agent := os.Args[2]
	id := map[string]string{"claude": "claude", "codex": "codex-cli", "opencode": "opencode"}[agent]
	if id == "" {
		fmt.Fprintf(os.Stderr, "unknown agent %q\n", agent)
		os.Exit(2)
	}

	flags := flag.NewFlagSet("_compat-probe", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	model := flags.String("model", "", "gateway model id")
	prompt := flags.String("prompt", "Reply with exactly COMPAT_OK and nothing else.", "probe prompt")
	keySlug := flags.String("key", "", "launch key slug")
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}
	modelID := strings.TrimSpace(*model)
	if modelID == "" {
		fmt.Fprintln(os.Stderr, "harnezpad: --model is required")
		os.Exit(2)
	}

	if dir := strings.TrimSpace(os.Getenv("HARNEZPAD_COMPAT_CODEX_HOME")); dir != "" {
		_ = os.Setenv("CODEX_HOME", dir)
	}

	result := compatProbeResult{Agent: agent, Model: modelID}

	launchToken, err := resolveCompatToken(m, *keySlug)
	if err != nil {
		result.Error = err.Error()
		writeCompatResult(result)
		os.Exit(1)
	}

	args, err := compatProbeArgs(m, id, modelID, *prompt, *keySlug)
	if err != nil {
		result.Error = err.Error()
		writeCompatResult(result)
		os.Exit(1)
	}

	cmd, err := m.CommandWithArgs(id, args)
	if err != nil {
		result.Error = err.Error()
		writeCompatResult(result)
		os.Exit(1)
	}
	if err := m.ApplyEnvironmentForModel(cmd, id, modelID, launchToken); err != nil {
		result.Error = err.Error()
		writeCompatResult(result)
		os.Exit(1)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	runErr := cmd.Run()
	result.DurationMs = time.Since(start).Milliseconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = runErr.Error()
		}
	}
	writeCompatResult(result)
	if result.ExitCode != 0 || result.Error != "" {
		os.Exit(1)
	}
	return true
}

func compatProbeArgs(m *app.Manager, id, modelID, prompt, keySlug string) ([]string, error) {
	switch id {
	case "codex-cli":
		models, discoverErr := m.ListModelsForKey(context.Background(), keySlug)
		if discoverErr != nil {
			models = []gateway.Model{{ID: modelID}}
		}
		if err := launch.Configure(m, modelID, models); err != nil {
			return nil, err
		}
		codexArgs := []string{
			"exec",
			"-s", "read-only",
			"--dangerously-bypass-approvals-and-sandbox",
			prompt,
		}
		return launch.LaunchArgs(m, codexArgs, modelID)
	case "claude":
		return launch.ClaudeLaunchArgs([]string{
			"-p", prompt,
			"--disallowed-tools", "Bash,Edit,Write",
		}, modelID), nil
	case "opencode":
		return []string{
			"run",
			"--pure",
			"-m", "harnezpad/" + modelID,
			prompt,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported agent integration %q", id)
	}
}

func resolveCompatToken(m *app.Manager, keySlug string) (string, error) {
	if token := strings.TrimSpace(os.Getenv("HARNEZPAD_COMPAT_TOKEN")); token != "" {
		return token, nil
	}
	return m.ResolveLaunchToken(keySlug)
}

func writeCompatResult(result compatProbeResult) {
	data, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "harnezpad: marshal compat probe result: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
