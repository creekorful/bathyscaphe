package main

import (
	"github.com/creekorful/trandoshan/internal/configapi"
	"os"
)

func main() {
	app := configapi.GetApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
