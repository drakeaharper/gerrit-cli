package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <change-id> [patchset]",
	Short: "Fetch a change",
	Long:  `Fetch a change and checkout to FETCH_HEAD.`,
	Args:  cobra.RangeArgs(1, 2),
	Run:   runFetch,
}

func runFetch(cmd *cobra.Command, args []string) {
	changeID := args[0]
	patchset := ""
	if len(args) > 1 {
		patchset = args[1]
	}
	fmt.Printf("Fetch command not yet implemented for change %s patchset %s\n", changeID, patchset)
}