package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"harnezpad/internal/gateway"
	"harnezpad/internal/update"
)

const nativeContractToken = "native-contract-session"

func TestNativeBearerProtectsEveryAPIPath(t *testing.T) {
	t.Setenv("HARNEZPAD_DEBUG", "1")
	manager := NewManager()
	manager.Updater = update.NewAppUpdater("dev", "")
	mux := http.NewServeMux()
	registerAPIHandlers(mux, manager, CLIStatus{})
	handler := requireNativeBearer(nativeContractToken, mux)

	for _, route := range nativeAPIRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

type nativeAPIRoute struct {
	method string
	path   string
}

func nativeAPIRoutes() []nativeAPIRoute {
	return []nativeAPIRoute{
		{http.MethodGet, "/api/cli-status"},
		{http.MethodGet, "/api/settings"},
		{http.MethodPut, "/api/settings"},
		{http.MethodPost, "/api/settings/validate"},
		{http.MethodGet, "/api/models"},
		{http.MethodGet, "/api/models/catalog"},
		{http.MethodGet, "/api/account"},
		{http.MethodGet, "/api/keys/capabilities"},
		{http.MethodGet, "/api/keys"},
		{http.MethodPost, "/api/keys"},
		{http.MethodPost, "/api/keys/register"},
		{http.MethodPut, "/api/keys/key-id"},
		{http.MethodDelete, "/api/keys/key-id"},
		{http.MethodPost, "/api/keys/key-id/delete"},
		{http.MethodPost, "/api/keys/key-id/default"},
		{http.MethodPost, "/api/keys/key-id/block"},
		{http.MethodPost, "/api/keys/key-id/unblock"},
		{http.MethodGet, "/api/update"},
		{http.MethodPost, "/api/update/install"},
		{http.MethodGet, "/api/integrations"},
		{http.MethodPost, "/api/integrations/claude/start"},
		{http.MethodPost, "/api/integrations/claude/stop"},
		{http.MethodPost, "/api/debug/toggle-sidebar"},
	}
}

func TestNativeCORSPreflightCoversEveryAPIPath(t *testing.T) {
	manager := NewManager()
	manager.Updater = update.NewAppUpdater("dev", "")
	mux := http.NewServeMux()
	registerAPIHandlers(mux, manager, CLIStatus{})
	handler := nativeCORSMiddleware(requireNativeBearer(nativeContractToken, mux))

	for _, route := range nativeAPIRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, route.path, nil)
			req.Header.Set("Origin", "zero://app")
			req.Header.Set("Access-Control-Request-Method", route.method)
			req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != "zero://app" {
				t.Fatalf("allow origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestHybridWebViewBootstrapContract(t *testing.T) {
	handler, _ := newNativeContractHandler(t)
	routes := []struct {
		path   string
		decode func([]byte) error
	}{
		{"/api/cli-status", func(body []byte) error {
			var value CLIStatus
			return json.Unmarshal(body, &value)
		}},
		{"/api/settings", func(body []byte) error {
			var value SettingsResponse
			return json.Unmarshal(body, &value)
		}},
		{"/api/models", func(body []byte) error {
			var value []Model
			return json.Unmarshal(body, &value)
		}},
		{"/api/models/catalog", func(body []byte) error {
			var value []gateway.ModelCatalogEntry
			return json.Unmarshal(body, &value)
		}},
		{"/api/account", func(body []byte) error {
			var value gateway.AccountSummary
			return json.Unmarshal(body, &value)
		}},
		{"/api/keys/capabilities", func(body []byte) error {
			var value struct {
				Supported bool   `json:"supported"`
				Reason    string `json:"reason"`
			}
			return json.Unmarshal(body, &value)
		}},
		{"/api/keys", func(body []byte) error {
			var value gateway.KeyListPage
			return json.Unmarshal(body, &value)
		}},
		{"/api/update", func(body []byte) error {
			var value update.UpdateStatus
			return json.Unmarshal(body, &value)
		}},
	}
	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			response := nativeContractRequestFromOrigin(t, handler, http.MethodGet, route.path, "", "zero://app")
			assertNativeStatus(t, response, http.StatusOK)
			if response.Header().Get("Access-Control-Allow-Origin") != "zero://app" {
				t.Fatalf("allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
			}
			if err := route.decode(response.Body.Bytes()); err != nil {
				t.Fatalf("typed decode: %v body=%s", err, response.Body.String())
			}
		})
	}
}

