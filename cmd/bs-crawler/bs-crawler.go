package main

import (
	"github.com/creekorful/bathyscaphe/internal/crawler"
	"github.com/creekorful/bathyscaphe/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&crawler.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
