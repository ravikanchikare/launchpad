package provider

import "testing"

func TestProfileResolvesPrefixedProviderEndpoints(t *testing.T) {
	profile, err := NewProfile(KindLiteLLM, "https://provider.example/team/v1/", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.OpenAIBaseURL(); got != "https://provider.example/team/v1" {
		t.Fatalf("OpenAIBaseURL() = %q", got)
	}
	if got := profile.AnthropicBaseURL(); got != "https://provider.example/team" {
		t.Fatalf("AnthropicBaseURL() = %q", got)
	}
	if got := profile.OpenAIModelsURL(); got != "https://provider.example/team/v1/models" {
		t.Fatalf("OpenAIModelsURL() = %q", got)
	}
	if got := profile.LiteLLMModelsURL(); got != "https://provider.example/team/model_group/info" {
		t.Fatalf("LiteLLMModelsURL() = %q", got)
	}
	if got := profile.AnthropicURL("/v1/messages"); got != "https://provider.example/team/v1/messages" {
		t.Fatalf("AnthropicURL() = %q", got)
	}
}

func TestProfileUsesModelsURLOverride(t *testing.T) {
	profile, err := NewProfile(
		KindOpenAICompatible,
		"https://provider.example/v1",
		"https://catalog.example/custom/models/",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.OpenAIModelsURL(); got != "https://catalog.example/custom/models" {
		t.Fatalf("OpenAIModelsURL() = %q", got)
	}
}

func TestProfileRejectsUnknownKind(t *testing.T) {
	if _, err := NewProfile("unknown", "https://provider.example", ""); err == nil {
		t.Fatal("NewProfile accepted an unknown provider kind")
	}
}
