package credentials

import "testing"

func TestResolveUsesProviderAPIKey(t *testing.T) {
	t.Setenv("LAUNCHPAD_PROVIDER_API_KEY", "provider-key")
	t.Setenv("LAUNCHPAD_DISABLE_KEYCHAIN", "1")
	token, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if token != "provider-key" {
		t.Fatalf("Resolve() = %q", token)
	}
}
