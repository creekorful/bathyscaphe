package main

import (
	"github.com/creekorful/trandoshan/internal/scheduler"
	"os"
)

func main() {
	app := scheduler.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
