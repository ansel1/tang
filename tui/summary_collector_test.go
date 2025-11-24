package tui

import (
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
)

// TestSummaryCollectorBasic tests basic functionality of SummaryCollector
func TestSummaryCollectorBasic(t *testing.T) {
	collector := NewSummaryCollector()

	// Create test events
	now := time.Now()
	events := []parser.TestEvent{
		{
			Time:    now,
			Action:  "run",
			Package: "example.com/pkg1",
			Test:    "TestA",
		},
		{
			Time:    now.Add(100 * time.Millisecond),
			Action:  "output",
			Package: "example.com/pkg1",
			Test:    "TestA",
			Output:  "test output line 1\n",
		},
		{
			Time:    now.Add(200 * time.Millisecond),
			Action:  "pass",
			Package: "example.com/pkg1",
			Test:    "TestA",
			Elapsed: 0.2,
		},
		{
			Time:    now.Add(300 * time.Millisecond),
			Action:  "run",
			Package: "example.com/pkg1",
			Test:    "TestB",
		},
		{
			Time:    now.Add(400 * time.Millisecond),
			Action:  "fail",
			Package: "example.com/pkg1",
			Test:    "TestB",
			Elapsed: 0.1,
		},
	}

	// Process events
	for _, event := range events {
		collector.AddTestEvent(event)
	}

	// Add package result
	collector.AddPackageResult("example.com/pkg1", "FAIL", 500*time.Millisecond)

	// Get summary
	packages, testResults, startTime, endTime := collector.GetSummary()

	// Verify package results
	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.Name != "example.com/pkg1" {
		t.Errorf("Expected package name 'example.com/pkg1', got '%s'", pkg.Name)
	}
	if pkg.Status != "FAIL" {
		t.Errorf("Expected package status 'FAIL', got '%s'", pkg.Status)
	}
	if pkg.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.PassedTests)
	}
	if pkg.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", pkg.FailedTests)
	}
	if pkg.Elapsed != 500*time.Millisecond {
		t.Errorf("Expected elapsed time 500ms, got %v", pkg.Elapsed)
	}

	// Verify test results
	if len(testResults) != 2 {
		t.Errorf("Expected 2 test results, got %d", len(testResults))
	}

	// Find TestA and TestB
	var testA, testB *TestResult
	for _, tr := range testResults {
		if tr.Name == "TestA" {
			testA = tr
		} else if tr.Name == "TestB" {
			testB = tr
		}
	}

	if testA == nil {
		t.Fatal("TestA not found in results")
	}
	if testA.Status != "pass" {
		t.Errorf("Expected TestA status 'pass', got '%s'", testA.Status)
	}
	if testA.Elapsed != 200*time.Millisecond {
		t.Errorf("Expected TestA elapsed 200ms, got %v", testA.Elapsed)
	}
	if len(testA.Output) != 1 {
		t.Errorf("Expected 1 output line for TestA, got %d", len(testA.Output))
	}

	if testB == nil {
		t.Fatal("TestB not found in results")
	}
	if testB.Status != "fail" {
		t.Errorf("Expected TestB status 'fail', got '%s'", testB.Status)
	}

	// Verify timing
	if startTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if endTime.IsZero() {
		t.Error("End time should not be zero")
	}
	if !endTime.After(startTime) {
		t.Error("End time should be after start time")
	}
}

