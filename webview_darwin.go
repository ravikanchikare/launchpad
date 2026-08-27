//go:build darwin

package main

import (
	"fmt"
	"os"

	webview "github.com/webview/webview_go"
)

func runUI(url string) {
	// Allow override to browser for testing
	if os.Getenv("LAUNCHPAD_BROWSER") == "1" {
		openBrowser(url)
		select {}
	}
	w := webview.New(false)
	if w == nil {
		fmt.Fprintln(os.Stderr, "failed to create webview, falling back to browser")
		openBrowser(url)
		select {}
	}
	defer w.Destroy()
	w.SetTitle("Launchpad")
	w.SetSize(1100, 720, webview.HintNone)
	// Centered, resizable, with dev tools if needed
	w.Navigate(url)
	// Run blocks until window is closed
	w.Run()
	// When window closes, exit
	os.Exit(0)
}
