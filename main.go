package main

import (
	"os"

	"github.com/rogersnm/compass/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
