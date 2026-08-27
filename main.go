package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"launchpad/internal/cli"
	"launchpad/internal/server"
	"launchpad/internal/store"
)

//go:embed ui/dist
var uiDist embed.FS

func main() {
	devMode := false
	for _, a := range os.Args[1:] {
		if a == "-dev" || a == "--dev" {
			devMode = true
			break
		}
	}
	if devMode {
		filtered := []string{os.Args[0]}
		for _, a := range os.Args[1:] {
			if a != "-dev" && a != "--dev" {
				filtered = append(filtered, a)
			}
		}
		os.Args = filtered
	}

	if handled, code := cli.Run(context.Background(), os.Args[0], os.Args[1:], cli.IO{
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
	}); handled {
		if code != 0 {
			os.Exit(code)
		}
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	st := &store.Store{}
	if p := strings.TrimSpace(os.Getenv("LAUNCHPAD_DB_PATH")); p != "" {
		st.DBPath = p
	}
	if devMode {
		if p := strings.TrimSpace(os.Getenv("LAUNCHPAD_APP_DB_PATH")); p != "" {
			st.DBPath = p
		}
	}

	srv := &server.Server{Store: st, Logger: slog.Default()}

	var ln net.Listener
	var err error
	if devMode {
		ln, err = net.Listen("tcp", "127.0.0.1:3001")
		if err != nil {
			fmt.Fprintf(os.Stderr, "devMode requires port 3001 free: %v\n", err)
			os.Exit(1)
		}
		found := false
		for _, addr := range []string{"127.0.0.1:5173", "localhost:5173"} {
			c, err := net.Dial("tcp", addr)
			if err == nil {
				c.Close()
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintln(os.Stderr, "Vite dev server not running on http://localhost:5173 — run `npm run dev` in ui/ first")
			os.Exit(1)
		}
		fmt.Println("Launchpad dev mode: API on http://127.0.0.1:3001, UI on http://localhost:5173")
	} else {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
	}
	port := ln.Addr().(*net.TCPAddr).Port
	srv.ClaudeGatewayURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	handler := srv.Handler()
	mux := http.NewServeMux()
	mux.Handle("/api/", handler)
	mux.Handle("/v1/", handler)
	if !devMode {
		dist, _ := fs.Sub(uiDist, "ui/dist")
		fileServer := http.FileServer(http.FS(dist))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				if _, err := fs.Stat(dist, strings.TrimPrefix(r.URL.Path, "/")); err == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
			data, err := fs.ReadFile(dist, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://localhost:5173"+r.URL.String(), http.StatusTemporaryRedirect)
		})
	}

	httpSrv := &http.Server{Handler: mux}

	fmt.Printf("Launchpad listening on http://127.0.0.1:%d\n", port)
	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	var url string
	if devMode {
		url = "http://localhost:5173/"
	} else {
		url = fmt.Sprintf("http://127.0.0.1:%d/", port)
	}
	if os.Getenv("LAUNCHPAD_HEADLESS") == "1" {
		select {}
	}
	runUI(url)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
	fmt.Printf("Opened %s\n", url)
}
