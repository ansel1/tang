package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
)

func TestPackageSummaryLastOutput(t *testing.T) {
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok  \tgithub.com/test/pkg1\t0.10s\n",
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	for _, evt := range events {
		_, _ = m.Update(evt)
	}

	output := m.String()

	// The output should contain the last output line (with tabs expanded)
	// The original line is "ok  \tgithub.com/test/pkg1\t0.10s"
	// After tab expansion, it becomes "ok      github.com/test/pkg1    0.10s"
	expected := "ok      github.com/test/pkg1    0.10s"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain last output line '%s'.\nGot:\n%s", expected, output)
	}

	// It should NOT contain the package name as the left part (although the output line contains it,
	// we want to ensure the *summary line* is using the output line.
	// Since the output line contains the package name, checking for the output line is sufficient
	// to prove we are using it, provided the package name alone wouldn't match the full line check).
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string // duration string to parse
		expected string
	}{
		// Test cases < 60s (should use X.Xs format)
		{
			name:     "zero duration",
			duration: "0s",
			expected: "0.0s",
		},
		{
			name:     "small duration",
			duration: "5.2s",
			expected: "5.2s",
		},
		{
			name:     "boundary case 59.9s",
			duration: "59.9s",
			expected: "59.9s",
		},
		{
			name:     "boundary case exactly 60s",
			duration: "60s",
			expected: "00:01:00.000",
		},
		// Test cases >= 60s (should use HH:MM:SS.mmm format)
		{
			name:     "just over 60s",
			duration: "60.1s",
			expected: "00:01:00.100",
		},
		{
			name:     "1 minute 30 seconds",
			duration: "1m30s",
			expected: "00:01:30.000",
		},
		{
			name:     "boundary case 3599.9s (just under 1 hour)",
			duration: "3599.9s",
			expected: "00:59:59.900",
		},
		{
			name:     "boundary case exactly 3600s (1 hour)",
			duration: "3600s",
			expected: "01:00:00.000",
		},
		{
			name:     "1 hour 23 minutes 45 seconds 678 milliseconds",
			duration: "1h23m45s678ms",
			expected: "01:23:45.678",
		},
		{
			name:     "multiple hours",
			duration: "2h30m15s",
			expected: "02:30:15.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the duration string
			d, err := parseDuration(tt.duration)
			if err != nil {
				t.Fatalf("Failed to parse duration %s: %v", tt.duration, err)
			}

			result := formatDuration(d)
			if result != tt.expected {
				t.Errorf("formatDuration(%s) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// parseDuration is a helper function to parse duration strings for testing.
// It handles the standard Go duration format (e.g., "1h23m45s678ms").
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
