package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	showAll bool
)

var commentsCmd = &cobra.Command{
	Use:   "comments <change-id>",
	Short: "View comments on a change",
	Long:  `View review comments on a specific change.`,
	Args:  cobra.ExactArgs(1),
	Run:   runComments,
}

func init() {
	commentsCmd.Flags().BoolVar(&showAll, "all", false, "Show all comments (default: unresolved only)")
}

func runComments(cmd *cobra.Command, args []string) {
	changeID := args[0]
	fmt.Printf("Comments command not yet implemented for change %s\n", changeID)
}