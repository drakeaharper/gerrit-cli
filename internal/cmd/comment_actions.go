package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	addUnresolved  bool
	addBatch       string
	resolveThread  int
	resolveMessage string
)

// batchCommentInput is the JSON shape accepted by `comments add --batch`.
type batchCommentInput struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Unresolved *bool  `json:"unresolved,omitempty"`
}

var commentsReplyCmd = &cobra.Command{
	Use:   "reply <change-id>",
	Short: "Reply to an inline comment thread",
	Long: `Reply to an existing inline comment thread on a change.

Examples:
  gerry comments reply 12345 -t 1 -m "Thanks, fixed"
  gerry comments reply 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentsReply,
}

var commentsAddCmd = &cobra.Command{
	Use:   "add <change-id>",
	Short: "Add a new inline comment on a file",
	Long: `Add a new inline comment on a specific file and line of a change.

Use --batch to post many comments in a single REST call. The batch file is a
JSON array of objects with fields: file, line, message, unresolved (optional,
defaults to true). Pass "-" to read from stdin.

Examples:
  gerry comments add 12345 -f main.go -l 42 -m "Consider renaming this"
  gerry comments add 12345 --batch comments.json
  cat comments.json | gerry comments add 12345 --batch -
  gerry comments add 12345   # interactive prompts

Batch JSON shape:
  [
    {"file": "main.go", "line": 10, "message": "rename"},
    {"file": "main.go", "line": 42, "message": "nit", "unresolved": false},
    {"file": "util.go", "line": 7,  "message": "dead code"}
  ]`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentsAdd,
}

var commentsResolveCmd = &cobra.Command{
	Use:   "resolve <change-id>",
	Short: "Mark a comment thread as resolved",
	Long: `Mark an unresolved comment thread as resolved.

Examples:
  gerry comments resolve 12345 -t 1
  gerry comments resolve 12345 -t 1 -m "Fixed in latest PS"
  gerry comments resolve 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentsResolve,
}

var commentsUnresolveCmd = &cobra.Command{
	Use:   "unresolve <change-id>",
	Short: "Mark a comment thread as unresolved",
	Long: `Mark a resolved comment thread as unresolved.

Examples:
  gerry comments unresolve 12345 -t 1
  gerry comments unresolve 12345   # interactive picker`,
	Args: cobra.ExactArgs(1),
	RunE: runCommentsUnresolve,
}

func init() {
	commentsReplyCmd.Flags().StringVarP(&replyMessage, "message", "m", "", "Reply message")
	commentsReplyCmd.Flags().IntVarP(&replyThread, "thread", "t", 0, "Thread index (from gerry comments output)")

	commentsAddCmd.Flags().StringVarP(&addFile, "file", "f", "", "File path to comment on")
	commentsAddCmd.Flags().IntVarP(&addLine, "line", "l", 0, "Line number to comment on")
	commentsAddCmd.Flags().StringVarP(&addMessage, "message", "m", "", "Comment message")
	commentsAddCmd.Flags().BoolVar(&addUnresolved, "unresolved", true, "Mark new comment thread as unresolved")
	commentsAddCmd.Flags().StringVar(&addBatch, "batch", "", "Path to JSON file with multiple comments, or \"-\" for stdin")

	commentsResolveCmd.Flags().IntVarP(&resolveThread, "thread", "t", 0, "Thread index (from gerry comments output)")
	commentsResolveCmd.Flags().StringVarP(&resolveMessage, "message", "m", "", "Optional message (default: \"Done\")")

	commentsUnresolveCmd.Flags().IntVarP(&resolveThread, "thread", "t", 0, "Thread index (from gerry comments output)")
	commentsUnresolveCmd.Flags().StringVarP(&resolveMessage, "message", "m", "", "Optional message")
}

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

