package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"harnezpad/internal/keys"
)

const inferenceOnlyReason = "Your current key cannot manage Gateway keys. Create a Full Access key named management-key in the Gateway dashboard and paste it in Settings."

type KeySummary struct {
	ID         string   `json:"id"`
	Alias      string   `json:"alias"`
	Slug       string   `json:"slug"`
	Models     []string `json:"models,omitempty"`
	AllModels  bool     `json:"allModels"`
	Blocked    bool     `json:"blocked"`
	Spend      float64  `json:"spend,omitempty"`
	MaxBudget  *float64 `json:"maxBudget,omitempty"`
	Active     bool     `json:"active"`
	Management bool     `json:"management"`
	Default    bool     `json:"default"`
}

type KeyListPage struct {
	Keys        []KeySummary `json:"keys"`
	TotalCount  int          `json:"totalCount"`
	CurrentPage int          `json:"currentPage"`
	TotalPages  int          `json:"totalPages"`
}

type CreateKeyInput struct {
	Alias  string
	Models []string
}

type CreateKeyResult struct {
	Key     string
	Summary KeySummary
}

type UpdateKeyInput struct {
	Alias   *string
	Models  []string
	Blocked *bool
}

type rawKeyObject struct {
	Token     string   `json:"token"`
	KeyAlias  *string  `json:"key_alias"`
	Models    []string `json:"models"`
	Blocked   *bool    `json:"blocked"`
	Spend     *float64 `json:"spend"`
	MaxBudget *float64 `json:"max_budget"`
	KeyType   *string  `json:"key_type"`
}

type keyInfoResponse struct {
	Key  string       `json:"key"`
	Info rawKeyObject `json:"info"`
}

type keyListResponse struct {
	Keys        []json.RawMessage `json:"keys"`
	TotalCount  *int              `json:"total_count"`
	CurrentPage *int              `json:"current_page"`
	TotalPages  *int              `json:"total_pages"`
}

type generateKeyRequest struct {
	KeyAlias string   `json:"key_alias"`
	Models   []string `json:"models,omitempty"`
}

type generateKeyResponse struct {
	Key      string   `json:"key"`
	KeyAlias *string  `json:"key_alias"`
	Models   []string `json:"models"`
	Token    string   `json:"token"`
	Blocked  *bool    `json:"blocked"`
	Spend    *float64 `json:"spend"`
}

type updateKeyRequest struct {
	Key      string    `json:"key"`
	KeyAlias *string   `json:"key_alias,omitempty"`
	Models   *[]string `json:"models,omitempty"`
	Blocked  *bool     `json:"blocked,omitempty"`
}

type deleteKeyRequest struct {
	Keys []string `json:"keys"`
}

func (c *Client) KeyCapabilities(ctx context.Context) (supported bool, reason string, err error) {
	resp, raw, err := c.doRequest(ctx, http.MethodGet, "/key/info", nil)
	if err != nil {
		return false, "", err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return true, "", nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, inferenceOnlyReason, nil
	default:
		return false, "", fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
}

func (c *Client) ValidateManagementKey(ctx context.Context) error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("management key is not configured")
	}
	resp, raw, err := c.doRequest(ctx, http.MethodGet, "/key/info", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body := strings.ToLower(strings.TrimSpace(string(raw)))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if strings.Contains(body, "expir") {
			return fmt.Errorf("management key is expired")
		}
		return fmt.Errorf("management key is invalid")
	}
	return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
}

func IsManagementKeyAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "expired") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "not configured")
}

func (c *Client) CurrentKeyToken(ctx context.Context) (string, error) {
	var out keyInfoResponse
	if err := c.doJSON(ctx, http.MethodGet, "/key/info", nil, &out); err != nil {
		return "", err
	}
	if out.Info.Token != "" {
		return out.Info.Token, nil
	}
	return strings.TrimSpace(out.Key), nil
}

func (c *Client) ListKeys(ctx context.Context, page, size int) (KeyListPage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 50
	}
	path := fmt.Sprintf("/key/list?return_full_object=true&page=%d&size=%d&sort_by=created_at&sort_order=desc", page, size)
	var raw keyListResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return KeyListPage{}, err
	}
	activeID, _ := c.CurrentKeyToken(ctx)
	canManage, _, _ := c.KeyCapabilities(ctx)
	summaries := make([]KeySummary, 0, len(raw.Keys))
	for _, item := range raw.Keys {
		summary, ok := parseKeySummary(item, activeID)
		if ok {
			summary.Management = summary.Management || (canManage && summary.Active)
			summaries = append(summaries, summary)
		}
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].Management && !summaries[j].Management
	})
	return KeyListPage{
		Keys:        summaries,
		TotalCount:  intOrDefault(raw.TotalCount, len(summaries)),
		CurrentPage: intOrDefault(raw.CurrentPage, page),
		TotalPages:  intOrDefault(raw.TotalPages, 1),
	}, nil
}

