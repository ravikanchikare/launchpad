package app

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestSFSymbolHandlerRejectsUnknown(t *testing.T) {
	mux := http.NewServeMux()
	registerSFSymbolHandler(mux)

	req := httptest.NewRequest(http.MethodGet, "/ui/sfsymbol/not-allowed.png", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPrewarmSFSymbols(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := PrewarmSFSymbols(); err != nil {
		t.Fatal(err)
	}
	png, ok := lookupSFSymbolPNG("gearshape", 18)
	if !ok || len(png) == 0 {
		t.Fatal("expected prewarmed gearshape icon")
	}

	mux := http.NewServeMux()
	registerSFSymbolHandler(mux)
	req := httptest.NewRequest(http.MethodGet, "/ui/sfsymbol/gearshape.png?size=18", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
