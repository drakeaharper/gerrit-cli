package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	mineAuthor      string
	mineOwner       string
	mineProject     string
	mineStatus      string
	mineSince       string
	mineUntil       string
	mineQuery       string
	mineLimit       int
	mineFormat      string
	mineOutput      string
	mineUnresolved  bool
	mineThread      bool
	mineNoMessages  bool
	mineRawMessages bool
	mineConcurrency int
	minePageSize    int
)

var commentsMineCmd = &cobra.Command{
	Use:   "mine",
	Short: "Crawl every comment you left across many changes",
	Long: `Collect the comments one person left across every change matching a query.

Gerrit has no "all comments by user" endpoint, so this walks the search API for
matching changes and pulls each change's comments, keeping only the ones by the
target author. Comments are grouped into their reply threads, so a reply from
the change owner is available as context with --thread.

By default it collects your own comments on every change you have commented on.
Narrow it with --owner, --project, --since, or --query.

Examples:
  # Everything you have ever said, newest change first
  gerry comments mine

  # Your review comments on one author's changes, as JSON for further analysis
  gerry comments mine --owner ashafovaloff@instructure.com --format json -o review-log.json

  # Include the replies you got, so each thread reads as a conversation
  gerry comments mine --owner ashafovaloff@instructure.com --thread

  # Only what is still unresolved, this quarter, in one repo
  gerry comments mine --unresolved --since 2026-04-01 --project canvas-lms

  # Someone else's review comments, as a markdown digest
  gerry comments mine --author ashafovaloff@instructure.com --format markdown`,
	Args: cobra.NoArgs,
	RunE: runCommentsMine,
}

func init() {
	commentsMineCmd.Flags().StringVar(&mineAuthor, "author", "self", "Whose comments to collect (email, username, or 'self')")
	commentsMineCmd.Flags().StringVar(&mineOwner, "owner", "", "Only changes owned by this account (email, username, or 'self')")
	commentsMineCmd.Flags().StringVar(&mineProject, "project", "", "Only changes in this project")
	commentsMineCmd.Flags().StringVar(&mineStatus, "status", "", "Only changes with this status (open, merged, abandoned)")
	commentsMineCmd.Flags().StringVar(&mineSince, "since", "", "Only changes updated on or after this date (YYYY-MM-DD)")
	commentsMineCmd.Flags().StringVar(&mineUntil, "until", "", "Only changes updated before this date (YYYY-MM-DD)")
	commentsMineCmd.Flags().StringVarP(&mineQuery, "query", "q", "", "Extra raw Gerrit query terms to AND in")
	commentsMineCmd.Flags().IntVarP(&mineLimit, "limit", "n", 0, "Maximum changes to crawl (0 = all matching)")
	commentsMineCmd.Flags().StringVarP(&mineFormat, "format", "f", "text", "Output format: text, json, markdown")
	commentsMineCmd.Flags().StringVarP(&mineOutput, "output", "o", "", "Write output to a file instead of stdout")
	commentsMineCmd.Flags().BoolVar(&mineUnresolved, "unresolved", false, "Only threads that are still unresolved")
	commentsMineCmd.Flags().BoolVar(&mineThread, "thread", false, "Include other people's comments in each matched thread")
	commentsMineCmd.Flags().BoolVar(&mineNoMessages, "no-messages", false, "Skip change-level messages (inline comments only)")
	commentsMineCmd.Flags().BoolVar(&mineRawMessages, "raw-messages", false, "Keep Gerrit's auto-generated change messages (\"Patch Set 1: (5 comments)\")")
	commentsMineCmd.Flags().IntVar(&mineConcurrency, "concurrency", 8, "Number of changes to fetch in parallel")
	commentsMineCmd.Flags().IntVar(&minePageSize, "page-size", 100, "Changes per search API page")
}

// MinedComment is one comment in a mined thread.
type MinedComment struct {
	Author   string `json:"author"`
	Email    string `json:"email,omitempty"`
	Mine     bool   `json:"mine"`
	PatchSet int    `json:"patch_set,omitempty"`
	Updated  string `json:"updated,omitempty"`
	Message  string `json:"message"`
}

