package main

import (
	"github.com/creekorful/trandoshan/internal/blacklister"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&blacklister.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
