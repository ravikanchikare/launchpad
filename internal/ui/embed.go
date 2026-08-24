package ui

import (
	"embed"
	"io/fs"
)

//go:embed ui.html css js
var content embed.FS

// HTML is the embedded launcher shell used for version-label injection.
var HTML string

func init() {
	data, err := content.ReadFile("ui.html")
	if err != nil {
		panic("read embedded ui.html: " + err.Error())
	}
	HTML = string(data)
}

// Static serves css/ and js/ from the embedded UI bundle.
func Static() fs.FS {
	sub, err := fs.Sub(content, ".")
	if err != nil {
		panic("ui static filesystem: " + err.Error())
	}
	return sub
}
