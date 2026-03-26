package main

import (
	"fmt"
	"os"

	"github.com/txperl/snipfly/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
