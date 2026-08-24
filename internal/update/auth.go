package update

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	headerHarnezPadTimestamp = "X-HarnezPad-Timestamp"
	headerHarnezPadSignature = "X-HarnezPad-Signature"
)

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		vals := values[key]
		sort.Strings(vals)
		for _, value := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

func signClientRequest(req *http.Request, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("update signing secret is not configured")
	}
	timestamp := time.Now().Unix()
	payload := strings.Join([]string{
		req.Method,
		req.URL.Path,
		canonicalQuery(req.URL.Query()),
		strconv.FormatInt(timestamp, 10),
	}, "\n")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	req.Header.Set(headerHarnezPadTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(headerHarnezPadSignature, hex.EncodeToString(mac.Sum(nil)))
	return nil
}

func needsClientAuth(rawURL, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || !strings.HasPrefix(parsed.Path, "/api/") {
		return false
	}
	return parsed.Query().Get("sig") == ""
}

func (u *AppUpdater) authorizedGet(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", "harnezpad/"+u.currentVersion)
	if needsClientAuth(rawURL, u.signingSecret) {
		if err := signClientRequest(req, u.signingSecret); err != nil {
			return nil, err
		}
	}
	return u.client.Do(req)
}
