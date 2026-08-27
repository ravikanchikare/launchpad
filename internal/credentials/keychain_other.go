//go:build !darwin

package credentials

import "errors"

func loadKeychain(string) string { return "" }

func SaveToKeychain(string, string) error {
	return errors.New("Keychain storage is only supported on macOS")
}
