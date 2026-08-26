package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    http.DefaultClient,
	}
}

func (c *Client) apiBase() string {
	return strings.TrimRight(c.BaseURL, "/")
}

func (c *Client) doRequest(ctx context.Context, method, path string, in any) (*http.Response, []byte, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiBase()+path, body)
	if err != nil {
		return nil, nil, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if err != nil {
		return resp, nil, err
	}
	return resp, raw, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out any) error {
	resp, raw, err := c.doRequest(ctx, method, path, in)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("gateway returned %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}
