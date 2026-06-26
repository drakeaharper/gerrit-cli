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
	RunE:  runFailures,
}

func runFailures(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	utils.Debugf("Fetching failure links for change %s", changeID)

	client := gerrit.NewRESTClient(cfg)
	messages, err := client.GetChangeMessages(changeID)
	if err != nil {
		return fmt.Errorf("failed to get change messages: %w", err)
	}

	result := findMostRecentFailure(messages)
	if result.SummaryLink == "" {
		utils.Info("No build failure links found from Service Cloud Jenkins")
		return nil
	}

	printFailures(result)
	return nil
}

func printFailures(result failureResult) {
	fmt.Printf("%s %s\n", utils.BoldRed("Build Failed"), utils.Dim("/o\\"))
	fmt.Println()

	fmt.Printf("%s\n", utils.BoldWhite("Summary report"))
	fmt.Printf("  %s\n", utils.Cyan(result.SummaryLink))

	for _, s := range result.Sections {
		fmt.Println()
		fmt.Printf("%s\n", utils.BoldWhite(fmt.Sprintf("%s (%d)", s.Title, len(s.Failures))))
		for _, f := range s.Failures {
			fmt.Printf("  %s %s\n", utils.BoldRed("✗"), f.Name)
			fmt.Printf("    %s\n", utils.Dim(f.Link))
		}
	}
}

// buildFailure is a single failed job/test linked in a failure section.
type buildFailure struct {
	Name string
	Link string
}

// failureSection is a named group of failures, e.g. "Test failures" or "Build failures".
type failureSection struct {
	Title    string
	Failures []buildFailure
}

// failureResult holds the build summary link and all failure sections.
type failureResult struct {
	SummaryLink string
	Sections    []failureSection
}

var (
	summaryLinkPattern = regexp.MustCompile(`https://jenkins\.inst-ci\.net/job/Canvas/job/[^/]+/\d+/+build-summary-report/`)
	// a section header line, e.g. "Test failures:" or "Build failures:"
	sectionHeaderPattern = regexp.MustCompile(`^\s*([A-Za-z][^:\[\]()]*failures)\s*:\s*$`)
	// a markdown link, e.g. "[Name](url)"
	markdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)
)

func findMostRecentFailure(messages []gerrit.ChangeMessageInfo) failureResult {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		author := msg.Author.DisplayName()
		if !strings.Contains(strings.ToLower(author), "service cloud jenkins") {
			continue
		}

		if !strings.Contains(msg.Message, "Verified-1") {
			continue
		}

		summary := summaryLinkPattern.FindString(msg.Message)
		if summary == "" {
			continue
		}

		return failureResult{
			SummaryLink: summary,
			Sections:    parseFailureSections(msg.Message),
		}
	}

	return failureResult{}
}

// parseFailureSections walks the message line by line, grouping markdown-link
// items under the most recent "... failures:" header.
func parseFailureSections(message string) []failureSection {
	var sections []failureSection
	current := -1

	for _, line := range strings.Split(message, "\n") {
		if h := sectionHeaderPattern.FindStringSubmatch(line); h != nil {
			sections = append(sections, failureSection{Title: strings.TrimSpace(h[1])})
			current = len(sections) - 1
			continue
		}

		if current == -1 {
			continue
		}

		if m := markdownLinkPattern.FindStringSubmatch(line); m != nil {
			sections[current].Failures = append(sections[current].Failures, buildFailure{Name: m[1], Link: m[2]})
		}
	}

	return sections
}
