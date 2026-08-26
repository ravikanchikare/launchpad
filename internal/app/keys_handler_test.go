package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"harnezpad/internal/gateway"
)

func TestKeysHandlersCapabilities(t *testing.T) {
	m := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/key/info" {
			_ = json.NewEncoder(w).Encode(map[string]any{"key": "hash", "info": map[string]any{"token": "hash"}})
			return
		}
		http.NotFound(w, r)
	})
	mux := http.NewServeMux()
	registerKeysHandlers(mux, m)

	req := httptest.NewRequest(http.MethodGet, "/api/keys/capabilities", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Supported bool   `json:"supported"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.Supported {
		t.Fatalf("supported=false reason=%q", out.Reason)
	}
}

func TestKeysHandlersCreateReturnsSecretOnce(t *testing.T) {
	m := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key":       "sk-created-secret",
				"key_alias": "HarnezPad",
				"token":     "hash-created",
				"models":    []string{"gpt-5"},
			})
		default:
			http.NotFound(w, r)
		}
	})
	mux := http.NewServeMux()
	registerKeysHandlers(mux, m)

	req := httptest.NewRequest(http.MethodPost, "/api/keys", strings.NewReader(`{"alias":"harnezpad","models":["gpt-5"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Key     string             `json:"key"`
		Summary gateway.KeySummary `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Key != "sk-created-secret" {
		t.Fatalf("key = %q", out.Key)
	}
	if out.Summary.ID != "hash-created" {
		t.Fatalf("summary = %+v", out.Summary)
	}
}

func TestKeysHandlersListSanitized(t *testing.T) {
	m := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "hash1",
				"info": map[string]any{
					"token":     "hash1",
					"key_alias": "Mine",
				},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     "hash1",
						"key_alias": "Mine",
						"models":    []string{},
					},
				},
				"total_count":  1,
				"current_page": 1,
				"total_pages":  1,
			})
		default:
			http.NotFound(w, r)
		}
	})
	mux := http.NewServeMux()
	registerKeysHandlers(mux, m)

	req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "sk-") {
		t.Fatalf("list leaked secret: %s", body)
	}
}

func TestKeysHandlersPostDelete(t *testing.T) {
	var deleted string
	hash := "b9677108f49c1ba952ad6de5e014e71cd78fe199320b2b5620877e8503ec3c04"
	activeHash := "a1111111111111111111111111111111111111111111111111111111111111111"
	m := newTestManager(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": activeHash,
				"info": map[string]any{
					"token":     activeHash,
					"key_alias": "Management Key",
				},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     activeHash,
						"key_alias": "Management Key",
					},
					map[string]any{
						"token":     hash,
						"key_alias": "mutable-key",
						"key_type":  "default",
					},
				},
			})
		case "/key/delete":
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			var body struct {
				Keys []string `json:"keys"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Keys) == 1 {
				deleted = body.Keys[0]
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	m.Settings.DefaultKeySlug = "mutable-key"
	mux := http.NewServeMux()
	registerKeysHandlers(mux, m)

	req := httptest.NewRequest(http.MethodPost, "/api/keys/"+hash+"/delete", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if deleted != hash {
		t.Fatalf("deleted = %q", deleted)
	}
	if m.DefaultKeySlug() != "management-key" {
		t.Fatalf("default key slug = %q, want management-key", m.DefaultKeySlug())
	}
}

func newTestManager(t *testing.T, handler http.HandlerFunc) *Manager {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	configPathOverride = filepath.Join(t.TempDir(), "settings.json")
	t.Cleanup(func() { configPathOverride = "" })
	return &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "test-token"}}
}

func TestManagerKeyMethodsRequireToken(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example"}}
	if _, reason, err := m.KeyCapabilities(context.Background()); err != nil || reason == "" {
		t.Fatalf("capabilities = %q err=%v", reason, err)
	}
	if _, err := m.ListKeys(context.Background(), 1, 10); err == nil {
		t.Fatal("expected list keys to fail without token")
	}
}