// MinedThread is a reply chain containing at least one comment by the author.
type MinedThread struct {
	File       string         `json:"file"`
	Line       int            `json:"line,omitempty"`
	Unresolved bool           `json:"unresolved"`
	Comments   []MinedComment `json:"comments"`
}

// MinedMessage is a change-level (non-inline) message by the author.
type MinedMessage struct {
	Date    string `json:"date,omitempty"`
	Tag     string `json:"tag,omitempty"`
	Message string `json:"message"`
}

// MinedChange holds everything the author said on one change.
type MinedChange struct {
	Number   int            `json:"number"`
	Subject  string         `json:"subject"`
	Project  string         `json:"project"`
	Branch   string         `json:"branch,omitempty"`
	Status   string         `json:"status"`
	Owner    string         `json:"owner,omitempty"`
	Created  string         `json:"created,omitempty"`
	Updated  string         `json:"updated,omitempty"`
	URL      string         `json:"url,omitempty"`
	Threads  []MinedThread  `json:"threads,omitempty"`
	Messages []MinedMessage `json:"change_messages,omitempty"`
}

// MinedResult is the top-level payload for --format json.
type MinedResult struct {
	Author        string        `json:"author"`
	Query         string        `json:"query"`
	ChangeCount   int           `json:"change_count"`
	CommentCount  int           `json:"comment_count"`
	ThreadCount   int           `json:"thread_count"`
	MessageCount  int           `json:"message_count"`
	SkippedErrors []string      `json:"skipped_errors,omitempty"`
	Changes       []MinedChange `json:"changes"`
}

func runCommentsMine(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	switch mineFormat {
	case "text", "json", "markdown":
	default:
		return fmt.Errorf("unsupported format %q (want text, json, or markdown)", mineFormat)
	}
	if mineConcurrency < 1 {
		return fmt.Errorf("--concurrency must be at least 1")
	}
	if minePageSize < 1 {
		return fmt.Errorf("--page-size must be at least 1")
	}

	client := gerrit.NewRESTClient(cfg)

	author, err := resolveAccount(client, mineAuthor)
	if err != nil {
		return fmt.Errorf("failed to resolve --author %q: %w", mineAuthor, err)
	}

	query := buildMineQuery(mineAuthor, mineOwner, mineProject, mineStatus, mineSince, mineUntil, mineQuery)
	utils.Debugf("Query: %s", query)

	changes, err := crawlChanges(client, query, mineLimit, minePageSize)
	if err != nil {
		return fmt.Errorf("failed to search changes: %w", err)
	}
	if len(changes) == 0 {
		fmt.Fprintln(os.Stderr, "No changes matched.")
		return nil
	}

	progressf("Crawling %d changes for comments by %s...\n", len(changes), author.DisplayName())

	result := collectComments(client, cfg, changes, author)
	result.Author = author.DisplayName()
	result.Query = query

	var rendered string
	switch mineFormat {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		rendered = string(data) + "\n"
	case "markdown":
		rendered = renderMineMarkdown(result)
	default:
		rendered = renderMineText(result)
	}

	if mineOutput != "" {
		if err := os.WriteFile(mineOutput, []byte(rendered), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", mineOutput, err)
		}
		fmt.Printf("Wrote %s (%d comments across %d changes)\n", mineOutput, result.CommentCount, result.ChangeCount)
		return nil
	}

	fmt.Print(rendered)
	return nil
}

// progressf writes crawl progress to stderr so it never pollutes piped output.
func progressf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

// buildMineQuery assembles the Gerrit search query from the filter flags. The
// author is prefiltered server-side with commentby: so the crawl only touches
// changes that can possibly contain a matching comment.
func buildMineQuery(author, owner, project, status, since, until, extra string) string {
	var terms []string

	if author != "" {
		terms = append(terms, "commentby:"+author)
	}
	if owner != "" {
		terms = append(terms, "owner:"+owner)
	}
	if project != "" {
		terms = append(terms, "project:"+project)
	}
	if status != "" {
		terms = append(terms, "status:"+status)
	}
	if since != "" {
		terms = append(terms, "after:"+since)
	}
	if until != "" {
		terms = append(terms, "before:"+until)
	}
	if extra = strings.TrimSpace(extra); extra != "" {
		terms = append(terms, extra)
	}

	return strings.Join(terms, " ")
}

