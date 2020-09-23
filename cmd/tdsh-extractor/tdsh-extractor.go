package main

import (
	"github.com/creekorful/trandoshan/internal/extractor"
	"os"
)

func main() {
	app := extractor.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
