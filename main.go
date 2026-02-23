package main

import (
	"fmt"
	"os"

	"github.com/quiet-circles/hyperlocalise/cmd"
)

var version = ""

func main() {
	if err := cmd.Execute(version); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
