package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	showAll bool
)

var commentsCmd = &cobra.Command{
	Use:   "comments <change-id>",
	Short: "View and manage comments on a change",
	Long: `View, reply to, add, resolve, or unresolve comments on a change.

Subcommands:
  reply      Reply to an inline comment thread
  add        Add a new inline comment on a file
  resolve    Mark a comment thread as resolved
  unresolve  Mark a comment thread as unresolved

When called without a subcommand, displays comments on the change.`,
	Args: cobra.ArbitraryArgs,
	RunE: runComments,
}

func init() {
	commentsCmd.Flags().BoolVar(&showAll, "all", false, "Show all comments (default: unresolved only)")
	commentsCmd.AddCommand(commentsReplyCmd)
	commentsCmd.AddCommand(commentsAddCmd)
	commentsCmd.AddCommand(commentsResolveCmd)
	commentsCmd.AddCommand(commentsUnresolveCmd)
}

func runComments(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		cmd.Help()
		return nil
	}
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

	utils.Debugf("Fetching comments for change %s", changeID)

	threads, err := getOrderedThreads(cfg, changeID, showAll)
	if err != nil {
		return err
	}

	if len(threads) == 0 {
		if showAll {
			fmt.Println("No comments found on this change.")
		} else {
			fmt.Println("No unresolved comment threads found. Use --all to show all comments.")
		}
		return nil
	}

	displayThreads(threads)
	return nil
}

// Comment is the display-layer representation of a comment, normalized across API sources.
type Comment struct {
	ID         string
	PatchSet   int
	File       string
	Line       int
	Author     string
	Message    string
	Updated    string
	Unresolved bool
	InReplyTo  string
}

func getCommentsREST(cfg *config.Config, changeID string) ([]Comment, error) {
	client := gerrit.NewRESTClient(cfg)
	commentsData, err := client.GetChangeComments(changeID)
	if err != nil {
		return nil, err
	}

	return parseRESTComments(commentsData), nil
}

func getCommentsSSH(cfg *config.Config, changeID string) ([]Comment, error) {
	client := gerrit.NewSSHClient(cfg)

	output, err := client.GetChangeDetails(changeID)
	if err != nil {
		return nil, err
	}

	// Parse the JSON output — SSH returns untyped data for comments
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var changeData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &changeData); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}

		if _, hasType := changeData["type"]; hasType {
			continue
		}

		return parseSSHComments(changeData), nil
	}

	return nil, nil
}

func parseRESTComments(commentsData map[string][]gerrit.CommentInfo) []Comment {
	var comments []Comment

	for filename, fileComments := range commentsData {
		for _, ci := range fileComments {
			comments = append(comments, Comment{
				ID:         ci.ID,
				PatchSet:   ci.PatchSet,
				File:       filename,
				Line:       ci.Line,
				Author:     ci.Author.DisplayName(),
				Message:    ci.Message,
				Updated:    ci.Updated,
				Unresolved: ci.Unresolved,
				InReplyTo:  ci.InReplyTo,
			})
		}
	}

	return comments
}

func parseSSHComments(changeData map[string]interface{}) []Comment {
	var comments []Comment

	if commentsSection, ok := changeData["comments"].([]interface{}); ok {
		for _, commentData := range commentsSection {
			if comment, ok := commentData.(map[string]interface{}); ok {
				c := Comment{
					Message: getSSHStringValue(comment, "message"),
					Updated: getSSHStringValue(comment, "timestamp"),
					File:    getSSHStringValue(comment, "file"),
				}

				if line, ok := comment["line"].(float64); ok {
					c.Line = int(line)
				}

				if reviewer, ok := comment["reviewer"].(map[string]interface{}); ok {
					c.Author = getAuthorName(reviewer)
				}

				comments = append(comments, c)
			}
		}
	}

	return comments
}

// getSSHStringValue reads a string from a raw SSH JSON map.
func getSSHStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getOrderedThreads fetches comments and returns ordered, filtered threads.
func getOrderedThreads(cfg *config.Config, changeID string, includeResolved bool) ([][]Comment, error) {
	comments, err := getCommentsREST(cfg, changeID)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		comments, err = getCommentsSSH(cfg, changeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get comments: %w", err)
		}
	}

	if len(comments) == 0 {
		return nil, nil
	}

	threads := buildCommentThreads(comments)
	threads = markThreadResolution(threads)

	if !includeResolved {
		var filtered [][]Comment
		for _, thread := range threads {
			if len(thread) > 0 && thread[0].Unresolved {
				filtered = append(filtered, thread)
			}
		}
		threads = filtered
	}

	sort.Slice(threads, func(i, j int) bool {
		if threads[i][0].File != threads[j][0].File {
			return threads[i][0].File < threads[j][0].File
		}
		return threads[i][0].Line < threads[j][0].Line
	})

	return threads, nil
}

