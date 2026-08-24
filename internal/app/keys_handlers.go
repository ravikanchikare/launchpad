package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"harnezpad/internal/gateway"
	"harnezpad/internal/keys"
)

func (m *Manager) KeyCapabilities(ctx context.Context) (bool, string, error) {
	if err := m.RequireGatewayToken(); err != nil {
		return false, err.Error(), nil
	}
	return m.gatewayClient().KeyCapabilities(ctx)
}

func (m *Manager) ListKeys(ctx context.Context, page, size int) (gateway.KeyListPage, error) {
	if err := m.RequireGatewayToken(); err != nil {
		return gateway.KeyListPage{}, err
	}
	return m.gatewayClient().ListKeys(ctx, page, size)
}

func (m *Manager) CreateKey(ctx context.Context, in gateway.CreateKeyInput) (gateway.CreateKeyResult, error) {
	if err := m.RequireGatewayToken(); err != nil {
		return gateway.CreateKeyResult{}, err
	}
	return m.gatewayClient().GenerateKey(ctx, in)
}

func (m *Manager) UpdateKey(ctx context.Context, keyID string, in gateway.UpdateKeyInput) error {
	if err := m.RequireGatewayToken(); err != nil {
		return err
	}
	return m.gatewayClient().UpdateKey(ctx, keyID, in)
}

func (m *Manager) DeleteKey(ctx context.Context, keyID string) error {
	if err := m.RequireGatewayToken(); err != nil {
		return err
	}
	return m.gatewayClient().DeleteKey(ctx, keyID)
}

func (m *Manager) SetKeyBlocked(ctx context.Context, keyID string, blocked bool) error {
	if err := m.RequireGatewayToken(); err != nil {
		return err
	}
	return m.gatewayClient().SetKeyBlocked(ctx, keyID, blocked)
}

func (m *Manager) annotateKeyList(page gateway.KeyListPage) gateway.KeyListPage {
	defaultSlug := m.DefaultKeySlug()
	for i := range page.Keys {
		page.Keys[i].Default = page.Keys[i].Slug == defaultSlug
	}
	return page
}

func (m *Manager) keySlugByID(ctx context.Context, keyID string) (string, error) {
	page, err := m.ListKeys(ctx, 1, 100)
	if err != nil {
		return "", err
	}
	for _, key := range page.Keys {
		if key.ID == keyID {
			return key.Slug, nil
		}
	}
	return "", fmt.Errorf("key not found")
}

func registerKeysHandlers(mux *http.ServeMux, m *Manager) {
	mux.HandleFunc("/api/keys/capabilities", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		supported, reason, err := m.KeyCapabilities(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(struct {
			Supported bool   `json:"supported"`
			Reason    string `json:"reason,omitempty"`
		}{supported, reason})
	})

	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			size, _ := strconv.Atoi(r.URL.Query().Get("size"))
			list, err := m.ListKeys(r.Context(), page, size)
			if err != nil {
				writeGatewayError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(m.annotateKeyList(list))
		case http.MethodPost:
			var body struct {
				Alias  string   `json:"alias"`
				Models []string `json:"models"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			result, err := m.CreateKey(r.Context(), gateway.CreateKeyInput{
				Alias:  body.Alias,
				Models: body.Models,
			})
			if err != nil {
				writeGatewayError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(struct {
				Key     string             `json:"key"`
				Summary gateway.KeySummary `json:"summary"`
			}{result.Key, result.Summary})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/keys/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Slug  string `json:"slug"`
			Token string `json:"token"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := keys.ValidateSlug(body.Slug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if err := m.saveStoredNamedKey(body.Slug, body.Token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/api/keys/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "api" || parts[1] != "keys" {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		keyID := parts[2]
		if decoded, err := url.PathUnescape(keyID); err == nil {
			keyID = decoded
		}
		if strings.HasPrefix(keyID, "sk-") {
			http.Error(w, "invalid key id", http.StatusBadRequest)
			return
		}
		action := ""
		if len(parts) > 3 {
			action = parts[3]
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case action == "" && r.Method == http.MethodPut:
			var body struct {
				Alias  *string  `json:"alias"`
				Models []string `json:"models"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			in := gateway.UpdateKeyInput{Alias: body.Alias}
			if body.Models != nil {
				in.Models = body.Models
			}
			if err := m.UpdateKey(r.Context(), keyID, in); err != nil {
				writeGatewayError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case action == "" && r.Method == http.MethodDelete:
			if err := m.DeleteKey(r.Context(), keyID); err != nil {
				writeGatewayError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case action == "delete" && r.Method == http.MethodPost:
			slug, slugErr := m.keySlugByID(r.Context(), keyID)
			if err := m.DeleteKey(r.Context(), keyID); err != nil {
				writeGatewayError(w, err)
				return
			}
			if slugErr == nil && slug != "" && slug != keys.ManagementSlug {
				_ = m.deleteStoredNamedKey(slug)
			}
			if slugErr == nil && slug == m.DefaultKeySlug() {
				m.Settings.DefaultKeySlug = keys.ManagementSlug
				if err := m.Save(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else if err := m.ensureDefaultKeyAvailable(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case action == "default" && r.Method == http.MethodPost:
			slug, err := m.keySlugByID(r.Context(), keyID)
			if err != nil {
				writeGatewayError(w, err)
				return
			}
			if slug == "" {
				http.Error(w, "key not found", http.StatusNotFound)
				return
			}
			m.Settings.DefaultKeySlug = slug
			if err := m.Save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case action == "block" && r.Method == http.MethodPost:
			if err := m.SetKeyBlocked(r.Context(), keyID, true); err != nil {
				writeGatewayError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case action == "unblock" && r.Method == http.MethodPost:
			if err := m.SetKeyBlocked(r.Context(), keyID, false); err != nil {
				writeGatewayError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func writeGatewayError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": UserFacingError(err)})
}
