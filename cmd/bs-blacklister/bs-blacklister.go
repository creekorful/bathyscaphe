package main

import (
	"github.com/darkspot-org/bathyscaphe/internal/blacklister"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&blacklister.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
