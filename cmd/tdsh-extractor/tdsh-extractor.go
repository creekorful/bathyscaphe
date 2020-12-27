package main

import (
	"github.com/creekorful/trandoshan/internal/extractor"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&extractor.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
