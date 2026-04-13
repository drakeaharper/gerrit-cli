package cmd

import (
	"fmt"
	"net/url"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	detailed   bool
	reviewer   bool
	listLimit  int
	listStatus string
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
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 25, "Maximum number of changes to show")
	listCmd.Flags().StringVar(&listStatus, "status", "open", "Filter by status (open, merged, abandoned)")
}

func runList(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	// Build query based on flags
	var query string
	if reviewer {
		query = fmt.Sprintf("reviewer:%s status:%s", cfg.User, listStatus)
	} else {
		query = fmt.Sprintf("owner:%s status:%s", cfg.User, listStatus)
	}

	utils.Debugf("Query: %s", query)

	// Try REST API first, fall back to SSH if needed
	changes, err := listChangesREST(cfg, query, listLimit)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		changes, err = listChangesSSH(cfg, query, listLimit)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to list changes: %w", err))
		}
	}

	if len(changes) == 0 {
		if reviewer {
			fmt.Println("No changes found that need your review.")
		} else {
			fmt.Println("No changes found.")
		}
		return
	}

	// Display results
	if detailed {
		displayDetailedChanges(changes)
	} else {
		displaySimpleChanges(changes)
	}
}

func listChangesREST(cfg *config.Config, query string, limit int) ([]gerrit.Change, error) {
	client := gerrit.NewRESTClient(cfg)
	encodedQuery := url.QueryEscape(query)
	return client.ListChanges(encodedQuery, limit)
}

func listChangesSSH(cfg *config.Config, query string, limit int) ([]gerrit.Change, error) {
	client := gerrit.NewSSHClient(cfg)

	output, err := client.ExecuteCommandArgs("query", "--format=JSON", "--current-patch-set", fmt.Sprintf("limit:%d", limit), query)
	if err != nil {
		return nil, err
	}

	return parseSSHChanges(output), nil
}

func displaySimpleChanges(changes []gerrit.Change) {
	headers := []string{"Change", "Subject", "CR", "QR", "LR", "Verified", "Updated"}
	var rows [][]string

	for _, change := range changes {
		rows = append(rows, []string{
			utils.BoldCyan(change.ChangeNumberStr()),
			utils.TruncateString(change.Subject, 60),
			getLabelStatus(change, "Code-Review"),
			getLabelStatus(change, "QA-Review"),
			getLabelStatus(change, "Lint-Review"),
			getLabelStatus(change, "Verified"),
			utils.FormatTimeAgo(change.UpdatedTime()),
		})
	}

	fmt.Print(utils.FormatTable(headers, rows, 2))
}
