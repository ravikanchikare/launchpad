package update

import (
	"net/http"
	"net/url"
	"testing"
)

func TestSignClientRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/api/update?arch=arm64&channel=test&os=darwin&version=0.5.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := signClientRequest(req, "test-secret"); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get(headerHarnezPadTimestamp) == "" || req.Header.Get(headerHarnezPadSignature) == "" {
		t.Fatal("expected signed headers")
	}
}

func TestNeedsClientAuth(t *testing.T) {
	if needsClientAuth("https://example.com/api/release?file=a.zip&expires=1&sig=abc", "secret") {
		t.Fatal("pre-signed URLs should not require client auth")
	}
	if !needsClientAuth("https://example.com/api/artifact?release_id=1&asset_id=2", "secret") {
		t.Fatal("unsigned API URLs should require client auth")
	}
	if needsClientAuth("https://example.com/files/a.zip", "secret") {
		t.Fatal("non-API URLs should not require client auth")
	}
}

func TestCanonicalQuery(t *testing.T) {
	values := url.Values{}
	values.Set("version", "0.5.0")
	values.Set("os", "darwin")
	values.Set("arch", "arm64")
	values.Set("channel", "test")
	if canonicalQuery(values) != "arch=arm64&channel=test&os=darwin&version=0.5.0" {
		t.Fatal("unexpected canonical query")
	}
}
