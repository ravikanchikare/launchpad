package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"launchpad/internal/config"
	"launchpad/internal/credentials"
	"launchpad/internal/provider"
	"launchpad/internal/store"
)

type claudeRoute struct {
	ID              string
	DisplayName     string
	TargetModel     string
	CreatedAt       string
	Family          string
	IsFamilyDefault bool
}

var claudeRouteSlots = []claudeRoute{
	{ID: "claude-fable-5", CreatedAt: "2026-06-09T00:00:00Z", Family: "fable", IsFamilyDefault: true},
	{ID: "claude-opus-5", CreatedAt: "2026-07-24T00:00:00Z", Family: "opus", IsFamilyDefault: true},
	{ID: "claude-sonnet-5", CreatedAt: "2026-06-30T00:00:00Z", Family: "sonnet", IsFamilyDefault: true},
	{ID: "claude-haiku-4-5-20251001", CreatedAt: "2025-10-01T00:00:00Z", Family: "haiku", IsFamilyDefault: true},
	{ID: "claude-sonnet-4-6", CreatedAt: "2025-11-18T00:00:00Z", Family: "sonnet"},
}

func (s *Server) getClaudeProviderModels(w http.ResponseWriter, r *http.Request) {
	routes, _, _, err := s.resolveClaudeRoutes(r.Context())
	if err != nil {
		writeClaudeProviderError(w, http.StatusBadGateway, err)
		return
	}
	type model struct {
		ID                  string `json:"id"`
		Type                string `json:"type"`
		DisplayName         string `json:"display_name"`
		CreatedAt           string `json:"created_at"`
		MaxTokens           int    `json:"max_tokens"`
		AnthropicFamilyTier string `json:"anthropic_family_tier"`
		IsFamilyDefault     bool   `json:"is_family_default"`
	}
	models := make([]model, 0, len(routes))
	for _, route := range routes {
		models = append(models, model{
			ID:                  route.ID,
			Type:                "model",
			DisplayName:         route.DisplayName,
			CreatedAt:           route.CreatedAt,
			MaxTokens:           64_000,
			AnthropicFamilyTier: route.Family,
			IsFamilyDefault:     route.IsFamilyDefault,
		})
	}
	firstID, lastID := "", ""
	if len(models) > 0 {
		firstID, lastID = models[0].ID, models[len(models)-1].ID
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": models, "first_id": firstID, "last_id": lastID, "has_more": false,
	})
}

func (s *Server) postClaudeProviderMessage(w http.ResponseWriter, r *http.Request) {
	routes, settings, apiKey, err := s.resolveClaudeRoutes(r.Context())
	if err != nil {
		writeClaudeProviderError(w, http.StatusBadGateway, err)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (64<<20)+1))
	if err != nil {
		writeClaudeProviderError(w, http.StatusBadRequest, err)
		return
	}
	if len(body) > 64<<20 {
		writeClaudeProviderError(w, http.StatusRequestEntityTooLarge, errors.New("request body is too large"))
		return
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		writeClaudeProviderError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))
		return
	}
	var requestedModel string
	if err := json.Unmarshal(payload["model"], &requestedModel); err != nil {
		writeClaudeProviderError(w, http.StatusBadRequest, errors.New("model is required"))
		return
	}
	var targetModel string
	for _, route := range routes {
		if route.ID == requestedModel {
			targetModel = route.TargetModel
			break
		}
	}
	if targetModel == "" {
		writeClaudeProviderError(w, http.StatusBadRequest, fmt.Errorf("unknown Claude model route %q", requestedModel))
		return
	}
	payload["model"], _ = json.Marshal(targetModel)
	rewritten, err := json.Marshal(payload)
	if err != nil {
		writeClaudeProviderError(w, http.StatusInternalServerError, err)
		return
	}

	upstreamURL, err := url.Parse(strings.TrimRight(settings.ProviderURL, "/") + r.URL.Path)
	if err != nil {
		writeClaudeProviderError(w, http.StatusInternalServerError, err)
		return
	}
	upstreamRequest, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL.String(), bytes.NewReader(rewritten))
	if err != nil {
		writeClaudeProviderError(w, http.StatusInternalServerError, err)
		return
	}
	for name, values := range r.Header {
		if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Cookie") || strings.EqualFold(name, "Host") {
			continue
		}
		for _, value := range values {
			upstreamRequest.Header.Add(name, value)
		}
	}
	upstreamRequest.Header.Set("Authorization", "Bearer "+apiKey)
	upstreamRequest.Header.Set("X-Api-Key", apiKey)
	upstreamRequest.Header.Set("Content-Type", "application/json")

	response, err := s.providerHTTPClient().Do(upstreamRequest)
	if err != nil {
		writeClaudeProviderError(w, http.StatusBadGateway, fmt.Errorf("connect to LiteLLM provider: %w", err))
		return
	}
	defer response.Body.Close()
	for name, values := range response.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	if err := copyClaudeProviderResponse(w, response.Body); err != nil {
		s.log().Warn("stream Claude provider response", "error", err)
	}
}

