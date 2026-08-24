package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user_id": "user@example.com",
				"user_info": map[string]any{
					"teams":      []string{"example-team"},
					"user_email": "user@example.com",
				},
			})
		case "/team/example-team/members/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user_id":    "user@example.com",
				"team_id":    "example-team",
				"team_alias": "example-team",
				"role":       "user",
				"user_email": "user@example.com",
				"spend":      45.81,
				"litellm_budget_table": map[string]any{
					"max_budget":      500.0,
					"rpm_limit":       120,
					"budget_reset_at": "2026-08-01T00:00:00Z",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	summary, err := client.AccountSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.TeamID != "example-team" || summary.Role != "user" {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Spend != 45.81 || summary.MaxBudget != 500 {
		t.Fatalf("budget fields = %+v", summary)
	}
	if summary.RPMLimit == nil || *summary.RPMLimit != 120 {
		t.Fatalf("rpm = %+v", summary.RPMLimit)
	}
	if summary.BudgetResetAt != "2026-08-01T00:00:00Z" {
		t.Fatalf("reset = %q", summary.BudgetResetAt)
	}
}

func TestAccountSummaryNoTeams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user_id": "user@example.com",
			"user_info": map[string]any{
				"teams": []string{},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	_, err := client.AccountSummary(context.Background())
	if err == nil {
		t.Fatal("expected error for empty teams")
	}
}
