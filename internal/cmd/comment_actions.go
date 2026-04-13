package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	replyMessage   string
	replyThread    int
	addFile        string
	addLine        int
	addMessage     string
	resolveThread  int
	resolveMessage string
)

// reply subcommand

var commentsReplyCmd = &cobra.Command{
	Use:   "reply <change-id>",
	Short: "Reply to an inline comment thread",
	Long: `Reply to an existing inline comment thread on a change.

Examples:
  gerry comments reply 12345 -t 1 -m "Thanks, fixed"
  gerry comments reply 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	Run:  runCommentsReply,
}

// add subcommand

var commentsAddCmd = &cobra.Command{
	Use:   "add <change-id>",
	Short: "Add a new inline comment on a file",
	Long: `Add a new inline comment on a specific file and line of a change.

Examples:
  gerry comments add 12345 -f main.go -l 42 -m "Consider renaming this"
  gerry comments add 12345   # interactive prompts`,
	Args: cobra.ExactArgs(1),
	Run:  runCommentsAdd,
}

// resolve subcommand

var commentsResolveCmd = &cobra.Command{
	Use:   "resolve <change-id>",
	Short: "Mark a comment thread as resolved",
	Long: `Mark an unresolved comment thread as resolved.

Examples:
  gerry comments resolve 12345 -t 1
  gerry comments resolve 12345 -t 1 -m "Fixed in latest PS"
  gerry comments resolve 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	Run:  runCommentsResolve,
}

// unresolve subcommand

var commentsUnresolveCmd = &cobra.Command{
	Use:   "unresolve <change-id>",
	Short: "Mark a comment thread as unresolved",
	Long: `Mark a resolved comment thread as unresolved.

Examples:
  gerry comments unresolve 12345 -t 1
  gerry comments unresolve 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	Run:  runCommentsUnresolve,
}

func init() {
	commentsReplyCmd.Flags().StringVarP(&replyMessage, "message", "m", "", "Reply message")
	commentsReplyCmd.Flags().IntVarP(&replyThread, "thread", "t", 0, "Thread index (from gerry comments output)")

	commentsAddCmd.Flags().StringVarP(&addFile, "file", "f", "", "File path to comment on")
	commentsAddCmd.Flags().IntVarP(&addLine, "line", "l", 0, "Line number to comment on")
	commentsAddCmd.Flags().StringVarP(&addMessage, "message", "m", "", "Comment message")

	commentsResolveCmd.Flags().IntVarP(&resolveThread, "thread", "t", 0, "Thread index (from gerry comments output)")
	commentsResolveCmd.Flags().StringVarP(&resolveMessage, "message", "m", "", "Optional message (default: \"Done\")")

	commentsUnresolveCmd.Flags().IntVarP(&resolveThread, "thread", "t", 0, "Thread index (from gerry comments output)")
	commentsUnresolveCmd.Flags().StringVarP(&resolveMessage, "message", "m", "", "Optional message")
}

// helpers

func boolPtr(b bool) *bool { return &b }

func getCurrentRevision(client *gerrit.RESTClient, changeID string) (string, error) {
	change, err := client.GetChange(changeID)
	if err != nil {
		return "", fmt.Errorf("failed to get change details: %w", err)
	}
	if change.CurrentRevision == "" {
		return "", fmt.Errorf("could not determine current revision for change %s", changeID)
	}
	return change.CurrentRevision, nil
}

func loadConfigAndClient() (*config.Config, *gerrit.RESTClient) {
	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}
	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}
	if cfg.HTTPPassword == "" {
		utils.ExitWithError(fmt.Errorf("this command requires REST API access; run 'gerry init' to configure HTTP credentials"))
	}
	return cfg, gerrit.NewRESTClient(cfg)
}

func selectThread(threads [][]Comment, threadIdx int, label string) []Comment {
	if threadIdx > 0 {
		if threadIdx > len(threads) {
			utils.ExitWithError(fmt.Errorf("thread index %d out of range (1-%d)", threadIdx, len(threads)))
		}
		return threads[threadIdx-1]
	}

	// Build interactive picker options
	options := make([]string, len(threads))
	for i, thread := range threads {
		if len(thread) == 0 {
			continue
		}
		first := thread[0]
		msg := strings.Split(strings.TrimSpace(first.Message), "\n")[0]
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		lineStr := ""
		if first.Line > 0 {
			lineStr = fmt.Sprintf(":%d", first.Line)
		}
		options[i] = fmt.Sprintf("[%d] %s%s (%s) — %s", i+1, first.File, lineStr, first.Author, msg)
	}

	var selected int
	prompt := &survey.Select{
		Message: label,
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		utils.ExitWithError(fmt.Errorf("cancelled: %w", err))
	}
	return threads[selected]
}

func promptMessage(flagValue string, label string) string {
	if flagValue != "" {
		return flagValue
	}
	var msg string
	prompt := &survey.Input{
		Message: label,
	}
	if err := survey.AskOne(prompt, &msg, survey.WithValidator(survey.Required)); err != nil {
		utils.ExitWithError(fmt.Errorf("cancelled: %w", err))
	}
	return msg
}

// command implementations

