package tui

import (
	"strings"
	"testing"
	"time"
)

func TestSmartRendering(t *testing.T) {
	m := NewModel(false, 1.0)
	m.TerminalWidth = 80
	m.TerminalHeight = 20 // Small height to force elision

	// Add some dummy packages and tests
	// Package 1: Running, mixed tests
	pkg1 := NewPackageState("pkg1")
	m.Packages["pkg1"] = pkg1
	m.PackageOrder = append(m.PackageOrder, "pkg1")

	// Test 1: Passed (Low priority)
	t1 := NewTestState("TestPassed", "pkg1")
	t1.Status = "passed"
	t1.SummaryLine = "=== RUN   TestPassed"
	pkg1.Tests["TestPassed"] = t1
	pkg1.TestOrder = append(pkg1.TestOrder, "TestPassed")
	pkg1.Passed++

	// Test 2: Failed (High priority)
	t2 := NewTestState("TestFailed", "pkg1")
	t2.Status = "failed"
	t2.SummaryLine = "=== RUN   TestFailed"
	t2.AddOutputLine("Error: something went wrong")
	t2.AddOutputLine("    at file.go:10")
	pkg1.Tests["TestFailed"] = t2
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailed")
	pkg1.Failed++

	// Test 3: Running (Medium priority)
	t3 := NewTestState("TestRunning", "pkg1")
	t3.Status = "running"
	t3.SummaryLine = "=== RUN   TestRunning"
	t3.AddOutputLine("Log: doing work")
	pkg1.Tests["TestRunning"] = t3
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunning")
	pkg1.Running++

	// Package 2: Running, just passed tests
	pkg2 := NewPackageState("pkg2")
	m.Packages["pkg2"] = pkg2
	m.PackageOrder = append(m.PackageOrder, "pkg2")

	t4 := NewTestState("TestPassed2", "pkg2")
	t4.Status = "passed"
	t4.SummaryLine = "=== RUN   TestPassed2"
	pkg2.Tests["TestPassed2"] = t4
	pkg2.TestOrder = append(pkg2.TestOrder, "TestPassed2")
	pkg2.Passed++

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
	t5 := NewTestState("TestRunningNew", "pkg1")
	t5.Status = "running"
	t5.SummaryLine = "=== RUN   TestRunningNew"
	t5.StartTime = t3.StartTime.Add(time.Second) // Newer
	pkg1.Tests["TestRunningNew"] = t5
	pkg1.TestOrder = append(pkg1.TestOrder, "TestRunningNew")
	pkg1.Running++

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
	t6 := NewTestState("TestFailedNew", "pkg1")
	t6.Status = "failed"
	t6.SummaryLine = "=== RUN   TestFailedNew"
	t6.StartTime = t2.StartTime.Add(time.Second)
	pkg1.Tests["TestFailedNew"] = t6
	pkg1.TestOrder = append(pkg1.TestOrder, "TestFailedNew")
	pkg1.Failed++

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
