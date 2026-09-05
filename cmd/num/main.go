package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/JangoCG/nummion/internal/cmd"
)

var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		}
	}
	if err := cmd.NewRootCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(cmd.ExitCode(err))
	}
}