func runCommentsReply(cmd *cobra.Command, args []string) {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid change ID: %w", err))
	}

	cfg, client := loadConfigAndClient()

	// Fetch unresolved threads only — matches default `gerry comments` view
	// so that -t N indices are consistent with what the user sees
	threads, err := getOrderedThreads(cfg, changeID, false)
	if err != nil {
		utils.ExitWithError(err)
	}
	if len(threads) == 0 {
		fmt.Println("No unresolved comment threads found.")
		return
	}

	thread := selectThread(threads, replyThread, "Select thread to reply to:")
	message := promptMessage(replyMessage, "Reply message:")

	// Get the last comment in the thread to reply to
	lastComment := thread[len(thread)-1]
	if lastComment.ID == "" {
		utils.ExitWithError(fmt.Errorf("cannot reply: comment ID not available (REST API required)"))
	}

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		utils.ExitWithError(err)
	}

	comments := map[string][]gerrit.ReviewComment{
		lastComment.File: {
			{
				InReplyTo: lastComment.ID,
				Message:   message,
			},
		},
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		utils.ExitWithError(fmt.Errorf("failed to post reply: %w", err))
	}

	fmt.Printf("%s Reply posted to %s:%d\n", utils.Green("✓"), lastComment.File, lastComment.Line)
}

func runCommentsAdd(cmd *cobra.Command, args []string) {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid change ID: %w", err))
	}

	_, client := loadConfigAndClient()

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		utils.ExitWithError(err)
	}

	// Get file path
	filePath := addFile
	if filePath == "" {
		files, err := client.GetChangeFiles(changeID, revision)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get file list: %w", err))
		}

		fileNames := make([]string, 0, len(files))
		for name := range files {
			if name == "/COMMIT_MSG" {
				continue
			}
			fileNames = append(fileNames, name)
		}
		sort.Strings(fileNames)

		if len(fileNames) == 0 {
			utils.ExitWithError(fmt.Errorf("no files found in change"))
		}

		var selected int
		prompt := &survey.Select{
			Message: "Select file to comment on:",
			Options: fileNames,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			utils.ExitWithError(fmt.Errorf("cancelled: %w", err))
		}
		filePath = fileNames[selected]
	}

	// Get line number
	line := addLine
	if line == 0 {
		var lineStr string
		prompt := &survey.Input{
			Message: "Line number:",
		}
		if err := survey.AskOne(prompt, &lineStr, survey.WithValidator(survey.Required)); err != nil {
			utils.ExitWithError(fmt.Errorf("cancelled: %w", err))
		}
		line, err = strconv.Atoi(lineStr)
		if err != nil || line <= 0 {
			utils.ExitWithError(fmt.Errorf("invalid line number: %s", lineStr))
		}
	}

	message := promptMessage(addMessage, "Comment message:")

	comments := map[string][]gerrit.ReviewComment{
		filePath: {
			{
				Line:    line,
				Message: message,
			},
		},
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		utils.ExitWithError(fmt.Errorf("failed to post comment: %w", err))
	}

	fmt.Printf("%s Comment added to %s:%d\n", utils.Green("✓"), filePath, line)
}

func runCommentsResolve(cmd *cobra.Command, args []string) {
	runResolveAction(args, true)
}

func runCommentsUnresolve(cmd *cobra.Command, args []string) {
	runResolveAction(args, false)
}

func runResolveAction(args []string, resolve bool) {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid change ID: %w", err))
	}

	cfg, client := loadConfigAndClient()

	// Fetch threads filtered by the opposite state
	// resolve: show unresolved threads; unresolve: show resolved threads
	threads, err := getOrderedThreads(cfg, changeID, true)
	if err != nil {
		utils.ExitWithError(err)
	}

	// Filter to relevant threads
	var filtered [][]Comment
	for _, thread := range threads {
		if len(thread) == 0 {
			continue
		}
		if resolve && thread[0].Unresolved {
			filtered = append(filtered, thread)
		} else if !resolve && !thread[0].Unresolved {
			filtered = append(filtered, thread)
		}
	}

	if len(filtered) == 0 {
		if resolve {
			fmt.Println("No unresolved threads to resolve.")
		} else {
			fmt.Println("No resolved threads to unresolve.")
		}
		return
	}

	label := "Select thread to resolve:"
	if !resolve {
		label = "Select thread to unresolve:"
	}
	thread := selectThread(filtered, resolveThread, label)

	// Determine message
	message := resolveMessage
	if message == "" && resolve {
		message = "Done"
	}
	if message == "" && !resolve {
		message = promptMessage("", "Message:")
	}

	lastComment := thread[len(thread)-1]
	if lastComment.ID == "" {
		utils.ExitWithError(fmt.Errorf("cannot modify thread: comment ID not available (REST API required)"))
	}

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		utils.ExitWithError(err)
	}

	unresolved := !resolve
	comments := map[string][]gerrit.ReviewComment{
		lastComment.File: {
			{
				InReplyTo:  lastComment.ID,
				Message:    message,
				Unresolved: boolPtr(unresolved),
			},
		},
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		action := "resolve"
		if !resolve {
			action = "unresolve"
		}
		utils.ExitWithError(fmt.Errorf("failed to %s thread: %w", action, err))
	}

	if resolve {
		fmt.Printf("%s Thread on %s:%d marked as resolved\n", utils.Green("✓"), lastComment.File, lastComment.Line)
	} else {
		fmt.Printf("%s Thread on %s:%d marked as unresolved\n", utils.Yellow("!"), lastComment.File, lastComment.Line)
	}
}
