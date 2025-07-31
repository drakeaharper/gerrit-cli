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
	Short: "View comments on a change",
	Long:  `View review comments on a specific change.`,
	Args:  cobra.ExactArgs(1),
	Run:   runComments,
}

func init() {
	commentsCmd.Flags().BoolVar(&showAll, "all", false, "Show all comments (default: unresolved only)")
}

func runComments(cmd *cobra.Command, args []string) {
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

	// Try REST API first, fall back to SSH if needed
	comments, err := getCommentsREST(cfg, changeID)
	if err != nil {
		utils.Warnf("REST API failed: %v", err)
		utils.Info("Falling back to SSH...")
		comments, err = getCommentsSSH(cfg, changeID)
		if err != nil {
			utils.ExitWithError(fmt.Errorf("failed to get comments: %w", err))
		}
	}

	if len(comments) == 0 {
		fmt.Println("No comments found on this change.")
		return
	}

	displayComments(comments)
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
	File      string
	Line      int
	Author    string
	Message   string
	Updated   string
	Unresolved bool
	InReplyTo string
}

func parseRESTComments(commentsData map[string]interface{}) []Comment {
	var comments []Comment
	
	for filename, fileComments := range commentsData {
		if commentsList, ok := fileComments.([]interface{}); ok {
			for _, commentData := range commentsList {
				if comment, ok := commentData.(map[string]interface{}); ok {
					c := Comment{
						File:    filename,
						Message: getStringValue(comment, "message"),
						Updated: getStringValue(comment, "updated"),
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

func displayComments(comments []Comment) {
	// Build thread structure
	threads := buildCommentThreads(comments)
	
	// Mark thread resolution status for all threads
	threads = markThreadResolution(threads)
	
	// Filter unresolved threads if --all not specified
	if !showAll {
		unresolvedThreads := [][]Comment{}
		for _, thread := range threads {
			if len(thread) > 0 && thread[0].Unresolved {
				unresolvedThreads = append(unresolvedThreads, thread)
			}
		}
		threads = unresolvedThreads
		
		if len(threads) == 0 {
			fmt.Println("No unresolved comment threads found. Use --all to show all comments.")
			return
		}
	}
	
	// Flatten threads back to comments for display
	comments = []Comment{}
	for _, thread := range threads {
		comments = append(comments, thread...)
	}
	
	// Sort comments by file, then line
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].File != comments[j].File {
			return comments[i].File < comments[j].File
		}
		return comments[i].Line < comments[j].Line
	})
	
	// Group comments by file
	fileGroups := make(map[string][]Comment)
	for _, comment := range comments {
		fileGroups[comment.File] = append(fileGroups[comment.File], comment)
	}
	
	// Display grouped comments
	fileNames := make([]string, 0, len(fileGroups))
	for fileName := range fileGroups {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)
	
	for i, fileName := range fileNames {
		if i > 0 {
			fmt.Println()
		}
		
		fmt.Printf("%s %s\n", utils.BoldCyan("File:"), utils.BoldWhite(fileName))
		fmt.Println(strings.Repeat("=", len(fileName)+6))
		
		for _, comment := range fileGroups[fileName] {
			fmt.Printf("%s %s", utils.BoldBlue("Author:"), comment.Author)
			if comment.Line > 0 {
				fmt.Printf(" %s %s", utils.Gray("Line:"), utils.Yellow(fmt.Sprintf("%d", comment.Line)))
			}
			if comment.Updated != "" {
				fmt.Printf(" %s %s", utils.Gray("Updated:"), utils.FormatTimeAgo(comment.Updated))
			}
			if showAll {
				// When showing all comments, display thread resolution status
				if comment.Unresolved {
					fmt.Printf(" %s", utils.BoldRed("[UNRESOLVED]"))
				} else {
					fmt.Printf(" %s", utils.Green("[RESOLVED]"))
				}
			} else if comment.Unresolved {
				// When filtering, only show UNRESOLVED marker
				fmt.Printf(" %s", utils.BoldRed("[UNRESOLVED]"))
			}
			fmt.Println()
			
			// Format message with proper indentation
			messageLines := strings.Split(strings.TrimSpace(comment.Message), "\n")
			for _, line := range messageLines {
				fmt.Printf("  %s\n", line)
			}
			
			fmt.Println()
		}
	}
	
	// Summary - count threads not individual comments
	totalThreads := len(threads)
	unresolvedThreads := 0
	for _, thread := range threads {
		if len(thread) > 0 && thread[0].Unresolved {
			unresolvedThreads++
		}
	}
	
	if showAll {
		fmt.Printf("Total threads: %s", utils.BoldWhite(fmt.Sprintf("%d", totalThreads)))
		if unresolvedThreads > 0 {
			fmt.Printf(" (%s unresolved, %s resolved)", 
				utils.BoldRed(fmt.Sprintf("%d", unresolvedThreads)),
				utils.Green(fmt.Sprintf("%d", totalThreads-unresolvedThreads)))
		}
		fmt.Println()
	} else {
		fmt.Printf("Unresolved threads: %s\n", utils.BoldRed(fmt.Sprintf("%d", totalThreads)))
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