# Analyze Command

The `gerry analyze` command fetches and analyzes merged changes across all repositories or a specific repository within a date range. It provides comprehensive statistics about contributions, repositories, and timelines with automatic pagination support.

## Features

- **Cross-repository analysis**: Analyze changes across all repositories in Gerrit or filter by specific repo
- **Date range filtering**: Specify start and end dates for analysis
- **Automatic pagination**: Handles large result sets automatically (up to 10,000 changes by default)
- **Multiple output formats**: Generate reports in Markdown, JSON, or CSV format
- **Save to file**: Export analysis results to a file for sharing or archiving
- **Detailed statistics**: Get insights into:
  - Changes by repository
  - Changes by author
  - Timeline analysis (changes per month)
  - Top contributors
  - Repository distribution

## Usage

### Basic Usage

Analyze all repositories for the current year:
```bash
gerry analyze --start-date 2025-01-01 --end-date 2025-12-31
```

### Analyze a Specific Repository

Filter by a specific repository (project):
```bash
gerry analyze --repo canvas-lms --start-date 2025-01-01
```

### Save Output to File

Save the markdown report to a file:
```bash
gerry analyze --start-date 2025-01-01 --output analysis_2025.md
```

### Export as JSON

Get structured JSON output for programmatic processing:
```bash
gerry analyze --start-date 2025-01-01 --format json --output changes.json
```

### Export as CSV

Export data as CSV for spreadsheet analysis:
```bash
gerry analyze --start-date 2025-01-01 --format csv --output changes.csv
```

### Analyze Recent Activity

Analyze the last 30 days in a specific repository:
```bash
gerry analyze --repo canvas-lms --start-date 2025-11-10 --end-date 2025-12-10
```

### Advanced Options

Configure pagination settings for very large result sets:
```bash
gerry analyze \
  --start-date 2025-01-01 \
  --page-size 1000 \
  --max-changes 50000 \
  --output large_analysis.md
```

## Command Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--start-date` | `-s` | Start date (YYYY-MM-DD) | Beginning of current year |
| `--end-date` | `-e` | End date (YYYY-MM-DD) | Today |
| `--repo` | `-r` | Filter by specific repository | All repositories |
| `--format` | `-f` | Output format: markdown, json, csv | markdown |
| `--output` | `-o` | Output file path | stdout |
| `--page-size` | | Results per page | 500 |
| `--max-changes` | | Maximum total changes to fetch | 10000 |

## Output Format Examples

### Markdown Output

The default markdown format includes:

```markdown
# Gerrit Change Analysis

**Analysis Period:** 2025-01-01 to 2025-12-10
**Repository:** All repositories
**Generated:** 2025-12-10 14:30:00
**Total Changes:** 1,234

## Changes by Repository

| Repository | Change Count |
|------------|--------------|
| canvas-lms | 856 |
| outcomes | 234 |
| quizzes | 144 |

## Changes by Author

| Author | Change Count | Repositories |
|--------|--------------|--------------|
| Drake Harper | 123 | 3 |
| Eric Saupe | 98 | 2 |
...

## Timeline Analysis

Changes merged per month:

| Month | Change Count |
|-------|--------------|
| 2025-01 | 156 |
| 2025-02 | 198 |
...

## Top 20 Contributors

| Rank | Author | Changes | Repositories |
|------|--------|---------|--------------|
| 1 | Drake Harper | 123 | 3 |
| 2 | Eric Saupe | 98 | 2 |
...
```

### JSON Output

The JSON format provides structured data with metadata and analysis:

```json
{
  "metadata": {
    "start_date": "2025-01-01",
    "end_date": "2025-12-10",
    "repository": "",
    "generated_at": "2025-12-10T14:30:00Z",
    "total_changes": 1234
  },
  "changes": [
    {
      "_number": 384465,
      "project": "canvas-lms",
      "subject": "Add dark mode support",
      "owner": {
        "name": "Drake Harper",
        "email": "drake@example.com"
      },
      "status": "MERGED",
      "created": "2025-01-15T10:00:00Z",
      "updated": "2025-01-16T15:30:00Z",
      "submitted": "2025-01-16T15:30:00Z"
    }
  ],
  "analysis": {
    "by_author": [...],
    "by_repository": [...],
    "timeline": [...]
  }
}
```

