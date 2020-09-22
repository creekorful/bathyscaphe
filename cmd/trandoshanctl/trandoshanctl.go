package main

import (
	"github.com/creekorful/trandoshan/internal/trandoshanctl"
	"os"
)

func main() {
	app := trandoshanctl.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
