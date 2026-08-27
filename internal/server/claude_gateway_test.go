package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"launchpad/internal/gateway"
	"launchpad/internal/store"
)

func TestClaudeGatewayAdvertisesCompatibleRoutesAndRewritesModels(t *testing.T) {
	var forwardedModel string
	var forwardedAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/model_group/info":
			http.Error(w, "virtual key", http.StatusForbidden)
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":[{"id":"claude-opus-5"},{"id":"claude-sonnet-5"},{"id":"gpt-5.6-luna"},{"id":"gpt-5.6-sol"}]}`))
		case "/v1/messages":
			forwardedAuthorization = r.Header.Get("Authorization")
			var payload struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			forwardedModel = payload.Model
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"message","content":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LITELLM_BASE_URL", upstream.URL)
	t.Setenv("LITELLM_API_KEY", "upstream-secret")
	t.Setenv("LAUNCHPAD_DISABLE_KEYCHAIN", "1")
	srv := &Server{
		Store:       &store.Store{DBPath: home + "/db.sqlite"},
		GatewayHTTP: upstream.Client(),
	}
	handler := srv.Handler()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("models status=%d body=%s", response.Code, response.Body.String())
	}
	var catalog struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		id, displayName string
	}{
		{"claude-fable-5", "gpt-5.6-luna"},
		{"claude-opus-5", "claude-opus-5"},
		{"claude-sonnet-5", "claude-sonnet-5"},
		{"claude-haiku-4-5-20251001", "gpt-5.6-sol"},
	}
	if len(catalog.Data) != len(want) {
		t.Fatalf("catalog = %#v", catalog.Data)
	}
	for index := range want {
		if catalog.Data[index].ID != want[index].id || catalog.Data[index].DisplayName != want[index].displayName {
			t.Fatalf("catalog[%d] = %#v, want %#v", index, catalog.Data[index], want[index])
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(
		`{"model":"claude-fable-5","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`,
	))
	request.Header.Set("Authorization", "Bearer launchpad")
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("message status=%d body=%s", response.Code, response.Body.String())
	}
	if forwardedModel != "gpt-5.6-luna" {
		t.Fatalf("forwarded model = %q, want gpt-5.6-luna", forwardedModel)
	}
	if forwardedAuthorization != "Bearer upstream-secret" {
		t.Fatalf("forwarded authorization = %q", forwardedAuthorization)
	}
}

func TestAssignClaudeRoutesDoesNotHideCatalogModelsBehindDuplicateMappings(t *testing.T) {
	catalog := []gateway.Model{
		{ID: "claude-opus-5"},
		{ID: "claude-sonnet-5"},
		{ID: "gpt-5.6-luna"},
		{ID: "gpt-5.6-sol"},
	}
	saved := store.ClaudeConfig{
		Fable5:   "gpt-5.6-sol",
		Opus5:    "claude-sonnet-5",
		Sonnet5:  "claude-sonnet-5",
		Haiku45:  "gpt-5.6-luna",
		Sonnet46: "gpt-5.6-luna",
	}

	routes := assignClaudeRoutes(catalog, saved)
	got := make(map[string]bool)
	for _, route := range routes {
		got[route.TargetModel] = true
	}
	for _, model := range catalog {
		if !got[model.ID] {
			t.Fatalf("catalog model %q missing from routes %#v", model.ID, routes)
		}
	}
	if len(routes) != len(catalog) {
		t.Fatalf("routes = %#v, want one route per unique catalog model", routes)
	}
}
