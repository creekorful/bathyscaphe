package main

import (
	"github.com/darkspot-org/bathyscaphe/internal/configapi"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&configapi.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
