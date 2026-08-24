//go:build darwin

package platform

import (
	"errors"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/webview/webview_go"
)

var activeNativeWindow struct {
	sync.RWMutex
	view webview.WebView
}

func ShowSettingsFromMenu() {
	showViewFromMenu("settings")
}

func ShowHelpFromMenu() {
	showViewFromMenu("help")
}

func ToggleSidebarFromNative() {
	scheduleMenuEval("toggleAppSidebar()")
}

func showViewFromMenu(name string) {
	scheduleMenuEval("showView('" + name + "')")
}

func scheduleMenuEval(script string) {
	activeNativeWindow.RLock()
	w := activeNativeWindow.view
	activeNativeWindow.RUnlock()
	if w == nil {
		return
	}
	// Never call w.Dispatch from a cgo export stack frame; defer to the next
	// goroutine so AppKit toolbar/menu handlers can return before Eval runs.
	go func(view webview.WebView, js string) {
		view.Dispatch(func() {
			view.Eval(js)
		})
	}(w, script)
}

func showViewFromMenuEval(script string) {
	scheduleMenuEval(script)
}

func openExternalURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("HarnezPad can open only valid HTTPS links")
	}
	return exec.Command("/usr/bin/open", raw).Run()
}

func QuitFromMenu() {
	activeNativeWindow.RLock()
	w := activeNativeWindow.view
	activeNativeWindow.RUnlock()
	if w == nil {
		return
	}
	go func(view webview.WebView) {
		view.Dispatch(func() {
			view.Terminate()
		})
	}(w)
}

func RunNativeWindow(addr string, beforeNavigate func() error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w := webview.New(false)
	if w == nil {
		panic("unable to create native webview")
	}
	activeNativeWindow.Lock()
	activeNativeWindow.view = w
	activeNativeWindow.Unlock()
	defer func() {
		UninstallMenuBar()
		activeNativeWindow.Lock()
		activeNativeWindow.view = nil
		activeNativeWindow.Unlock()
		w.Destroy()
	}()
	w.SetTitle("HarnezPad")
	w.SetSize(1030, 760, webview.HintNone)
	InstallMenuBar(w.Window())
	if beforeNavigate != nil {
		if err := beforeNavigate(); err != nil {
			log.Printf("HarnezPad UI prewarm failed: %v", err)
		}
	}
	_ = w.Bind("harnezpadClipboardWrite", func(value string) error {
		cmd := exec.Command("/usr/bin/pbcopy")
		cmd.Stdin = strings.NewReader(value)
		return cmd.Run()
	})
	_ = w.Bind("harnezpadOpenExternal", openExternalURL)
	w.Navigate(addr)
	w.Run()
}

func ShowNativeUpdateAlert(title, message string) {
	showNativeUpdateAlert(title, message)
}

func ShowNativeUpdateConfirm(title, message string) bool {
	return showNativeUpdateConfirm(title, message)
}
