package format

import (
	"testing"
	"time"

	"github.com/ansel1/tang/results"
)

// TestComputeSummaryBasic tests basic summary computation with mixed results.
func TestComputeSummaryBasic(t *testing.T) {
	run := results.NewRun(1)

	// Package 1: 2 pass, 1 fail
	pkg1 := &results.PackageResult{
		Name:      "pkg1",
		Status:    results.StatusFailed,
		Elapsed:   5 * time.Second,
		TestOrder: []string{"TestA", "TestB", "TestC"},
	}
	pkg1.Counts.Passed = 2
	pkg1.Counts.Failed = 1
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")

	run.TestResults["pkg1/TestA"] = &results.TestResult{
		Package: "pkg1", Name: "TestA", Status: results.StatusPassed, Elapsed: 1.0 * time.Second,
	}
	run.TestResults["pkg1/TestB"] = &results.TestResult{
		Package: "pkg1", Name: "TestB", Status: results.StatusFailed, Elapsed: 1.0 * time.Second,
	}
	run.TestResults["pkg1/TestC"] = &results.TestResult{
		Package: "pkg1", Name: "TestC", Status: results.StatusPassed, Elapsed: 1.0 * time.Second,
	}

	// Package 2: 1 pass, 1 skip
	pkg2 := &results.PackageResult{
		Name:      "pkg2",
		Status:    results.StatusPassed,
		Elapsed:   3 * time.Second,
		TestOrder: []string{"TestD", "TestE"},
	}
	pkg2.Counts.Passed = 1
	pkg2.Counts.Skipped = 1
	run.Packages["pkg2"] = pkg2
	run.PackageOrder = append(run.PackageOrder, "pkg2")

	run.TestResults["pkg2/TestD"] = &results.TestResult{
		Package: "pkg2", Name: "TestD", Status: results.StatusPassed, Elapsed: 1.0 * time.Second,
	}
	run.TestResults["pkg2/TestE"] = &results.TestResult{
		Package: "pkg2", Name: "TestE", Status: results.StatusSkipped, Elapsed: 1.0 * time.Second,
	}

	// Compute summary
	summary := ComputeSummary(run, 10*time.Second)

	// Verify overall statistics
	if summary.TotalTests != 5 {
		t.Errorf("Expected 5 total tests, got %d", summary.TotalTests)
	}
	if summary.PassedTests != 3 {
		t.Errorf("Expected 3 passed tests, got %d", summary.PassedTests)
	}
	if summary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", summary.FailedTests)
	}
	if summary.SkippedTests != 1 {
		t.Errorf("Expected 1 skipped test, got %d", summary.SkippedTests)
	}
	if summary.PackageCount != 2 {
		t.Errorf("Expected 2 packages, got %d", summary.PackageCount)
	}

	// Verify failures collection
	if len(summary.Failures) != 1 {
		t.Errorf("Expected 1 failure, got %d", len(summary.Failures))
	}
	if len(summary.Failures) > 0 && summary.Failures[0].Name != "TestB" {
		t.Errorf("Expected failure TestB, got %s", summary.Failures[0].Name)
	}

	// Verify skipped collection
	if len(summary.Skipped) != 1 {
		t.Errorf("Expected 1 skipped test, got %d", len(summary.Skipped))
	}
	if len(summary.Skipped) > 0 && summary.Skipped[0].Name != "TestE" {
		t.Errorf("Expected skipped TestE, got %s", summary.Skipped[0].Name)
	}

	// Verify package statistics
	if summary.FastestPackage == nil {
		t.Error("Expected fastest package to be set")
	} else if summary.FastestPackage.Name != "pkg2" {
		t.Errorf("Expected fastest package to be pkg2, got %s", summary.FastestPackage.Name)
	}

	if summary.SlowestPackage == nil {
		t.Error("Expected slowest package to be set")
	} else if summary.SlowestPackage.Name != "pkg1" {
		t.Errorf("Expected slowest package to be pkg1, got %s", summary.SlowestPackage.Name)
	}

	if summary.MostTestsPackage == nil {
		t.Error("Expected most tests package to be set")
	} else if summary.MostTestsPackage.Name != "pkg1" {
		t.Errorf("Expected most tests package to be pkg1, got %s", summary.MostTestsPackage.Name)
	}
}

