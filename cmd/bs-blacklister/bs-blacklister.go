package main

import (
	"github.com/creekorful/bathyscaphe/internal/blacklister"
	"github.com/creekorful/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&blacklister.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
