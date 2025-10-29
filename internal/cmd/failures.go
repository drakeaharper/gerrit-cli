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
	// Validate change ID
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

	// Use REST API to get messages
	client := gerrit.NewRESTClient(cfg)
	messages, err := client.GetChangeMessages(changeID)
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to get change messages: %w", err))
	}

	// Find the most recent failure link from Service Cloud Jenkins
	failureLink := findMostRecentFailureLink(messages)
	if failureLink == "" {
		utils.Info("No build failure links found from Service Cloud Jenkins")
		return
	}

	fmt.Println(failureLink)
}

// findMostRecentFailureLink searches through messages in reverse order (most recent first)
// to find the latest build failure link from Service Cloud Jenkins
func findMostRecentFailureLink(messages []map[string]interface{}) string {
	// Regular expression to match Jenkins build failure links
	// Pattern: https://jenkins.inst-ci.net/job/Canvas/job/<branch>/<build-number>//build-summary-report/
	jenkinsLinkPattern := regexp.MustCompile(`https://jenkins\.inst-ci\.net/job/Canvas/job/[^/]+/\d+//build-summary-report/`)

	// Iterate through messages in reverse order (most recent first)
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		// Check if the message is from Service Cloud Jenkins
		author := getAuthorFromMessage(message)
		if !strings.Contains(strings.ToLower(author), "service cloud jenkins") {
			continue
		}

		// Check if this is a Verified -1 message
		messageText := getStringValue(message, "message")
		if !strings.Contains(messageText, "Verified-1") {
			continue
		}

		// Extract the Jenkins link from the message
		matches := jenkinsLinkPattern.FindString(messageText)
		if matches != "" {
			return matches
		}
	}

	return ""
}

// getAuthorFromMessage extracts the author name from a message
func getAuthorFromMessage(message map[string]interface{}) string {
	if author, ok := message["author"].(map[string]interface{}); ok {
		if name := getStringValue(author, "name"); name != "" {
			return name
		}
		if username := getStringValue(author, "username"); username != "" {
			return username
		}
		if email := getStringValue(author, "email"); email != "" {
			return email
		}
	}
	return ""
}
