package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"harnezpad/internal/gateway"
)

func TestAccountHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user_info": map[string]any{"teams": []string{"team-a"}},
			})
		case "/team/team-a/members/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"team_id":    "team-a",
				"team_alias": "Team A",
				"role":       "user",
				"spend":      12.5,
				"litellm_budget_table": map[string]any{
					"max_budget": 100.0,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "token"}}
	req := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		summary, err := m.AccountSummary(r.Context())
		if err != nil {
			writeGatewayError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(summary)
	})
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var summary gateway.AccountSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.TeamID != "team-a" || summary.Spend != 12.5 || summary.MaxBudget != 100 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestAccountHandlerMissingKey(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "http://example"}}
	req := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/account", func(w http.ResponseWriter, r *http.Request) {
		summary, err := m.AccountSummary(r.Context())
		if err != nil {
			writeGatewayError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(summary)
	})
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestModelCatalogHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model_group/info" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"model_group": "alpha",
					"providers":   []string{"openai"},
				},
			},
		})
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "token"}}
	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/models/catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		entries, err := m.ListModelCatalog(r.Context())
		if err != nil {
			writeGatewayError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(entries)
	})
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"alpha"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestManagerAccountSummaryRequiresToken(t *testing.T) {
	m := &Manager{}
	_, err := m.AccountSummary(context.Background())
	if err == nil || !strings.Contains(err.Error(), "management key") {
		t.Fatalf("err = %v", err)
	}
}