// TestSummaryCollectorMultiplePackages tests handling of multiple packages
func TestSummaryCollectorMultiplePackages(t *testing.T) {
	collector := NewSummaryCollector()

	now := time.Now()

	// Package 1 events
	collector.AddTestEvent(parser.TestEvent{
		Time:    now,
		Action:  "run",
		Package: "example.com/pkg1",
		Test:    "TestA",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(100 * time.Millisecond),
		Action:  "pass",
		Package: "example.com/pkg1",
		Test:    "TestA",
		Elapsed: 0.1,
	})
	collector.AddPackageResult("example.com/pkg1", "ok", 100*time.Millisecond)

	// Package 2 events
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(200 * time.Millisecond),
		Action:  "run",
		Package: "example.com/pkg2",
		Test:    "TestB",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(300 * time.Millisecond),
		Action:  "skip",
		Package: "example.com/pkg2",
		Test:    "TestB",
		Elapsed: 0.1,
	})
	collector.AddPackageResult("example.com/pkg2", "ok", 100*time.Millisecond)

	// Get summary
	packages, testResults, _, _ := collector.GetSummary()

	// Verify packages
	if len(packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(packages))
	}

	// Verify package order (chronological)
	if packages[0].Name != "example.com/pkg1" {
		t.Errorf("Expected first package to be pkg1, got %s", packages[0].Name)
	}
	if packages[1].Name != "example.com/pkg2" {
		t.Errorf("Expected second package to be pkg2, got %s", packages[1].Name)
	}

	// Verify test counts
	if packages[0].PassedTests != 1 {
		t.Errorf("Expected pkg1 to have 1 passed test, got %d", packages[0].PassedTests)
	}
	if packages[1].SkippedTests != 1 {
		t.Errorf("Expected pkg2 to have 1 skipped test, got %d", packages[1].SkippedTests)
	}

	// Verify test results
	if len(testResults) != 2 {
		t.Errorf("Expected 2 test results, got %d", len(testResults))
	}
}

// TestSummaryCollectorOutputAccumulation tests that output lines are accumulated correctly
func TestSummaryCollectorOutputAccumulation(t *testing.T) {
	collector := NewSummaryCollector()

	now := time.Now()

	// Test with multiple output lines
	collector.AddTestEvent(parser.TestEvent{
		Time:    now,
		Action:  "run",
		Package: "example.com/pkg",
		Test:    "TestWithOutput",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(10 * time.Millisecond),
		Action:  "output",
		Package: "example.com/pkg",
		Test:    "TestWithOutput",
		Output:  "line 1\n",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(20 * time.Millisecond),
		Action:  "output",
		Package: "example.com/pkg",
		Test:    "TestWithOutput",
		Output:  "line 2\n",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(30 * time.Millisecond),
		Action:  "output",
		Package: "example.com/pkg",
		Test:    "TestWithOutput",
		Output:  "line 3\n",
	})
	collector.AddTestEvent(parser.TestEvent{
		Time:    now.Add(100 * time.Millisecond),
		Action:  "fail",
		Package: "example.com/pkg",
		Test:    "TestWithOutput",
		Elapsed: 0.1,
	})

	// Get summary
	_, testResults, _, _ := collector.GetSummary()

	if len(testResults) != 1 {
		t.Fatalf("Expected 1 test result, got %d", len(testResults))
	}

	test := testResults[0]
	if len(test.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(test.Output))
	}

	expectedOutput := []string{"line 1", "line 2", "line 3"}
	for i, expected := range expectedOutput {
		if i >= len(test.Output) {
			t.Errorf("Missing output line %d", i)
			continue
		}
		if test.Output[i] != expected {
			t.Errorf("Expected output line %d to be '%s', got '%s'", i, expected, test.Output[i])
		}
	}
}

