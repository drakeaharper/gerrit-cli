package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/drakeaharper/gerrit-cli/internal/config"
	"github.com/drakeaharper/gerrit-cli/internal/gerrit"
	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	analyzeStartDate string
	analyzeEndDate   string
	analyzeRepo      string
	analyzeFormat    string
	analyzeOutput    string
	analyzePageSize  int
	analyzeMaxLimit  int
	analyzeTimeout   int
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze merged changes across repositories",
	Long: `Analyze merged changes across all repositories or a specific repository within a date range.
This command fetches all merged changes and generates statistics about contributions,
repositories, and timelines. Supports pagination for large result sets.

Examples:
  # Analyze all repositories for the current year
  gerry analyze --start-date 2025-01-01 --end-date 2025-12-31

  # Analyze a specific repository
  gerry analyze --repo canvas-lms --start-date 2025-01-01

  # Save output to a file
  gerry analyze --start-date 2025-01-01 --output analysis.md

  # Get JSON output
  gerry analyze --start-date 2025-01-01 --format json --output changes.json

  # Analyze last 30 days in a specific repo
  gerry analyze --repo canvas-lms --start-date 2025-11-10 --end-date 2025-12-10
`,
	Run: runAnalyze,
}

func init() {
	// Calculate default dates
	now := time.Now()
	startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	analyzeCmd.Flags().StringVarP(&analyzeStartDate, "start-date", "s", startOfYear.Format("2006-01-02"), "Start date (YYYY-MM-DD)")
	analyzeCmd.Flags().StringVarP(&analyzeEndDate, "end-date", "e", now.Format("2006-01-02"), "End date (YYYY-MM-DD)")
	analyzeCmd.Flags().StringVarP(&analyzeRepo, "repo", "r", "", "Filter by specific repository (project)")
	analyzeCmd.Flags().StringVarP(&analyzeFormat, "format", "f", "markdown", "Output format: markdown, json, csv")
	analyzeCmd.Flags().StringVarP(&analyzeOutput, "output", "o", "", "Output file (default: stdout)")
	analyzeCmd.Flags().IntVar(&analyzePageSize, "page-size", 500, "Number of results per page")
	analyzeCmd.Flags().IntVar(&analyzeMaxLimit, "max-changes", 10000, "Maximum total changes to fetch (safety limit)")
	analyzeCmd.Flags().IntVar(&analyzeTimeout, "timeout", 300, "Request timeout in seconds (default: 300)")
}

type AnalysisData struct {
	StartDate    string                   `json:"start_date"`
	EndDate      string                   `json:"end_date"`
	Repository   string                   `json:"repository,omitempty"`
	GeneratedAt  string                   `json:"generated_at"`
	TotalChanges int                      `json:"total_changes"`
	Changes      []map[string]interface{} `json:"changes"`
}

func runAnalyze(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to load configuration: %w", err))
	}

	if err := cfg.Validate(); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid configuration: %w", err))
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", analyzeStartDate); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err))
	}
	if _, err := time.Parse("2006-01-02", analyzeEndDate); err != nil {
		utils.ExitWithError(fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err))
	}

	utils.Infof("Analyzing changes from %s to %s", analyzeStartDate, analyzeEndDate)
	if analyzeRepo != "" {
		utils.Infof("Repository filter: %s", analyzeRepo)
	} else {
		utils.Info("Analyzing all repositories")
	}

	// Fetch all changes with pagination
	// Use configurable timeout for analyze operations which can be slow
	timeout := time.Duration(analyzeTimeout) * time.Second
	utils.Debugf("Using timeout: %v", timeout)
	client := gerrit.NewRESTClientWithTimeout(cfg, timeout)
	changes, err := fetchAllChangesWithPagination(client)
	if err != nil {
		utils.ExitWithError(fmt.Errorf("failed to fetch changes: %w", err))
	}

	if len(changes) == 0 {
		utils.Info("No changes found in the specified date range")
		return
	}

	fmt.Printf("%s Fetched %d total changes\n", color.GreenString("✓"), len(changes))

	// Create analysis data
	analysisData := AnalysisData{
		StartDate:    analyzeStartDate,
		EndDate:      analyzeEndDate,
		Repository:   analyzeRepo,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		TotalChanges: len(changes),
		Changes:      changes,
	}

	// Generate output in requested format
	var output string
	switch strings.ToLower(analyzeFormat) {
	case "markdown", "md":
		output = generateMarkdownReport(analysisData)
	case "json":
		output = generateJSONReport(analysisData)
	case "csv":
		output = generateCSVReport(analysisData)
	default:
		utils.ExitWithError(fmt.Errorf("unknown format: %s (supported: markdown, json, csv)", analyzeFormat))
	}

	// Write output
	if analyzeOutput != "" {
		if err := utils.WriteFile(analyzeOutput, []byte(output)); err != nil {
			utils.ExitWithError(fmt.Errorf("failed to write output file: %w", err))
		}
		fmt.Printf("%s Report saved to: %s\n", color.GreenString("✓"), analyzeOutput)
	} else {
		fmt.Print(output)
	}
}

