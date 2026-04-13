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
	teamDetailed    bool
	teamLimit       int
	teamStatus      string
	teamAllVerified bool
	teamFilter      string
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Show changes where you are a reviewer or CC'd",
	Long:  `Show changes where you are either a reviewer or CC'd on.`,
	Run:   runTeam,
}

func init() {
	teamCmd.Flags().BoolVar(&teamDetailed, "detailed", false, "Show detailed information")
	teamCmd.Flags().IntVarP(&teamLimit, "limit", "n", 25, "Maximum number of changes to show")
	teamCmd.Flags().StringVar(&teamStatus, "status", "open", "Filter by status (open, merged, abandoned)")
	teamCmd.Flags().BoolVar(&teamAllVerified, "all-verified", false, "Include changes with all verified states (default: only Verified+1)")
	teamCmd.Flags().StringVarP(&teamFilter, "filter", "f", "", "Additional Gerrit query filter (e.g., 'ownerin:learning-experience' or '-owner:user@example.com')")
}

func runTeam(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	// Build query to find changes where user is reviewer or CC'd
	// Using the same query patterns as Gerrit web UI, but exclude merged by default
	// By default, only show Verified+1 changes unless --all-verified is specified
	verifiedFilter := ""
	if !teamAllVerified {
		verifiedFilter = " label:Verified=1"
	}

	var query string
	if teamStatus == "open" {
		// CC query: is:open -is:ignored -is:wip cc:self
		// Reviewer query: is:open -owner:self -is:wip -is:ignored reviewer:self
		// Both exclude merged changes
		query = fmt.Sprintf("(is:open -is:ignored -is:wip -status:merged%s cc:%s OR is:open -owner:%s -is:wip -is:ignored -status:merged%s reviewer:%s)",
			verifiedFilter, cfg.User, cfg.User, verifiedFilter, cfg.User)
	} else if teamStatus == "merged" {
		// Allow merged changes if explicitly requested
		query = fmt.Sprintf("(status:merged%s cc:%s OR status:merged%s reviewer:%s)", verifiedFilter, cfg.User, verifiedFilter, cfg.User)
	} else {
		// For abandoned or other statuses, exclude merged
		query = fmt.Sprintf("(status:%s -status:merged%s cc:%s OR status:%s -status:merged%s reviewer:%s)", teamStatus, verifiedFilter, cfg.User, teamStatus, verifiedFilter, cfg.User)
	}

	// Append custom filter if provided
	if teamFilter != "" {
		query = fmt.Sprintf("(%s) %s", query, teamFilter)
	}

	utils.Debugf("Query: %s", query)

	// Try REST API first, fall back to SSH if needed
	changes, err := listTeamChangesREST(cfg, query, teamLimit)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		changes, err = listTeamChangesSSH(cfg, query, teamLimit)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to list changes: %w", err))
		}
	}

	if len(changes) == 0 {
		fmt.Println("No changes found where you are a reviewer or CC'd.")
		return
	}

	// Display results
	if teamDetailed {
		displayDetailedChanges(changes)
	} else {
		displayTeamSimpleChanges(changes)
	}
}

func listTeamChangesREST(cfg *config.Config, query string, limit int) ([]map[string]interface{}, error) {
	client := gerrit.NewRESTClient(cfg)
	encodedQuery := url.QueryEscape(query)
	return client.ListChanges(encodedQuery, limit)
}

func listTeamChangesSSH(cfg *config.Config, query string, limit int) ([]map[string]interface{}, error) {
	client := gerrit.NewSSHClient(cfg)

	output, err := client.ExecuteCommandArgs("query", "--format=JSON", "--current-patch-set", fmt.Sprintf("limit:%d", limit), query)
	if err != nil {
		return nil, err
	}

	return parseSSHChanges(output), nil
}

func displayTeamSimpleChanges(changes []map[string]interface{}) {
	headers := []string{"Change", "Subject", "Owner", "CR", "QR", "LR", "Verified", "Updated"}
	var rows [][]string

	for _, change := range changes {
		changeNum := getStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getStringValue(change, "number")
		}

		subject := getStringValue(change, "subject")
		subject = utils.TruncateString(subject, 45)

		owner := getOwnerName(change)

		updated := getStringValue(change, "updated")
		if updated == "" {
			updated = getStringValue(change, "lastUpdated")
		}
		updated = utils.FormatTimeAgo(updated)

		codeReview := getLabelStatus(change, "Code-Review")
		qr := getLabelStatus(change, "QA-Review")
		lr := getLabelStatus(change, "Lint-Review")
		verified := getLabelStatus(change, "Verified")

		rows = append(rows, []string{
			utils.BoldCyan(changeNum),
			subject,
			owner,
			codeReview,
			qr,
			lr,
			verified,
			updated,
		})
	}

	fmt.Print(utils.FormatTable(headers, rows, 2))
}
