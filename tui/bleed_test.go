package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/results"
)

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
		Name:      "pkg1",
		TestOrder: []string{"TestBleed"},
		Status:    results.StatusRunning,
		StartTime: time.Now(),
	}
	pkg.Counts.Running = 1
	run.Packages["pkg1"] = pkg
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs = 1

	test := &results.TestResult{
		Package:     "pkg1",
		Name:        "TestBleed",
		Status:      results.StatusRunning,
		SummaryLine: "=== RUN   TestBleed",
		Output:      []string{"\033[31mThis is red text"},
		StartTime:   time.Now(),
	}
	run.TestResults["pkg1/TestBleed"] = test
	run.Counts.Running++

	// Render
	output := m.View()

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
