package main

import (
	"os"

	"github.com/drakeaharper/gerrit-cli/internal/cmd"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if err := cmd.Execute(Version, BuildTime); err != nil {
		os.Exit(1)
	}
}
