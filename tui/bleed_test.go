package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/ansel1/tang/results"
)

// TestFaintOutputLine verifies that captured test output lines are rendered
// with lipgloss Faint styling (the "dim" SGR attribute, ESC[2m) so that log
// lines are visually de-emphasized relative to test name/status lines.
func TestFaintOutputLine(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 20

	run := results.NewRun(1)
	run.Status = results.StatusRunning

	state := collector.State()
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	pkg := &results.PackageResult{
		Name:         "pkg1",
		TestOrder:    []string{"TestFaint"},
		DisplayOrder: []string{"TestFaint"},
		Status:       results.StatusRunning,
		StartTime:    time.Now(),
	}
	pkg.Counts.Running = 1
	run.Packages["pkg1"] = pkg
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs = 1

	test := &results.TestResult{
		Package:        "pkg1",
		Name:           "TestFaint",
		Status:         results.StatusRunning,
		SummaryLine:    "=== RUN   TestFaint",
		Output:         []string{"hello log line"},
		StartTime:      time.Now(),
		LastResumeTime: time.Now(),
	}
	run.TestResults["pkg1/TestFaint"] = test
	run.Counts.Running++

	output := m.String()

	// Locate the rendered output line.
	var foundLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "hello log line") {
			foundLine = line
			break
		}
	}
	if foundLine == "" {
		t.Fatal("could not find output line in rendered view")
	}

	// The rendered payload must include the expected faint SGR produced by
	// the model's dimStyle. Compare via lipgloss.Render so we pick up
	// whatever escape sequence lipgloss chooses for Faint(true) on the
	// active renderer.
	expected := lipgloss.NewStyle().Faint(true).Render("hello log line")
	if !strings.Contains(foundLine, expected) {
		t.Errorf("expected output line to contain faint-rendered payload %q, got: %q", expected, foundLine)
	}

	// Header line (test name) must NOT be wrapped in the faint style.
	var headerLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "=== RUN   TestFaint") {
			headerLine = line
			break
		}
	}
	if headerLine == "" {
		t.Fatal("could not find test header line in rendered view")
	}
	faintHeader := lipgloss.NewStyle().Faint(true).Render("=== RUN   TestFaint")
	if strings.Contains(headerLine, faintHeader) {
		t.Errorf("test header line must not be faint-styled, got: %q", headerLine)
	}
}

func TestBleedProtection(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 20

	// Create a package and test
	// Create a run and package
	run := results.NewRun(1)
	run.Status = results.StatusRunning

	state := collector.State()
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	pkg := &results.PackageResult{
		Name:         "pkg1",
		TestOrder:    []string{"TestBleed"},
		DisplayOrder: []string{"TestBleed"},
		Status:       results.StatusRunning,
		StartTime:    time.Now(),
	}
	pkg.Counts.Running = 1
	run.Packages["pkg1"] = pkg
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs = 1

	test := &results.TestResult{
		Package:        "pkg1",
		Name:           "TestBleed",
		Status:         results.StatusRunning,
		SummaryLine:    "=== RUN   TestBleed",
		Output:         []string{"\033[31mThis is red text"},
		StartTime:      time.Now(),
		LastResumeTime: time.Now(),
	}
	run.TestResults["pkg1/TestBleed"] = test
	run.Counts.Running++

	// Render
	output := m.String()

	// Check if the output line is followed by a reset sequence
	lines := strings.Split(output, "\n")
	var foundLine string
	for _, line := range lines {
		if strings.Contains(line, "This is red text") {
			foundLine = line
			break
		}
	}

	if foundLine == "" {
		t.Fatal("Could not find the test output line in View()")
	}

	if !strings.Contains(foundLine, "\033[0m") {
		t.Errorf("Expected line to contain reset sequence, but got: %q", foundLine)
	}
}
