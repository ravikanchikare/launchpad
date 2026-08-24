package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"harnezpad/internal/update"
)

func TestNativeSessionTokenIsRandom256Bits(t *testing.T) {
	first, err := newNativeSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newNativeSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("consecutive session tokens matched")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("token entropy bytes = %d, want 32", len(decoded))
	}
}

func TestNativeBearerAuthentication(t *testing.T) {
	const token = "test-native-session-token"
	handler := requireNativeBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for name, authorization := range map[string]string{
		"missing":    "",
		"wrong":      "Bearer wrong",
		"wrong type": token,
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://helper.invalid/api/cli-status", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", authorization)
			recorder := newResponseRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.status, http.StatusUnauthorized)
			}
			if recorder.header.Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q", recorder.header.Get("WWW-Authenticate"))
			}
		})
	}

	req, err := http.NewRequest(http.MethodGet, "http://helper.invalid/api/cli-status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := newResponseRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.status != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.status, http.StatusNoContent)
	}
	if recorder.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", recorder.header.Get("Cache-Control"))
	}
}

func TestNativeCORSPolicy(t *testing.T) {
	const token = "test-native-session-token"
	called := 0
	handler := nativeCORSMiddleware(requireNativeBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusNoContent)
	})))

	for _, origin := range []string{"zero://app", "http://127.0.0.1:5173"} {
		t.Run("preflight "+origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/settings", nil)
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", http.MethodPut)
			req.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
			req.Header.Set("Access-Control-Request-Private-Network", "true")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Fatalf("allow origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
			}
			if rec.Header().Get("Access-Control-Allow-Private-Network") != "true" {
				t.Fatalf("allow private network = %q", rec.Header().Get("Access-Control-Allow-Private-Network"))
			}
		})
	}
	if called != 0 {
		t.Fatalf("preflight reached API handler %d times", called)
	}

	t.Run("trusted actual request still requires bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		req.Header.Set("Origin", "zero://app")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "zero://app" {
			t.Fatalf("allow origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("trusted actual request with bearer succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		req.Header.Set("Origin", "zero://app")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent || called != 1 {
			t.Fatalf("status = %d called=%d", rec.Code, called)
		}
	})

	for name, configure := range map[string]func(*http.Request){
		"untrusted origin": func(req *http.Request) {
			req.Header.Set("Origin", "https://example.com")
		},
		"untrusted header": func(req *http.Request) {
			req.Header.Set("Origin", "zero://app")
			req.Header.Set("Access-Control-Request-Headers", "authorization, x-untrusted")
		},
		"untrusted method": func(req *http.Request) {
			req.Header.Set("Origin", "zero://app")
			req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/settings", nil)
			req.Header.Set("Access-Control-Request-Method", http.MethodGet)
			req.Header.Set("Access-Control-Request-Headers", "authorization")
			configure(req)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code < 400 {
				t.Fatalf("status = %d", rec.Code)
			}
		})
	}

	t.Run("originless native request remains supported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent || called != 2 {
			t.Fatalf("status = %d called=%d", rec.Code, called)
		}
	})
}

func TestNativeHelperReadyHandshakeAndAuthenticatedAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runNativeHelper(ctx, nativeHelperOptions{
			parentPID:  os.Getpid(),
			stdout:     stdoutWriter,
			stderr:     &stderr,
			parentPoll: 10 * time.Millisecond,
			statusPoll: time.Hour,
			initializeManager: func(manager *Manager, _ *log.Logger) CLIStatus {
				manager.Settings = Settings{GatewayURL: DefaultGateway}
				manager.Updater = update.NewAppUpdater("dev", "")
				return CLIStatus{Installed: true, Path: "/tmp/harnezpad"}
			},
		})
	}()

	line, err := bufio.NewReader(stdoutReader).ReadString('\n')
	if err != nil {
		t.Fatalf("read ready event: %v", err)
	}
	var ready nativeHelperEvent
	if err := json.Unmarshal([]byte(line), &ready); err != nil {
		t.Fatalf("decode ready event %q: %v", line, err)
	}
	if ready.Type != "ready" || !strings.HasPrefix(ready.URL, "http://127.0.0.1:") || ready.Token == "" {
		t.Fatalf("ready event = %+v", ready)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 3 || raw["type"] != "ready" || raw["url"] != ready.URL || raw["token"] != ready.Token {
		t.Fatalf("first stdout object = %v", raw)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	preflight, err := http.NewRequest(http.MethodOptions, ready.URL+"/api/settings", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "zero://app")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "authorization")
	preflightResponse, err := client.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	preflightResponse.Body.Close()
	if preflightResponse.StatusCode != http.StatusNoContent || preflightResponse.Header.Get("Access-Control-Allow-Origin") != "zero://app" {
		t.Fatalf("preflight status/origin = %d/%q", preflightResponse.StatusCode, preflightResponse.Header.Get("Access-Control-Allow-Origin"))
	}

	unauthorized, err := client.Get(ready.URL + "/api/cli-status")
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthorized.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, ready.URL+"/api/cli-status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+ready.Token)
	req.Header.Set("Origin", "zero://app")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authenticated status = %d", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != "zero://app" {
		t.Fatalf("authenticated allow origin = %q", response.Header.Get("Access-Control-Allow-Origin"))
	}
	var status CLIStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Installed || status.Path != "/tmp/harnezpad" {
		t.Fatalf("CLI status = %+v", status)
	}

	cancel()
	stdoutReader.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("helper shutdown: %v; stderr: %s", err, stderr.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("helper did not shut down after cancellation")
	}
}

func TestMonitorParentDetectsExit(t *testing.T) {
	command := exec.Command("/bin/sh", "-c", "sleep 0.05")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	exited := monitorParent(ctx, command.Process.Pid, 10*time.Millisecond)
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-ctx.Done():
		t.Fatal("parent exit was not detected")
	}
}

type responseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	return r.body.Write(body)
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
}
