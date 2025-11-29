package results

import (
	"testing"
	"time"

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
		collector.handleTestEvent(evt)
	}

	// Sleep briefly to ensure some wall clock time has passed (if needed for elapsed calc)
	// Note: results.Collector currently uses event.Elapsed for finished items.
	// For interrupted items, it doesn't calculate elapsed time yet in GetState.
	// But we can check status.

	state := collector.GetState()
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

	// Package should have default status (empty or running) since it never completed
	// results.Collector sets status on pass/fail/skip.
	// Initial status is empty string? No, NewPackageResult doesn't set status.
	// Wait, NewPackageResult isn't used in handleTestEvent, it creates struct literal.
	// Status defaults to "".
	// We might want to set it to "running" initially?
	// Let's check collector.go.
	// It creates &PackageResult{...}. Status is empty.
	// If I want to verify it's NOT "ok" or "FAIL", that's fine.
	if pkg.Status == "ok" || pkg.Status == "FAIL" {
		t.Errorf("Expected package status to be incomplete, got '%s'", pkg.Status)
	}

	// Should have 1 passed test (TestOne completed)
	if pkg.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.PassedTests)
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
	if testOne.Status != "pass" {
		t.Errorf("Expected TestOne status 'pass', got '%s'", testOne.Status)
	}

	testTwo := run.TestResults["github.com/test/pkg1/TestTwo"]
	if testTwo == nil {
		t.Fatal("TestTwo not found in test results")
	}
	if testTwo.Status != "running" {
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
		collector.handleTestEvent(evt)
	}

	state := collector.GetState()
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
	if pkg1.Status != "ok" {
		t.Errorf("Expected pkg1 status 'ok', got '%s'", pkg1.Status)
	}
	if pkg1.PassedTests != 1 {
		t.Errorf("Expected pkg1 to have 1 passed test, got %d", pkg1.PassedTests)
	}

	// Package 2 should be incomplete
	if pkg2 == nil {
		t.Fatal("Package 2 not found")
	}
	if pkg2.Status == "ok" || pkg2.Status == "FAIL" {
		t.Errorf("Expected pkg2 status to be incomplete, got '%s'", pkg2.Status)
	}
	if pkg2.FailedTests != 1 {
		t.Errorf("Expected pkg2 to have 1 failed test, got %d", pkg2.FailedTests)
	}

	// Package 3 should be incomplete
	if pkg3 == nil {
		t.Fatal("Package 3 not found")
	}
	if pkg3.Status == "ok" || pkg3.Status == "FAIL" {
		t.Errorf("Expected pkg3 status to be incomplete, got '%s'", pkg3.Status)
	}
}
