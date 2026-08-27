package provider

import (
	"errors"
	"net/url"
	"strings"
)

type Kind string

const (
	KindLiteLLM          Kind = "litellm"
	KindOpenAICompatible Kind = "openai-compatible"
)

type Profile struct {
	Kind      Kind
	BaseURL   string
	ModelsURL string
}

func NewProfile(kind Kind, baseURL, modelsURL string) (Profile, error) {
	if kind == "" {
		kind = KindLiteLLM
	}
	if kind != KindLiteLLM && kind != KindOpenAICompatible {
		return Profile{}, errors.New("provider kind must be litellm or openai-compatible")
	}
	baseURL, err := normalizeURL(baseURL, "provider URL")
	if err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(modelsURL) != "" {
		modelsURL, err = normalizeURL(modelsURL, "models URL")
		if err != nil {
			return Profile{}, err
		}
	}
	return Profile{Kind: kind, BaseURL: baseURL, ModelsURL: modelsURL}, nil
}

func (p Profile) OpenAIBaseURL() string {
	parsed, _ := url.Parse(p.BaseURL)
	if !strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/v1") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	}
	return parsed.String()
}

func (p Profile) AnthropicBaseURL() string {
	parsed, _ := url.Parse(p.BaseURL)
	parsed.Path = strings.TrimSuffix(strings.TrimRight(parsed.Path, "/"), "/v1")
	return strings.TrimRight(parsed.String(), "/")
}

func (p Profile) OpenAIModelsURL() string {
	if p.ModelsURL != "" {
		return p.ModelsURL
	}
	return appendURLPath(p.OpenAIBaseURL(), "/models")
}

func (p Profile) LiteLLMModelsURL() string {
	return appendURLPath(p.AnthropicBaseURL(), "/model_group/info")
}

func (p Profile) AnthropicURL(requestPath string) string {
	return appendURLPath(p.AnthropicBaseURL(), requestPath)
}

func normalizeURL(value, name string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return "", errors.New(name + " must be an absolute URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", errors.New(name + " must use http or https")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New(name + " cannot contain a query or fragment")
	}
	return value, nil
}

func appendURLPath(baseURL, suffix string) string {
	parsed, _ := url.Parse(baseURL)
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(suffix, "/")
	return parsed.String()
}
