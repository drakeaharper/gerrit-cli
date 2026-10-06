package cmd

import (
	"fmt"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var unshareReviewers []string

var unshareCmd = &cobra.Command{
	Use:   "unshare <change-id>...",
	Short: "Remove reviewers or CCs from changes (defaults to you)",
	Long: `Remove reviewers or CCs from one or more Gerrit changes.

With no --reviewer, removes you. Removing a reviewer also deletes any votes
they cast on the change.

Examples:
  gerry unshare 12345
  gerry unshare 12345 23456 34567
  gerry unshare 12345 -r john.doe -r learning-experience`,
	Args: cobra.MinimumNArgs(1),
	RunE: runUnshare,
}

func init() {
	unshareCmd.Flags().StringArrayVarP(&unshareReviewers, "reviewer", "r", nil, "Reviewer or CC to remove (repeatable, default: you)")
}

func runUnshare(cmd *cobra.Command, args []string) error {
	for _, changeID := range args {
		if err := utils.ValidateChangeID(changeID); err != nil {
			return fmt.Errorf("invalid change ID %s: %w", changeID, err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	client := gerrit.NewRESTClient(cfg)

	accounts := unshareReviewers
	if len(accounts) == 0 {
		accounts = []string{"self"}
	}

	failed := 0
	for _, changeID := range args {
		for _, account := range accounts {
			utils.Debugf("Removing %s from change %s", account, changeID)
			if err := client.RemoveReviewer(changeID, account); err != nil {
				utils.Errorf("Failed to remove %s from %s: %v", account, changeID, err)
				failed++
				continue
			}
			utils.Infof("Removed %s from %s", account, changeID)
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d removal(s) failed", failed)
	}
	return nil
}
