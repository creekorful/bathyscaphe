package main

import (
	"github.com/creekorful/bathyscaphe/internal/configapi"
	"github.com/creekorful/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&configapi.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
