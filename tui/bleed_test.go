package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/ansel1/tang/results"
)

// TestFaintOutputLine verifies that for running tests, the output is inline
// on the same line as the test name, and the entire line is in bright style
// (not faint) since it's now part of the test line.
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

	test := results.NewTestResult("pkg1", "TestFaint")
	test.Latest().Status = results.StatusRunning
	test.Latest().SummaryLine = "=== RUN   TestFaint"
	test.Latest().Output = []string{"hello log line"}
	test.Latest().StartTime = time.Now()
	test.Latest().LastResumeTime = time.Now()
	run.TestResults["pkg1/TestFaint"] = test
	run.Counts.Running++

	output := m.String()

	// For running tests, the output is inline on the same line as the test name.
	// Find the line that contains both the test name and the output.
	var foundLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "TestFaint") && strings.Contains(line, "hello log line") {
			foundLine = line
			break
		}
	}
	if foundLine == "" {
		t.Fatal("could not find test line with output in rendered view")
	}

	// The line should contain the "=== CONT" prefix and be in bright style
	if !strings.Contains(foundLine, "=== CONT") {
		t.Errorf("expected line to contain === CONT prefix, got: %q", foundLine)
	}

	// Check for bright white bold style (\x1b[1;97m)
	const brightWhiteBold = "\x1b[1;97m"
	if !strings.Contains(foundLine, brightWhiteBold) {
		t.Errorf("expected line to be in bright white bold style, got: %q", foundLine)
	}

	// The output should NOT be faint-styled (it's now part of the test line)
	faintStyle := lipgloss.NewStyle().Faint(true).Render("hello log line")
	if strings.Contains(foundLine, faintStyle) {
		t.Errorf("output should not be faint-styled for running tests, got: %q", foundLine)
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

	test := results.NewTestResult("pkg1", "TestBleed")
	test.Latest().Status = results.StatusRunning
	test.Latest().SummaryLine = "=== RUN   TestBleed"
	test.Latest().Output = []string{"\033[31mThis is red text"}
	test.Latest().StartTime = time.Now()
	test.Latest().LastResumeTime = time.Now()
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
