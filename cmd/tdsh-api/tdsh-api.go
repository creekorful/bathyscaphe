package main

import (
	"github.com/creekorful/trandoshan/internal/api"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&api.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
