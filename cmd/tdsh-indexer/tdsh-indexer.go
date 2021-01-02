package main

import (
	"github.com/creekorful/trandoshan/internal/indexer"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&indexer.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
