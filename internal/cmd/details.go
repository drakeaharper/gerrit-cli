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

	utils.Debugf("Fetching details for change %s", changeID)

	change, err := getChangeDetailsREST(cfg, changeID)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		change, err = getChangeDetailsSSH(cfg, changeID)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get change details: %w", err))
		}
	}

	displayChangeDetails(change)

	if showFiles {
		fmt.Println()
		displayChangeFiles(cfg, changeID, change)
	}
}

func getChangeDetailsREST(cfg *config.Config, changeID string) (*gerrit.Change, error) {
	client := gerrit.NewRESTClient(cfg)
	return client.GetChange(changeID)
}

func getChangeDetailsSSH(cfg *config.Config, changeID string) (*gerrit.Change, error) {
	client := gerrit.NewSSHClient(cfg)

	output, err := client.GetChangeDetails(changeID)
	if err != nil {
		return nil, err
	}

	return parseSSHChangeDetail(output)
}

func displayChangeDetails(change *gerrit.Change) {
	fmt.Printf("%s %s\n", utils.BoldCyan("Change:"), utils.BoldWhite(change.ChangeNumberStr()))
	fmt.Printf("%s %s\n", utils.BoldCyan("Subject:"), change.Subject)
	fmt.Printf("%s %s\n", utils.BoldCyan("Status:"), utils.FormatChangeStatus(change.Status))
	fmt.Printf("%s %s\n", utils.BoldCyan("Project:"), change.Project)
	fmt.Printf("%s %s\n", utils.BoldCyan("Branch:"), change.Branch)

	if change.Topic != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Topic:"), change.Topic)
	}

	fmt.Printf("%s %s\n", utils.BoldCyan("Owner:"), change.Owner.DisplayName())

	if psNum := change.CurrentPatchSetNumber(); psNum > 0 {
		fmt.Printf("%s %d\n", utils.BoldCyan("Patch Set:"), psNum)
	}

	if change.Created != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Created:"), utils.FormatTimeAgo(change.Created))
	}

	if updated := change.UpdatedTime(); updated != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("Updated:"), utils.FormatTimeAgo(updated))
	}

	if change.URL != "" {
		fmt.Printf("%s %s\n", utils.BoldCyan("URL:"), utils.Cyan(change.URL))
	}

	// Display review labels and scores
	fmt.Println()
	displayReviewLabels(change)

	// Display reviewers
	fmt.Println()
	displayReviewers(change)

	// Display commit message
	if change.CommitMessage != "" {
		fmt.Println()
		fmt.Printf("%s\n", utils.BoldCyan("Description:"))
		fmt.Println(strings.Repeat("-", 50))
		for _, line := range strings.Split(change.CommitMessage, "\n") {
			fmt.Printf("  %s\n", line)
		}
	} else if change.CurrentRevision != "" {
		if rev, ok := change.Revisions[change.CurrentRevision]; ok && rev.Commit.Message != "" {
			fmt.Println()
			fmt.Printf("%s\n", utils.BoldCyan("Description:"))
			fmt.Println(strings.Repeat("-", 50))
			for _, line := range strings.Split(rev.Commit.Message, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
	}
}

func displayReviewLabels(change *gerrit.Change) {
	fmt.Printf("%s\n", utils.BoldCyan("Review Labels:"))

	if len(change.Labels) > 0 {
		labelNames := make([]string, 0, len(change.Labels))
		for label := range change.Labels {
			labelNames = append(labelNames, label)
		}
		sort.Strings(labelNames)

		for _, labelName := range labelNames {
			labelData, ok := change.Labels[labelName].(map[string]interface{})
			if !ok {
				continue
			}

			fmt.Printf("  %s: ", utils.BoldWhite(labelName))

			if approved, ok := labelData["approved"].(map[string]interface{}); ok {
				if value, ok := approved["value"]; ok {
					fmt.Printf("%s by %s", utils.FormatScore(labelName, value), getAuthorName(approved))
				}
			} else if rejected, ok := labelData["rejected"].(map[string]interface{}); ok {
				if value, ok := rejected["value"]; ok {
					fmt.Printf("%s by %s", utils.FormatScore(labelName, value), getAuthorName(rejected))
				}
			} else if all, ok := labelData["all"].([]interface{}); ok && len(all) > 0 {
				var votes []string
				for _, vote := range all {
					if voteData, ok := vote.(map[string]interface{}); ok {
						if value, ok := voteData["value"]; ok {
							votes = append(votes, fmt.Sprintf("%s by %s", utils.FormatScore(labelName, value), getAuthorName(voteData)))
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

func displayReviewers(change *gerrit.Change) {
	fmt.Printf("%s\n", utils.BoldCyan("Reviewers:"))

	if len(change.Reviewers) > 0 {
		if reviewerList := change.Reviewers["REVIEWER"]; len(reviewerList) > 0 {
			for _, r := range reviewerList {
				fmt.Printf("  • %s\n", r.DisplayName())
			}
		} else {
			fmt.Printf("  %s\n", utils.Gray("No reviewers assigned"))
		}

		if ccList := change.Reviewers["CC"]; len(ccList) > 0 {
			fmt.Printf("\n%s\n", utils.BoldCyan("CC:"))
			for _, cc := range ccList {
				fmt.Printf("  • %s\n", cc.DisplayName())
			}
		}
	} else {
		fmt.Printf("  %s\n", utils.Gray("No reviewers assigned"))
	}
}

func displayChangeFiles(cfg *config.Config, changeID string, change *gerrit.Change) {
	fmt.Printf("%s\n", utils.BoldCyan("Changed Files:"))

	if change.CurrentRevision == "" {
		fmt.Printf("  %s\n", utils.Gray("Could not determine current revision"))
		return
	}

	client := gerrit.NewRESTClient(cfg)
	files, err := client.GetChangeFiles(changeID, change.CurrentRevision)
	if err != nil {
		fmt.Printf("  %s: %v\n", utils.Gray("Could not fetch files"), err)
		return
	}

	if len(files) == 0 {
		fmt.Printf("  %s\n", utils.Gray("No files found"))
		return
	}

	fileNames := make([]string, 0, len(files))
	for fileName := range files {
		if fileName != "/COMMIT_MSG" {
			fileNames = append(fileNames, fileName)
		}
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		fi := files[fileName]
		var statusIcon string
		switch fi.Status {
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

		var changes string
		if fi.LinesInserted > 0 || fi.LinesDeleted > 0 {
			changes = fmt.Sprintf(" (%s%d %s%d)",
				utils.Green("+"), fi.LinesInserted,
				utils.Red("-"), fi.LinesDeleted)
		}

		fmt.Printf("  %s%s%s\n", statusIcon, fileName, changes)
	}
}
