package main

import (
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/creekorful/trandoshan/internal/scheduler"
	"os"
)

func main() {
	app := process.MakeApp(&scheduler.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
