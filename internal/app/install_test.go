package app

import "testing"

func TestPackagedCLIPathSupportsLegacyAndNativeBundles(t *testing.T) {
	for name, test := range map[string]struct {
		executable string
		want       string
	}{
		"legacy Cocoa host": {
			executable: "/Applications/HarnezPad.app/Contents/MacOS/HarnezPad",
			want:       "/Applications/HarnezPad.app/Contents/Resources/harnezpad",
		},
		"native helper": {
			executable: "/Applications/HarnezPad.app/Contents/Resources/harnezpad",
			want:       "/Applications/HarnezPad.app/Contents/Resources/harnezpad",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := packagedCLIPath(test.executable)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPackagedCLIPathRejectsUnpackagedHelper(t *testing.T) {
	if _, err := packagedCLIPath("/tmp/harnezpad"); err == nil {
		t.Fatal("expected unpackaged helper path to be rejected")
	}
}
