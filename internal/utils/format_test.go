package utils

import (
	"testing"
	"time"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
		{"", 5, ""},
		{"abcdef", 6, "abcdef"},
		{"abcdefg", 6, "abc..."},
		// Multi-byte characters
		{"日本語テスト", 4, "日..."},
		{"🎉🎉🎉", 3, "🎉🎉🎉"},
		{"🎉🎉🎉🎉", 3, "🎉🎉🎉"},
	}

	for _, tt := range tests {
		got := TruncateString(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestFormatScore(t *testing.T) {
	// Disable color for testing
	tests := []struct {
		label string
		value interface{}
	}{
		{"Code-Review", float64(2)},
		{"Code-Review", float64(-1)},
		{"Code-Review", 0},
		{"Verified", "1"},
	}

	for _, tt := range tests {
		got := FormatScore(tt.label, tt.value)
		if got == "" {
			t.Errorf("FormatScore(%q, %v) returned empty string", tt.label, tt.value)
		}
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no color", "no color"},
	}

	for _, tt := range tests {
		got := stripANSI(tt.input)
		if got != tt.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPadString(t *testing.T) {
	tests := []struct {
		input string
		width int
		want  int // expected length including padding
	}{
		{"hello", 10, 10},
		{"hello", 3, 5}, // no truncation
	}

	for _, tt := range tests {
		got := PadString(tt.input, tt.width)
		if len(got) != tt.want {
			t.Errorf("PadString(%q, %d) has len %d, want %d", tt.input, tt.width, len(got), tt.want)
		}
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		input time.Time
		want  string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5 minutes ago"},
		{now.Add(-1 * time.Minute), "1 minute ago"},
		{now.Add(-2 * time.Hour), "2 hours ago"},
		{now.Add(-1 * time.Hour), "1 hour ago"},
		{now.Add(-48 * time.Hour), "2 days ago"},
		{now.Add(-24 * time.Hour), "1 day ago"},
	}

	for _, tt := range tests {
		got := timeAgo(tt.input)
		if got != tt.want {
			t.Errorf("timeAgo(%v) = %q, want %q", now.Sub(tt.input), got, tt.want)
		}
	}
}

func TestFormatTable(t *testing.T) {
	headers := []string{"A", "B"}
	rows := [][]string{
		{"hello", "world"},
		{"foo", "bar"},
	}

	result := FormatTable(headers, rows, 2)
	if result == "" {
		t.Error("FormatTable returned empty string")
	}

	// Empty rows should return empty
	result = FormatTable(headers, nil, 2)
	if result != "" {
		t.Errorf("FormatTable with no rows should be empty, got %q", result)
	}
}
