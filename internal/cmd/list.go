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

func listChangesREST(cfg *config.Config, query string, limit int) ([]map[string]interface{}, error) {
	client := gerrit.NewRESTClient(cfg)
	encodedQuery := url.QueryEscape(query)
	return client.ListChanges(encodedQuery, limit)
}

func listChangesSSH(cfg *config.Config, query string, limit int) ([]map[string]interface{}, error) {
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

func displaySimpleChanges(changes []map[string]interface{}) {
	headers := []string{"Change", "Subject", "Status", "Updated"}
	var rows [][]string
	
	for _, change := range changes {
		changeNum := getStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getStringValue(change, "number")
		}
		
		subject := getStringValue(change, "subject")
		subject = utils.TruncateString(subject, 60)
		
		status := getStringValue(change, "status")
		status = utils.FormatChangeStatus(status)
		
		updated := getStringValue(change, "updated")
		if updated == "" {
			updated = getStringValue(change, "lastUpdated")
		}
		updated = utils.FormatTimeAgo(updated)
		
		rows = append(rows, []string{
			utils.BoldCyan(changeNum),
			subject,
			status,
			updated,
		})
	}
	
	fmt.Print(utils.FormatTable(headers, rows, 2))
}

func displayDetailedChanges(changes []map[string]interface{}) {
	for i, change := range changes {
		if i > 0 {
			fmt.Println()
		}
		
		changeNum := getStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getStringValue(change, "number")
		}
		
		subject := getStringValue(change, "subject")
		status := getStringValue(change, "status")
		updated := getStringValue(change, "updated")
		if updated == "" {
			updated = getStringValue(change, "lastUpdated")
		}
		
		project := getStringValue(change, "project")
		branch := getStringValue(change, "branch")
		owner := getOwnerName(change)
		
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

func getStringValue(data map[string]interface{}, key string) string {
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

func getOwnerName(change map[string]interface{}) string {
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