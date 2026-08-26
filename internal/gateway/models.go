package gateway

import (
	"context"
	"net/http"
	"sort"
)

type ModelCatalogEntry struct {
	ID                 string   `json:"id"`
	Providers          []string `json:"providers,omitempty"`
	Mode               string   `json:"mode,omitempty"`
	MaxInputTokens     *float64 `json:"maxInputTokens,omitempty"`
	MaxOutputTokens    *float64 `json:"maxOutputTokens,omitempty"`
	InputCostPerToken  *float64 `json:"inputCostPerToken,omitempty"`
	OutputCostPerToken *float64 `json:"outputCostPerToken,omitempty"`
	SupportsVision     bool     `json:"supportsVision"`
	SupportsTools      bool     `json:"supportsTools"`
	SupportsReasoning  bool     `json:"supportsReasoning"`
	SupportsWebSearch  bool     `json:"supportsWebSearch"`
	HealthStatus       string   `json:"healthStatus,omitempty"`
}

type modelGroupInfoResponse struct {
	Data []rawModelGroup `json:"data"`
}

type rawModelGroup struct {
	ModelGroup              string   `json:"model_group"`
	Providers               []string `json:"providers"`
	Mode                    *string  `json:"mode"`
	MaxInputTokens          *float64 `json:"max_input_tokens"`
	MaxOutputTokens         *float64 `json:"max_output_tokens"`
	InputCostPerToken       *float64 `json:"input_cost_per_token"`
	OutputCostPerToken      *float64 `json:"output_cost_per_token"`
	SupportsVision          bool     `json:"supports_vision"`
	SupportsFunctionCalling bool     `json:"supports_function_calling"`
	SupportsReasoning       bool     `json:"supports_reasoning"`
	SupportsWebSearch       bool     `json:"supports_web_search"`
	HealthStatus            *string  `json:"health_status"`
}

func (c *Client) ListModelGroups(ctx context.Context) ([]ModelCatalogEntry, error) {
	var raw modelGroupInfoResponse
	if err := c.doJSON(ctx, http.MethodGet, "/model_group/info", nil, &raw); err != nil {
		return nil, err
	}
	out := make([]ModelCatalogEntry, 0, len(raw.Data))
	for _, group := range raw.Data {
		entry := ModelCatalogEntry{
			ID:                 group.ModelGroup,
			Providers:          append([]string(nil), group.Providers...),
			MaxInputTokens:     group.MaxInputTokens,
			MaxOutputTokens:    group.MaxOutputTokens,
			InputCostPerToken:  group.InputCostPerToken,
			OutputCostPerToken: group.OutputCostPerToken,
			SupportsVision:     group.SupportsVision,
			SupportsTools:      group.SupportsFunctionCalling,
			SupportsReasoning:  group.SupportsReasoning,
			SupportsWebSearch:  group.SupportsWebSearch,
		}
		if group.Mode != nil {
			entry.Mode = *group.Mode
		}
		if group.HealthStatus != nil {
			entry.HealthStatus = *group.HealthStatus
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}