// TestPropertyTestCountConsistency is a property-based test that verifies test count consistency.
// **Feature: test-summary, Property 1: Test count consistency**
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4**
//
// Property: For any test run, the sum of passed, failed, and skipped tests in the summary
// SHALL equal the total number of test events with action "pass", "fail", or "skip".
func TestPropertyTestCountConsistency(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random test events
		events, expectedCounts := generateRandomTestEvents(i)

		// Create collector and process events
		collector := NewSummaryCollector()
		for _, event := range events {
			collector.AddTestEvent(event)
		}

		// Add package results for all packages
		packages := make(map[string]bool)
		for _, event := range events {
			if event.Package != "" {
				packages[event.Package] = true
			}
		}
		for pkg := range packages {
			collector.AddPackageResult(pkg, "ok", 100*time.Millisecond)
		}

		// Get summary
		pkgResults, testResults, _, _ := collector.GetSummary()

		// Count tests by status from test results
		actualPassed := 0
		actualFailed := 0
		actualSkipped := 0
		for _, tr := range testResults {
			switch tr.Status {
			case "pass":
				actualPassed++
			case "fail":
				actualFailed++
			case "skip":
				actualSkipped++
			}
		}

		// Verify counts match expected
		if actualPassed != expectedCounts.passed {
			t.Errorf("Iteration %d: Expected %d passed tests, got %d", i, expectedCounts.passed, actualPassed)
		}
		if actualFailed != expectedCounts.failed {
			t.Errorf("Iteration %d: Expected %d failed tests, got %d", i, expectedCounts.failed, actualFailed)
		}
		if actualSkipped != expectedCounts.skipped {
			t.Errorf("Iteration %d: Expected %d skipped tests, got %d", i, expectedCounts.skipped, actualSkipped)
		}

		// Verify sum equals total
		actualTotal := actualPassed + actualFailed + actualSkipped
		expectedTotal := expectedCounts.passed + expectedCounts.failed + expectedCounts.skipped
		if actualTotal != expectedTotal {
			t.Errorf("Iteration %d: Expected total %d tests, got %d", i, expectedTotal, actualTotal)
		}

		// Verify package aggregation matches
		totalFromPackages := 0
		for _, pkg := range pkgResults {
			totalFromPackages += pkg.PassedTests + pkg.FailedTests + pkg.SkippedTests
		}
		if totalFromPackages != expectedTotal {
			t.Errorf("Iteration %d: Package aggregation mismatch: expected %d, got %d", i, expectedTotal, totalFromPackages)
		}
	}
}

// testCounts holds expected test counts for property testing
type testCounts struct {
	passed  int
	failed  int
	skipped int
}

// generateRandomTestEvents generates a random sequence of test events for property testing.
// It returns the events and the expected counts of passed/failed/skipped tests.
func generateRandomTestEvents(seed int) ([]parser.TestEvent, testCounts) {
	// Use seed for deterministic randomness
	// Simple pseudo-random based on seed
	rng := func(n int) int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed % n
	}

	now := time.Now()
	events := make([]parser.TestEvent, 0)
	counts := testCounts{}

	// Generate 1-20 packages
	numPackages := rng(20) + 1
	packages := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		packages[i] = "example.com/pkg" + string(rune('A'+i))
	}

	// Generate 1-50 tests across packages
	numTests := rng(50) + 1
	for i := 0; i < numTests; i++ {
		pkg := packages[rng(numPackages)]
		testName := "Test" + string(rune('A'+i%26)) + string(rune('0'+i/26))

		// Add run event
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100) * time.Millisecond),
			Action:  "run",
			Package: pkg,
			Test:    testName,
		})

		// Randomly add output events (0-3)
		numOutputs := rng(4)
		for j := 0; j < numOutputs; j++ {
			events = append(events, parser.TestEvent{
				Time:    now.Add(time.Duration(i*100+j*10) * time.Millisecond),
				Action:  "output",
				Package: pkg,
				Test:    testName,
				Output:  "output line\n",
			})
		}

		// Add final status event (pass/fail/skip)
		statusRoll := rng(100)
		var action string
		if statusRoll < 70 {
			// 70% pass
			action = "pass"
			counts.passed++
		} else if statusRoll < 85 {
			// 15% fail
			action = "fail"
			counts.failed++
		} else {
			// 15% skip
			action = "skip"
			counts.skipped++
		}

		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100+50) * time.Millisecond),
			Action:  action,
			Package: pkg,
			Test:    testName,
			Elapsed: 0.05,
		})
	}

	return events, counts
}

