package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateManagementKeyExpired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"key expired"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	err := client.ValidateManagementKey(context.Background())
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("err = %v", err)
	}
	if !IsManagementKeyAuthError(err) {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestValidateManagementKeyValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "hash123",
			"info": map[string]any{
				"token": "hash123",
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	if err := client.ValidateManagementKey(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestKeyCapabilitiesSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/info" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "hash123",
			"info": map[string]any{
				"token":     "hash123",
				"key_alias": "HarnezPad",
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	supported, reason, err := client.KeyCapabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !supported || reason != "" {
		t.Fatalf("supported=%v reason=%q", supported, reason)
	}
}

func TestKeyCapabilitiesInferenceOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	supported, reason, err := client.KeyCapabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if supported || !strings.Contains(reason, "cannot manage") {
		t.Fatalf("supported=%v reason=%q", supported, reason)
	}
}

func TestListKeysSanitizesSecrets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "hash-active",
				"info": map[string]any{
					"token":     "hash-active",
					"key_alias": "Active Key",
				},
			})
		case r.URL.Path == "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     "hash-active",
						"key_alias": "Active Key",
						"models":    []string{"gpt-5"},
						"blocked":   false,
						"spend":     1.25,
					},
					map[string]any{
						"token":     "hash-other",
						"key_alias": "Other Key",
						"models":    []string{},
						"blocked":   true,
					},
				},
				"total_count":  2,
				"current_page": 1,
				"total_pages":  1,
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	page, err := client.ListKeys(context.Background(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Keys) != 2 {
		t.Fatalf("keys len = %d", len(page.Keys))
	}
	if !page.Keys[0].Active || page.Keys[0].Alias != "Active Key" || page.Keys[0].Models[0] != "gpt-5" {
		t.Fatalf("first key = %+v", page.Keys[0])
	}
	if !page.Keys[0].Management {
		t.Fatalf("active management-capable key should be marked management: %+v", page.Keys[0])
	}
	if !page.Keys[1].AllModels || !page.Keys[1].Blocked {
		t.Fatalf("second key = %+v", page.Keys[1])
	}
	for _, key := range page.Keys {
		if strings.HasPrefix(key.ID, "sk-") {
			t.Fatalf("secret leaked in id: %q", key.ID)
		}
	}
}

func TestGenerateKeyReturnsPlaintextOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key/generate" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body generateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.KeyAlias != "harnezpad" {
			t.Fatalf("alias = %q", body.KeyAlias)
		}
		_ = json.NewEncoder(w).Encode(generateKeyResponse{
			Key:      "sk-test-secret-value",
			KeyAlias: strPtr("harnezpad"),
			Token:    "hash-new",
			Models:   body.Models,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	result, err := client.GenerateKey(context.Background(), CreateKeyInput{
		Alias:  "harnezpad",
		Models: []string{"kimi-k3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != "sk-test-secret-value" {
		t.Fatalf("key = %q", result.Key)
	}
	if result.Summary.ID != "hash-new" || result.Summary.Alias != "harnezpad" {
		t.Fatalf("summary = %+v", result.Summary)
	}
}

func TestUpdateDeleteBlockKey(t *testing.T) {
	var blocked bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "hash-active",
				"info": map[string]any{
					"token":     "hash-active",
					"key_alias": "Management Key",
				},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     "hash-active",
						"key_alias": "Management Key",
						"key_type":  "management",
					},
					map[string]any{
						"token":     "hash123",
						"key_alias": "Mutable Key",
						"key_type":  "default",
					},
				},
			})
		case "/key/update":
			var body updateKeyRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Key != "hash123" {
				t.Fatalf("update body = %+v", body)
			}
			if body.KeyAlias != nil {
				if *body.KeyAlias != "renamed-key" {
					t.Fatalf("alias update body = %+v", body)
				}
			}
			if body.Blocked != nil {
				blocked = *body.Blocked
			}
			w.WriteHeader(http.StatusOK)
		case "/key/delete":
			var body deleteKeyRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Keys) != 1 || body.Keys[0] != "hash123" {
				t.Fatalf("delete body = %+v", body)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	alias := "renamed-key"
	if err := client.UpdateKey(context.Background(), "hash123", UpdateKeyInput{Alias: &alias}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetKeyBlocked(context.Background(), "hash123", true); err != nil || !blocked {
		t.Fatalf("block failed: blocked=%v err=%v", blocked, err)
	}
	if err := client.SetKeyBlocked(context.Background(), "hash123", false); err != nil || blocked {
		t.Fatalf("unblock failed: blocked=%v err=%v", blocked, err)
	}
	if err := client.DeleteKey(context.Background(), "hash123"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateKeyDistinguishesOmittedAndEmptyModels(t *testing.T) {
	var updateBodies []map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key":  "hash-active",
				"info": map[string]any{"token": "hash-active"},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{"token": "hash-active", "key_type": "management"},
					map[string]any{"token": "hash123", "key_alias": "mutable-key"},
				},
			})
		case "/key/update":
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			updateBodies = append(updateBodies, body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	alias := "renamed-key"
	if err := client.UpdateKey(context.Background(), "hash123", UpdateKeyInput{Alias: &alias}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateKey(context.Background(), "hash123", UpdateKeyInput{Models: []string{}}); err != nil {
		t.Fatal(err)
	}
	if len(updateBodies) != 2 {
		t.Fatalf("update bodies = %d", len(updateBodies))
	}
	if _, present := updateBodies[0]["models"]; present {
		t.Fatalf("alias-only update unexpectedly included models: %s", updateBodies[0]["models"])
	}
	models, present := updateBodies[1]["models"]
	if !present || string(models) != "[]" {
		t.Fatalf("empty-model update encoded models as %s, present=%v", models, present)
	}
}

func strPtr(value string) *string {
	return &value
}

func TestManagementKeyCannotBeModified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "mgmt",
				"info": map[string]any{
					"token":     "mgmt",
					"key_alias": "Management Key",
				},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     "mgmt",
						"key_alias": "Management Key",
						"key_type":  "management",
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	if err := client.DeleteKey(context.Background(), "mgmt"); err == nil || !strings.Contains(err.Error(), "management keys") {
		t.Fatalf("delete err = %v", err)
	}
	alias := "Renamed"
	if err := client.UpdateKey(context.Background(), "mgmt", UpdateKeyInput{Alias: &alias}); err == nil || !strings.Contains(err.Error(), "management keys") {
		t.Fatalf("update err = %v", err)
	}
}

func TestListKeysMarksManagementType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key/info":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"key": "hash-active",
				"info": map[string]any{
					"token":     "hash-active",
					"key_alias": "Active Key",
				},
			})
		case "/key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []any{
					map[string]any{
						"token":     "hash-active",
						"key_alias": "Active Key",
						"key_type":  "management",
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "token")
	page, err := client.ListKeys(context.Background(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Keys) != 1 || !page.Keys[0].Management {
		t.Fatalf("keys = %+v", page.Keys)
	}
}
