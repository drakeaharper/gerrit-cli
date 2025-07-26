package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	showFiles bool
)

var detailsCmd = &cobra.Command{
	Use:   "details <change-id>",
	Short: "Show change details",
	Long:  `Show comprehensive information about a change including files, reviewers, and scores.`,
	Args:  cobra.ExactArgs(1),
	Run:   runDetails,
}

func init() {
	detailsCmd.Flags().BoolVar(&showFiles, "files", false, "Show list of changed files")
}

func runDetails(cmd *cobra.Command, args []string) {
	changeID := args[0]
	
	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	utils.Debugf("Fetching details for change %s", changeID)

	// Try REST API first, fall back to SSH if needed
	change, err := getChangeDetailsREST(cfg, changeID)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		change, err = getChangeDetailsSSH(cfg, changeID)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get change details: %w", err))
		}
	}

	displayChangeDetails(change, showFiles)

	// Show files if requested
	if showFiles {
		fmt.Println()
		displayChangeFiles(cfg, changeID, change)
	}
}

func getChangeDetailsREST(cfg *config.Config, changeID string) (map[string]interface{}, error) {
	client := gerrit.NewRESTClient(cfg)
	return client.GetChange(changeID)
}

func getChangeDetailsSSH(cfg *config.Config, changeID string) (map[string]interface{}, error) {
	client := gerrit.NewSSHClient(cfg)
	
	// Get change details with comments
	output, err := client.GetChangeDetails(changeID)
	if err != nil {
		return nil, err
	}

	// Parse the JSON output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
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
		
		return change, nil
	}
	
	return nil, fmt.Errorf("no valid change data found")
}

func displayChangeDetails(change map[string]interface{}, showFiles bool) {
	// Basic information
	changeNum := getStringValue(change, "_number")
	if changeNum == "" {
		changeNum = getStringValue(change, "number")
	}
	
	subject := getStringValue(change, "subject")
	status := getStringValue(change, "status")
	project := getStringValue(change, "project")
	branch := getStringValue(change, "branch")
	topic := getStringValue(change, "topic")
	
	// Owner information
	owner := getOwnerName(change)
	
	// Timestamps
	created := getStringValue(change, "created")
	updated := getStringValue(change, "updated")
	if updated == "" {
		updated = getStringValue(change, "lastUpdated")
	}
	
	// URLs
	url := getStringValue(change, "url")
	
	// Current revision info
	currentRevision := getStringValue(change, "current_revision")
	var patchSetNum string
	if revisions, ok := change["revisions"].(map[string]interface{}); ok {
		if currentRev, ok := revisions[currentRevision].(map[string]interface{}); ok {
			patchSetNum = getStringValue(currentRev, "_number")
		}
	}
	
	// Display basic info
	fmt.Printf("%s %s\n", utils.BoldCyan("Change:"), utils.BoldWhite(changeNum))
	fmt.Printf("%s %s\n", utils.BoldCyan("Subject:"), subject)
	fmt.Printf("%s %s\n", utils.BoldCyan("Status:"), utils.FormatChangeStatus(status))
	fmt.Printf("%s %s\n", utils.BoldCyan("Project:"), project)
	fmt.Printf("%s %s\n", utils.BoldCyan("Branch:"), branch)
	
	if topic != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Topic:"), topic)
	}
	
	fmt.Printf("%s %s\n", utils.BoldCyan("Owner:"), owner)
	
	if patchSetNum != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Patch Set:"), patchSetNum)
	}
	
	if created != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Created:"), utils.FormatTimeAgo(created))
	}
	
	if updated != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Updated:"), utils.FormatTimeAgo(updated))
	}
	
	if url != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("URL:"), utils.Cyan(url))
	}
	
	// Display review labels and scores
	fmt.Println()
	displayReviewLabels(change)
	
	// Display reviewers
	fmt.Println()
	displayReviewers(change)
	
	// Display message/description
	if message := getStringValue(change, "message"); message != "" {
		fmt.Println()
		fmt.Printf("%s\n", utils.BoldCyan("Description:"))
		fmt.Println(strings.Repeat("-", 50))
		
		// Format commit message nicely
		lines := strings.Split(message, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	}
}