// TestPropertyPackageResultCompleteness is a property-based test that verifies package result completeness.
// **Feature: test-summary, Property 2: Package result completeness**
// **Validates: Requirements 1.1, 1.2**
//
// Property: For any test run, every package that produces test events SHALL appear in the package summary section.
func TestPropertyPackageResultCompleteness(t *testing.T) {
	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random test events across multiple packages
		events, expectedPackages := generateRandomPackageEvents(i)

		// Create collector and process events
		collector := NewSummaryCollector()
		for _, event := range events {
			collector.AddTestEvent(event)
		}

		// Add package results for all packages that had test events
		for pkg := range expectedPackages {
			collector.AddPackageResult(pkg, "ok", 100*time.Millisecond)
		}

		// Get summary
		pkgResults, _, _, _ := collector.GetSummary()

		// Build set of packages in results
		actualPackages := make(map[string]bool)
		for _, pkg := range pkgResults {
			actualPackages[pkg.Name] = true
		}

		// Verify all expected packages appear in results
		for expectedPkg := range expectedPackages {
			if !actualPackages[expectedPkg] {
				t.Errorf("Iteration %d: Package %s produced test events but does not appear in summary", i, expectedPkg)
			}
		}

		// Verify no extra packages appear in results
		for actualPkg := range actualPackages {
			if !expectedPackages[actualPkg] {
				t.Errorf("Iteration %d: Package %s appears in summary but had no test events", i, actualPkg)
			}
		}

		// Verify package count matches
		if len(actualPackages) != len(expectedPackages) {
			t.Errorf("Iteration %d: Expected %d packages in summary, got %d", i, len(expectedPackages), len(actualPackages))
		}
	}
}

// generateRandomPackageEvents generates random test events across multiple packages.
// It returns the events and a set of packages that should appear in the summary.
func generateRandomPackageEvents(seed int) ([]parser.TestEvent, map[string]bool) {
	// Use seed for deterministic randomness
	rng := func(n int) int {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		return seed % n
	}

	now := time.Now()
	events := make([]parser.TestEvent, 0)
	expectedPackages := make(map[string]bool)

	// Generate 1-15 packages
	numPackages := rng(15) + 1
	packages := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		// Create diverse package names
		packages[i] = "github.com/test/pkg" + string(rune('A'+i%26))
		if i >= 26 {
			packages[i] += string(rune('0' + i/26))
		}
	}

	// Generate 1-40 tests distributed across packages
	numTests := rng(40) + 1
	for i := 0; i < numTests; i++ {
		// Pick a random package for this test
		pkg := packages[rng(numPackages)]
		expectedPackages[pkg] = true

		testName := "Test" + string(rune('A'+i%26))
		if i >= 26 {
			testName += string(rune('0' + i/26))
		}

		// Add run event
		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100) * time.Millisecond),
			Action:  "run",
			Package: pkg,
			Test:    testName,
		})

		// Randomly add 0-2 output events
		numOutputs := rng(3)
		for j := 0; j < numOutputs; j++ {
			events = append(events, parser.TestEvent{
				Time:    now.Add(time.Duration(i*100+j*10) * time.Millisecond),
				Action:  "output",
				Package: pkg,
				Test:    testName,
				Output:  "test output\n",
			})
		}

		// Add final status event (pass/fail/skip)
		statusRoll := rng(100)
		var action string
		if statusRoll < 75 {
			action = "pass"
		} else if statusRoll < 90 {
			action = "fail"
		} else {
			action = "skip"
		}

		events = append(events, parser.TestEvent{
			Time:    now.Add(time.Duration(i*100+50) * time.Millisecond),
			Action:  action,
			Package: pkg,
			Test:    testName,
			Elapsed: 0.05,
		})
	}

	return events, expectedPackages
}

