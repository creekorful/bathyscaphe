package main

import (
	"github.com/creekorful/trandoshan/internal/api"
	"os"
)

func main() {
	app := api.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
