//go:build darwin

package platform

import (
	"bytes"
	"image/png"
	"runtime"
	"testing"
)

func TestSFSymbolPNG(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pngBytes, err := SFSymbolPNG("gearshape", 18)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(pngBytes)); err != nil {
		t.Fatalf("invalid png: %v", err)
	}
}

func TestSFSymbolPNGUnknown(t *testing.T) {
	if _, err := SFSymbolPNG("not.a.real.symbol.name.xyz", 18); err == nil {
		t.Fatal("expected unknown symbol error")
	}
}
