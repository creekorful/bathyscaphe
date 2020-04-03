package main

import (
	"github.com/creekorful/trandoshan-crawler/internal/feeder"
	"os"
)

func main() {
	app := feeder.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
