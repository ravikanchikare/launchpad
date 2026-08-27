package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"launchpad/internal/config"
	"launchpad/internal/credentials"
	"launchpad/internal/gateway"
	"launchpad/internal/launch"
	"launchpad/internal/store"
)

type Server struct {
	Store            *store.Store
	Logger           *slog.Logger
	ClaudeGatewayURL string
	GatewayHTTP      *http.Client
}

func (s *Server) log() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/settings", s.getSettings)
	mux.HandleFunc("POST /api/v1/settings", s.postSettings)
	mux.HandleFunc("GET /api/v1/launcher/config", s.getLauncherConfig)
	mux.HandleFunc("POST /api/v1/launcher/config", s.postLauncherConfig)
	mux.HandleFunc("GET /api/v1/integrations", s.getIntegrations)
	mux.HandleFunc("GET /api/v1/harnesses", s.getIntegrations)
	mux.HandleFunc("GET /api/v1/harnesses/{id}", s.getHarness)
	mux.HandleFunc("POST /api/v1/harnesses/{id}/launch", s.postHarnessLaunch)
	mux.HandleFunc("POST /api/v1/harnesses/{id}/restore", s.postHarnessRestore)
	mux.HandleFunc("GET /api/v1/apps/claude", s.getClaudeApp)
	mux.HandleFunc("POST /api/v1/apps/claude", s.postClaudeApp)
	mux.HandleFunc("GET /api/v1/apps/claude/models", s.getClaudeModels)
	mux.HandleFunc("POST /api/v1/apps/claude/restart", s.postClaudeRestart)
	mux.HandleFunc("POST /api/v1/apps/claude/reset", s.postClaudeReset)
	mux.HandleFunc("GET /v1/models", s.getClaudeGatewayModels)
	mux.HandleFunc("POST /v1/messages", s.postClaudeGatewayMessage)
	mux.HandleFunc("POST /v1/messages/count_tokens", s.postClaudeGatewayMessage)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) getLauncherConfig(w http.ResponseWriter, r *http.Request) {
	settings, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, keyErr := credentials.Resolve()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"gatewayUrl":       settings.GatewayURL,
		"cliName":          config.CLIName(),
		"apiKeyConfigured": keyErr == nil,
	})
}

func (s *Server) postLauncherConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		GatewayURL *string `json:"gatewayUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	settings, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if request.GatewayURL != nil {
		settings.GatewayURL = strings.TrimSpace(*request.GatewayURL)
	}
	if err := config.Save(settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.getLauncherConfig(w, r)
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.Settings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"settings": map[string]any{
			"showAppsInMenu":    st.ShowAppsInMenu,
			"autoUpdateEnabled": st.AutoUpdateEnabled,
		},
	})
}

func (s *Server) postSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShowAppsInMenu    *bool `json:"showAppsInMenu"`
		AutoUpdateEnabled *bool `json:"autoUpdateEnabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	st, err := s.Store.Settings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if req.ShowAppsInMenu != nil {
		st.ShowAppsInMenu = *req.ShowAppsInMenu
	}
	if req.AutoUpdateEnabled != nil {
		st.AutoUpdateEnabled = *req.AutoUpdateEnabled
	}
	if err := s.Store.SetSettings(st); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.log().Info("settings updated", "showAppsInMenu", st.ShowAppsInMenu, "autoUpdateEnabled", st.AutoUpdateEnabled)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"settings": map[string]any{
			"showAppsInMenu":    st.ShowAppsInMenu,
			"autoUpdateEnabled": st.AutoUpdateEnabled,
		},
	})
}

