package results

import (
	"testing"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
)

func TestCollectorInterruptedPackage(t *testing.T) {
	collector := NewCollector()

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
		collector.Push(engine.Event{Type: engine.EventTest, TestEvent: evt})
	}

	state := collector.State()
	if len(state.Runs) != 1 {
		t.Fatalf("Expected 1 run, got %d", len(state.Runs))
	}
	run := state.Runs[0]

	// Should have 1 package
	if len(run.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(run.Packages))
	}

	pkg := run.Packages["github.com/test/pkg1"]
	if pkg.Name != "github.com/test/pkg1" {
		t.Errorf("Expected package name 'github.com/test/pkg1', got '%s'", pkg.Name)
	}

	if pkg.Status == StatusPassed || pkg.Status == StatusFailed {
		t.Errorf("Expected package status to be incomplete, got '%s'", pkg.Status)
	}

	// Should have 1 passed test (TestOne completed)
	if pkg.Counts.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.Counts.Passed)
	}

	// Should have 2 test results total
	if len(run.TestResults) != 2 {
		t.Fatalf("Expected 2 test results, got %d", len(run.TestResults))
	}

	// Verify TestOne is marked as passed
	testOne := run.TestResults["github.com/test/pkg1/TestOne"]
	if testOne == nil {
		t.Fatal("TestOne not found in test results")
	}
	if testOne.Status != StatusPassed {
		t.Errorf("Expected TestOne status 'pass', got '%s'", testOne.Status)
	}

	testTwo := run.TestResults["github.com/test/pkg1/TestTwo"]
	if testTwo == nil {
		t.Fatal("TestTwo not found in test results")
	}
	if testTwo.Status != StatusRunning {
		t.Errorf("Expected TestTwo status 'running', got '%s'", testTwo.Status)
	}
}

func TestCollectorMultipleInterruptedPackages(t *testing.T) {
	collector := NewCollector()

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
		// Start pkg2 BEFORE pkg1 finishes to keep run alive
		{
			Time:    time.Now(),
			Action:  "run",
			Package: "github.com/test/pkg2",
			Test:    "TestB",
		},
		{
			Time:    time.Now(),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.1,
		},
		// Package 2 continues...
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
		collector.Push(engine.Event{Type: engine.EventTest, TestEvent: evt})
	}

	state := collector.State()
	if len(state.Runs) != 1 {
		t.Fatalf("Expected 1 run, got %d", len(state.Runs))
	}
	run := state.Runs[0]

	// Should have 3 packages
	if len(run.Packages) != 3 {
		t.Fatalf("Expected 3 packages, got %d", len(run.Packages))
	}

	pkg1 := run.Packages["github.com/test/pkg1"]
	pkg2 := run.Packages["github.com/test/pkg2"]
	pkg3 := run.Packages["github.com/test/pkg3"]

	// Package 1 should be complete with status "ok"
	if pkg1 == nil {
		t.Fatal("Package 1 not found")
	}
	if pkg1.Status != StatusPassed {
		t.Errorf("Expected pkg1 status 'passed', got '%s'", pkg1.Status)
	}
	if pkg1.Counts.Passed != 1 {
		t.Errorf("Expected pkg1 to have 1 passed test, got %d", pkg1.Counts.Passed)
	}

	// Package 2 should be incomplete
	if pkg2 == nil {
		t.Fatal("Package 2 not found")
	}
	if pkg2.Status == StatusPassed || pkg2.Status == StatusFailed {
		t.Errorf("Expected pkg2 status to be incomplete, got '%s'", pkg2.Status)
	}
	if pkg2.Counts.Failed != 1 {
		t.Errorf("Expected pkg2 to have 1 failed test, got %d", pkg2.Counts.Failed)
	}

	// Package 3 should be incomplete
	if pkg3 == nil {
		t.Fatal("Package 3 not found")
	}
	if pkg3.Status == StatusPassed || pkg3.Status == StatusFailed {
		t.Errorf("Expected pkg3 status to be incomplete, got '%s'", pkg3.Status)
	}
}

