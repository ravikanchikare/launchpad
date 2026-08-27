//go:build !darwin

package main

func runUI(url string) {
	openBrowser(url)
	select {}
}