func displayThreads(threads [][]Comment) {
	type indexedThread struct {
		index  int
		thread []Comment
	}
	fileThreads := make(map[string][]indexedThread)
	for i, thread := range threads {
		if len(thread) == 0 {
			continue
		}
		fileName := thread[0].File
		fileThreads[fileName] = append(fileThreads[fileName], indexedThread{index: i + 1, thread: thread})
	}

	fileNames := make([]string, 0, len(fileThreads))
	for fileName := range fileThreads {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)

	for i, fileName := range fileNames {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("%s %s\n", utils.BoldCyan("File:"), utils.BoldWhite(fileName))
		fmt.Println(strings.Repeat("=", len(fileName)+6))

		for _, it := range fileThreads[fileName] {
			thread := it.thread
			firstComment := thread[0]

			status := utils.BoldRed("[UNRESOLVED]")
			if !firstComment.Unresolved {
				status = utils.Green("[RESOLVED]")
			}
			lineStr := ""
			if firstComment.Line > 0 {
				lineStr = fmt.Sprintf(" %s %s", utils.Gray("Line:"), utils.Yellow(fmt.Sprintf("%d", firstComment.Line)))
			}
			fmt.Printf("%s%s %s\n", utils.BoldWhite(fmt.Sprintf("[%d]", it.index)), lineStr, status)

			for _, comment := range thread {
				fmt.Printf("  %s %s", utils.BoldBlue("Author:"), comment.Author)
				if comment.Updated != "" {
					fmt.Printf(" %s %s", utils.Gray("Updated:"), utils.FormatTimeAgo(comment.Updated))
				}
				fmt.Println()

				messageLines := strings.Split(strings.TrimSpace(comment.Message), "\n")
				for _, line := range messageLines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Println()
			}
		}
	}

	totalThreads := len(threads)
	unresolvedCount := 0
	for _, thread := range threads {
		if len(thread) > 0 && thread[0].Unresolved {
			unresolvedCount++
		}
	}

	resolvedCount := totalThreads - unresolvedCount
	if resolvedCount > 0 && unresolvedCount > 0 {
		fmt.Printf("Threads: %s (%s unresolved, %s resolved)\n",
			utils.BoldWhite(fmt.Sprintf("%d", totalThreads)),
			utils.BoldRed(fmt.Sprintf("%d", unresolvedCount)),
			utils.Green(fmt.Sprintf("%d", resolvedCount)))
	} else if unresolvedCount > 0 {
		fmt.Printf("Unresolved threads: %s\n", utils.BoldRed(fmt.Sprintf("%d", totalThreads)))
	} else {
		fmt.Printf("Threads: %s (all resolved)\n", utils.Green(fmt.Sprintf("%d", totalThreads)))
	}
}

// buildCommentThreads groups comments into threads by Gerrit reply-chain
// (in_reply_to), not by (file, line). A thread is the chain rooted at a comment
// with no in_reply_to (or one whose parent is missing from the set), so multiple
// independent threads can share the same line.
//
// When comments carry no IDs (e.g. SSH-sourced data, where in_reply_to is also
// unavailable), it falls back to the legacy (file, line) grouping.
func buildCommentThreads(comments []Comment) [][]Comment {
	hasIDs := false
	for _, c := range comments {
		if c.ID != "" {
			hasIDs = true
			break
		}
	}
	if !hasIDs {
		return buildCommentThreadsByLine(comments)
	}

	byID := make(map[string]Comment, len(comments))
	children := make(map[string][]Comment)
	for _, c := range comments {
		if c.ID != "" {
			byID[c.ID] = c
		}
	}
	for _, c := range comments {
		children[c.InReplyTo] = append(children[c.InReplyTo], c)
	}
	for parent := range children {
		sort.Slice(children[parent], func(i, j int) bool {
			return children[parent][i].Updated < children[parent][j].Updated
		})
	}

	// Roots are comments with no parent in this set: an empty in_reply_to, or
	// an in_reply_to pointing at a comment we don't have.
	var roots []Comment
	for _, c := range comments {
		if c.InReplyTo == "" {
			continue
		}
		if _, ok := byID[c.InReplyTo]; ok {
			continue
		}
		// Parent missing — treat this comment as a root.
		roots = append(roots, c)
	}
	roots = append(roots, children[""]...)
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Updated < roots[j].Updated
	})

	threads := [][]Comment{}
	for _, root := range roots {
		var chain []Comment
		walkThreadChain(root, children, &chain)
		threads = append(threads, chain)
	}

	return threads
}

// walkThreadChain depth-first collects a comment and all its descendant replies.
func walkThreadChain(c Comment, children map[string][]Comment, chain *[]Comment) {
	*chain = append(*chain, c)
	if c.ID == "" {
		return
	}
	for _, child := range children[c.ID] {
		walkThreadChain(child, children, chain)
	}
}

func buildCommentThreadsByLine(comments []Comment) [][]Comment {
	threadMap := make(map[string][]Comment)
	var order []string

	for _, comment := range comments {
		threadKey := fmt.Sprintf("%s:%d", comment.File, comment.Line)
		if _, seen := threadMap[threadKey]; !seen {
			order = append(order, threadKey)
		}
		threadMap[threadKey] = append(threadMap[threadKey], comment)
	}

	threads := [][]Comment{}
	for _, key := range order {
		thread := threadMap[key]
		sort.Slice(thread, func(i, j int) bool {
			return thread[i].Updated < thread[j].Updated
		})
		threads = append(threads, thread)
	}

	return threads
}

func markThreadResolution(threads [][]Comment) [][]Comment {
	for _, thread := range threads {
		if len(thread) == 0 {
			continue
		}

		lastComment := thread[len(thread)-1]
		isResolved := !lastComment.Unresolved || strings.EqualFold(strings.TrimSpace(lastComment.Message), "Done")

		for i := range thread {
			thread[i].Unresolved = !isResolved
		}
	}

	return threads
}
