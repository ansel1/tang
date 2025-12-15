package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/results"
)

func TestSmartRendering(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 20 // Small height to force elision

	// Create a run
	run := results.NewRun(1)
	run.Status = results.StatusRunning

	state := collector.State()
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	// Packet 1: Running, mixed tests
	pkg1 := &results.PackageResult{
		Name:          "pkg1",
		Status:        results.StatusRunning,
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
		TestOrder:     make([]string, 0),
	}
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs++

	// Test 1: Passed (Low priority)
	t1 := &results.TestResult{
		Package:       "pkg1",
		Name:          "TestPassed",
		Status:        results.StatusPassed,
		SummaryLine:   "=== RUN   TestPassed",
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
	}
	run.TestResults["pkg1/TestPassed"] = t1
	pkg1.TestOrder = append(pkg1.TestOrder, "TestPassed")
	pkg1.Counts.Passed++
	run.Counts.Passed++

	// Test 2: Failed (High priority)
	t2 := &results.TestResult{
		Package:       "pkg1",
		Name:          "TestFailed",
		Status:        results.StatusFailed,
		SummaryLine:   "=== RUN   TestFailed",
		Output:        []string{"Error: something went wrong", "    at file.go:10"},
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
	}
	run.TestResults["pkg1/TestFailed"] = t2
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailed")
	pkg1.Counts.Failed++
	run.Counts.Failed++

	// Test 3: Running (Medium priority)
	t3 := &results.TestResult{
		Package:       "pkg1",
		Name:          "TestRunning",
		Status:        results.StatusRunning,
		SummaryLine:   "=== RUN   TestRunning",
		Output:        []string{"Log: doing work"},
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
	}
	run.TestResults["pkg1/TestRunning"] = t3
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunning")
	pkg1.Counts.Running++
	run.Counts.Running++

	// Package 2: Running, just passed tests
	pkg2 := &results.PackageResult{
		Name:          "pkg2",
		Status:        results.StatusRunning,
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
		TestOrder:     make([]string, 0),
	}
	run.Packages["pkg2"] = pkg2
	run.PackageOrder = append(run.PackageOrder, "pkg2")
	run.RunningPkgs++

	t4 := &results.TestResult{
		Package:       "pkg2",
		Name:          "TestPassed2",
		Status:        results.StatusPassed,
		SummaryLine:   "=== RUN   TestPassed2",
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
	}
	run.TestResults["pkg2/TestPassed2"] = t4
	pkg2.TestOrder = append(pkg2.TestOrder, "TestPassed2")
	pkg2.Counts.Passed++
	run.Counts.Passed++

	// Calculate expected lines:
	// Fixed: Summary(1) + Separator(1) + Headers(2) = 4 lines
	// Available: 20 - 4 = 16 lines

	// Test lines needed:
	// t1 (Passed): 1 line (Summary)
	// t2 (Failed): 1 (Summary) (Output elided)
	// t3 (Running): 1 (Summary) + 1 (Output) = 2 lines
	// t4 (Passed): 1 line (Summary)
	// Total needed: 1+1+2+1 = 5 lines.
	// 7 < 16, so ALL should be visible.

	output := m.View()
	if !strings.Contains(output, "TestPassed") {
		t.Error("Expected TestPassed to be visible")
	}
	if !strings.Contains(output, "TestFailed") {
		t.Error("Expected TestFailed to be visible")
	}
	if strings.Contains(output, "Error: something went wrong") {
		t.Error("Expected failed output to be elided")
	}

	// Now reduce height to force elision
	// We need to elide low priority first.
	// Priority: Running (1), Failed (2), Passed (3)
	// Lines: t3(2), t2(3), t1(1), t4(1)

	// Set height to allow Headers + Summary + Separator + 2 lines
	// Total = 4 + 2 = 6 lines.
	// Available for tests = 2 lines.
	// Should show: t3 (Running, 2 lines) -> Takes all 2.
	// t2 (Failed), t1 (Passed), t4 (Passed) should be hidden.

	m.TerminalHeight = 4 + 2 // 6 lines total
	output = m.View()
	if !strings.Contains(output, "TestRunning") {
		t.Error("Expected TestRunning to be visible with height 6 (priority 1)")
	}
	if strings.Contains(output, "TestFailed") {
		t.Error("Expected TestFailed to be elided with height 6 (priority 2 vs 1)")
	}
	if strings.Contains(output, "TestPassed") {
		t.Error("Expected TestPassed to be elided with height 6")
	}

	// Test Recency: Add another running test, started later
	t5 := &results.TestResult{
		Package:       "pkg1",
		Name:          "TestRunningNew",
		Status:        results.StatusRunning,
		SummaryLine:   "=== RUN   TestRunningNew",
		StartTime:     t3.StartTime.Add(time.Second), // Newer
		WallStartTime: t3.WallStartTime.Add(time.Second),
	}
	run.TestResults["pkg1/TestRunningNew"] = t5
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunningNew")
	pkg1.Counts.Running++
	run.Counts.Running++

	// Available = 2 lines.
	// Should show: t5 (Running, Newer) -> Takes 1 line (no output).
	// Wait, t5 has no output, so 1 line.
	// t3 (Running, Older) -> Needs 2 lines.
	// If we allocate t5 first (1 line), we have 1 line left.
	// t3 needs 2 lines. It might get 1 line (truncated) or 0 if logic requires full fit?
	// Logic: if availableLines >= len(lines) -> full. else if availableLines > 0 -> partial.
	// So t5 gets 1 line. 1 left. t3 gets 1 line (Summary).

	output = m.View()
	if !strings.Contains(output, "TestRunningNew") {
		t.Error("Expected TestRunningNew (newer) to be visible")
	}
	if !strings.Contains(output, "TestRunning") {
		t.Error("Expected TestRunning (older) to be partially visible")
	}
	if strings.Contains(output, "doing work") {
		t.Error("Expected TestRunning output to be elided (only summary fits)")
	}

	// Test Recency with Failed tests
	// Add another failed test, newer than t2
	t6 := &results.TestResult{
		Package:       "pkg1",
		Name:          "TestFailedNew",
		Status:        results.StatusFailed,
		SummaryLine:   "=== RUN   TestFailedNew",
		StartTime:     t2.StartTime.Add(time.Second),
		WallStartTime: t2.WallStartTime.Add(time.Second),
	}
	run.TestResults["pkg1/TestFailedNew"] = t6
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailedNew")
	pkg1.Counts.Failed++
	run.Counts.Failed++

	// Set height to allow Running tests + 1 line for failed
	// Running: t5(1) + t3(2) = 3 lines.
	// Available = 3 + 1 = 4 lines.
	// Running take 3 lines. 1 line left for Failed.
	// Should go to t6 (Newer Failed).

	m.TerminalHeight = 4 + 4 // 8 lines total
	output = m.View()
	if !strings.Contains(output, "TestFailedNew") {
		t.Error("Expected TestFailedNew (newer failed) to be visible")
	}
	if strings.Contains(output, "TestFailed") && !strings.Contains(output, "TestFailedNew") {
		t.Error("Expected TestFailed (older) to be elided in favor of TestFailedNew")
	}
}
