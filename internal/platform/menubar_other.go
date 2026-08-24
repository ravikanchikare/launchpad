//go:build !darwin

package platform

import "unsafe"

func InstallMenuBar(_ unsafe.Pointer)   {}
func UninstallMenuBar()                 {}
func showNativeUpdateAlert(_, _ string) {}
func DebugToggleSidebar()               {}
