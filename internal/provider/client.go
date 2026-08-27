package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type Model struct {
	ID                 string
	Providers          []string
	InputCostPerToken  *float64
	OutputCostPerToken *float64
	SupportsTools      bool
	HealthStatus       string
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	resp, body, err := c.get(ctx, "/model_group/info")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return c.listOpenAIModels(ctx)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("LiteLLM model discovery returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var raw struct {
		Data []struct {
			ModelGroup              string   `json:"model_group"`
			Providers               []string `json:"providers"`
			InputCostPerToken       *float64 `json:"input_cost_per_token"`
			OutputCostPerToken      *float64 `json:"output_cost_per_token"`
			SupportsFunctionCalling bool     `json:"supports_function_calling"`
			HealthStatus            *string  `json:"health_status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode LiteLLM model catalog: %w", err)
	}
	models := make([]Model, 0, len(raw.Data))
	for _, item := range raw.Data {
		id := strings.TrimSpace(item.ModelGroup)
		if id == "" {
			continue
		}
		model := Model{
			ID:                 id,
			Providers:          append([]string(nil), item.Providers...),
			InputCostPerToken:  item.InputCostPerToken,
			OutputCostPerToken: item.OutputCostPerToken,
			SupportsTools:      item.SupportsFunctionCalling,
		}
		if item.HealthStatus != nil {
			model.HealthStatus = *item.HealthStatus
		}
		models = append(models, model)
	}
	return sortedModels(models)
}

func (c *Client) listOpenAIModels(ctx context.Context) ([]Model, error) {
	resp, body, err := c.get(ctx, "/v1/models")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("OpenAI-compatible model discovery returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var raw struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode OpenAI-compatible model catalog: %w", err)
	}
	models := make([]Model, 0, len(raw.Data))
	for _, item := range raw.Data {
		if id := strings.TrimSpace(item.ID); id != "" {
			models = append(models, Model{ID: id})
		}
	}
	return sortedModels(models)
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to LiteLLM provider: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp, nil, err
	}
	return resp, body, nil
}

func sortedModels(models []Model) ([]Model, error) {
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	if len(models) == 0 {
		return nil, fmt.Errorf("provider returned no models")
	}
	return models, nil
}
