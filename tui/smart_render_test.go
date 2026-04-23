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
	now := time.Now()
	pkg1 := &results.PackageResult{
		Name:          "pkg1",
		Status:        results.StatusRunning,
		StartTime:     now,
		WallStartTime: now,
		TestOrder:     make([]string, 0),
		DisplayOrder:  make([]string, 0),
	}
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs++

	// Test 1: Passed (Low priority)
	t1 := results.NewTestResult("pkg1", "TestPassed")
	t1.Latest().Status = results.StatusPassed
	t1.Latest().SummaryLine = "=== RUN   TestPassed"
	t1.Latest().StartTime = now
	t1.Latest().WallStartTime = now
	t1.Latest().LastResumeTime = now
	run.TestResults["pkg1/TestPassed"] = t1
	pkg1.TestOrder = append(pkg1.TestOrder, "TestPassed")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestPassed")
	pkg1.Counts.Passed++
	run.Counts.Passed++

	// Test 2: Failed (High priority)
	t2 := results.NewTestResult("pkg1", "TestFailed")
	t2.Latest().Status = results.StatusFailed
	t2.Latest().SummaryLine = "=== RUN   TestFailed"
	t2.Latest().Output = []string{"Error: something went wrong", "    at file.go:10"}
	t2.Latest().StartTime = now
	t2.Latest().WallStartTime = now
	t2.Latest().LastResumeTime = now
	run.TestResults["pkg1/TestFailed"] = t2
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailed")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestFailed")
	pkg1.Counts.Failed++
	run.Counts.Failed++

	// Test 3: Running (High priority)
	t3 := results.NewTestResult("pkg1", "TestRunning")
	t3.Latest().Status = results.StatusRunning
	t3.Latest().SummaryLine = "=== RUN   TestRunning"
	t3.Latest().Output = []string{"Log: doing work"}
	t3.Latest().StartTime = now
	t3.Latest().WallStartTime = now
	t3.Latest().LastResumeTime = now
	run.TestResults["pkg1/TestRunning"] = t3
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunning")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestRunning")
	pkg1.Counts.Running++
	run.Counts.Running++

	// Package 2: Running, just passed tests
	pkg2 := &results.PackageResult{
		Name:          "pkg2",
		Status:        results.StatusRunning,
		StartTime:     time.Now(),
		WallStartTime: time.Now(),
		TestOrder:     make([]string, 0),
		DisplayOrder:  make([]string, 0),
	}
	run.Packages["pkg2"] = pkg2
	run.PackageOrder = append(run.PackageOrder, "pkg2")
	run.RunningPkgs++

	t4 := results.NewTestResult("pkg2", "TestPassed2")
	t4.Latest().Status = results.StatusPassed
	t4.Latest().SummaryLine = "=== RUN   TestPassed2"
	t4.Latest().StartTime = now
	t4.Latest().WallStartTime = now
	t4.Latest().LastResumeTime = now
	run.TestResults["pkg2/TestPassed2"] = t4
	pkg2.TestOrder = append(pkg2.TestOrder, "TestPassed2")
	pkg2.DisplayOrder = append(pkg2.DisplayOrder, "TestPassed2")
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

	output := m.String()
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
	output = m.String()
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
	t5 := results.NewTestResult("pkg1", "TestRunningNew")
	t5.Latest().Status = results.StatusRunning
	t5.Latest().SummaryLine = "=== RUN   TestRunningNew"
	t5.Latest().StartTime = t3.StartTime().Add(time.Second) // Newer
	t5.Latest().WallStartTime = t3.WallStartTime().Add(time.Second)
	t5.Latest().LastResumeTime = t3.WallStartTime().Add(time.Second)
	run.TestResults["pkg1/TestRunningNew"] = t5
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunningNew")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestRunningNew")
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

	output = m.String()
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
	t6 := results.NewTestResult("pkg1", "TestFailedNew")
	t6.Latest().Status = results.StatusFailed
	t6.Latest().SummaryLine = "=== RUN   TestFailedNew"
	t6.Latest().StartTime = t2.StartTime().Add(time.Second)
	t6.Latest().WallStartTime = t2.WallStartTime().Add(time.Second)
	t6.Latest().LastResumeTime = t2.WallStartTime().Add(time.Second)
	run.TestResults["pkg1/TestFailedNew"] = t6
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailedNew")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestFailedNew")
	pkg1.Counts.Failed++
	run.Counts.Failed++

	// Set height to allow Running tests + 1 line for failed
	// Running: t5(1) + t3(2) = 3 lines.
	// Available = 3 + 1 = 4 lines.
	// Running take 3 lines. 1 line left for Failed.
	// Should go to t6 (Newer Failed).

	m.TerminalHeight = 4 + 4 // 8 lines total
	output = m.String()
	if !strings.Contains(output, "TestFailedNew") {
		t.Error("Expected TestFailedNew (newer failed) to be visible")
	}
	if strings.Contains(output, "TestFailed") && !strings.Contains(output, "TestFailedNew") {
		t.Error("Expected TestFailed (older) to be elided in favor of TestFailedNew")
	}
}

