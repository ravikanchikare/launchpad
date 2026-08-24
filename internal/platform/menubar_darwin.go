//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include "menubar_darwin.h"
#include <stdlib.h>
*/
import "C"

import "unsafe"

func InstallMenuBar(window unsafe.Pointer) {
	C.harnezpadInstallMenuBar(window)
}

func UninstallMenuBar() {
	C.harnezpadUninstallMenuBar()
}

func showAboutFromMenu() {
	C.harnezpadPresentAbout()
}

func showHelpFromMenu() {
	ShowHelpFromMenu()
}

//export harnezpadShowSettings
func harnezpadShowSettings() {
	ShowSettingsFromMenu()
}

//export harnezpadCheckForUpdates
func harnezpadCheckForUpdates() {
	if CheckUpdatesHandler != nil {
		CheckUpdatesHandler()
	}
}

//export harnezpadShowAbout
func harnezpadShowAbout() {
	showAboutFromMenu()
}

//export harnezpadShowHelp
func harnezpadShowHelp() {
	showHelpFromMenu()
}

func showNativeUpdateAlert(title, message string) {
	cTitle := C.CString(title)
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cMessage))
	C.harnezpadPresentUpdateAlert(cTitle, cMessage)
}

func showNativeUpdateConfirm(title, message string) bool {
	cTitle := C.CString(title)
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cMessage))
	return C.harnezpadPresentUpdateConfirm(cTitle, cMessage) != 0
}

//export harnezpadQuit
func harnezpadQuit() {
	QuitFromMenu()
}

//export harnezpadToggleSidebar
func harnezpadToggleSidebar() {
	ToggleSidebarFromNative()
}

// DebugToggleSidebar invokes the native titlebar sidebar toggle (for crash tests).
func DebugToggleSidebar() {
	ToggleSidebarFromNative()
}
