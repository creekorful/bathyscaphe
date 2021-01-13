package main

import (
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/scheduler"
	"os"
)

func main() {
	app := process.MakeApp(&scheduler.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
