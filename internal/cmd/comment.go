package cmd

import (
	"fmt"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var commentMessage string

var commentCmd = &cobra.Command{
	Use:   "comment <change-id>",
	Short: "Post a change-level comment (no vote)",
	Long: `Post a plain, change-level (non-inline) comment on a change without
casting any label vote. Useful for responding to review feedback when the
inline thread is on a line a later patch set removed.

For inline comments and thread replies, see 'gerry comments'.

Examples:
  gerry comment 12345 -m "here is why the line was removed"`,
	Args: cobra.ExactArgs(1),
	RunE: runComment,
}

func init() {
	commentCmd.Flags().StringVarP(&commentMessage, "message", "m", "", "Comment message")
}

func runComment(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	message, err := promptMessage(commentMessage, "Comment message:")
	if err != nil {
		return err
	}

	_, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		return err
	}

	if err := client.PostReview(changeID, revision, message); err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	fmt.Printf("%s Comment posted on %s\n", utils.Green("✓"), changeID)
	return nil
}