// TestCollectorFinishInterruptedRun tests that finishCurrentRun properly marks
// interrupted packages and computes their elapsed time
func TestCollectorFinishInterruptedRun(t *testing.T) {
	collector := NewCollector()

	startTime := time.Now().Add(-2 * time.Second) // 2 seconds ago
	events := []parser.TestEvent{
		// Package 1: Completes normally
		{
			Time:    startTime,
			Action:  "run",
			Package: "github.com/test/pkg1",
			Test:    "TestA",
		},
		{
			Time:    startTime.Add(100 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Test:    "TestA",
			Elapsed: 0.1,
		},
		// Start pkg2 BEFORE pkg1 finishes to keep run alive
		{
			Time:    startTime.Add(150 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg2",
			Test:    "TestB",
		},
		{
			Time:    startTime.Add(200 * time.Millisecond),
			Action:  "pass",
			Package: "github.com/test/pkg1",
			Elapsed: 0.2,
		},
		// Package 2: Never completes
		{
			Time:    startTime.Add(400 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg2",
			Test:    "TestB",
			Output:  "some test output\n",
		},
		// Package 3: Never completes and has an output line
		{
			Time:    startTime.Add(500 * time.Millisecond),
			Action:  "run",
			Package: "github.com/test/pkg3",
			Test:    "TestC",
		},
		{
			Time:    startTime.Add(600 * time.Millisecond),
			Action:  "output",
			Package: "github.com/test/pkg3",
			Output:  "ERR | Some error message\n",
		},
	}

	for _, evt := range events {
		collector.Push(engine.Event{Type: engine.EventTest, TestEvent: evt})
	}

	// Verify state before finishCurrentRun
	{
		run := collector.State().CurrentRun
		if run == nil {
			t.Fatal("Expected a current run before finish")
		}

		pkg1 := run.Packages["github.com/test/pkg1"]
		pkg2 := run.Packages["github.com/test/pkg2"]
		pkg3 := run.Packages["github.com/test/pkg3"]

		if pkg1.Status != StatusPassed {
			t.Errorf("Expected pkg1 status 'ok' before finish, got '%s'", pkg1.Status)
		}

		if pkg2.Status != StatusRunning {
			t.Errorf("Expected pkg2 status 'running' before finish, got '%s'", pkg2.Status)
		}

		if pkg3.Status != StatusRunning {
			t.Errorf("Expected pkg3 status 'running' before finish, got '%s'", pkg3.Status)
		}
	}

	// Now finish the run
	collector.Finish()

	// Verify interrupted packages were handled correctly
	{
		state := collector.State()
		if state.CurrentRun != nil {
			t.Error("Expected CurrentRun to be nil after finish")
		}

		if len(state.Runs) != 1 {
			t.Fatalf("Expected 1 run, got %d", len(state.Runs))
		}

		run := state.Runs[0]

		// Check EndTime was set
		if run.LastEventTime.IsZero() {
			t.Error("Expected EndTime to be set")
		}

		pkg1 := run.Packages["github.com/test/pkg1"]
		pkg2 := run.Packages["github.com/test/pkg2"]
		pkg3 := run.Packages["github.com/test/pkg3"]

		// pkg1 should remain "ok"
		if pkg1.Status != StatusPassed {
			t.Errorf("Expected pkg1 status 'ok', got '%s'", pkg1.Status)
		}

		// pkg2 should be marked as "interrupted"
		if pkg2.Status != StatusInterrupted {
			t.Errorf("Expected pkg2 status 'interrupted', got '%s'", pkg2.Status)
		}

		// pkg3 should be marked as "interrupted"
		if pkg3.Status != StatusInterrupted {
			t.Errorf("Expected pkg3 status 'interrupted', got '%s'", pkg3.Status)
		}

		// pkg2 and pkg3 should have computed elapsed times
		if pkg2.Elapsed == 0 {
			t.Error("Expected pkg2 to have non-zero elapsed time")
		}
		if pkg3.Elapsed == 0 {
			t.Error("Expected pkg3 to have non-zero elapsed time")
		}

		// Elapsed times should be reasonable (at least the time since their start)
		expectedMinElapsed := time.Since(startTime.Add(300 * time.Millisecond))
		if pkg2.Elapsed < expectedMinElapsed-100*time.Millisecond {
			t.Errorf("pkg2 elapsed time %v seems too small (expected at least %v)", pkg2.Elapsed, expectedMinElapsed)
		}

		expectedMinElapsed = time.Since(startTime.Add(500 * time.Millisecond))
		if pkg3.Elapsed < expectedMinElapsed-100*time.Millisecond {
			t.Errorf("pkg3 elapsed time %v seems too small (expected at least %v)", pkg3.Elapsed, expectedMinElapsed)
		}
	}
}
