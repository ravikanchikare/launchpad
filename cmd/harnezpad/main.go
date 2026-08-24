package main

import (
	"harnezpad/internal/app"
	"harnezpad/internal/cli"
)

func main() {
	if cli.Run() {
		return
	}
	app.RunDesktop()
}