// TestComputeSummarySlowTests tests slow test detection and sorting.
func TestComputeSummarySlowTests(t *testing.T) {
	run := results.NewRun(1)

	pkg1 := &results.PackageResult{
		Name:    "pkg1",
		Status:  results.StatusPassed,
		Elapsed: 60 * time.Second,
	}
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")

	tests := []struct {
		name    string
		elapsed float64
	}{
		{"TestFast", 5.0},
		{"TestSlow1", 15.0},
		{"TestSlow2", 25.0},
		{"TestSlow3", 12.0},
	}

	for _, test := range tests {
		run.TestResults["pkg1/"+test.name] = &results.TestResult{
			Package: "pkg1", Name: test.name, Status: results.StatusPassed, Elapsed: time.Duration(test.elapsed * float64(time.Second)),
		}
		pkg1.TestOrder = append(pkg1.TestOrder, test.name)
	}

	// Compute summary with 10s threshold
	summary := ComputeSummary(run, 10*time.Second)

	// Verify slow tests detected
	if len(summary.SlowTests) != 3 {
		t.Errorf("Expected 3 slow tests, got %d", len(summary.SlowTests))
	}

	// Verify slow tests are sorted by duration (descending)
	if len(summary.SlowTests) >= 3 {
		if summary.SlowTests[0].Name != "TestSlow2" {
			t.Errorf("Expected first slow test to be TestSlow2 (25s), got %s", summary.SlowTests[0].Name)
		}
		if summary.SlowTests[1].Name != "TestSlow1" {
			t.Errorf("Expected second slow test to be TestSlow1 (15s), got %s", summary.SlowTests[1].Name)
		}
		if summary.SlowTests[2].Name != "TestSlow3" {
			t.Errorf("Expected third slow test to be TestSlow3 (12s), got %s", summary.SlowTests[2].Name)
		}
	}
}

// TestComputeSummaryEmptyResults tests summary with no tests.
func TestComputeSummaryEmptyResults(t *testing.T) {
	run := results.NewRun(1)
	summary := ComputeSummary(run, 10*time.Second)

	if summary.TotalTests != 0 {
		t.Errorf("Expected 0 total tests, got %d", summary.TotalTests)
	}
	if summary.PackageCount != 0 {
		t.Errorf("Expected 0 packages, got %d", summary.PackageCount)
	}
	if len(summary.Failures) != 0 {
		t.Errorf("Expected 0 failures, got %d", len(summary.Failures))
	}
	if len(summary.Skipped) != 0 {
		t.Errorf("Expected 0 skipped, got %d", len(summary.Skipped))
	}
	if len(summary.SlowTests) != 0 {
		t.Errorf("Expected 0 slow tests, got %d", len(summary.SlowTests))
	}
}

// TestComputeSummaryAllPass tests summary with all passing tests.
func TestComputeSummaryAllPass(t *testing.T) {
	run := results.NewRun(1)

	pkg1 := &results.PackageResult{
		Name:      "pkg1",
		Status:    results.StatusPassed,
		Elapsed:   6 * time.Second,
		TestOrder: []string{"TestA", "TestB", "TestC"},
	}
	pkg1.Counts.Passed = 3
	run.Packages["pkg1"] = pkg1
	run.PackageOrder = append(run.PackageOrder, "pkg1")

	for _, name := range []string{"TestA", "TestB", "TestC"} {
		run.TestResults["pkg1/"+name] = &results.TestResult{
			Package: "pkg1", Name: name, Status: results.StatusPassed, Elapsed: 1.0 * time.Second,
		}
	}

	summary := ComputeSummary(run, 10*time.Second)

	if summary.TotalTests != 3 {
		t.Errorf("Expected 3 total tests, got %d", summary.TotalTests)
	}
	if summary.PassedTests != 3 {
		t.Errorf("Expected 3 passed tests, got %d", summary.PassedTests)
	}
	if summary.FailedTests != 0 {
		t.Errorf("Expected 0 failed tests, got %d", summary.FailedTests)
	}
	if summary.SkippedTests != 0 {
		t.Errorf("Expected 0 skipped tests, got %d", summary.SkippedTests)
	}
	if len(summary.Failures) != 0 {
		t.Errorf("Expected empty failures list, got %d", len(summary.Failures))
	}
	if len(summary.Skipped) != 0 {
		t.Errorf("Expected empty skipped list, got %d", len(summary.Skipped))
	}
}