// resolveAccount turns "self", an email, or a username into a Gerrit account.
func resolveAccount(client *gerrit.RESTClient, who string) (*gerrit.Account, error) {
	if who == "" || who == "self" {
		return client.GetSelf()
	}

	accounts, err := client.QueryAccounts(url.QueryEscape(who))
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no account found")
	}
	if len(accounts) > 1 {
		utils.Debugf("%d accounts matched %q, using %s", len(accounts), who, accounts[0].DisplayName())
	}

	return &accounts[0], nil
}

// crawlChanges pages through the search API until the query is exhausted or the
// limit is reached. limit <= 0 means every matching change.
func crawlChanges(client *gerrit.RESTClient, query string, limit, pageSize int) ([]gerrit.Change, error) {
	encoded := url.QueryEscape(query)
	var all []gerrit.Change

	for start := 0; ; {
		n := pageSize
		if limit > 0 && limit-len(all) < n {
			n = limit - len(all)
		}

		page, err := client.ListChangesPage(encoded, n, start)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		all = append(all, page...)
		utils.Debugf("Fetched %d changes (total %d)", len(page), len(all))

		if limit > 0 && len(all) >= limit {
			break
		}
		if !page[len(page)-1].MoreChanges {
			break
		}
		start += len(page)
	}

	return all, nil
}

// collectComments fetches each change's comments in parallel and keeps the
// threads that contain a comment by the target author.
func collectComments(client *gerrit.RESTClient, cfg *config.Config, changes []gerrit.Change, author *gerrit.Account) *MinedResult {
	mined := make([]*MinedChange, len(changes))
	errs := make([]string, len(changes))

	var (
		wg   sync.WaitGroup
		sem  = make(chan struct{}, mineConcurrency)
		mu   sync.Mutex
		done int
	)

	for i, ch := range changes {
		wg.Add(1)
		go func(i int, ch gerrit.Change) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mc, err := mineChange(client, cfg, ch, author)
			if err != nil {
				errs[i] = fmt.Sprintf("%d: %v", ch.ChangeNumber(), err)
			} else {
				mined[i] = mc
			}

			mu.Lock()
			done++
			if done%25 == 0 || done == len(changes) {
				progressf("  %d/%d changes\n", done, len(changes))
			}
			mu.Unlock()
		}(i, ch)
	}
	wg.Wait()

	result := &MinedResult{}
	for i := range mined {
		if errs[i] != "" {
			result.SkippedErrors = append(result.SkippedErrors, errs[i])
		}
		mc := mined[i]
		if mc == nil || (len(mc.Threads) == 0 && len(mc.Messages) == 0) {
			continue
		}
		result.Changes = append(result.Changes, *mc)
		result.ThreadCount += len(mc.Threads)
		result.MessageCount += len(mc.Messages)
		for _, t := range mc.Threads {
			for _, c := range t.Comments {
				if c.Mine {
					result.CommentCount++
				}
			}
		}
	}
	result.CommentCount += result.MessageCount
	result.ChangeCount = len(result.Changes)

	return result
}

