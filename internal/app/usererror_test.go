package app

import (
	"errors"
	"testing"
)

func TestUserFacingError(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"management key is expired", "Management key is expired. Create a replacement in Virtual Keys."},
		{"gateway token is not configured; save it in HarnezPad Settings before launching", "Management key is not configured. Save it in HarnezPad Settings before launching."},
		{"gateway returned 401: {\"error\":\"forbidden\"}", "Management key is invalid or expired."},
		{"gateway returned 502: upstream unavailable", "Could not reach the Gateway. Check your connection and try again."},
		{"save token: keychain locked", "Could not save the management key to Keychain."},
		{"self-update is unavailable for development builds", "Updates are not available in this build."},
		{"Enter a valid management key", "Enter a valid management key"},
	}
	for _, tc := range tests {
		if got := UserFacingError(errors.New(tc.in)); got != tc.want {
			t.Fatalf("UserFacingError(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
