package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cherryPickCmd = &cobra.Command{
	Use:   "cherry-pick <change-id> [patchset]",
	Short: "Cherry-pick a change",
	Long:  `Fetch and cherry-pick a change.`,
	Args:  cobra.RangeArgs(1, 2),
	Run:   runCherryPick,
}

func runCherryPick(cmd *cobra.Command, args []string) {
	changeID := args[0]
	patchset := ""
	if len(args) > 1 {
		patchset = args[1]
	}
	fmt.Printf("Cherry-pick command not yet implemented for change %s patchset %s\n", changeID, patchset)
}