func copyClaudeProviderResponse(w http.ResponseWriter, body io.Reader) error {
	buffer := make([]byte, 32<<10)
	flusher, _ := w.(http.Flusher)
	for {
		count, readErr := body.Read(buffer)
		if count > 0 {
			if _, writeErr := w.Write(buffer[:count]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func (s *Server) resolveClaudeRoutes(ctx context.Context) ([]claudeRoute, config.Settings, string, error) {
	settings, err := config.Load()
	if err != nil {
		return nil, config.Settings{}, "", err
	}
	apiKey, err := credentials.Resolve()
	if err != nil {
		return nil, config.Settings{}, "", err
	}
	client := provider.NewClient(settings.ProviderURL, apiKey)
	client.HTTP = s.providerHTTPClient()
	catalog, err := client.ListModels(ctx)
	if err != nil {
		return nil, config.Settings{}, "", err
	}
	claudeConfig, err := s.Store.ClaudeConfig()
	if err != nil {
		return nil, config.Settings{}, "", err
	}
	routes := assignClaudeRoutes(catalog, claudeConfig)
	if len(routes) == 0 {
		return nil, config.Settings{}, "", errors.New("provider returned no models for Claude")
	}
	return routes, settings, apiKey, nil
}

func assignClaudeRoutes(catalog []provider.Model, saved store.ClaudeConfig) []claudeRoute {
	available := make(map[string]bool, len(catalog))
	remaining := make([]string, 0, len(catalog))
	for _, model := range catalog {
		available[model.ID] = true
		remaining = append(remaining, model.ID)
	}
	savedModels := []string{saved.Fable5, saved.Opus5, saved.Sonnet5, saved.Haiku45, saved.Sonnet46}
	assigned := make([]string, len(claudeRouteSlots))
	used := make(map[string]bool)
	for index, model := range savedModels {
		if available[model] && !used[model] {
			assigned[index] = model
			used[model] = true
		}
	}
	for index, slot := range claudeRouteSlots {
		if assigned[index] == "" && available[slot.ID] && !used[slot.ID] {
			assigned[index] = slot.ID
			used[slot.ID] = true
		}
	}
	next := 0
	for index := range assigned {
		if assigned[index] != "" {
			continue
		}
		for next < len(remaining) && used[remaining[next]] {
			next++
		}
		if next == len(remaining) {
			break
		}
		assigned[index] = remaining[next]
		used[remaining[next]] = true
		next++
	}
	routes := make([]claudeRoute, 0, len(assigned))
	for index, target := range assigned {
		if target == "" {
			continue
		}
		route := claudeRouteSlots[index]
		route.DisplayName = target
		route.TargetModel = target
		routes = append(routes, route)
	}
	return routes
}

func (s *Server) providerHTTPClient() *http.Client {
	if s.ProviderHTTP != nil {
		return s.ProviderHTTP
	}
	return http.DefaultClient
}

func writeClaudeProviderError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":  "error",
		"error": map[string]string{"type": "api_error", "message": err.Error()},
	})
}
