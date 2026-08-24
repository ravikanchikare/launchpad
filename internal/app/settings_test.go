package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"harnezpad/internal/keys"
)

func TestValidateManagementTokenRejectsTestInvalidKey(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example"}}
	if err := m.ValidateManagementToken(context.Background(), keys.TestInvalidManagementKey); err == nil {
		t.Fatal("expected test invalid key to be rejected")
	}
}

func TestValidateManagementKeyBeforeSaveAllowsTestInvalidKey(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example"}}
	if err := m.validateManagementKeyBeforeSave(context.Background(), keys.TestInvalidManagementKey); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsResponseTestInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL}}
	err := m.SaveManagementToken(context.Background(), "sk-not-valid")
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("err = %v", err)
	}
}

func TestSettingsResponseMissingToken(t *testing.T) {
	m := &Manager{Settings: Settings{GatewayURL: "https://gateway.example"}}
	resp := m.SettingsResponse(context.Background())
	if resp.TokenConfigured || resp.TokenValid || resp.SetupReason != "missing" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestSettingsResponseExpiredToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"key expired"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "expired-token"}}
	resp := m.SettingsResponse(context.Background())
	if !resp.TokenConfigured || resp.TokenValid || resp.SetupReason != "expired" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestSettingsResponseValidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "hash123",
			"info": map[string]any{
				"token": "hash123",
			},
		})
	}))
	defer srv.Close()

	m := &Manager{Settings: Settings{GatewayURL: srv.URL, Token: "good-token"}}
	resp := m.SettingsResponse(context.Background())
	if !resp.TokenConfigured || !resp.TokenValid || resp.SetupReason != "" {
		t.Fatalf("response = %+v", resp)
	}
}
