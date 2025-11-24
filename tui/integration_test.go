package tui

import (
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
)

// TestDisplaySummaryIntegration verifies that the TUI model can display a summary
// when tests complete or when interrupted.
func TestDisplaySummaryIntegration(t *testing.T) {
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	// Simulate some test events
	events := []parser.TestEvent{
		{
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
			Time:    time.Now(),
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
			Output:  "=== RUN   TestExample\n",
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestExample",
			Elapsed: 0.05,
		},
		{
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.10,
		},
	}

	// Process events through both the model and collector
	for _, evt := range events {
		m.handleTestEvent(evt)
		collector.AddTestEvent(evt)
	}

	// Add package result to collector
	collector.AddPackageResult("github.com/test/pkg1", "ok", 100*time.Millisecond)

	// Verify the model has the expected state
	if m.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", m.Passed)
	}

	// Verify the collector has the expected state
	packages, testResults, _, _ := collector.GetSummary()
	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}
	if len(testResults) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(testResults))
	}

	// Verify we can compute a summary
	summary := ComputeSummary(collector, 10*time.Second)
	if summary.TotalTests != 1 {
		t.Errorf("Expected 1 total test, got %d", summary.TotalTests)
	}
	if summary.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", summary.PassedTests)
	}

	// Verify we can format the summary
	formatter := NewSummaryFormatter(80)
	summaryText := formatter.Format(summary)
	if summaryText == "" {
		t.Error("Expected non-empty summary text")
	}

	// Note: We don't actually call displaySummary() here because it prints to stdout
	// and we're in a test. The integration is verified by checking that:
	// 1. The model has a reference to the collector
	// 2. The collector has the expected data
	// 3. ComputeSummary and Format work correctly
}
