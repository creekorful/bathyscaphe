package main

import (
	"github.com/creekorful/trandoshan/internal/archiver"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&archiver.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
