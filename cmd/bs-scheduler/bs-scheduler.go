package main

import (
	"github.com/creekorful/bathyscaphe/internal/process"
	"github.com/creekorful/bathyscaphe/internal/scheduler"
	"os"
)

func main() {
	app := process.MakeApp(&scheduler.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
