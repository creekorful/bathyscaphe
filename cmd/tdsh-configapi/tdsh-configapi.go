package main

import (
	"github.com/creekorful/trandoshan/internal/configapi"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&configapi.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