func fetchAllChangesWithPagination(client *gerrit.RESTClient) ([]map[string]interface{}, error) {
	// Build query
	var queryParts []string
	queryParts = append(queryParts, "status:merged")
	queryParts = append(queryParts, fmt.Sprintf("after:%s", analyzeStartDate))
	queryParts = append(queryParts, fmt.Sprintf("before:%s", analyzeEndDate))

	if analyzeRepo != "" {
		queryParts = append(queryParts, fmt.Sprintf("project:%s", analyzeRepo))
	}

	query := strings.Join(queryParts, " ")
	utils.Debugf("Query: %s", query)

	var allChanges []map[string]interface{}
	start := 0

	for start < analyzeMaxLimit {
		// Build query path with pagination
		encodedQuery := url.QueryEscape(query)
		path := fmt.Sprintf("changes/?q=%s&n=%d&start=%d&o=DETAILED_ACCOUNTS&o=DETAILED_LABELS&o=MESSAGES",
			encodedQuery, analyzePageSize, start)

		utils.Debugf("Fetching page at offset %d (total so far: %d)", start, len(allChanges))

		// Show progress to user
		if start > 0 {
			fmt.Printf("\rFetching changes... %d so far", len(allChanges))
		} else {
			fmt.Printf("Fetching changes...")
		}

		resp, err := client.Get(path)
		if err != nil {
			fmt.Println() // Clear progress line
			return nil, err
		}

		var pageChanges []map[string]interface{}
		if err := json.Unmarshal(resp, &pageChanges); err != nil {
			fmt.Println() // Clear progress line
			return nil, fmt.Errorf("failed to parse changes: %w", err)
		}

		if len(pageChanges) == 0 {
			utils.Debugf("No more results found")
			break
		}

		utils.Debugf("Fetched %d changes in this page", len(pageChanges))
		allChanges = append(allChanges, pageChanges...)

		// Check if we got a full page
		if len(pageChanges) < analyzePageSize {
			utils.Debugf("Received partial page (%d < %d), no more results", len(pageChanges), analyzePageSize)
			break
		}

		// Check for _more_changes indicator on the last change
		if len(pageChanges) > 0 {
			lastChange := pageChanges[len(pageChanges)-1]
			if moreChanges, ok := lastChange["_more_changes"].(bool); !ok || !moreChanges {
				utils.Debugf("No _more_changes indicator, pagination complete")
				break
			}
		}

		start += len(pageChanges)
	}

	// Clear progress line
	if len(allChanges) > 0 {
		fmt.Printf("\rFetching changes... %d total\n", len(allChanges))
	}

	return allChanges, nil
}

func generateMarkdownReport(data AnalysisData) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Gerrit Change Analysis\n\n")
	sb.WriteString(fmt.Sprintf("**Analysis Period:** %s to %s\n", data.StartDate, data.EndDate))
	if data.Repository != "" {
		sb.WriteString(fmt.Sprintf("**Repository:** %s\n", data.Repository))
	} else {
		sb.WriteString("**Repository:** All repositories\n")
	}
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Total Changes:** %d\n\n", data.TotalChanges))

	// Changes by Repository
	if data.Repository == "" {
		sb.WriteString("## Changes by Repository\n\n")
		repoStats := analyzeByRepository(data.Changes)
		sb.WriteString("| Repository | Change Count |\n")
		sb.WriteString("|------------|-------------|\n")
		for _, stat := range repoStats {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", stat.Name, stat.Count))
		}
		sb.WriteString("\n")
	}

	// Changes by Author
	sb.WriteString("## Changes by Author\n\n")
	authorStats := analyzeByAuthor(data.Changes)
	sb.WriteString("| Author | Change Count | Repositories |\n")
	sb.WriteString("|--------|--------------|-------------|\n")
	for _, stat := range authorStats {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", stat.Name, stat.Count, stat.RepoCount))
	}
	sb.WriteString("\n")

	// Timeline Analysis
	sb.WriteString("## Timeline Analysis\n\n")
	sb.WriteString("Changes merged per month:\n\n")
	timelineStats := analyzeTimeline(data.Changes)
	sb.WriteString("| Month | Change Count |\n")
	sb.WriteString("|-------|-------------|\n")
	for _, stat := range timelineStats {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", stat.Name, stat.Count))
	}
	sb.WriteString("\n")

	// Top Contributors
	sb.WriteString("## Top 20 Contributors\n\n")
	sb.WriteString("| Rank | Author | Changes | Repositories |\n")
	sb.WriteString("|------|--------|---------|-------------|\n")
	for i, stat := range authorStats {
		if i >= 20 {
			break
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %d | %d |\n", i+1, stat.Name, stat.Count, stat.RepoCount))
	}
	sb.WriteString("\n")

	sb.WriteString("---\n")
	sb.WriteString("*Generated by gerry analyze*\n")

	return sb.String()
}

