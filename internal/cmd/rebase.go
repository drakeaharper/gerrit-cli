package cmd

import (
	"fmt"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	rebaseBase           string
	rebaseAllowConflicts bool
)

var rebaseCmd = &cobra.Command{
	Use:   "rebase <change-id>",
	Short: "Rebase a change onto a new base",
	Long: `Rebase a Gerrit change using the server-side rebase API.

By default, rebases the change onto the target branch's HEAD.
Use --base to specify a different base commit or change.

Examples:
  gerry rebase 12345
  gerry rebase 12345 --base main
  gerry rebase 12345 --base abc123def
  gerry rebase 12345 --base 67890~2
  gerry rebase 12345 --allow-conflicts`,
	Args: cobra.ExactArgs(1),
	Run:  runRebase,
}

func init() {
	rebaseCmd.Flags().StringVarP(&rebaseBase, "base", "b", "", "Base to rebase onto (commit SHA, branch, or change~patchset)")
	rebaseCmd.Flags().BoolVar(&rebaseAllowConflicts, "allow-conflicts", false, "Allow rebasing with conflicts (creates conflict markers)")
}

func runRebase(cmd *cobra.Command, args []string) {
	changeID := args[0]

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

	fmt.Printf("Rebasing change %s", utils.BoldCyan(changeID))
	if rebaseBase != "" {
		fmt.Printf(" onto %s", utils.BoldYellow(rebaseBase))
	}
	fmt.Println("...")

	change, err := client.RebaseChange(changeID, rebaseBase, rebaseAllowConflicts)
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to rebase change: %w", err))
	}

	fmt.Printf("%s Change rebased successfully!\n", color.GreenString("✓"))

	// Display some info about the rebased change
	if subject, ok := change["subject"].(string); ok {
		fmt.Printf("Subject: %s\n", subject)
	}
	if revisions, ok := change["revisions"].(map[string]interface{}); ok {
		fmt.Printf("New patchset count: %d\n", len(revisions))
	}
}
