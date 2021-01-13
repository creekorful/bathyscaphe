package main

import (
	"github.com/darkspot-org/bathyscaphe/internal/indexer"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&indexer.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
