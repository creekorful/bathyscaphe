package main

import (
	"github.com/creekorful/trandoshan/internal/crawler"
	"github.com/creekorful/trandoshan/internal/process"
	"os"
)

func main() {
	app := process.MakeApp(&crawler.State{})
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