func TestNativeAPIRouteMethods(t *testing.T) {
	handler, _ := newNativeContractHandler(t)
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/settings"},
		{http.MethodGet, "/api/settings/validate"},
		{http.MethodPost, "/api/models"},
		{http.MethodPost, "/api/models/catalog"},
		{http.MethodPost, "/api/account"},
		{http.MethodPost, "/api/keys/capabilities"},
		{http.MethodPatch, "/api/keys"},
		{http.MethodGet, "/api/keys/register"},
		{http.MethodGet, "/api/keys/key-id"},
		{http.MethodGet, "/api/keys/key-id/delete"},
		{http.MethodGet, "/api/keys/key-id/default"},
		{http.MethodGet, "/api/keys/key-id/block"},
		{http.MethodGet, "/api/keys/key-id/unblock"},
		{http.MethodPost, "/api/update"},
		{http.MethodGet, "/api/update/install"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := nativeContractRequest(t, handler, test.method, test.path, "")
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestNativeAPIResponseAndMutationContracts(t *testing.T) {
	handler, stub := newNativeContractHandler(t)

	response := nativeContractRequest(t, handler, http.MethodGet, "/api/settings", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONFields(t, response, "gatewayUrl", "tokenConfigured", "tokenValid", "defaultKeySlug")
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}

	response = nativeContractRequest(t, handler, http.MethodPost, "/api/settings/validate", `{"token":"replacement-token"}`)
	assertNativeStatus(t, response, http.StatusNoContent)
	response = nativeContractRequest(t, handler, http.MethodPut, "/api/settings", `{"defaultKeySlug":"contract-key"}`)
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONValue(t, response, "defaultKeySlug", "contract-key")

	response = nativeContractRequest(t, handler, http.MethodGet, "/api/models", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONArrayFirstFields(t, response, "id", "owned_by")
	response = nativeContractRequest(t, handler, http.MethodGet, "/api/models/catalog", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONArrayFirstFields(t, response, "id", "supportsVision", "supportsTools", "supportsReasoning", "supportsWebSearch")

	response = nativeContractRequest(t, handler, http.MethodGet, "/api/account", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONFields(t, response, "teamId", "teamAlias", "spend", "maxBudget")

	response = nativeContractRequest(t, handler, http.MethodGet, "/api/keys/capabilities", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONValue(t, response, "supported", true)
	response = nativeContractRequest(t, handler, http.MethodGet, "/api/keys?page=1&size=20", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONFields(t, response, "keys", "totalCount", "currentPage", "totalPages")

	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys", `{"alias":"created-key","models":["model-a"]}`)
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONFields(t, response, "key", "summary")
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys/register", `{"slug":"created-key","token":"created-secret"}`)
	assertNativeStatus(t, response, http.StatusNoContent)
	if stub.registeredSlug != "created-key" || stub.registeredToken != "created-secret" {
		t.Fatalf("registered key = %q %q", stub.registeredSlug, stub.registeredToken)
	}

	response = nativeContractRequest(t, handler, http.MethodPut, "/api/keys/mutable-id", `{"alias":"renamed-key","models":[]}`)
	assertNativeStatus(t, response, http.StatusNoContent)
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys/mutable-id/default", "")
	assertNativeStatus(t, response, http.StatusNoContent)
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys/mutable-id/block", "")
	assertNativeStatus(t, response, http.StatusNoContent)
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys/mutable-id/unblock", "")
	assertNativeStatus(t, response, http.StatusNoContent)
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/keys/mutable-id/delete", "")
	assertNativeStatus(t, response, http.StatusNoContent)
	if stub.deletedID != "mutable-id" || stub.deletedSlug != "contract-key" {
		t.Fatalf("deleted id/slug = %q/%q", stub.deletedID, stub.deletedSlug)
	}

	response = nativeContractRequest(t, handler, http.MethodGet, "/api/update", "")
	assertNativeStatus(t, response, http.StatusOK)
	assertJSONFields(t, response, "currentVersion", "channel", "available", "downloaded")
	response = nativeContractRequest(t, handler, http.MethodPost, "/api/update/install", "")
	assertNativeStatus(t, response, http.StatusConflict)

	if len(stub.updateBodies) < 3 {
		t.Fatalf("key update calls = %d, want edit + block + unblock", len(stub.updateBodies))
	}
}

type nativeContractGateway struct {
	registeredSlug  string
	registeredToken string
	deletedID       string
	deletedSlug     string
	updateBodies    []map[string]json.RawMessage
}

func newNativeContractHandler(t *testing.T) (http.Handler, *nativeContractGateway) {
	t.Helper()
	stub := &nativeContractGateway{}
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "model-a", "owned_by": "provider-a"}}})
		case "/model_group/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{
				"model_group": "model-a", "providers": []string{"provider-a"}, "supports_vision": true,
			}}})
		case "/user/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"user_info": map[string]any{"teams": []string{"team-a"}}})
		case "/team/team-a/members/me":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"team_id": "team-a", "team_alias": "Team A", "spend": 4.5,
				"litellm_budget_table": map[string]any{"max_budget": 50.0},
			})
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "management-id", "info": map[string]any{"token": "management-id", "key_alias": "Management Key"},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{"token": "management-id", "key_alias": "Management Key", "key_type": "management"},
					map[string]any{"token": "mutable-id", "key_alias": "contract-key", "models": []string{"model-a"}},
				},
				"total_count": 2, "current_page": 1, "total_pages": 1,
			})
		case "/key/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "created-secret", "token": "created-id", "key_alias": "created-key", "models": []string{"model-a"},
			})
		case "/key/update":
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			stub.updateBodies = append(stub.updateBodies, body)
			w.WriteHeader(http.StatusOK)
		case "/key/delete":
			var body struct {
				Keys []string `json:"keys"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body.Keys) == 1 {
				stub.deletedID = body.Keys[0]
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(gateway.Close)

	configPathOverride = filepath.Join(t.TempDir(), "settings.json")
	t.Cleanup(func() { configPathOverride = "" })
	manager := NewManager()
	manager.Settings = Settings{GatewayURL: gateway.URL, Token: "management-token", DefaultKeySlug: "management-key"}
	manager.Updater = update.NewAppUpdater("dev", "")
	manager.saveNamedKey = func(slug, token string) error {
		stub.registeredSlug = slug
		stub.registeredToken = token
		return nil
	}
	manager.deleteNamedKey = func(slug string) error {
		stub.deletedSlug = slug
		return nil
	}
	mux := http.NewServeMux()
	registerAPIHandlers(mux, manager, CLIStatus{Installed: true, Path: "/tmp/harnezpad"})
	return nativeCORSMiddleware(requireNativeBearer(nativeContractToken, mux)), stub
}

func nativeContractRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return nativeContractRequestFromOrigin(t, handler, method, path, body, "")
}

func nativeContractRequestFromOrigin(t *testing.T, handler http.Handler, method, path, body, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+nativeContractToken)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertNativeStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d, body=%s", response.Code, want, response.Body.String())
	}
}

func assertJSONFields(t *testing.T, response *httptest.ResponseRecorder, fields ...string) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON: %v body=%s", err, response.Body.String())
	}
	for _, field := range fields {
		if _, ok := body[field]; !ok {
			t.Fatalf("missing field %q in %v", field, body)
		}
	}
}

func assertJSONValue(t *testing.T, response *httptest.ResponseRecorder, field string, want any) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON: %v body=%s", err, response.Body.String())
	}
	if body[field] != want {
		t.Fatalf("%s = %#v, want %#v", field, body[field], want)
	}
}

func assertJSONArrayFirstFields(t *testing.T, response *httptest.ResponseRecorder, fields ...string) {
	t.Helper()
	var body []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON: %v body=%s", err, response.Body.String())
	}
	if len(body) == 0 {
		t.Fatal("expected nonempty JSON array")
	}
	for _, field := range fields {
		if _, ok := body[0][field]; !ok {
			t.Fatalf("missing field %q in %v", field, body[0])
		}
	}
}
