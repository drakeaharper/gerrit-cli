package cmd

import (
	"fmt"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var retriggerCmd = &cobra.Command{
	Use:   "retrigger <change-id>",
	Short: "Retrigger Canvas LMS build for a change",
	Long:  `Posts a comment with __TRIGGER_CANVAS_LMS__ to retrigger the build pipeline.`,
	Args:  cobra.ExactArgs(1),
	RunE: runRetrigger,
}

func runRetrigger(cmd *cobra.Command, args []string) error {
	changeID := args[0]

	// Validate change ID
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	utils.Debugf("Retriggering build for change %s", changeID)

	client := gerrit.NewRESTClient(cfg)

	// Post the trigger comment
	if err := client.PostReview(changeID, "current", "__TRIGGER_CANVAS_LMS__"); err != nil {
		return fmt.Errorf("failed to post retrigger comment: %w", err)
	}

	utils.Info("Build retrigger comment posted successfully")
	return nil
}