### CSV Output

CSV format for spreadsheet applications:

```csv
change_number,project,subject,owner_name,owner_email,status,created,updated,submitted
384465,canvas-lms,"Add dark mode support",Drake Harper,drake@example.com,MERGED,2025-01-15T10:00:00Z,2025-01-16T15:30:00Z,2025-01-16T15:30:00Z
384466,outcomes,"Fix grade calculation",Eric Saupe,eric@example.com,MERGED,2025-01-16T09:00:00Z,2025-01-17T11:00:00Z,2025-01-17T11:00:00Z
```

## Use Cases

### Team Contribution Reports

Generate monthly contribution reports for your team:
```bash
gerry analyze \
  --start-date 2025-11-01 \
  --end-date 2025-11-30 \
  --output team_report_nov_2025.md
```

### Project Activity Analysis

Analyze activity in a specific project:
```bash
gerry analyze \
  --repo canvas-lms \
  --start-date 2025-01-01 \
  --format json \
  --output canvas_activity.json
```

### Historical Data Export

Export historical data for external analysis:
```bash
gerry analyze \
  --start-date 2024-01-01 \
  --end-date 2024-12-31 \
  --format csv \
  --output 2024_changes.csv
```

### Cross-Repo Comparison

Compare contribution patterns across all repositories:
```bash
gerry analyze \
  --start-date 2025-01-01 \
  --output cross_repo_2025.md
```

## Query Details

The analyze command uses Gerrit's query syntax with the following filters:

- `status:merged` - Only includes merged changes
- `after:YYYY-MM-DD` - Changes submitted after start date
- `before:YYYY-MM-DD` - Changes submitted before end date
- `project:name` - Filter by specific repository (when --repo is specified)

For more information about Gerrit query syntax, see:
https://gerrit-review.googlesource.com/Documentation/user-search.html

## Performance Considerations

- **Pagination**: The command automatically handles pagination with a default page size of 500 changes
- **Safety Limit**: By default, fetches up to 10,000 changes to prevent excessive API calls
- **Large Date Ranges**: For very large date ranges, consider breaking the analysis into smaller periods
- **Network**: Analysis time depends on the number of changes and network speed

## Tips

1. **Start Small**: Begin with a specific repository and short date range to test
2. **Progress Feedback**: Use `--verbose` flag to see detailed progress information
3. **Save Results**: Always save results to a file for large analyses
4. **Multiple Formats**: Generate both Markdown (for reading) and CSV (for analysis)
5. **Regular Reports**: Automate monthly/quarterly reports with scheduled scripts

## Troubleshooting

### No Changes Found

If you get "No changes found", verify:
- Date range is correct (YYYY-MM-DD format)
- Repository name is correct (use exact Gerrit project name)
- Changes were actually merged in the specified period

### Authentication Errors

Ensure your Gerrit configuration is set up correctly:
```bash
gerry init
```

### Timeout Issues

For very large result sets, increase the max-changes limit or reduce the date range:
```bash
gerry analyze --start-date 2025-01-01 --max-changes 50000
```

## Integration with Other Tools

### Piping to Tools

Pipe markdown output to a viewer:
```bash
gerry analyze --start-date 2025-01-01 | glow
```

### Converting to Other Formats

Convert CSV to Excel-compatible format:
```bash
gerry analyze --format csv --start-date 2025-01-01 | iconv -f UTF-8 -t WINDOWS-1252 > changes.csv
```

### Combining with jq

Process JSON output with jq:
```bash
gerry analyze --format json --start-date 2025-01-01 | \
  jq '.analysis.by_author | sort_by(-.Count) | .[0:5]'
```