func mineChange(client *gerrit.RESTClient, cfg *config.Config, ch gerrit.Change, author *gerrit.Account) (*MinedChange, error) {
	num := ch.ChangeNumberStr()

	mc := &MinedChange{
		Number:  ch.ChangeNumber(),
		Subject: ch.Subject,
		Project: ch.Project,
		Branch:  ch.Branch,
		Status:  ch.Status,
		Owner:   ch.Owner.DisplayName(),
		Created: ch.Created,
		Updated: ch.UpdatedTime(),
		URL:     changeWebURL(cfg, ch),
	}

	commentsData, err := client.GetChangeComments(num)
	if err != nil {
		return nil, err
	}

	threads := markThreadResolution(buildCommentThreads(parseRESTComments(commentsData)))
	for _, thread := range threads {
		if !threadHasAuthor(thread, author) {
			continue
		}
		if len(thread) == 0 || (mineUnresolved && !thread[0].Unresolved) {
			continue
		}

		mt := MinedThread{
			File:       thread[0].File,
			Line:       thread[0].Line,
			Unresolved: thread[0].Unresolved,
		}
		for _, c := range thread {
			isAuthor := commentBy(c, author)
			if !isAuthor && !mineThread {
				continue
			}
			mt.Comments = append(mt.Comments, MinedComment{
				Author:   c.Author,
				Email:    c.AuthorEmail,
				Mine:     isAuthor,
				PatchSet: c.PatchSet,
				Updated:  c.Updated,
				Message:  c.Message,
			})
		}
		mc.Threads = append(mc.Threads, mt)
	}

	sort.SliceStable(mc.Threads, func(i, j int) bool {
		if mc.Threads[i].File != mc.Threads[j].File {
			return mc.Threads[i].File < mc.Threads[j].File
		}
		return mc.Threads[i].Line < mc.Threads[j].Line
	})

	// Change-level messages are a separate endpoint, and unresolved has no
	// meaning for them — skip when the caller asked only for open threads.
	if !mineNoMessages && !mineUnresolved {
		messages, err := client.GetChangeMessages(num)
		if err != nil {
			return nil, err
		}
		for _, m := range messages {
			if !accountMatches(m.Author, author) {
				continue
			}
			if !mineRawMessages && messageIsBoilerplate(m) {
				continue
			}
			mc.Messages = append(mc.Messages, MinedMessage{
				Date:    m.Date,
				Tag:     m.Tag,
				Message: m.Message,
			})
		}
	}

	return mc, nil
}

// changeWebURL builds the browser URL for a change, preferring what Gerrit reported.
func changeWebURL(cfg *config.Config, ch gerrit.Change) string {
	if ch.URL != "" {
		return ch.URL
	}
	if cfg.Server == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/c/%s/+/%d", cfg.Server, ch.Project, ch.ChangeNumber())
}

// autoMessageLine matches the scaffolding Gerrit wraps around a review post:
// the "Patch Set 3:" header (with optional label votes) and the "(4 comments)"
// receipt it appends when the post carried inline comments.
var autoMessageLine = regexp.MustCompile(`^(Patch Set \d+:.*|\(\d+ comments?\)|Build Started .*|Uploaded patch set \d+.*)$`)

// messageIsBoilerplate reports whether a change message carries no prose of its
// own. Posting inline comments always generates a change message like
// "Patch Set 1:\n\n(5 comments)", which would otherwise dominate the output and
// duplicate the inline comments it is only announcing.
func messageIsBoilerplate(m gerrit.ChangeMessageInfo) bool {
	if strings.HasPrefix(m.Tag, "autogenerated:") {
		return true
	}

	for _, line := range strings.Split(m.Message, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || autoMessageLine.MatchString(line) {
			continue
		}
		return false
	}

	return true
}

func threadHasAuthor(thread []Comment, author *gerrit.Account) bool {
	for _, c := range thread {
		if commentBy(c, author) {
			return true
		}
	}
	return false
}

// commentBy reports whether a comment was written by the target account,
// preferring the numeric account ID and falling back to email or display name.
func commentBy(c Comment, author *gerrit.Account) bool {
	return accountMatches(gerrit.Account{
		AccountID: c.AuthorID,
		Email:     c.AuthorEmail,
		Name:      c.Author,
	}, author)
}

func accountMatches(got gerrit.Account, want *gerrit.Account) bool {
	if want == nil {
		return false
	}
	if got.AccountID != 0 && want.AccountID != 0 {
		return got.AccountID == want.AccountID
	}
	if got.Email != "" && want.Email != "" {
		return strings.EqualFold(got.Email, want.Email)
	}
	if got.Username != "" && want.Username != "" {
		return strings.EqualFold(got.Username, want.Username)
	}
	return got.Name != "" && strings.EqualFold(got.Name, want.Name)
}

