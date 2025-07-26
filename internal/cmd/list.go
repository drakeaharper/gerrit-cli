package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	detailed bool
	reviewer bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List changes",
	Long:  `List your open changes or changes that need your review.`,
	Run:   runList,
}

func init() {
	listCmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed information")
	listCmd.Flags().BoolVar(&reviewer, "reviewer", false, "Show changes that need your review")
}

func runList(cmd *cobra.Command, args []string) {
	fmt.Println("List command not yet implemented")
}