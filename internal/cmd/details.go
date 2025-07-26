package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var detailsCmd = &cobra.Command{
	Use:   "details <change-id>",
	Short: "Show change details",
	Long:  `Show comprehensive information about a change including files, reviewers, and scores.`,
	Args:  cobra.ExactArgs(1),
	Run:   runDetails,
}

func runDetails(cmd *cobra.Command, args []string) {
	changeID := args[0]
	fmt.Printf("Details command not yet implemented for change %s\n", changeID)
}