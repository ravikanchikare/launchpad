package keys

import "testing"

func TestSlugify(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"HarnezPad Launcher", "harnezpad-launcher"},
		{"Management Key", "management-key"},
		{"my_key-2", "my-key-2"},
		{"  Mixed Case  ", "mixed-case"},
	} {
		if got := Slugify(tc.in); got != tc.want {
			t.Fatalf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	if err := ValidateSlug("harnezpad"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSlug("Bad Name"); err == nil {
		t.Fatal("expected invalid slug")
	}
}
