package main

import (
	"fmt"
	"os"

	"lexware-cli/internal/cmd"
)

var version = "dev"

func main() {
	if err := cmd.NewRootCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(cmd.ExitCode(err))
	}
}
