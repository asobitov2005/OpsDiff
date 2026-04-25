package main

import (
	"fmt"
	"os"

	"github.com/asobitov2005/OpsDiff/internal/app"
)

func main() {
	if err := app.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