// revisionForComment returns the revision-id to post a reply against. Gerrit
// accepts a numeric patch set identifier as a revision-id, so anchoring the
// reply to the parent comment's patch set keeps the new comment on the same
// line (and thus the same thread) as the comment it replies to. Falls back to
// the current revision when the patch set is unknown (e.g. SSH-sourced data).
func revisionForComment(client *gerrit.RESTClient, changeID string, parent Comment) (string, error) {
	if parent.PatchSet > 0 {
		return strconv.Itoa(parent.PatchSet), nil
	}
	return getCurrentRevision(client, changeID)
}

func loadConfigAndClient() (*config.Config, *gerrit.RESTClient, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid configuration: %w", err)
	}
	if cfg.HTTPPassword == "" {
		return nil, nil, fmt.Errorf("this command requires REST API access; run 'gerry init' to configure HTTP credentials")
	}
	return cfg, gerrit.NewRESTClient(cfg), nil
}

func selectThread(threads [][]Comment, threadIdx int, label string) ([]Comment, error) {
	if threadIdx > 0 {
		if threadIdx > len(threads) {
			return nil, fmt.Errorf("thread index %d out of range (1-%d)", threadIdx, len(threads))
		}
		return threads[threadIdx-1], nil
	}

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
		return nil, fmt.Errorf("cancelled: %w", err)
	}
	return threads[selected], nil
}

func promptMessage(flagValue string, label string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	var msg string
	prompt := &survey.Input{
		Message: label,
	}
	if err := survey.AskOne(prompt, &msg, survey.WithValidator(survey.Required)); err != nil {
		return "", fmt.Errorf("cancelled: %w", err)
	}
	return msg, nil
}

func runCommentsReply(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	cfg, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	threads, err := getOrderedThreads(cfg, changeID, false)
	if err != nil {
		return err
	}
	if len(threads) == 0 {
		fmt.Println("No unresolved comment threads found.")
		return nil
	}

	thread, err := selectThread(threads, replyThread, "Select thread to reply to:")
	if err != nil {
		return err
	}
	message, err := promptMessage(replyMessage, "Reply message:")
	if err != nil {
		return err
	}

	lastComment := thread[len(thread)-1]
	if lastComment.ID == "" {
		return fmt.Errorf("cannot reply: comment ID not available (REST API required)")
	}

	revision, err := revisionForComment(client, changeID, lastComment)
	if err != nil {
		return err
	}

	comments := map[string][]gerrit.ReviewComment{
		lastComment.File: {
			{
				InReplyTo: lastComment.ID,
				Line:      lastComment.Line,
				Message:   message,
			},
		},
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		return fmt.Errorf("failed to post reply: %w", err)
	}

	fmt.Printf("%s Reply posted to %s:%d\n", utils.Green("✓"), lastComment.File, lastComment.Line)
	return nil
}

func runCommentsAdd(cmd *cobra.Command, args []string) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	_, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	revision, err := getCurrentRevision(client, changeID)
	if err != nil {
		return err
	}

	if addBatch != "" {
		return runCommentsAddBatch(client, changeID, revision, addBatch)
	}

	filePath := addFile
	if filePath == "" {
		files, err := client.GetChangeFiles(changeID, revision)
		if err != nil {
			return fmt.Errorf("failed to get file list: %w", err)
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
			return fmt.Errorf("no files found in change")
		}

		var selected int
		prompt := &survey.Select{
			Message: "Select file to comment on:",
			Options: fileNames,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return fmt.Errorf("cancelled: %w", err)
		}
		filePath = fileNames[selected]
	}

	line := addLine
	if line == 0 {
		var lineStr string
		prompt := &survey.Input{
			Message: "Line number:",
		}
		if err := survey.AskOne(prompt, &lineStr, survey.WithValidator(survey.Required)); err != nil {
			return fmt.Errorf("cancelled: %w", err)
		}
		line, err = strconv.Atoi(lineStr)
		if err != nil || line <= 0 {
			return fmt.Errorf("invalid line number: %s", lineStr)
		}
	}

	message, err := promptMessage(addMessage, "Comment message:")
	if err != nil {
		return err
	}

	comments := map[string][]gerrit.ReviewComment{
		filePath: {
			{
				Line:       line,
				Message:    message,
				Unresolved: boolPtr(addUnresolved),
			},
		},
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	state := "unresolved"
	if !addUnresolved {
		state = "resolved"
	}
	fmt.Printf("%s Comment added to %s:%d (%s)\n", utils.Green("✓"), filePath, line, state)
	return nil
}

