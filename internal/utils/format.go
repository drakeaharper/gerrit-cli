package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	// Color functions for different types of output
	BoldWhite   = color.New(color.FgWhite, color.Bold).SprintFunc()
	BoldGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	BoldRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	BoldYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	BoldBlue    = color.New(color.FgBlue, color.Bold).SprintFunc()
	BoldCyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
	Green       = color.New(color.FgGreen).SprintFunc()
	Red         = color.New(color.FgRed).SprintFunc()
	Yellow      = color.New(color.FgYellow).SprintFunc()
	Blue        = color.New(color.FgBlue).SprintFunc()
	Cyan        = color.New(color.FgCyan).SprintFunc()
	Gray        = color.New(color.FgHiBlack).SprintFunc()
	Dim         = color.New(color.Faint).SprintFunc()
)

func FormatChangeStatus(status string) string {
	switch strings.ToUpper(status) {
	case "NEW", "OPEN":
		return Green(status)
	case "MERGED":
		return BoldGreen(status)
	case "ABANDONED":
		return Red(status)
	case "DRAFT":
		return Yellow(status)
	default:
		return status
	}
}

func FormatScore(label string, value interface{}) string {
	var score int
	switch v := value.(type) {
	case float64:
		score = int(v)
	case int:
		score = v
	case string:
		score, _ = strconv.Atoi(v)
	default:
		return Gray("?")
	}

	switch {
	case score > 0:
		return BoldGreen(fmt.Sprintf("+%d", score))
	case score < 0:
		return BoldRed(fmt.Sprintf("%d", score))
	default:
		return Gray("0")
	}
}

func FormatTimeAgo(timestamp interface{}) string {
	var t time.Time
	
	switch v := timestamp.(type) {
	case string:
		// Try different time formats
		formats := []string{
			"2006-01-02 15:04:05.000000000",
			"2006-01-02 15:04:05",
			time.RFC3339,
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, v); err == nil {
				t = parsed
				break
			}
		}
	case time.Time:
		t = v
	case float64:
		t = time.Unix(int64(v), 0)
	case int64:
		t = time.Unix(v, 0)
	default:
		return Gray("unknown")
	}
	
	if t.IsZero() {
		return Gray("unknown")
	}
	
	return Dim(timeAgo(t))
}

func timeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func PadString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// stripANSI removes ANSI color codes from a string for accurate length calculation
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func FormatTable(headers []string, rows [][]string, padding int) string {
	if len(rows) == 0 {
		return ""
	}
	
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	
	for _, row := range rows {
		for i, cell := range row {
			// Use stripped length for width calculation to ignore ANSI color codes
			cellLen := len(stripANSI(cell))
			if i < len(widths) && cellLen > widths[i] {
				widths[i] = cellLen
			}
		}
	}
	
	var result strings.Builder
	
	// Headers
	for i, header := range headers {
		if i > 0 {
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(BoldWhite(PadString(header, widths[i])))
	}
	result.WriteString("\n")
	
	// Separator
	for i := range headers {
		if i > 0 {
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(strings.Repeat("-", widths[i]))
	}
	result.WriteString("\n")
	
	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				result.WriteString(strings.Repeat(" ", padding))
			}
			if i < len(widths) {
				// Calculate padding based on visual length, not string length
				visualLen := len(stripANSI(cell))
				paddingNeeded := widths[i] - visualLen
				if paddingNeeded > 0 {
					result.WriteString(cell + strings.Repeat(" ", paddingNeeded))
				} else {
					result.WriteString(cell)
				}
			} else {
				result.WriteString(cell)
			}
		}
		result.WriteString("\n")
	}
	
	return result.String()
}

func ParseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}