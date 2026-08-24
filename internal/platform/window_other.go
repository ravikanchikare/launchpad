//go:build !darwin

package platform

import "log"

func ShowSettingsFromMenu() {}

func ShowHelpFromMenu() {}

func QuitFromMenu() {}

func RunNativeWindow(addr string, beforeNavigate func() error) {
	if beforeNavigate != nil {
		if err := beforeNavigate(); err != nil {
			log.Printf("HarnezPad UI prewarm failed: %v", err)
		}
	}
	log.Printf("HarnezPad UI available at %s (native window requires macOS)", addr)
	select {}
}

func ShowNativeUpdateAlert(title, message string) {
	log.Printf("%s: %s", title, message)
}

func ShowNativeUpdateConfirm(title, message string) bool {
	log.Printf("%s: %s (auto-decline on non-macOS)", title, message)
	return false
}

func CheckForUpdatesFromMenu() {
	if CheckUpdatesHandler != nil {
		CheckUpdatesHandler()
	}
}
