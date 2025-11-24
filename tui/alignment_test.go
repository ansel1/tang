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

	// The line should fit within TerminalWidth (80)
	// Note: lipgloss.Width might not account for tabs correctly in the assertion either if we don't expand them first.
	// But the terminal (and our view function) should ensure it fits.
	// If the view function renders tabs, lipgloss.Width(summaryLine) might return < 80, but visually it is > 80.
	// However, if we expand tabs in the test to check visual width:
	expandedLine := strings.ReplaceAll(summaryLine, "\t", "        ") // Approximation
	// Better: use a tab writer or simple expansion loop if we want exactness,
	// but simply checking if the right aligned part is pushed too far is enough.

	// If the bug exists, the right part (counts) will be pushed far to the right.
	// The right part should contain "✓ 1".
	if !strings.Contains(summaryLine, "✓ 1") {
		t.Errorf("Summary line missing counts: %q", summaryLine)
	}

	// Check if the line is too long (visually)
	// If we just check string length, it might be short because \t is 1 char.
	// But if the padding calculation was wrong, the padding string will be huge.
	// Let's check the length of the padding.
	// The line structure is: "  " + leftPart + padding + "  " + rightPart
	// If leftPart has tabs, lipgloss thinks it's short, so padding is large.
	// But leftPart prints long. So total visual length = long + large_padding + right > 80.

	// We can check if the string length (bytes) is suspiciously large?
	// No, padding is spaces.
	// If leftPart is "ok\tpkg\t0.1s" (len ~15), but visual len ~30.
	// lipgloss.Width says 15. Available = 80 - right - 2 - 15 = ~50.
	// Padding = 50 spaces.
	// Printed: "ok\tpkg\t0.1s" (30 cols) + 50 spaces + right.
	// Total cols = 30 + 50 + right = 80 + right > 80.
	// So the line will wrap or overflow.

	// If we expand tabs in the captured output string (assuming 8 spaces), we can check width.
	visualWidth := 0
	for _, r := range summaryLine {
		if r == '\t' {
			visualWidth += (8 - (visualWidth % 8))
		} else {
			visualWidth += lipgloss.Width(string(r))
		}
	}

	if visualWidth > m.TerminalWidth {
		t.Errorf("Summary line visual width %d exceeds terminal width %d. Line: %q", visualWidth, m.TerminalWidth, summaryLine)
	}
}
