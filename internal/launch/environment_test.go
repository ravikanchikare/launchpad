package launch

import (
	"strings"
	"testing"
)

func TestIsolatedChildEnvironment(t *testing.T) {
	t.Setenv("HARNEZPAD_PARENT_ENV_TEST", "parent")
	env := IsolatedChildEnvironment(
		map[string]string{"HARNEZPAD_PARENT_ENV_TEST": "child", "HARNEZPAD_CHILD_ENV_TEST": "value"},
		[]string{"AWS_PROFILE"},
	)
	values := make(map[string]string)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	if values["HARNEZPAD_PARENT_ENV_TEST"] != "child" || values["HARNEZPAD_CHILD_ENV_TEST"] != "value" {
		t.Fatalf("overrides missing: %v", values)
	}
	if _, ok := values["AWS_PROFILE"]; ok {
		t.Fatalf("AWS_PROFILE should be removed from child environment")
	}
}

func TestClaudeLaunchArgsExcludeUserSettings(t *testing.T) {
	got := ClaudeLaunchArgs([]string{"--verbose"}, "model-a")
	joined := strings.Join(got, "|")
	if joined != "--model|model-a|--verbose|--setting-sources|project,local" {
		t.Fatalf("Claude args = %v", got)
	}
}
