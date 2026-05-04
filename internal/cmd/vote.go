package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	voteCR       int
	voteQA       int
	voteLint     int
	votePR       int
	voteVerified int
	voteCRSet    bool
	voteQASet    bool
	voteLintSet  bool
	votePRSet    bool
	voteVerSet   bool
	voteLabels   []string
	voteMessage  string
)

var voteCmd = &cobra.Command{
	Use:   "vote <change-id>",
	Short: "Vote on review labels for a change",
	Long: `Post label votes (Code-Review, QA-Review, Product-Review, Lint-Review, Verified)
on a Gerrit change. Use shortcut flags for common labels or -l NAME=VALUE for any label.

Examples:
  gerry vote 12345 --cr +2
  gerry vote 12345 --cr +1 --qa +1 -m "LGTM"
  gerry vote 12345 --pr +1 --verified +1
  gerry vote 12345 -l Code-Review=+2 -l QA-Review=+1`,
	Args: cobra.ExactArgs(1),
	RunE: runVote,
}

func init() {
	voteCmd.Flags().IntVar(&voteCR, "cr", 0, "Code-Review vote (-2..+2)")
	voteCmd.Flags().IntVar(&voteQA, "qa", 0, "QA-Review vote")
	voteCmd.Flags().IntVar(&votePR, "pr", 0, "Product-Review vote")
	voteCmd.Flags().IntVar(&voteLint, "lint", 0, "Lint-Review vote")
	voteCmd.Flags().IntVar(&voteVerified, "verified", 0, "Verified vote")
	voteCmd.Flags().StringArrayVarP(&voteLabels, "label", "l", nil, "Arbitrary label vote NAME=VALUE (repeatable)")
	voteCmd.Flags().StringVarP(&voteMessage, "message", "m", "", "Optional message to attach")
}

func runVote(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	labels := map[string]int{}
	if cmd.Flags().Changed("cr") {
		labels["Code-Review"] = voteCR
	}
	if cmd.Flags().Changed("qa") {
		labels["QA-Review"] = voteQA
	}
	if cmd.Flags().Changed("pr") {
		labels["Product-Review"] = votePR
	}
	if cmd.Flags().Changed("lint") {
		labels["Lint-Review"] = voteLint
	}
	if cmd.Flags().Changed("verified") {
		labels["Verified"] = voteVerified
	}
	for _, raw := range voteLabels {
		name, val, ok := strings.Cut(raw, "=")
		if !ok {
			return fmt.Errorf("invalid --label value %q (expected NAME=VALUE)", raw)
		}
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		if name == "" {
			return fmt.Errorf("invalid --label value %q (empty name)", raw)
		}
		n, err := strconv.Atoi(strings.TrimPrefix(val, "+"))
		if err != nil {
			return fmt.Errorf("invalid --label value %q: %w", raw, err)
		}
		labels[name] = n
	}

	if len(labels) == 0 {
		return fmt.Errorf("at least one vote flag is required (--cr, --qa, --pr, --lint, --verified, or -l NAME=VALUE)")
	}

	_, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		return err
	}

	if err := client.PostVote(changeID, revision, voteMessage, labels); err != nil {
		return fmt.Errorf("failed to post vote: %w", err)
	}

	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(labels))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s%s", name, formatVote(labels[name])))
	}
	fmt.Printf("%s Voted on %s: %s\n", utils.Green("✓"), changeID, strings.Join(parts, ", "))
	return nil
}

func formatVote(v int) string {
	if v > 0 {
		return fmt.Sprintf("+%d", v)
	}
	return strconv.Itoa(v)
}
