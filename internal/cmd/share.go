package cmd

import (
	"fmt"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	shareReviewers []string
	shareCCs       []string
)

var shareCmd = &cobra.Command{
	Use:   "share <change-id>",
	Short: "Add reviewers or CCs to a change",
	Long: `Add reviewers or CCs to a Gerrit change.

Examples:
  gerry share 12345 -r john.doe
  gerry share 12345 --cc learning-experience
  gerry share 12345 -r alice -r bob --cc my-team`,
	Args: cobra.ExactArgs(1),
	Run:  runShare,
}

func init() {
	shareCmd.Flags().StringArrayVarP(&shareReviewers, "reviewer", "r", nil, "Add reviewer (can be user or group, repeatable)")
	shareCmd.Flags().StringArrayVar(&shareCCs, "cc", nil, "Add CC (can be user or group, repeatable)")
}

func runShare(cmd *cobra.Command, args []string) {
	changeID := args[0]

	if len(shareReviewers) == 0 && len(shareCCs) == 0 {
		utils.ExitWithError(fmt.Errorf("at least one --reviewer (-r) or --cc is required"))
	}

	if err := utils.ValidateChangeID(changeID); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid change ID: %w", err))
	}

	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	client := gerrit.NewRESTClient(cfg)

	// Add reviewers
	for _, reviewer := range shareReviewers {
		utils.Debugf("Adding reviewer %s to change %s", reviewer, changeID)
		if err := client.AddReviewer(changeID, reviewer, "REVIEWER"); err != nil {
			utils.ExitWithError(fmt.Errorf("failed to add reviewer %s: %w", reviewer, err))
		}
		utils.Infof("Added reviewer: %s", reviewer)
	}

	// Add CCs
	for _, cc := range shareCCs {
		utils.Debugf("Adding CC %s to change %s", cc, changeID)
		if err := client.AddReviewer(changeID, cc, "CC"); err != nil {
			utils.ExitWithError(fmt.Errorf("failed to add CC %s: %w", cc, err))
		}
		utils.Infof("Added CC: %s", cc)
	}
}
