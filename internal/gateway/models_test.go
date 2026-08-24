package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModelGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model_group/info" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"model_group":              "gpt-5.5",
					"providers":                []string{"openai"},
					"mode":                     "chat",
					"max_input_tokens":         1050000.0,
					"max_output_tokens":        128000.0,
					"input_cost_per_token":     0.000005,
					"output_cost_per_token":    0.00003,
					"supports_vision":          true,
					"supports_function_calling": true,
					"supports_reasoning":       true,
					"supports_web_search":      true,
				},
				map[string]any{
					"model_group": "kimi-k3",
					"providers":   []string{"baseten"},
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	entries, err := client.ListModelGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d", len(entries))
	}
	if entries[0].ID != "gpt-5.5" || entries[1].ID != "kimi-k3" {
		t.Fatalf("sorted ids = %q, %q", entries[0].ID, entries[1].ID)
	}
	gpt := entries[0]
	if gpt.Mode != "chat" || !gpt.SupportsVision || !gpt.SupportsTools || !gpt.SupportsWebSearch {
		t.Fatalf("gpt entry = %+v", gpt)
	}
	if gpt.InputCostPerToken == nil || *gpt.InputCostPerToken != 0.000005 {
		t.Fatalf("input cost = %+v", gpt.InputCostPerToken)
	}
}
