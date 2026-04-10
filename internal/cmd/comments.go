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
	Run:  runComments,
}

func init() {
	commentsCmd.Flags().BoolVar(&showAll, "all", false, "Show all comments (default: unresolved only)")
	commentsCmd.AddCommand(commentsReplyCmd)
	commentsCmd.AddCommand(commentsAddCmd)
	commentsCmd.AddCommand(commentsResolveCmd)
	commentsCmd.AddCommand(commentsUnresolveCmd)
}

func runComments(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Help()
		return
	}
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

	utils.Debugf("Fetching comments for change %s", changeID)

	threads, err := getOrderedThreads(cfg, changeID, showAll)
	if err != nil {
		utils.ExitWithError(err)
	}

	if len(threads) == 0 {
		if showAll {
			fmt.Println("No comments found on this change.")
		} else {
			fmt.Println("No unresolved comment threads found. Use --all to show all comments.")
		}
		return
	}

	displayThreads(threads)
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

	// Get change details with comments
	output, err := client.GetChangeDetails(changeID)
	if err != nil {
		return nil, err
	}

	// Parse the JSON output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var changeData map[string]interface{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if err := utils.ParseJSON([]byte(line), &changeData); err != nil {
			utils.Debugf("Failed to parse line: %s", line)
			continue
		}

		// Skip the stats line
		if _, hasType := changeData["type"]; hasType {
			continue
		}

		break // Use the first valid change data
	}

	return parseSSHComments(changeData), nil
}

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

func parseRESTComments(commentsData map[string]interface{}) []Comment {
	var comments []Comment

	for filename, fileComments := range commentsData {
		if commentsList, ok := fileComments.([]interface{}); ok {
			for _, commentData := range commentsList {
				if comment, ok := commentData.(map[string]interface{}); ok {
					c := Comment{
						ID:      getStringValue(comment, "id"),
						File:    filename,
						Message: getStringValue(comment, "message"),
						Updated: getStringValue(comment, "updated"),
					}

					if patchSet, ok := comment["patch_set"].(float64); ok {
						c.PatchSet = int(patchSet)
					}

					if line, ok := comment["line"].(float64); ok {
						c.Line = int(line)
					}

					if author, ok := comment["author"].(map[string]interface{}); ok {
						c.Author = getAuthorName(author)
					}

					if unresolved, ok := comment["unresolved"].(bool); ok {
						c.Unresolved = unresolved
					}

					c.InReplyTo = getStringValue(comment, "in_reply_to")

					comments = append(comments, c)
				}
			}
		}
	}

	return comments
}

func parseSSHComments(changeData map[string]interface{}) []Comment {
	var comments []Comment

	// SSH API has a different structure - comments are nested in the change data
	if commentsSection, ok := changeData["comments"].([]interface{}); ok {
		for _, commentData := range commentsSection {
			if comment, ok := commentData.(map[string]interface{}); ok {
				c := Comment{
					File:    getStringValue(comment, "file"),
					Message: getStringValue(comment, "message"),
					Updated: getStringValue(comment, "timestamp"),
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

// getOrderedThreads fetches comments and returns ordered, filtered threads.
// If includeResolved is false, only unresolved threads are returned.
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

	// Sort threads by file then line for stable ordering
	sort.Slice(threads, func(i, j int) bool {
		if threads[i][0].File != threads[j][0].File {
			return threads[i][0].File < threads[j][0].File
		}
		return threads[i][0].Line < threads[j][0].Line
	})

	return threads, nil
}

func displayThreads(threads [][]Comment) {
	// Group threads by file, preserving thread indices
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

			// Thread header with index
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

				// Format message with proper indentation
				messageLines := strings.Split(strings.TrimSpace(comment.Message), "\n")
				for _, line := range messageLines {
					fmt.Printf("    %s\n", line)
				}
				fmt.Println()
			}
		}
	}

	// Summary
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

func getAuthorName(author map[string]interface{}) string {
	if name, ok := author["name"].(string); ok && name != "" {
		return name
	}
	if username, ok := author["username"].(string); ok && username != "" {
		return username
	}
	if email, ok := author["email"].(string); ok && email != "" {
		return email
	}
	return "unknown"
}

// buildCommentThreads groups comments into threads based on file and line number
func buildCommentThreads(comments []Comment) [][]Comment {
	// Group comments by file and line number
	threadMap := make(map[string][]Comment)

	for _, comment := range comments {
		// Comments on the same file and line are part of the same thread
		threadKey := fmt.Sprintf("%s:%d", comment.File, comment.Line)
		threadMap[threadKey] = append(threadMap[threadKey], comment)
	}

	// Convert map to slice of threads and sort each thread by timestamp
	threads := [][]Comment{}
	for _, thread := range threadMap {
		// Sort thread by timestamp (oldest first)
		sort.Slice(thread, func(i, j int) bool {
			return thread[i].Updated < thread[j].Updated
		})
		threads = append(threads, thread)
	}

	return threads
}

// markThreadResolution marks the resolution status of each thread based on its last comment
func markThreadResolution(threads [][]Comment) [][]Comment {
	for _, thread := range threads {
		if len(thread) == 0 {
			continue
		}

		// Thread is already sorted by timestamp, so last comment is most recent
		lastComment := thread[len(thread)-1]

		// A thread is considered resolved if:
		// 1. The last comment is explicitly marked as resolved (!Unresolved)
		// 2. The last comment's message is "Done" (case-insensitive)
		isResolved := !lastComment.Unresolved || strings.EqualFold(strings.TrimSpace(lastComment.Message), "Done")

		// Mark all comments in the thread with the thread's resolution status
		for i := range thread {
			thread[i].Unresolved = !isResolved
		}
	}

	return threads
}