func generateJSONReport(data AnalysisData) string {
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"start_date":    data.StartDate,
			"end_date":      data.EndDate,
			"repository":    data.Repository,
			"generated_at":  data.GeneratedAt,
			"total_changes": data.TotalChanges,
		},
		"changes": data.Changes,
		"analysis": map[string]interface{}{
			"by_author":     analyzeByAuthor(data.Changes),
			"by_repository": analyzeByRepository(data.Changes),
			"timeline":      analyzeTimeline(data.Changes),
		},
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error generating JSON: %v", err)
	}

	return string(jsonBytes)
}

func generateCSVReport(data AnalysisData) string {
	var sb strings.Builder

	// CSV Header
	sb.WriteString("change_number,project,subject,owner_name,owner_email,status,created,updated,submitted\n")

	// CSV Rows
	for _, change := range data.Changes {
		changeNum := getStringValue(change, "_number")
		if changeNum == "" {
			changeNum = getStringValue(change, "number")
		}

		project := getStringValue(change, "project")
		subject := strings.ReplaceAll(getStringValue(change, "subject"), ",", " ")
		subject = strings.ReplaceAll(subject, "\"", "'")

		ownerName := ""
		ownerEmail := ""
		if owner, ok := change["owner"].(map[string]interface{}); ok {
			ownerName = getStringValue(owner, "name")
			if ownerName == "" {
				ownerName = getStringValue(owner, "username")
			}
			ownerEmail = getStringValue(owner, "email")
		}

		status := getStringValue(change, "status")
		created := getStringValue(change, "created")
		updated := getStringValue(change, "updated")
		submitted := getStringValue(change, "submitted")

		sb.WriteString(fmt.Sprintf("%s,%s,\"%s\",%s,%s,%s,%s,%s,%s\n",
			changeNum, project, subject, ownerName, ownerEmail, status, created, updated, submitted))
	}

	return sb.String()
}

type Statistic struct {
	Name      string
	Count     int
	RepoCount int
}

func analyzeByRepository(changes []map[string]interface{}) []Statistic {
	repoCounts := make(map[string]int)

	for _, change := range changes {
		repo := getStringValue(change, "project")
		if repo != "" {
			repoCounts[repo]++
		}
	}

	var stats []Statistic
	for repo, count := range repoCounts {
		stats = append(stats, Statistic{Name: repo, Count: count})
	}

	// Sort by count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats
}

func analyzeByAuthor(changes []map[string]interface{}) []Statistic {
	authorCounts := make(map[string]int)
	authorRepos := make(map[string]map[string]bool)

	for _, change := range changes {
		author := ""
		if owner, ok := change["owner"].(map[string]interface{}); ok {
			author = getStringValue(owner, "name")
			if author == "" {
				author = getStringValue(owner, "username")
			}
			if author == "" {
				author = getStringValue(owner, "email")
			}
		}

		if author != "" {
			authorCounts[author]++

			repo := getStringValue(change, "project")
			if repo != "" {
				if authorRepos[author] == nil {
					authorRepos[author] = make(map[string]bool)
				}
				authorRepos[author][repo] = true
			}
		}
	}

	var stats []Statistic
	for author, count := range authorCounts {
		repoCount := len(authorRepos[author])
		stats = append(stats, Statistic{Name: author, Count: count, RepoCount: repoCount})
	}

	// Sort by count descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats
}

func analyzeTimeline(changes []map[string]interface{}) []Statistic {
	monthCounts := make(map[string]int)

	for _, change := range changes {
		submitted := getStringValue(change, "submitted")
		if submitted == "" {
			submitted = getStringValue(change, "updated")
		}

		if submitted != "" {
			// Extract YYYY-MM from the timestamp
			parts := strings.Split(submitted, "T")
			if len(parts) > 0 {
				dateParts := strings.Split(parts[0], "-")
				if len(dateParts) >= 2 {
					month := dateParts[0] + "-" + dateParts[1]
					monthCounts[month]++
				}
			}
		}
	}

	var stats []Statistic
	for month, count := range monthCounts {
		stats = append(stats, Statistic{Name: month, Count: count})
	}

	// Sort by month ascending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	return stats
}