func TestPausedTestPriority(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 20

	run := results.NewRun(1)
	run.Status = results.StatusRunning

	state := collector.State()
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	now := time.Now()

	pkg1 := &results.PackageResult{
		Name:          "pkg1",
		Status:        results.StatusRunning,
		StartTime:     now,
		WallStartTime: now,
		TestOrder:     make([]string, 0),
		DisplayOrder:  make([]string, 0),
	}
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.RunningPkgs++

	// Paused test (started first)
	tPaused := results.NewTestResult("pkg1", "TestPaused")
	tPaused.Latest().Status = results.StatusPaused
	tPaused.Latest().SummaryLine = "=== RUN   TestPaused"
	tPaused.Latest().StartTime = now
	tPaused.Latest().WallStartTime = now
	tPaused.Latest().LastResumeTime = now
	run.TestResults["pkg1/TestPaused"] = tPaused
	pkg1.TestOrder = append(pkg1.TestOrder, "TestPaused")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestPaused")
	pkg1.Counts.Running++
	run.Counts.Running++

	// Running test (started later)
	tRunning := results.NewTestResult("pkg1", "TestActive")
	tRunning.Latest().Status = results.StatusRunning
	tRunning.Latest().SummaryLine = "=== RUN   TestActive"
	tRunning.Latest().Output = []string{"doing stuff"}
	tRunning.Latest().StartTime = now.Add(time.Second)
	tRunning.Latest().WallStartTime = now.Add(time.Second)
	tRunning.Latest().LastResumeTime = now.Add(time.Second)
	run.TestResults["pkg1/TestActive"] = tRunning
	pkg1.TestOrder = append(pkg1.TestOrder, "TestActive")
	pkg1.DisplayOrder = append(pkg1.DisplayOrder, "TestActive")
	pkg1.Counts.Running++
	run.Counts.Running++

	// With enough space, both should be visible
	output := m.String()
	if !strings.Contains(output, "TestPaused") {
		t.Error("Expected TestPaused to be visible with plenty of space")
	}
	if !strings.Contains(output, "TestActive") {
		t.Error("Expected TestActive to be visible with plenty of space")
	}

	// Constrain to fit only TestActive (2 lines: summary + output).
	// Fixed lines: summary(1) + separator(1) + pkg header(1) = 3.
	// Available = TerminalHeight - 3 = 2. TestActive uses 2, TestPaused gets nothing.
	m.TerminalHeight = 5
	output = m.String()
	if !strings.Contains(output, "TestActive") {
		t.Error("Expected TestActive (running) to be visible over paused test")
	}
	if strings.Contains(output, "TestPaused") {
		t.Error("Expected TestPaused (paused) to be elided in favor of running test")
	}
}

// TestMultiExecutionTUI tests that during a second execution the test row
// reflects the latest execution while package/run counts remain cumulative
func TestMultiExecutionTUI(t *testing.T) {
	collector := results.NewCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80
	m.TerminalHeight = 20

	now := time.Now()

	// Create a run
	run := results.NewRun(1)
	run.Status = results.StatusRunning

	state := collector.State()
	state.Runs = append(state.Runs, run)
	state.CurrentRun = run

	// Package
	pkg := &results.PackageResult{
		Name:          "pkg1",
		Status:        results.StatusRunning,
		StartTime:     now,
		WallStartTime: now,
		TestOrder:     []string{"TestFoo"},
		DisplayOrder:  []string{"TestFoo"},
	}
	pkg.Counts.Failed = 1  // First execution failed
	pkg.Counts.Passed = 1  // Second execution passed
	pkg.Counts.Running = 1 // Second execution is running
	run.Packages["pkg1"] = pkg
	run.PackageOrder = append(run.PackageOrder, "pkg1")
	run.Counts.Failed = 1
	run.Counts.Passed = 1
	run.Counts.Running = 1
	run.RunningPkgs = 1

	// Test with 2 executions: first failed, second running
	tr := results.NewTestResult("pkg1", "TestFoo")
	// First execution - failed
	tr.Executions[0].Status = results.StatusFailed
	tr.Executions[0].SummaryLine = "--- FAIL: TestFoo (0.10s)"
	tr.Executions[0].Output = []string{"FAIL: first failure"}
	// Second execution - running
	tr.AppendExecution()
	tr.Latest().Status = results.StatusRunning
	tr.Latest().SummaryLine = "=== RUN   TestFoo"
	tr.Latest().Output = []string{"Running..."}

	run.TestResults["pkg1/TestFoo"] = tr

	output := m.String()

	// The TUI should show the latest execution (running) in the test row
	// and should show cumulative counts (1 failed, 1 passed, 1 running)

	// Check that running state is shown (not the failed state from first execution)
	if !strings.Contains(output, "TestFoo") {
		t.Error("Expected TestFoo to be visible")
	}

	// Check cumulative counts are displayed (should see both passed and running)
	// The counts should show: 1 passed, 1 running, 1 failed
	if !strings.Contains(output, "✓1") {
		t.Error("Expected to see 1 passed count")
	}
	if !strings.Contains(output, "✗1") {
		t.Error("Expected to see 1 failed count")
	}

	// Running count should not be negative
	if run.Counts.Running < 0 {
		t.Error("Running count should not be negative")
	}

	// Test should show running state (not failed from first execution)
	latestStatus := tr.Status()
	if latestStatus != results.StatusRunning {
		t.Errorf("Expected latest execution status to be Running, got %s", latestStatus)
	}

	// Verify the test output shows latest execution output
	latestOutput := tr.Output()
	if len(latestOutput) == 0 || latestOutput[0] != "Running..." {
		t.Error("Expected to see latest execution output")
	}
}
