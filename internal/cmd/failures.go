package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var failuresCmd = &cobra.Command{
	Use:   "failures <change-id>",
	Short: "Get the most recent build failure link",
	Long:  `Retrieves the most recent build failure link from Service Cloud Jenkins for a change.`,
	Args:  cobra.ExactArgs(1),
	Run:   runFailures,
}

func runFailures(cmd *cobra.Command, args []string) {
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

	utils.Debugf("Fetching failure links for change %s", changeID)

	client := gerrit.NewRESTClient(cfg)
	messages, err := client.GetChangeMessages(changeID)
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to get change messages: %w", err))
	}

	failureLink := findMostRecentFailureLink(messages)
	if failureLink == "" {
		utils.Info("No build failure links found from Service Cloud Jenkins")
		return
	}

	fmt.Println(failureLink)
}

func findMostRecentFailureLink(messages []gerrit.ChangeMessageInfo) string {
	jenkinsLinkPattern := regexp.MustCompile(`https://jenkins\.inst-ci\.net/job/Canvas/job/[^/]+/\d+//build-summary-report/`)

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		author := msg.Author.DisplayName()
		if !strings.Contains(strings.ToLower(author), "service cloud jenkins") {
			continue
		}

		if !strings.Contains(msg.Message, "Verified-1") {
			continue
		}

		if match := jenkinsLinkPattern.FindString(msg.Message); match != "" {
			return match
		}
	}

	return ""
}