func runCommentsResolve(cmd *cobra.Command, args []string) error {
	return runResolveAction(args, true)
}

func runCommentsUnresolve(cmd *cobra.Command, args []string) error {
	return runResolveAction(args, false)
}

func runResolveAction(args []string, resolve bool) error {
	changeID := args[0]
	if err := utils.ValidateChangeID(changeID); err != nil {
		return fmt.Errorf("invalid change ID: %w", err)
	}

	cfg, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	threads, err := getOrderedThreads(cfg, changeID, true)
	if err != nil {
		return err
	}

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
		return nil
	}

	label := "Select thread to resolve:"
	if !resolve {
		label = "Select thread to unresolve:"
	}
	thread, err := selectThread(filtered, resolveThread, label)
	if err != nil {
		return err
	}

	message := resolveMessage
	if message == "" && resolve {
		message = "Done"
	}
	if message == "" && !resolve {
		message, err = promptMessage("", "Message:")
		if err != nil {
			return err
		}
	}

	lastComment := thread[len(thread)-1]
	if lastComment.ID == "" {
		return fmt.Errorf("cannot modify thread: comment ID not available (REST API required)")
	}

	revision, err := revisionForComment(client, changeID, lastComment)
	if err != nil {
		return err
	}

	unresolved := !resolve
	comments := map[string][]gerrit.ReviewComment{
		lastComment.File: {
			{
				InReplyTo:  lastComment.ID,
				Line:       lastComment.Line,
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
		return fmt.Errorf("failed to %s thread: %w", action, err)
	}

	if resolve {
		fmt.Printf("%s Thread on %s:%d marked as resolved\n", utils.Green("✓"), lastComment.File, lastComment.Line)
	} else {
		fmt.Printf("%s Thread on %s:%d marked as unresolved\n", utils.Yellow("!"), lastComment.File, lastComment.Line)
	}
	return nil
}

func runCommentsAddBatch(client *gerrit.RESTClient, changeID, revision, source string) error {
	data, err := readBatchSource(source)
	if err != nil {
		return err
	}

	var inputs []batchCommentInput
	if err := json.Unmarshal(data, &inputs); err != nil {
		return fmt.Errorf("failed to parse batch JSON: %w", err)
	}
	if len(inputs) == 0 {
		return fmt.Errorf("batch contains no comments")
	}

	comments := make(map[string][]gerrit.ReviewComment, len(inputs))
	for i, in := range inputs {
		if in.File == "" {
			return fmt.Errorf("batch entry %d: file is required", i+1)
		}
		if in.Line <= 0 {
			return fmt.Errorf("batch entry %d (%s): line must be > 0", i+1, in.File)
		}
		if strings.TrimSpace(in.Message) == "" {
			return fmt.Errorf("batch entry %d (%s:%d): message is required", i+1, in.File, in.Line)
		}
		unresolved := in.Unresolved
		if unresolved == nil {
			unresolved = boolPtr(true)
		}
		comments[in.File] = append(comments[in.File], gerrit.ReviewComment{
			Line:       in.Line,
			Message:    in.Message,
			Unresolved: unresolved,
		})
	}

	if err := client.PostReviewWithComments(changeID, revision, comments); err != nil {
		return fmt.Errorf("failed to post batch comments: %w", err)
	}

	fmt.Printf("%s Posted %d comment(s) across %d file(s)\n",
		utils.Green("✓"), len(inputs), len(comments))
	return nil
}

func readBatchSource(source string) ([]byte, error) {
	if source == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch from stdin: %w", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch file %s: %w", source, err)
	}
	return data, nil
}
