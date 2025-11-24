package tui

import (
	"strings"
	"testing"

	"github.com/ansel1/tang/parser"
	"github.com/charmbracelet/lipgloss"
)

func TestPackageSummaryAlignmentWithTabs(t *testing.T) {
	m := NewModel(false, 1.0)
	m.TerminalWidth = 80

	// Create a package event with tabs in the output
	// "ok\tgithub.com/test/pkg1\t0.10s"
	// The tabs will expand, making the line longer than lipgloss.Width() thinks if not expanded.
	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
			Elapsed: 0.05,
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok\tgithub.com/test/pkg1\t0.10s\n",
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
	lines := strings.Split(output, "\n")

	// Find the package summary line
	var summaryLine string
	for _, line := range lines {
		if strings.Contains(line, "github.com/test/pkg1") {
			summaryLine = line
			break
		}
	}

	if summaryLine == "" {
		t.Fatal("Could not find package summary line")
	}

	// Verify the package summary line contains the expected test count
	if !strings.Contains(summaryLine, "✓ 1") {
		t.Errorf("Summary line missing counts: %q", summaryLine)
	}

	// Check the visual width using lipgloss.Width, which correctly handles ANSI codes
	visualWidth := lipgloss.Width(summaryLine)

	if visualWidth > m.TerminalWidth {
		t.Errorf("Summary line visual width %d exceeds terminal width %d. Line: %q", visualWidth, m.TerminalWidth, summaryLine)
	}
}
