//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include "sfsymbol_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

var ErrSFSymbolUnavailable = errors.New("SF Symbol is unavailable")

func SFSymbolPNG(name string, pointSize int) ([]byte, error) {
	if name == "" {
		return nil, ErrSFSymbolUnavailable
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	var length C.size_t
	data := C.HarnezPadSFSymbolPNG(cname, C.int(pointSize), &length)
	if data == nil || length == 0 {
		return nil, fmt.Errorf("%w: %q", ErrSFSymbolUnavailable, name)
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(length)), nil
}
