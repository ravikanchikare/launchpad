package launch

import (
	"encoding/json"
	"testing"
)

func TestParseModelFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"separate", []string{"--model", "kimi-k3"}, "kimi-k3"},
		{"equals", []string{"--model=claude-sonnet-4-5"}, "claude-sonnet-4-5"},
		{"absent", []string{"--verbose"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, forwarded, err := ParseModelFlag(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("model = %q, want %q", got, tc.want)
			}
			if len(forwarded) != len(tc.args) {
				t.Fatalf("forwarded args = %v, want %v", forwarded, tc.args)
			}
		})
	}
}

func TestBuildOpenCodeConfig(t *testing.T) {
	raw, err := BuildOpenCodeConfig("https://gateway.example/v1", "secret", "kimi-k3")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatal(err)
	}
	if config["model"] != "harnezpad/kimi-k3" {
		t.Fatalf("model = %v", config["model"])
	}
}

func TestEnsureModelArg(t *testing.T) {
	if got := EnsureModelArg([]string{"--verbose"}, "kimi-k3"); len(got) != 3 || got[0] != "--model" || got[1] != "kimi-k3" {
		t.Fatalf("EnsureModelArg default = %v", got)
	}
	args := []string{"--model", "existing", "--verbose"}
	if got := EnsureModelArg(args, "kimi-k3"); len(got) != len(args) || got[1] != "existing" {
		t.Fatalf("EnsureModelArg explicit = %v", got)
	}
}

func TestRestoreFlags(t *testing.T) {
	args := []string{"--model", "kimi-k3", "--restore", "--verbose"}
	if !HasRestoreFlag(args) {
		t.Fatal("expected restore flag")
	}
	if got := StripRestoreFlag(args); len(got) != 3 || got[2] != "--verbose" {
		t.Fatalf("StripRestoreFlag = %v", got)
	}
}

func TestParseKeyFlag(t *testing.T) {
	key, rest, err := ParseKeyFlag([]string{"--key", "harnezpad", "--verbose"})
	if err != nil || key != "harnezpad" || len(rest) != 1 || rest[0] != "--verbose" {
		t.Fatalf("ParseKeyFlag = %q %v err=%v", key, rest, err)
	}
}
