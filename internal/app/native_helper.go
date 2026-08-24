package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"harnezpad/internal/update"
	"harnezpad/internal/version"
)

const (
	defaultParentPollInterval = time.Second
	defaultStatusPollInterval = time.Second
)

var nativeTrustedOrigins = map[string]struct{}{
	"zero://app":            {},
	"http://127.0.0.1:5173": {},
}

var nativeCORSMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodDelete:  {},
	http.MethodOptions: {},
}

type nativeHelperEvent struct {
	Type   string               `json:"type"`
	URL    string               `json:"url,omitempty"`
	Token  string               `json:"token,omitempty"`
	Status *update.UpdateStatus `json:"status,omitempty"`
}

type nativeEventWriter struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func newNativeEventWriter(w io.Writer) *nativeEventWriter {
	return &nativeEventWriter{encoder: json.NewEncoder(w)}
}

func (w *nativeEventWriter) write(event nativeHelperEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.encoder.Encode(event)
}

type nativeHelperOptions struct {
	parentPID         int
	stdout            io.Writer
	stderr            io.Writer
	parentPoll        time.Duration
	statusPoll        time.Duration
	listen            func(network, address string) (net.Listener, error)
	initializeManager func(*Manager, *log.Logger) CLIStatus
}

// RunNativeHelper serves HarnezPad's JSON API for the Native SDK host.
// Stdout is reserved for NDJSON protocol events; all diagnostics are emitted
// on stderr so the host can parse the stream deterministically.
func RunNativeHelper(parentPID int) error {
	return runNativeHelper(context.Background(), nativeHelperOptions{
		parentPID: parentPID,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
	})
}

func runNativeHelper(parent context.Context, options nativeHelperOptions) error {
	if options.parentPID <= 1 {
		return fmt.Errorf("parent PID must be greater than 1")
	}
	if !processAlive(options.parentPID) {
		return fmt.Errorf("parent process %d is not running", options.parentPID)
	}
	if options.stdout == nil {
		options.stdout = io.Discard
	}
	if options.stderr == nil {
		options.stderr = io.Discard
	}
	if options.parentPoll <= 0 {
		options.parentPoll = defaultParentPollInterval
	}
	if options.statusPoll <= 0 {
		options.statusPoll = defaultStatusPollInterval
	}
	if options.listen == nil {
		options.listen = net.Listen
	}
	if options.initializeManager == nil {
		options.initializeManager = initializeNativeManager
	}

	logger := log.New(options.stderr, "harnezpad native helper: ", log.LstdFlags)
	manager := NewManager()
	manager.setChildOutput(options.stderr, options.stderr)
	cliStatus := options.initializeManager(manager, logger)
	if manager.Updater == nil {
		manager.Updater = update.NewAppUpdater(version.Version, version.DefaultUpdateAPI)
	}

	token, err := newNativeSessionToken()
	if err != nil {
		return fmt.Errorf("generate session token: %w", err)
	}
	listener, err := options.listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	registerAPIHandlers(mux, manager, cliStatus)
	server := &http.Server{
		Handler:           nativeCORSMiddleware(requireNativeBearer(token, mux)),
		ErrorLog:          logger,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    16 << 10,
	}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	events := newNativeEventWriter(options.stdout)
	url := "http://" + listener.Addr().String()
	// This exact object is the protocol handshake consumed by the Zig host and
	// must remain the first line written to stdout.
	if err := events.write(nativeHelperEvent{Type: "ready", URL: url, Token: token}); err != nil {
		_ = server.Close()
		return fmt.Errorf("write ready event: %w", err)
	}
	logger.Printf("listening on %s for parent %d", url, options.parentPID)

	startNativeUpdateEvents(ctx, manager.Updater, events, logger, options.statusPoll)
	parentExited := monitorParent(ctx, options.parentPID, options.parentPoll)

	var runErr error
	select {
	case <-ctx.Done():
	case <-parentExited:
		logger.Printf("parent process %d exited; shutting down", options.parentPID)
	case runErr = <-serveErr:
		if runErr != nil {
			runErr = fmt.Errorf("serve native API: %w", runErr)
		}
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shut down native API: %w", err)
	}
	return runErr
}

func nativeCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		if _, trusted := nativeTrustedOrigins[origin]; !trusted {
			writeNativeCORSFailure(w, http.StatusForbidden, "origin not allowed")
			return
		}

		appendVary(w.Header(), "Origin")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if r.Method == http.MethodOptions {
			appendVary(w.Header(), "Access-Control-Request-Method")
			appendVary(w.Header(), "Access-Control-Request-Headers")
			requestedMethod := strings.ToUpper(strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")))
			if requestedMethod == "" {
				writeNativeCORSFailure(w, http.StatusBadRequest, "CORS method is required")
				return
			}
			if _, allowed := nativeCORSMethods[requestedMethod]; !allowed || requestedMethod == http.MethodOptions {
				writeNativeCORSFailure(w, http.StatusMethodNotAllowed, "CORS method not allowed")
				return
			}
			if !nativeCORSHeadersAllowed(r.Header.Get("Access-Control-Request-Headers")) {
				writeNativeCORSFailure(w, http.StatusForbidden, "CORS headers not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			if strings.EqualFold(strings.TrimSpace(r.Header.Get("Access-Control-Request-Private-Network")), "true") {
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if _, allowed := nativeCORSMethods[r.Method]; !allowed {
			writeNativeCORSFailure(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func nativeCORSHeadersAllowed(raw string) bool {
	for _, header := range strings.Split(raw, ",") {
		header = strings.ToLower(strings.TrimSpace(header))
		if header == "" || header == "authorization" || header == "content-type" {
			continue
		}
		return false
	}
	return true
}

func appendVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, item := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func writeNativeCORSFailure(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func initializeNativeManager(manager *Manager, logger *log.Logger) CLIStatus {
	manager.Load()
	manager.Updater = update.NewAppUpdater(version.Version, version.DefaultUpdateAPI)
	upgraded, err := manager.Updater.CleanupAfterUpgrade()
	if err != nil {
		logger.Printf("update cleanup failed: %v", err)
	}
	cliStatus := InstallCLI()
	if !cliStatus.Installed {
		logger.Printf("CLI installation failed: %s", cliStatus.Error)
	} else if upgraded {
		logger.Printf("CLI reinstalled at %s after app update", cliStatus.Path)
	}
	return cliStatus
}

func newNativeSessionToken() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random[:]), nil
}

func requireNativeBearer(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := []byte(r.Header.Get("Authorization"))
		if len(provided) != len(expected) || subtle.ConstantTimeCompare(provided, expected) != 1 {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r)
	})
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func monitorParent(ctx context.Context, parentPID int, interval time.Duration) <-chan struct{} {
	exited := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !processAlive(parentPID) {
					exited <- struct{}{}
					return
				}
			}
		}
	}()
	return exited
}

func startNativeUpdateEvents(ctx context.Context, updater *update.AppUpdater, events *nativeEventWriter, logger *log.Logger, interval time.Duration) {
	updater.StartBackground(ctx, func(info update.UpdateInfo) {
		if err := updater.Download(ctx, info); err != nil {
			logger.Printf("update download failed: %v", err)
			return
		}
		logger.Printf("update %s is ready to install", info.Version)
	})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var previous []byte
		for {
			status := updater.Status()
			encoded, err := json.Marshal(status)
			if err == nil && !equalBytes(encoded, previous) {
				copyStatus := status
				if err := events.write(nativeHelperEvent{Type: "update-status", Status: &copyStatus}); err != nil {
					logger.Printf("write update status event: %v", err)
					return
				}
				previous = append(previous[:0], encoded...)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare(left, right) == 1
}