func (c *Client) GenerateKey(ctx context.Context, in CreateKeyInput) (CreateKeyResult, error) {
	alias := strings.TrimSpace(in.Alias)
	if err := keys.ValidateSlug(alias); err != nil {
		return CreateKeyResult{}, err
	}
	req := generateKeyRequest{KeyAlias: alias}
	if len(in.Models) > 0 {
		req.Models = in.Models
	}
	var out generateKeyResponse
	if err := c.doJSON(ctx, http.MethodPost, "/key/generate", req, &out); err != nil {
		return CreateKeyResult{}, err
	}
	if strings.TrimSpace(out.Key) == "" {
		return CreateKeyResult{}, fmt.Errorf("gateway did not return a key secret")
	}
	id := out.Token
	if id == "" {
		id = out.Key
	}
	summary := summarizeRaw(rawKeyObject{
		Token:    id,
		KeyAlias: out.KeyAlias,
		Models:   out.Models,
		Blocked:  out.Blocked,
		Spend:    out.Spend,
	}, "")
	summary.Alias = alias
	if out.KeyAlias != nil && strings.TrimSpace(*out.KeyAlias) != "" {
		summary.Alias = strings.TrimSpace(*out.KeyAlias)
	}
	return CreateKeyResult{Key: out.Key, Summary: summary}, nil
}

func (c *Client) UpdateKey(ctx context.Context, keyID string, in UpdateKeyInput) error {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("key id is required")
	}
	if err := c.ensureKeyMutable(ctx, keyID); err != nil {
		return err
	}
	req := updateKeyRequest{Key: keyID}
	if in.Alias != nil {
		alias := strings.TrimSpace(*in.Alias)
		if err := keys.ValidateSlug(alias); err != nil {
			return err
		}
		req.KeyAlias = &alias
	}
	if in.Models != nil {
		models := make([]string, len(in.Models))
		copy(models, in.Models)
		req.Models = &models
	}
	if in.Blocked != nil {
		req.Blocked = in.Blocked
	}
	return c.doJSON(ctx, http.MethodPost, "/key/update", req, nil)
}

func (c *Client) DeleteKey(ctx context.Context, keyID string) error {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("key id is required")
	}
	if err := c.ensureKeyMutable(ctx, keyID); err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, "/key/delete", deleteKeyRequest{Keys: []string{keyID}}, nil)
}

func (c *Client) SetKeyBlocked(ctx context.Context, keyID string, blocked bool) error {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return fmt.Errorf("key id is required")
	}
	return c.UpdateKey(ctx, keyID, UpdateKeyInput{Blocked: &blocked})
}

func (c *Client) ensureKeyMutable(ctx context.Context, keyID string) error {
	page, err := c.ListKeys(ctx, 1, 100)
	if err != nil {
		return err
	}
	for _, key := range page.Keys {
		if key.ID == keyID {
			if key.Management {
				return fmt.Errorf("management keys cannot be modified")
			}
			return nil
		}
	}
	return fmt.Errorf("key not found")
}

func parseKeySummary(raw json.RawMessage, activeID string) (KeySummary, bool) {
	if len(raw) == 0 {
		return KeySummary{}, false
	}
	if raw[0] == '"' {
		var token string
		if err := json.Unmarshal(raw, &token); err != nil {
			return KeySummary{}, false
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return KeySummary{}, false
		}
		return KeySummary{
			ID:     token,
			Alias:  displayAlias("", token),
			Slug:   keys.Slugify(displayAlias("", token)),
			Active: token == activeID,
		}, true
	}
	var obj rawKeyObject
	if err := json.Unmarshal(raw, &obj); err != nil {
		return KeySummary{}, false
	}
	return summarizeRaw(obj, activeID), true
}

func summarizeRaw(obj rawKeyObject, activeID string) KeySummary {
	id := strings.TrimSpace(obj.Token)
	if id == "" {
		return KeySummary{}
	}
	alias := ""
	if obj.KeyAlias != nil {
		alias = strings.TrimSpace(*obj.KeyAlias)
	}
	models := append([]string(nil), obj.Models...)
	blocked := obj.Blocked != nil && *obj.Blocked
	spend := 0.0
	if obj.Spend != nil {
		spend = *obj.Spend
	}
	return KeySummary{
		ID:         id,
		Alias:      displayAlias(alias, id),
		Slug:       keys.SlugFromAlias(displayAlias(alias, id)),
		Models:     models,
		AllModels:  len(models) == 0,
		Blocked:    blocked,
		Spend:      spend,
		MaxBudget:  obj.MaxBudget,
		Active:     activeID != "" && id == activeID,
		Management: keyTypeIsManagement(obj.KeyType),
	}
}

func keyTypeIsManagement(keyType *string) bool {
	if keyType == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(*keyType), "management")
}

func displayAlias(alias, id string) string {
	if alias != "" {
		return alias
	}
	if len(id) <= 12 {
		return id
	}
	return id[:6] + "…" + id[len(id)-4:]
}

func intOrDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