func (s *Server) getIntegrations(w http.ResponseWriter, r *http.Request) {
	infos := launch.ListInfos()
	cliName := config.CLIName()
	type resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Installed   bool   `json:"installed"`
		Command     string `json:"command,omitempty"`
	}
	installed := launch.IsInstalled("claude")
	out := []resp{
		{ID: "claude-desktop", Name: "Claude", Description: "Use gateway models in Claude Desktop", Installed: installed},
		{ID: "chatgpt", Name: "ChatGPT", Description: "Launch ChatGPT through your LiteLLM gateway", Installed: launch.IsInstalled("chatgpt"), Command: cliName + " launch chatgpt"},
	}
	enabledTerminal := map[string]bool{"claude": true, "codex": true, "opencode": true, "copilot": true}
	for _, inf := range infos {
		if !enabledTerminal[inf.Name] {
			continue
		}
		out = append(out, resp{ID: inf.Name, Name: inf.DisplayName, Description: inf.Description, Installed: inf.Installed, Command: cliName + " launch " + inf.Name})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getHarness(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	spec, err := launch.Lookup(id)
	if err != nil {
		http.Error(w, "unknown harness", 404)
		return
	}
	installed := false
	if spec.CheckInstalled != nil {
		installed = spec.CheckInstalled()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":          spec.Name,
		"name":        spec.DisplayName,
		"description": spec.Description,
		"installed":   installed,
		"command":     "launchpad launch " + spec.Name,
		"installUrl":  spec.InstallURL,
	})
}

func (s *Server) postHarnessLaunch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	spec, err := launch.Lookup(id)
	if err != nil {
		http.Error(w, "unknown harness", 404)
		return
	}
	s.log().Info("harness launch requested", "id", spec.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "would launch " + spec.Name})
}

func (s *Server) postHarnessRestore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.log().Info("harness restore requested", "id", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) getClaudeApp(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Store.ClaudeConfig()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	running, runningErr := launch.ClaudeDesktopRunning(r.Context())
	if runningErr != nil {
		s.log().Warn("could not check Claude Desktop process", "error", runningErr)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		store.ClaudeConfig
		Running bool `json:"running"`
	}{ClaudeConfig: cfg, Running: running})
}

func (s *Server) postClaudeApp(w http.ResponseWriter, r *http.Request) {
	var cfg struct {
		Fable5   *string `json:"fable_5"`
		Opus5    *string `json:"opus_5"`
		Sonnet5  *string `json:"sonnet_5"`
		Haiku45  *string `json:"haiku_4_5"`
		Sonnet46 *string `json:"sonnet_4_6"`
		AutoMode *bool   `json:"autoMode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}
	cur, err := s.Store.ClaudeConfig()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if cfg.Fable5 != nil {
		cur.Fable5 = *cfg.Fable5
	}
	if cfg.Opus5 != nil {
		cur.Opus5 = *cfg.Opus5
	}
	if cfg.Sonnet5 != nil {
		cur.Sonnet5 = *cfg.Sonnet5
	}
	if cfg.Haiku45 != nil {
		cur.Haiku45 = *cfg.Haiku45
	}
	if cfg.Sonnet46 != nil {
		cur.Sonnet46 = *cfg.Sonnet46
	}
	if cfg.AutoMode != nil {
		cur.AutoMode = *cfg.AutoMode
	}
	if err := s.Store.SetClaudeConfig(cur); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.log().Info("claude config updated", "cfg", cur)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cur)
}

func (s *Server) getClaudeModels(w http.ResponseWriter, r *http.Request) {
	settings, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apiKey, err := credentials.Resolve()
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	catalog, err := gateway.NewClient(settings.GatewayURL, apiKey).ListModels(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	models := make([]string, 0, len(catalog))
	for _, model := range catalog {
		models = append(models, model.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (s *Server) postClaudeRestart(w http.ResponseWriter, r *http.Request) {
	s.log().Info("claude restart requested")
	if _, err := credentials.Resolve(); err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	claudeConfig, err := s.Store.ClaudeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	profile := launch.ClaudeDesktopProfile{
		GatewayURL: s.ClaudeGatewayURL,
		APIKey:     "launchpad",
		AutoMode:   claudeConfig.AutoMode,
	}
	if profile.GatewayURL == "" {
		http.Error(w, "Launchpad Claude gateway is unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := launch.ConfigureClaudeDesktop(r.Context(), profile); err != nil {
		s.log().Error("claude restart failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "Claude Desktop is running"})
}

func (s *Server) postClaudeReset(w http.ResponseWriter, r *http.Request) {
	_ = s.Store.SetClaudeConfig(store.ClaudeConfig{AutoMode: false})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
