//go:build !darwin

package platform

import "errors"

func LoadGatewayToken() string { return "" }

func SaveGatewayToken(string) error {
	return errors.New("secure token storage is only supported on macOS")
}

func LoadNamedKey(string) string { return "" }

func SaveNamedKey(string, string) error {
	return errors.New("secure token storage is only supported on macOS")
}

func DeleteNamedKey(string) error {
	return errors.New("secure token storage is only supported on macOS")
}
