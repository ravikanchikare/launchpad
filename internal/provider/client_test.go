package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"model_group":"cheap-model","providers":["openai"],"supports_function_calling":true}]}`))
	}))
	defer server.Close()

	profile, err := NewProfile(KindLiteLLM, server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	models, err := NewClient(profile, "secret").ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "cheap-model" || !models[0].SupportsTools {
		t.Fatalf("models = %#v", models)
	}
}

func TestListModelsFallsBackForInferenceOnlyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/model_group/info":
			http.Error(w, `{"error":{"message":"management route forbidden"}}`, http.StatusForbidden)
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"model-b"},{"id":"model-a"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	profile, err := NewProfile(KindLiteLLM, server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	models, err := NewClient(profile, "virtual-key").ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "model-a" || models[1].ID != "model-b" {
		t.Fatalf("models = %#v", models)
	}
}

func TestOpenAICompatibleProviderSkipsLiteLLMDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prefix/v1/models" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"}]}`))
	}))
	defer server.Close()

	profile, err := NewProfile(KindOpenAICompatible, server.URL+"/prefix/v1", "")
	if err != nil {
		t.Fatal(err)
	}
	models, err := NewClient(profile, "key").ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "model-a" {
		t.Fatalf("models = %#v", models)
	}
}

func TestModelsURLOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"}]}`))
	}))
	defer server.Close()

	profile, err := NewProfile(KindLiteLLM, "https://unused.example", server.URL+"/catalog")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient(profile, "key").ListModels(context.Background()); err != nil {
		t.Fatal(err)
	}
}
