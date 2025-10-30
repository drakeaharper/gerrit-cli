package main

import (
	"os"

	"github.com/drakeaharper/gerrit-cli/internal/cmd"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if err := cmd.Execute(Version, BuildTime); err != nil {
		utils.ExitWithError(err)
		os.Exit(1)
	}
}
