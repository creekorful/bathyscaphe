package main

import (
	"github.com/creekorful/bathyscaphe/internal/indexer"
	"github.com/creekorful/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&indexer.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
