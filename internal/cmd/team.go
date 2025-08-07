package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	teamDetailed bool
	teamLimit    int
	teamStatus   string
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
	var query string
	if teamStatus == "open" {
		// CC query: is:open -is:ignored -is:wip cc:self
		// Reviewer query: is:open -owner:self -is:wip -is:ignored reviewer:self
		// Both exclude merged changes
		query = fmt.Sprintf("(is:open -is:ignored -is:wip -status:merged cc:%s OR is:open -owner:%s -is:wip -is:ignored -status:merged reviewer:%s)", 
			cfg.User, cfg.User, cfg.User)
	} else if teamStatus == "merged" {
		// Allow merged changes if explicitly requested
		query = fmt.Sprintf("(status:merged cc:%s OR status:merged reviewer:%s)", cfg.User, cfg.User)
	} else {
		// For abandoned or other statuses, exclude merged
		query = fmt.Sprintf("(status:%s -status:merged cc:%s OR status:%s -status:merged reviewer:%s)", teamStatus, cfg.User, teamStatus, cfg.User)
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
		displayTeamDetailedChanges(changes)
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
	
	// Build SSH query command
	sshQuery := fmt.Sprintf("query --format=JSON --current-patch-set limit:%d %s", limit, query)
	output, err := client.ExecuteCommand(sshQuery)
	if err != nil {
		return nil, err
	}

	// Parse JSON lines output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var changes []map[string]interface{}
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var change map[string]interface{}
		if err := utils.ParseJSON([]byte(line), &change); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}
		
		// Skip the stats line
		if _, hasType := change["type"]; hasType {
			continue
		}
		
		changes = append(changes, change)
	}
	
	return changes, nil
}

func displayTeamSimpleChanges(changes []map[string]interface{}) {
	headers := []string{"Change", "Subject", "Status", "Verified", "Updated"}
	var rows [][]string
	
	for _, change := range changes {
		changeNum := getTeamStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getTeamStringValue(change, "number")
		}
		
		subject := getTeamStringValue(change, "subject")
		subject = utils.TruncateString(subject, 60)
		
		status := getTeamStringValue(change, "status")
		status = utils.FormatChangeStatus(status)
		
		updated := getTeamStringValue(change, "updated")
		if updated == "" {
			updated = getTeamStringValue(change, "lastUpdated")
		}
		updated = utils.FormatTimeAgo(updated)
		
		verified := getTeamVerifiedStatus(change)
		
		rows = append(rows, []string{
			utils.BoldCyan(changeNum),
			subject,
			status,
			verified,
			updated,
		})
	}
	
	fmt.Print(utils.FormatTable(headers, rows, 2))
}

func displayTeamDetailedChanges(changes []map[string]interface{}) {
	for i, change := range changes {
		if i > 0 {
			fmt.Println()
		}
		
		changeNum := getTeamStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getTeamStringValue(change, "number")
		}
		
		subject := getTeamStringValue(change, "subject")
		status := getTeamStringValue(change, "status")
		updated := getTeamStringValue(change, "updated")
		if updated == "" {
			updated = getTeamStringValue(change, "lastUpdated")
		}
		
		project := getTeamStringValue(change, "project")
		branch := getTeamStringValue(change, "branch")
		owner := getTeamOwnerName(change)
		
		fmt.Printf("%s %s\n", utils.BoldCyan("Change:"), utils.BoldWhite(changeNum))
		fmt.Printf("%s %s\n", utils.BoldCyan("Subject:"), subject)
		fmt.Printf("%s %s\n", utils.BoldCyan("Status:"), utils.FormatChangeStatus(status))
		fmt.Printf("%s %s\n", utils.BoldCyan("Project:"), project)
		fmt.Printf("%s %s\n", utils.BoldCyan("Branch:"), branch)
		fmt.Printf("%s %s\n", utils.BoldCyan("Owner:"), owner)
		fmt.Printf("%s %s\n", utils.BoldCyan("Updated:"), utils.FormatTimeAgo(updated))
		
		// Show review scores if available
		if labels, ok := change["labels"].(map[string]interface{}); ok {
			fmt.Printf("%s ", utils.BoldCyan("Reviews:"))
			var scores []string
			for label, data := range labels {
				if labelData, ok := data.(map[string]interface{}); ok {
					if approved, ok := labelData["approved"].(map[string]interface{}); ok {
						if value, ok := approved["value"]; ok {
							scores = append(scores, fmt.Sprintf("%s:%s", label, utils.FormatScore(label, value)))
						}
					} else if rejected, ok := labelData["rejected"].(map[string]interface{}); ok {
						if value, ok := rejected["value"]; ok {
							scores = append(scores, fmt.Sprintf("%s:%s", label, utils.FormatScore(label, value)))
						}
					}
				}
			}
			if len(scores) > 0 {
				fmt.Println(strings.Join(scores, " "))
			} else {
				fmt.Println(utils.Gray("none"))
			}
		}
	}
}


func getTeamStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', 0, 64)
		case int:
			return strconv.Itoa(v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func getTeamOwnerName(change map[string]interface{}) string {
	if owner, ok := change["owner"].(map[string]interface{}); ok {
		if name, ok := owner["name"].(string); ok && name != "" {
			return name
		}
		if username, ok := owner["username"].(string); ok && username != "" {
			return username
		}
		if email, ok := owner["email"].(string); ok && email != "" {
			return email
		}
	}
	return "unknown"
}

func getTeamVerifiedStatus(change map[string]interface{}) string {
	if labels, ok := change["labels"].(map[string]interface{}); ok {
		if verifiedData, exists := labels["Verified"].(map[string]interface{}); exists {
			// The most recent vote determines the status
			// Check if there's an approved reviewer (means +1)
			if _, hasApproved := verifiedData["approved"]; hasApproved {
				return utils.FormatScore("Verified", 1)
			}
			// Check if there's a rejected reviewer (means -1)
			if _, hasRejected := verifiedData["rejected"]; hasRejected {
				return utils.FormatScore("Verified", -1)
			}
			// No approved or rejected status
			return utils.Gray("0")
		}
	}
	// No verification status
	return utils.Gray("—")
}