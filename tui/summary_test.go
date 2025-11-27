package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/parser"
)

func TestPackageSummaryLastOutput(t *testing.T) {
	collector := NewSummaryCollector()
	m := NewModel(false, 1.0, collector)
	m.TerminalWidth = 80

	events := []parser.TestEvent{
		{
			Action:  "start",
			Package: "github.com/test/pkg1",
		},
		{
			Action:  "output",
			Package: "github.com/test/pkg1",
			Output:  "ok  \tgithub.com/test/pkg1\t0.10s\n",
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

	// The output should contain the last output line (with tabs expanded)
	// The original line is "ok  \tgithub.com/test/pkg1\t0.10s"
	// After tab expansion, it becomes "ok      github.com/test/pkg1    0.10s"
	expected := "ok      github.com/test/pkg1    0.10s"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain last output line '%s'.\nGot:\n%s", expected, output)
	}

	// It should NOT contain the package name as the left part (although the output line contains it,
	// we want to ensure the *summary line* is using the output line.
	// Since the output line contains the package name, checking for the output line is sufficient
	// to prove we are using it, provided the package name alone wouldn't match the full line check).
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string // duration string to parse
		expected string
	}{
		// Test cases < 60s (should use HH:MM:SS.mmm format now)
		{
			name:     "zero duration",
			duration: "0s",
			expected: "00:00:00.000",
		},
		{
			name:     "small duration",
			duration: "5.2s",
			expected: "00:00:05.200",
		},
		{
			name:     "boundary case 59.9s",
			duration: "59.9s",
			expected: "00:00:59.900",
		},
		{
			name:     "boundary case exactly 60s",
			duration: "60s",
			expected: "00:01:00.000",
		},
		// Test cases >= 60s (should use HH:MM:SS.mmm format)
		{
			name:     "just over 60s",
			duration: "60.1s",
			expected: "00:01:00.100",
		},
		{
			name:     "1 minute 30 seconds",
			duration: "1m30s",
			expected: "00:01:30.000",
		},
		{
			name:     "boundary case 3599.9s (just under 1 hour)",
			duration: "3599.9s",
			expected: "00:59:59.900",
		},
		{
			name:     "boundary case exactly 3600s (1 hour)",
			duration: "3600s",
			expected: "01:00:00.000",
		},
		{
			name:     "1 hour 23 minutes 45 seconds 678 milliseconds",
			duration: "1h23m45s678ms",
			expected: "01:23:45.678",
		},
		{
			name:     "multiple hours",
			duration: "2h30m15s",
			expected: "02:30:15.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the duration string
			d, err := parseDuration(tt.duration)
			if err != nil {
				t.Fatalf("Failed to parse duration %s: %v", tt.duration, err)
			}

			result := formatDuration(d)
			if result != tt.expected {
				t.Errorf("formatDuration(%s) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// parseDuration is a helper function to parse duration strings for testing.
// It handles the standard Go duration format (e.g., "1h23m45s678ms").
func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func TestSummaryCollectorInterruptedPackage(t *testing.T) {
	collector := NewSummaryCollector()

	// Simulate a package with tests that started but never completed
	startTime := time.Now()
	events := []parser.TestEvent{
		{
			Time:    startTime,
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
		},
		{
			Time:    startTime.Add(100 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
			Output:  "=== RUN   TestOne\n",
		},
		{
			Time:    startTime.Add(500 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
			Elapsed: 0.5,
		},
		{
			Time:    startTime.Add(600 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestTwo",
		},
		{
			Time:    startTime.Add(700 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestTwo",
			Output:  "=== RUN   TestTwo\n",
		},
		// TestTwo never completes - simulating interruption
		// Package never sends completion event
	}

	for _, evt := range events {
		collector.AddTestEvent(evt)
	}

	// Sleep briefly to ensure some wall clock time has passed
	time.Sleep(10 * time.Millisecond)

	// Get summary - should include the incomplete package
	packages, testResults, _, _ := collector.GetSummary()

	// Should have 1 package
	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.Name != "github.com/test/pkg1" {
		t.Errorf("Expected package name 'github.com/test/pkg1', got '%s'", pkg.Name)
	}

	// Package should have status "?" since it never completed
	if pkg.Status != "?" {
		t.Errorf("Expected package status '?', got '%s'", pkg.Status)
	}

	// Should have 1 passed test (TestOne completed)
	if pkg.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.PassedTests)
	}

	// Elapsed time should be > 0 (wall clock time since package started)
	if pkg.Elapsed <= 0 {
		t.Errorf("Expected elapsed time > 0, got %v", pkg.Elapsed)
	}

	// Elapsed time should be at least 10ms (our sleep time)
	if pkg.Elapsed < 10*time.Millisecond {
		t.Errorf("Expected elapsed time >= 10ms, got %v", pkg.Elapsed)
	}

	// Should have 2 test results total
	if len(testResults) != 2 {
		t.Fatalf("Expected 2 test results, got %d", len(testResults))
	}

	// Verify TestOne is marked as passed
	var testOne, testTwo *TestResult
	for _, tr := range testResults {
		if tr.Name == "TestOne" {
			testOne = tr
		} else if tr.Name == "TestTwo" {
			testTwo = tr
		}
	}

	if testOne == nil {
		t.Fatal("TestOne not found in test results")
	}
	if testOne.Status != "pass" {
		t.Errorf("Expected TestOne status 'pass', got '%s'", testOne.Status)
	}

	if testTwo == nil {
		t.Fatal("TestTwo not found in test results")
	}
	if testTwo.Status != "running" {
		t.Errorf("Expected TestTwo status 'running', got '%s'", testTwo.Status)
	}
}

func TestSummaryCollectorMultipleInterruptedPackages(t *testing.T) {
	collector := NewSummaryCollector()

	// Simulate multiple packages, some complete, some incomplete
	events := []parser.TestEvent{
		// Package 1: Complete
		{
			Time:    time.Now(),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestA",
		},
		{
			Time:    time.Now(),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestA",
			Elapsed: 0.1,
		},
		{
			Time:    time.Now(),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.1,
		},
		// Package 2: Incomplete (has tests but no package completion)
		{
			Time:    time.Now(),
			Action:  "run",
			Package: "github.com/test/pkg2",
			Test:    "TestB",
		},
		{
			Time:    time.Now(),
			Action:  "fail",
			Package: "github.com/test/pkg2",
			Test:    "TestB",
			Elapsed: 0.2,
		},
		// No package completion event for pkg2
		// Package 3: Incomplete (test still running)
		{
			Time:    time.Now(),
			Action:  "run",
			Package: "github.com/test/pkg3",
			Test:    "TestC",
		},
		// TestC never completes, package never completes
	}

	for _, evt := range events {
		collector.AddTestEvent(evt)
	}

	packages, _, _, _ := collector.GetSummary()

	// Should have 3 packages
	if len(packages) != 3 {
		t.Fatalf("Expected 3 packages, got %d", len(packages))
	}

	// Find each package
	var pkg1, pkg2, pkg3 *PackageResult
	for _, pkg := range packages {
		switch pkg.Name {
		case "github.com/test/pkg1":
			pkg1 = pkg
		case "github.com/test/pkg2":
			pkg2 = pkg
		case "github.com/test/pkg3":
			pkg3 = pkg
		}
	}

	// Package 1 should be complete with status "ok"
	if pkg1 == nil {
		t.Fatal("Package 1 not found")
	}
	if pkg1.Status != "ok" {
		t.Errorf("Expected pkg1 status 'ok', got '%s'", pkg1.Status)
	}
	if pkg1.PassedTests != 1 {
		t.Errorf("Expected pkg1 to have 1 passed test, got %d", pkg1.PassedTests)
	}

	// Package 2 should be incomplete with status "?"
	if pkg2 == nil {
		t.Fatal("Package 2 not found")
	}
	if pkg2.Status != "?" {
		t.Errorf("Expected pkg2 status '?', got '%s'", pkg2.Status)
	}
	if pkg2.FailedTests != 1 {
		t.Errorf("Expected pkg2 to have 1 failed test, got %d", pkg2.FailedTests)
	}

	// Package 3 should be incomplete with status "?"
	if pkg3 == nil {
		t.Fatal("Package 3 not found")
	}
	if pkg3.Status != "?" {
		t.Errorf("Expected pkg3 status '?', got '%s'", pkg3.Status)
	}
}

func TestSummaryCollectorInterruptedPackageElapsedTime(t *testing.T) {
	collector := NewSummaryCollector()

	// Simulate a package that runs for 5 seconds before interruption
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []parser.TestEvent{
		{
			Time:    baseTime,
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
		},
		{
			Time:    baseTime.Add(2 * time.Second),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
			Elapsed: 2.0,
		},
		{
			Time:    baseTime.Add(3 * time.Second),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestTwo",
		},
		{
			Time:    baseTime.Add(5 * time.Second),
			Action:  "output",
			Package: "github.com/test/pkg1",
			Test:    "TestTwo",
			Output:  "some output\n",
		},
		// Interrupted here - no completion for TestTwo or package
	}

	for _, evt := range events {
		collector.AddTestEvent(evt)
	}

	// Sleep to ensure wall clock time passes
	time.Sleep(50 * time.Millisecond)

	packages, _, _, _ := collector.GetSummary()

	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]

	// Elapsed should be based on wall clock time (at least 50ms)
	if pkg.Elapsed < 50*time.Millisecond {
		t.Errorf("Expected elapsed time >= 50ms, got %v", pkg.Elapsed)
	}

	t.Logf("Package elapsed time: %v", pkg.Elapsed)
}

func TestSummaryCollectorInterruptedPackageWithReplayRate(t *testing.T) {
	collector := NewSummaryCollector()

	// Simulate a package with tests
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []parser.TestEvent{
		{
			Time:    baseTime,
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
		},
		{
			Time:    baseTime.Add(1 * time.Second),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestOne",
			Elapsed: 1.0,
		},
		{
			Time:    baseTime.Add(2 * time.Second),
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestTwo",
		},
		// Interrupted - no completion
	}

	for _, evt := range events {
		collector.AddTestEvent(evt)
	}

	// Sleep for 100ms of wall clock time
	time.Sleep(100 * time.Millisecond)

	// Get summary with replay rate 0.01 (100x slower playback)
	// Wall time: 100ms
	// Simulated time: 100ms / 0.01 = 10,000ms = 10s
	packages, _, _, _ := collector.GetSummaryWithReplay(true, 0.01)

	if len(packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]

	// Elapsed should be at least 10s (100ms wall time / 0.01 rate)
	expectedMin := 10 * time.Second
	if pkg.Elapsed < expectedMin {
		t.Errorf("Expected elapsed time >= %v with replay rate 0.01, got %v", expectedMin, pkg.Elapsed)
	}

	t.Logf("Wall time: ~100ms, Replay rate: 0.01, Simulated elapsed time: %v", pkg.Elapsed)
}
