package main

import (
	"github.com/creekorful/trandoshan/internal/persister"
	"os"
)

func main() {
	app := persister.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
