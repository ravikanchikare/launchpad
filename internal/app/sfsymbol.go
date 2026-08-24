package app

import (
	"net/http"
	"strconv"
	"strings"
	"sync"

	"harnezpad/internal/platform"
)

var (
	sfsymbolCache    sync.Map
	sfsymbolPrewarmSz = []int{14, 15, 18, 20}
	allowedSFSymbols = map[string]struct{}{
		"sidebar.left":        {},
		"sidebar.leading":     {},
		"terminal":            {},
		"square.stack.3d.up":  {},
		"questionmark.circle": {},
		"key":                 {},
		"gearshape":           {},
		"doc.on.doc":          {},
		"arrow.down.circle":   {},
		"arrow.right.circle":  {},
		"arrow.right":         {},
		"chevron.left":        {},
		"plus":                {},
		"checkmark":           {},
		"pencil":              {},
		"ellipsis":            {},
	}
)

// PrewarmSFSymbols renders UI icons on the AppKit main thread before WebView load.
// HTTP handlers must only serve cached PNGs; AppKit is not thread-safe.
func PrewarmSFSymbols() error {
	for name := range allowedSFSymbols {
		for _, size := range sfsymbolPrewarmSz {
			if _, err := renderSFSymbolPNG(name, size); err != nil {
				return err
			}
		}
	}
	return nil
}

func registerSFSymbolHandler(mux *http.ServeMux) {
	mux.HandleFunc("/ui/sfsymbol/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/ui/sfsymbol/")
		name = strings.TrimSuffix(name, ".png")
		if _, ok := allowedSFSymbols[name]; !ok {
			http.NotFound(w, r)
			return
		}
		size := 18
		if raw := strings.TrimSpace(r.URL.Query().Get("size")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 12 && parsed <= 48 {
				size = parsed
			}
		}
		png, ok := lookupSFSymbolPNG(name, size)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(png)
	})
}

func lookupSFSymbolPNG(name string, size int) ([]byte, bool) {
	key := name + ":" + strconv.Itoa(size)
	cached, ok := sfsymbolCache.Load(key)
	if !ok {
		return nil, false
	}
	png, ok := cached.([]byte)
	return png, ok
}

func renderSFSymbolPNG(name string, size int) ([]byte, error) {
	key := name + ":" + strconv.Itoa(size)
	if cached, ok := sfsymbolCache.Load(key); ok {
		return cached.([]byte), nil
	}
	png, err := platform.SFSymbolPNG(name, size)
	if err != nil {
		return nil, err
	}
	sfsymbolCache.Store(key, png)
	return png, nil
}