func displayReviewLabels(change map[string]interface{}) {
	fmt.Printf("%s\n", utils.BoldCyan("Review Labels:"))
	
	if labels, ok := change["labels"].(map[string]interface{}); ok && len(labels) > 0 {
		// Sort labels for consistent output
		labelNames := make([]string, 0, len(labels))
		for label := range labels {
			labelNames = append(labelNames, label)
		}
		sort.Strings(labelNames)
		
		for _, labelName := range labelNames {
			labelData := labels[labelName].(map[string]interface{})
			
			fmt.Printf("  %s: ", utils.BoldWhite(labelName))
			
			// Check for approved/rejected values
			if approved, ok := labelData["approved"].(map[string]interface{}); ok {
				if value, ok := approved["value"]; ok {
					name := getAuthorName(approved)
					fmt.Printf("%s by %s", utils.FormatScore(labelName, value), name)
				}
			} else if rejected, ok := labelData["rejected"].(map[string]interface{}); ok {
				if value, ok := rejected["value"]; ok {
					name := getAuthorName(rejected)
					fmt.Printf("%s by %s", utils.FormatScore(labelName, value), name)
				}
			} else if all, ok := labelData["all"].([]interface{}); ok && len(all) > 0 {
				// Show all votes
				var votes []string
				for _, vote := range all {
					if voteData, ok := vote.(map[string]interface{}); ok {
						if value, ok := voteData["value"]; ok {
							name := getAuthorName(voteData)
							votes = append(votes, fmt.Sprintf("%s by %s", utils.FormatScore(labelName, value), name))
						}
					}
				}
				if len(votes) > 0 {
					fmt.Print(strings.Join(votes, ", "))
				} else {
					fmt.Print(utils.Gray("no votes"))
				}
			} else {
				fmt.Print(utils.Gray("no votes"))
			}
			fmt.Println()
		}
	} else {
		fmt.Printf("  %s\n", utils.Gray("No review labels"))
	}
}

func displayReviewers(change map[string]interface{}) {
	fmt.Printf("%s\n", utils.BoldCyan("Reviewers:"))
	
	if reviewers, ok := change["reviewers"].(map[string]interface{}); ok {
		if reviewerList, ok := reviewers["REVIEWER"].([]interface{}); ok && len(reviewerList) > 0 {
			for _, reviewer := range reviewerList {
				if reviewerData, ok := reviewer.(map[string]interface{}); ok {
					name := getAuthorName(reviewerData)
					fmt.Printf("  • %s\n", name)
				}
			}
		} else {
			fmt.Printf("  %s\n", utils.Gray("No reviewers assigned"))
		}
		
		if ccList, ok := reviewers["CC"].([]interface{}); ok && len(ccList) > 0 {
			fmt.Printf("\n%s\n", utils.BoldCyan("CC:"))
			for _, cc := range ccList {
				if ccData, ok := cc.(map[string]interface{}); ok {
					name := getAuthorName(ccData)
					fmt.Printf("  • %s\n", name)
				}
			}
		}
	} else {
		fmt.Printf("  %s\n", utils.Gray("No reviewers assigned"))
	}
}

func displayChangeFiles(cfg *config.Config, changeID string, change map[string]interface{}) {
	fmt.Printf("%s\n", utils.BoldCyan("Changed Files:"))
	
	// Get current revision
	currentRevision := getStringValue(change, "current_revision")
	if currentRevision == "" {
		fmt.Printf("  %s\n", utils.Gray("Could not determine current revision"))
		return
	}
	
	// Try to get files via REST API
	client := gerrit.NewRESTClient(cfg)
	files, err := client.GetChangeFiles(changeID, currentRevision)
	if err != nil {
		fmt.Printf("  %s: %v\n", utils.Gray("Could not fetch files"), err)
		return
	}
	
	if len(files) == 0 {
		fmt.Printf("  %s\n", utils.Gray("No files found"))
		return
	}
	
	// Sort files for consistent output
	fileNames := make([]string, 0, len(files))
	for fileName := range files {
		if fileName != "/COMMIT_MSG" { // Skip commit message pseudo-file
			fileNames = append(fileNames, fileName)
		}
	}
	sort.Strings(fileNames)
	
	for _, fileName := range fileNames {
		if fileData, ok := files[fileName].(map[string]interface{}); ok {
			status := getStringValue(fileData, "status")
			var statusIcon string
			switch status {
			case "A":
				statusIcon = utils.Green("+ ")
			case "M":
				statusIcon = utils.Yellow("~ ")
			case "D":
				statusIcon = utils.Red("- ")
			case "R":
				statusIcon = utils.Blue("→ ")
			default:
				statusIcon = "  "
			}
			
			// Show lines added/deleted if available
			var changes string
			if linesInserted, ok := fileData["lines_inserted"].(float64); ok {
				if linesDeleted, ok := fileData["lines_deleted"].(float64); ok {
					changes = fmt.Sprintf(" (%s%d %s%d)", 
						utils.Green("+"), int(linesInserted),
						utils.Red("-"), int(linesDeleted))
				}
			}
			
			fmt.Printf("  %s%s%s\n", statusIcon, fileName, changes)
		} else {
			fmt.Printf("  %s\n", fileName)
		}
	}
}