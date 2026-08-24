package app

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"harnezpad/assets"
	"harnezpad/internal/platform"
	"harnezpad/internal/ui"
	"harnezpad/internal/update"
	"harnezpad/internal/version"
)

func CurrentVersionLabel() string {
	label := version.Version
	if label != "dev" && !strings.HasPrefix(label, "v") {
		label = "v" + label
	}
	return label
}

func BackgroundContext() context.Context {
	return context.Background()
}

func CheckForUpdatesFromMenu() {
	m := Active
	if m == nil || m.Updater == nil {
		platform.ShowNativeUpdateAlert("HarnezPad", "Update checking is not available yet.")
		return
	}
	go func() {
		info, err := m.Updater.Check(BackgroundContext())
		if err != nil {
			platform.ShowNativeUpdateAlert("HarnezPad Updates", UserFacingError(err))
			return
		}
		if info == nil {
			platform.ShowNativeUpdateAlert("HarnezPad Updates", "HarnezPad is up to date ("+CurrentVersionLabel()+").")
			return
		}
		if err := m.Updater.Download(BackgroundContext(), *info); err != nil {
			platform.ShowNativeUpdateAlert("HarnezPad Updates", "HarnezPad "+info.Version+" is available but could not be downloaded. "+UserFacingError(err))
			return
		}
		message := "HarnezPad " + info.Version + " is ready to install. Restart now?"
		if !platform.ShowNativeUpdateConfirm("HarnezPad Updates", message) {
			return
		}
		if err := m.installPendingUpdate(); err != nil {
			platform.ShowNativeUpdateAlert("HarnezPad Updates", UserFacingError(err))
		}
	}()
}

func (m *Manager) installPendingUpdate() error {
	if m == nil || m.Updater == nil {
		return errors.New("update checking is not available yet")
	}
	status := m.Updater.Status()
	if status.Update == nil || !status.Downloaded {
		return errors.New("no verified update is ready")
	}
	if err := m.Updater.Install(*status.Update); err != nil {
		return err
	}
	if err := m.Updater.LaunchNewApp(); err != nil {
		return err
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}

var page = template.Must(template.New("page").Parse(ui.HTML))

func RunDesktop() {
	m := NewManager()
	Active = m
	defer func() { Active = nil }()

	platform.CheckUpdatesHandler = CheckForUpdatesFromMenu
	platform.CurrentVersionLabel = CurrentVersionLabel

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.Load()
	m.Updater = update.NewAppUpdater(version.Version, version.DefaultUpdateAPI)
	upgraded, err := m.Updater.CleanupAfterUpgrade()
	if err != nil {
		log.Printf("HarnezPad update cleanup failed: %v", err)
	}
	cliStatus := InstallCLI()
	if !cliStatus.Installed {
		log.Printf("CLI installation failed: %s", cliStatus.Error)
	} else if upgraded {
		log.Printf("CLI reinstalled at %s after app update", cliStatus.Path)
	}

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS))))
	registerSFSymbolHandler(mux)
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.FS(ui.Static()))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var rendered bytes.Buffer
		if err := page.Execute(&rendered, nil); err != nil {
			http.Error(w, "render HarnezPad UI", http.StatusInternalServerError)
			return
		}
		brand := `<div class="brand">HarnezPad <span style="color:var(--muted);font-size:11px;font-weight:500;margin-left:7px">` + template.HTMLEscapeString(CurrentVersionLabel()) + `</span></div>`
		body := strings.Replace(rendered.String(), `<div class="brand">HarnezPad</div>`, brand, 1)
		_, _ = io.WriteString(w, body)
	})
	registerAPIHandlers(mux, m, cliStatus)

	srv := &http.Server{Handler: mux}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	addr := "http://" + l.Addr().String()
	log.Println("HarnezPad listening on " + addr)
	go func() {
		if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()
	m.Updater.StartBackground(ctx, func(info update.UpdateInfo) {
		if err := m.Updater.Download(ctx, info); err != nil {
			log.Printf("HarnezPad update download failed: %v", err)
			return
		}
		log.Printf("HarnezPad update %s is ready; use the update controls to restart", info.Version)
	})
	time.Sleep(300 * time.Millisecond)
	platform.RunNativeWindow(addr, PrewarmSFSymbols)

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("HarnezPad HTTP shutdown: %v", err)
	}
	os.Exit(0)
}
