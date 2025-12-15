package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/charmbracelet/lipgloss"
)

func TestAlignment(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	now := time.Now()
	events := []parser.TestEvent{
		{
			Time:    now,
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Time:    now.Add(10 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
		},
		{
			Time:    now.Add(20 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
			Elapsed: 0.05,
		},
		{
			Time:    now.Add(30 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok\tgithub.com/test/pkg1\t0.10s\n",
		},
		{
			Time:    now.Add(40 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	for _, evt := range events {
		m.Update(EngineEventMsg(engine.Event{Type: engine.EventTest, TestEvent: evt}))
	}

	output := m.View()
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
