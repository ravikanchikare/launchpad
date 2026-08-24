package cli

import (
	"fmt"
	"os"

	"harnezpad/internal/app"
	"harnezpad/internal/launch"
)

func RestoreIntegration(m *app.Manager, id string) error {
	switch id {
	case "claude":
		fmt.Fprintln(os.Stderr, "Claude Code routing is process-scoped; no persistent HarnezPad configuration remains.")
		return nil
	case "codex-cli":
		if err := launch.RestoreCodex(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Codex restored to its usual profile.")
		return nil
	case "codex-desktop":
		if err := launch.ResetChatGPT(m); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "ChatGPT restored to its usual profile.")
		return nil
	default:
		return fmt.Errorf("--restore is not supported for %s", id)
	}
}
