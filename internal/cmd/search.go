package cmd

import (
	"fmt"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	searchLimit    int
	searchDetailed bool
)

var searchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search changes with a raw Gerrit query",
	Long: `Search changes using a raw Gerrit search query.

The query is passed through to Gerrit unchanged, so any Gerrit search
operator is supported.

Examples:
  gerry search "owner:ashafovaloff@instructure.com status:open i18next migrate label:Verified+1"
  gerry search "project:canvas-lms status:merged"
  gerry search 'message:"fix flaky" status:open'`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 25, "Maximum number of changes to show")
	searchCmd.Flags().BoolVar(&searchDetailed, "detailed", false, "Show detailed information")
}

func runSearch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	query := strings.Join(args, " ")
	utils.Debugf("Query: %s", query)

	// Try REST API first, fall back to SSH if needed
	changes, err := listChangesREST(cfg, query, searchLimit)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		changes, err = listChangesSSH(cfg, query, searchLimit)
		if err != nil {
			return fmt.Errorf("failed to search changes: %w", err)
		}
	}

	if len(changes) == 0 {
		fmt.Println("No changes found.")
		return nil
	}

	if searchDetailed {
		displayDetailedChanges(changes)
	} else {
		displaySimpleChanges(changes)
	}
	return nil
}
