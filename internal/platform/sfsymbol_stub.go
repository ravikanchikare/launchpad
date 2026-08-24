//go:build !darwin

package platform

import "errors"

var ErrSFSymbolUnavailable = errors.New("SF Symbols are only available on macOS")

func SFSymbolPNG(string, int) ([]byte, error) {
	return nil, ErrSFSymbolUnavailable
}
