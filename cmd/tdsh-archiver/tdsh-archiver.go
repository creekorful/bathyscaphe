package main

import (
	"github.com/creekorful/trandoshan/internal/archiver"
	"os"
)

func main() {
	app := archiver.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
