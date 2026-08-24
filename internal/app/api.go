package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"harnezpad/internal/keys"
	"harnezpad/internal/platform"
)

// registerAPIHandlers installs the application API shared by the legacy
// WebView host and the Native SDK helper. Authentication is intentionally
// applied by the helper's outer handler so legacy loopback behavior remains
// unchanged during the migration.
func registerAPIHandlers(mux *http.ServeMux, m *Manager, cliStatus CLIStatus) {
	mux.HandleFunc("/api/cli-status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(cliStatus)
	})
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if r.Method == http.MethodPut {
			var x struct {
				Token          string `json:"token"`
				DefaultKeySlug string `json:"defaultKeySlug"`
			}
			if json.NewDecoder(r.Body).Decode(&x) != nil {
				http.Error(w, "invalid settings", http.StatusBadRequest)
				return
			}
			if x.Token != "" {
				if err := m.SaveManagementToken(r.Context(), x.Token); err != nil {
					status := http.StatusBadRequest
					if strings.HasPrefix(err.Error(), "save management key:") || strings.HasPrefix(err.Error(), "save token:") {
						status = http.StatusInternalServerError
					}
					http.Error(w, UserFacingError(err), status)
					return
				}
			}
			if slug := strings.TrimSpace(x.DefaultKeySlug); slug != "" {
				if err := keys.ValidateSlug(slug); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				m.Settings.DefaultKeySlug = slug
			}
			if err := m.Save(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		_ = json.NewEncoder(w).Encode(m.SettingsResponse(r.Context()))
	})
	mux.HandleFunc("/api/settings/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var x struct {
			Token string `json:"token"`
		}
		if json.NewDecoder(r.Body).Decode(&x) != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if err := m.ValidateManagementToken(r.Context(), x.Token); err != nil {
			http.Error(w, UserFacingError(err), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		xs, err := m.ListModels(r.Context())
		if err != nil {
			writeGatewayError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(xs)
	})
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
	registerKeysHandlers(mux, m)
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("check") == "1" {
			info, err := m.Updater.Check(r.Context())
			if err != nil {
				http.Error(w, UserFacingError(err), http.StatusBadGateway)
				return
			}
			if info != nil {
				if err := m.Updater.Download(r.Context(), *info); err != nil {
					http.Error(w, UserFacingError(err), http.StatusBadGateway)
					return
				}
			}
		}
		_ = json.NewEncoder(w).Encode(m.Updater.Status())
	})
	mux.HandleFunc("/api/update/install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := m.installPendingUpdate(); err != nil {
			statusCode := http.StatusInternalServerError
			if err.Error() == "no verified update is ready" {
				statusCode = http.StatusConflict
			}
			http.Error(w, UserFacingError(err), statusCode)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if os.Getenv("HARNEZPAD_DEBUG") == "1" {
		mux.HandleFunc("/api/debug/toggle-sidebar", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			platform.DebugToggleSidebar()
			w.WriteHeader(http.StatusNoContent)
		})
	}
	mux.HandleFunc("/api/integrations", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(m.Integrations())
	})
	mux.HandleFunc("/api/integrations/", func(w http.ResponseWriter, r *http.Request) {
		p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(p) != 4 || p[0] != "api" || p[1] != "integrations" {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		var err error
		if p[3] == "start" {
			err = m.Start(p[2], r.URL.Query().Get("model"))
		} else if p[3] == "stop" {
			err = m.Stop(p[2])
		} else {
			err = errors.New("unknown action")
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