// TestProcessEvents tests the concurrent event consumer functionality
func TestProcessEvents(t *testing.T) {
	collector := NewSummaryCollector()

	// Create event channel
	events := make(chan engine.Event, 10)

	// Start ProcessEvents in a goroutine
	done := make(chan bool)
	go func() {
		collector.ProcessEvents(events)
		done <- true
	}()

	// Send test events
	now := time.Now()
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now,
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestA",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(100 * time.Millisecond),
			Action:  "pass",
			Package: "example.com/pkg",
			Test:    "TestA",
			Elapsed: 0.1,
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(200 * time.Millisecond),
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestB",
		},
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(300 * time.Millisecond),
			Action:  "fail",
			Package: "example.com/pkg",
			Test:    "TestB",
			Elapsed: 0.1,
		},
	}

	// Send EventComplete to signal end
	events <- engine.Event{
		Type: engine.EventComplete,
	}

	// Close channel
	close(events)

	// Wait for ProcessEvents to complete
	<-done

	// Add package result
	collector.AddPackageResult("example.com/pkg", "FAIL", 300*time.Millisecond)

	// Get summary and verify
	packages, testResults, startTime, endTime := collector.GetSummary()

	// Verify package results
	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages[0]
	if pkg.Name != "example.com/pkg" {
		t.Errorf("Expected package name 'example.com/pkg', got '%s'", pkg.Name)
	}
	if pkg.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.PassedTests)
	}
	if pkg.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", pkg.FailedTests)
	}

	// Verify test results
	if len(testResults) != 2 {
		t.Errorf("Expected 2 test results, got %d", len(testResults))
	}

	// Verify timing
	if startTime.IsZero() {
		t.Error("Start time should not be zero")
	}
	if endTime.IsZero() {
		t.Error("End time should not be zero")
	}
}

// TestProcessEventsIgnoresNonTestEvents tests that ProcessEvents ignores non-test events
func TestProcessEventsIgnoresNonTestEvents(t *testing.T) {
	collector := NewSummaryCollector()

	// Create event channel
	events := make(chan engine.Event, 10)

	// Start ProcessEvents in a goroutine
	done := make(chan bool)
	go func() {
		collector.ProcessEvents(events)
		done <- true
	}()

	now := time.Now()

	// Send various event types
	events <- engine.Event{
		Type:    engine.EventRawLine,
		RawLine: []byte("some raw output"),
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now,
			Action:  "run",
			Package: "example.com/pkg",
			Test:    "TestA",
		},
	}
	events <- engine.Event{
		Type:  engine.EventError,
		Error: nil,
	}
	events <- engine.Event{
		Type: engine.EventTest,
		TestEvent: parser.TestEvent{
			Time:    now.Add(100 * time.Millisecond),
			Action:  "pass",
			Package: "example.com/pkg",
			Test:    "TestA",
			Elapsed: 0.1,
		},
	}
	events <- engine.Event{
		Type: engine.EventComplete,
	}

	close(events)
	<-done

	// Verify only test events were processed
	_, testResults, _, _ := collector.GetSummary()
	if len(testResults) != 1 {
		t.Errorf("Expected 1 test result (only EventTest should be processed), got %d", len(testResults))
	}
}

// TestProcessEventsConcurrentGetSummary tests that GetSummary can be called while ProcessEvents is running
func TestProcessEventsConcurrentGetSummary(t *testing.T) {
	collector := NewSummaryCollector()

	// Create event channel
	events := make(chan engine.Event, 100)

	// Start ProcessEvents in a goroutine
	go collector.ProcessEvents(events)

	now := time.Now()

	// Send some test events
	for i := 0; i < 10; i++ {
		events <- engine.Event{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(time.Duration(i*100) * time.Millisecond),
				Action:  "run",
				Package: "example.com/pkg",
				Test:    "Test" + string(rune('A'+i)),
			},
		}
		events <- engine.Event{
			Type: engine.EventTest,
			TestEvent: parser.TestEvent{
				Time:    now.Add(time.Duration(i*100+50) * time.Millisecond),
				Action:  "pass",
				Package: "example.com/pkg",
				Test:    "Test" + string(rune('A'+i)),
				Elapsed: 0.05,
			},
		}

		// Call GetSummary concurrently while events are being processed
		// This should not cause a race condition
		_, testResults, _, _ := collector.GetSummary()

		// Verify we can read the current state
		if len(testResults) < i+1 {
			// It's okay if we haven't processed all events yet
			// The important thing is that we don't crash or deadlock
		}
	}

	// Send EventComplete
	events <- engine.Event{
		Type: engine.EventComplete,
	}
	close(events)

	// Give ProcessEvents time to complete
	time.Sleep(100 * time.Millisecond)

	// Final verification
	_, testResults, _, _ := collector.GetSummary()
	if len(testResults) != 10 {
		t.Errorf("Expected 10 test results, got %d", len(testResults))
	}
}
