package launch

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTerminalLaunchAdapters(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for _, test := range []struct {
		name string
		want []string
	}{
		{"codex", []string{`model_provider="launchpad"`, "https://gateway.example/v1", "OPENAI_API_KEY=secret"}},
		{"opencode", []string{"OPENCODE_CONFIG_CONTENT=", `"baseURL":"https://gateway.example/v1"`, `"model":"launchpad/model-a"`}},
		{"copilot", []string{"COPILOT_PROVIDER_BASE_URL=https://gateway.example/v1", "COPILOT_PROVIDER_API_KEY=secret", "COPILOT_MODEL=model-a"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, "output")
			script := "#!/bin/sh\nprintf 'args:%s\\n' \"$*\" > \"$LAUNCHPAD_TEST_OUTPUT\"\nenv >> \"$LAUNCHPAD_TEST_OUTPUT\"\n"
			if err := os.WriteFile(filepath.Join(dir, test.name), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("LAUNCHPAD_TEST_OUTPUT", output)
			t.Setenv("LITELLM_API_KEY", "must-not-leak")
			t.Setenv("ANTHROPIC_AUTH_TOKEN", "must-not-leak")
			runner := Runner{GatewayURL: "https://gateway.example", APIKey: "secret"}
			if err := runner.Run(context.Background(), test.name, "model-a", []string{"--verbose"}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !strings.Contains(string(data), want) {
					t.Fatalf("output missing %q:\n%s", want, data)
				}
			}
			for _, forbidden := range []string{"LITELLM_API_KEY=must-not-leak", "ANTHROPIC_AUTH_TOKEN=must-not-leak"} {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("output leaked %q:\n%s", forbidden, data)
				}
			}
		})
	}
}
