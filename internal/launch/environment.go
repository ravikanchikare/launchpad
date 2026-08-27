package launch

import (
	"os"
	"strings"
)

var ClaudeProviderEnvironment = []string{
	"AWS_PROFILE",
	"AWS_DEFAULT_PROFILE",
	"AWS_REGION",
	"AWS_DEFAULT_REGION",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",
	"AWS_CONFIG_FILE",
	"AWS_SHARED_CREDENTIALS_FILE",
	"AWS_SDK_LOAD_CONFIG",
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_MANTLE",
	"CLAUDE_CODE_USE_VERTEX",
	"ANTHROPIC_BEDROCK_BASE_URL",
	"ANTHROPIC_BEDROCK_MANTLE_BASE_URL",
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"CLAUDE_CODE_SUBAGENT_MODEL",
}

func IsolatedChildEnvironment(overrides map[string]string, unset []string) []string {
	removed := make(map[string]struct{}, len(unset)+len(overrides))
	for _, key := range unset {
		removed[key] = struct{}{}
	}
	for key := range overrides {
		removed[key] = struct{}{}
	}
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, skip := removed[key]; skip {
				continue
			}
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}