func renderMineText(r *MinedResult) string {
	var b strings.Builder

	for _, ch := range r.Changes {
		fmt.Fprintf(&b, "%s %s\n", utils.BoldWhite(fmt.Sprintf("[%d]", ch.Number)), utils.BoldCyan(ch.Subject))
		fmt.Fprintf(&b, "  %s %s  %s %s  %s %s\n",
			utils.Gray("project:"), ch.Project,
			utils.Gray("status:"), utils.FormatChangeStatus(ch.Status),
			utils.Gray("updated:"), utils.FormatTimeAgo(ch.Updated))
		if ch.URL != "" {
			fmt.Fprintf(&b, "  %s\n", utils.Gray(ch.URL))
		}
		b.WriteString("\n")

		for _, t := range ch.Threads {
			status := utils.Green("[RESOLVED]")
			if t.Unresolved {
				status = utils.BoldRed("[UNRESOLVED]")
			}
			loc := t.File
			if t.Line > 0 {
				loc = fmt.Sprintf("%s:%d", t.File, t.Line)
			}
			fmt.Fprintf(&b, "  %s %s\n", utils.BoldWhite(loc), status)
			for _, c := range t.Comments {
				who := c.Author
				if !c.Mine {
					who = utils.Gray(who + " (reply)")
				} else {
					who = utils.BoldBlue(who)
				}
				fmt.Fprintf(&b, "    %s\n", who)
				for _, line := range strings.Split(strings.TrimSpace(c.Message), "\n") {
					fmt.Fprintf(&b, "      %s\n", line)
				}
			}
			b.WriteString("\n")
		}

		for _, m := range ch.Messages {
			fmt.Fprintf(&b, "  %s\n", utils.BoldWhite("[change message]"))
			for _, line := range strings.Split(strings.TrimSpace(m.Message), "\n") {
				fmt.Fprintf(&b, "      %s\n", line)
			}
			b.WriteString("\n")
		}
	}

	fmt.Fprintf(&b, "%s comments across %s changes (%d threads, %d change messages)\n",
		utils.BoldWhite(fmt.Sprintf("%d", r.CommentCount)),
		utils.BoldWhite(fmt.Sprintf("%d", r.ChangeCount)),
		r.ThreadCount, r.MessageCount)
	for _, e := range r.SkippedErrors {
		fmt.Fprintf(&b, "%s %s\n", utils.BoldRed("skipped"), e)
	}

	return b.String()
}

func renderMineMarkdown(r *MinedResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Comments by %s\n\n", r.Author)
	fmt.Fprintf(&b, "Query: `%s`\n\n", r.Query)
	fmt.Fprintf(&b, "%d comments across %d changes (%d threads, %d change messages).\n\n",
		r.CommentCount, r.ChangeCount, r.ThreadCount, r.MessageCount)

	for _, ch := range r.Changes {
		if ch.URL != "" {
			fmt.Fprintf(&b, "## [%d](%s) %s\n\n", ch.Number, ch.URL, ch.Subject)
		} else {
			fmt.Fprintf(&b, "## %d %s\n\n", ch.Number, ch.Subject)
		}
		fmt.Fprintf(&b, "`%s` · %s · updated %s\n\n", ch.Project, ch.Status, ch.Updated)

		for _, t := range ch.Threads {
			loc := t.File
			if t.Line > 0 {
				loc = fmt.Sprintf("%s:%d", t.File, t.Line)
			}
			state := "resolved"
			if t.Unresolved {
				state = "unresolved"
			}
			fmt.Fprintf(&b, "### `%s` (%s)\n\n", loc, state)
			for _, c := range t.Comments {
				label := c.Author
				if !c.Mine {
					label += " (reply)"
				}
				fmt.Fprintf(&b, "**%s**\n\n", label)
				for _, line := range strings.Split(strings.TrimSpace(c.Message), "\n") {
					fmt.Fprintf(&b, "> %s\n", line)
				}
				b.WriteString("\n")
			}
		}

		for _, m := range ch.Messages {
			fmt.Fprintf(&b, "### change message\n\n")
			for _, line := range strings.Split(strings.TrimSpace(m.Message), "\n") {
				fmt.Fprintf(&b, "> %s\n", line)
			}
			b.WriteString("\n")
		}
	}

	if len(r.SkippedErrors) > 0 {
		b.WriteString("## Skipped\n\n")
		for _, e := range r.SkippedErrors {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}

	return b.String()
}
