package format

import (
	"strings"
	"testing"
	"time"

	"github.com/ansel1/tang/results"
)

func TestSummaryFormatterBasic(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 10

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "ok") {
		t.Error("Expected 'ok' status for passed package")
	}
	if !strings.Contains(output, "github.com/user/project/pkg1") {
		t.Error("Expected package name")
	}
	if !strings.Contains(output, "(1 packages)") {
		t.Error("Expected totals line with package count")
	}
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol")
	}
}

func TestSummaryFormatterWithFailures(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusFailed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Failed = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	failTest := results.NewTestResult("github.com/user/project/pkg1", "TestFailing")
	failTest.Latest().Status = results.StatusFailed
	failTest.Latest().Elapsed = 1 * time.Second
	failTest.Latest().Output = []string{"Error: expected true, got false", "at line 42"}
	run.TestResults["github.com/user/project/pkg1/TestFailing"] = failTest
	pkg1.TestOrder = []string{"TestFailing"}

	// Create TestExecutionEntry for the failure
	failEntry := &TestExecutionEntry{
		TestResult:      failTest,
		TestExecution:   failTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  2,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Failures:     []*TestExecutionEntry{failEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "FAIL") {
		t.Error("Expected FAIL label in output")
	}
	if !strings.Contains(output, "TestFailing") {
		t.Error("Expected test name in failures")
	}
	if !strings.Contains(output, "Error: expected true, got false") {
		t.Error("Expected failure output")
	}
}

func TestSummaryFormatterWithSkipped(t *testing.T) {
	formatter := NewSummaryFormatter(80, false, SummaryOptions{IncludeSkipped: true})

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Skipped = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	skipTest := results.NewTestResult("github.com/user/project/pkg1", "TestSkipped")
	skipTest.Latest().Status = results.StatusSkipped
	skipTest.Latest().Elapsed = 0
	skipTest.Latest().Output = []string{"Skipping: not implemented yet"}
	run.TestResults["github.com/user/project/pkg1/TestSkipped"] = skipTest
	pkg1.TestOrder = []string{"TestSkipped"}

	// Create TestExecutionEntry for the skipped test
	skipEntry := &TestExecutionEntry{
		TestResult:      skipTest,
		TestExecution:   skipTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  0,
		SkippedTests: 2,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped:      []*TestExecutionEntry{skipEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "SKIP") {
		t.Error("Expected SKIP label")
	}
	if !strings.Contains(output, "TestSkipped") {
		t.Error("Expected test name in skipped")
	}
	if !strings.Contains(output, "Skipping: not implemented yet") {
		t.Error("Expected skip reason")
	}
}

func TestSummaryFormatterWithSlowTests(t *testing.T) {
	formatter := NewSummaryFormatter(80, false, SummaryOptions{IncludeSlow: true})

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 65 * time.Second,
	}
	pkg1.Counts.Passed = 2

	slowTest := results.NewTestResult("github.com/user/project/pkg1", "TestSlow")
	slowTest.Latest().Status = results.StatusPassed
	slowTest.Latest().Elapsed = 65 * time.Second

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}
	run.TestResults["github.com/user/project/pkg1/TestSlow"] = slowTest
	pkg1.TestOrder = []string{"TestSlow"}

	// Create TestExecutionEntry for the slow test
	slowEntry := &TestExecutionEntry{
		TestResult:      slowTest,
		TestExecution:   slowTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   2,
		PassedTests:  2,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    65 * time.Second,
		PackageCount: 1,
		SlowTests:    []*TestExecutionEntry{slowEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "SLOW") {
		t.Error("Expected SLOW label")
	}
	if !strings.Contains(output, "TestSlow") {
		t.Error("Expected test name in slow tests")
	}
}

func TestSummaryFormatterSkippedHiddenByDefault(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Skipped = 2

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}

	skipTest := results.NewTestResult("github.com/user/project/pkg1", "TestSkipped")
	skipTest.Latest().Status = results.StatusSkipped
	skipTest.Latest().Elapsed = 0
	skipTest.Latest().Output = []string{"Skipping: not implemented yet"}
	run.TestResults["github.com/user/project/pkg1/TestSkipped"] = skipTest
	pkg1.TestOrder = []string{"TestSkipped"}

	// Create TestExecutionEntry for the skipped test
	skipEntry := &TestExecutionEntry{
		TestResult:      skipTest,
		TestExecution:   skipTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  0,
		SkippedTests: 2,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped:      []*TestExecutionEntry{skipEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "--- SKIP") {
		t.Error("Skipped test details should be hidden by default")
	}
	if strings.Contains(output, "TestSkipped") {
		t.Error("Skipped test name should be hidden by default")
	}
}

func TestSummaryFormatterSlowHiddenByDefault(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 65 * time.Second,
	}
	pkg1.Counts.Passed = 2

	slowTest := results.NewTestResult("github.com/user/project/pkg1", "TestSlow")
	slowTest.Latest().Status = results.StatusPassed
	slowTest.Latest().Elapsed = 65 * time.Second

	run := results.NewRun(1)
	run.Packages["github.com/user/project/pkg1"] = pkg1
	run.PackageOrder = []string{"github.com/user/project/pkg1"}
	run.TestResults["github.com/user/project/pkg1/TestSlow"] = slowTest
	pkg1.TestOrder = []string{"TestSlow"}

	// Create TestExecutionEntry for the slow test
	slowEntry := &TestExecutionEntry{
		TestResult:      slowTest,
		TestExecution:   slowTest.Latest(),
		Iteration:       1,
		TotalExecutions: 1,
	}

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   2,
		PassedTests:  2,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    65 * time.Second,
		PackageCount: 1,
		SlowTests:    []*TestExecutionEntry{slowEntry},
		Run:          run,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "SLOW") {
		t.Error("Slow test details should be hidden by default")
	}
}

func TestSummaryFormatterNoFailuresOrSkips(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 5 * time.Second,
	}
	pkg1.Counts.Passed = 10

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	if strings.Contains(output, "--- FAIL") {
		t.Error("FAIL details should be omitted when no failures")
	}
	if strings.Contains(output, "--- SKIP") {
		t.Error("SKIP details should be omitted when no skips")
	}
}

func TestSummaryFormatterTotalsLine(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "github.com/user/project/pkg1",
		Status:  results.StatusPassed,
		Elapsed: 3 * time.Second,
	}
	pkg1.Counts.Passed = 8
	pkg1.Counts.Failed = 1
	pkg1.Counts.Skipped = 1

	pkg2 := &results.PackageResult{
		Name:    "github.com/user/project/pkg2",
		Status:  results.StatusPassed,
		Elapsed: 2 * time.Second,
	}
	pkg2.Counts.Passed = 5

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1, pkg2},
		TotalTests:   15,
		PassedTests:  13,
		FailedTests:  1,
		SkippedTests: 1,
		TotalTime:    5 * time.Second,
		PackageCount: 2,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, "(2 packages)") {
		t.Error("Expected totals line with package count")
	}
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol in totals")
	}
	if !strings.Contains(output, SymbolFail) {
		t.Error("Expected fail symbol in totals")
	}
	if !strings.Contains(output, SymbolSkip) {
		t.Error("Expected skip symbol in totals")
	}
}

func TestSummaryFormatterSymbols(t *testing.T) {
	formatter := NewSummaryFormatter(80, false)

	pkg1 := &results.PackageResult{
		Name:    "pkg1",
		Status:  results.StatusPassed,
		Elapsed: 1 * time.Second,
	}
	pkg1.Counts.Passed = 1

	pkg2 := &results.PackageResult{
		Name:    "pkg2",
		Status:  results.StatusFailed,
		Elapsed: 1 * time.Second,
	}
	pkg2.Counts.Failed = 1

	summary := &Summary{
		Packages:     []*results.PackageResult{pkg1, pkg2},
		TotalTests:   2,
		PassedTests:  1,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    2 * time.Second,
		PackageCount: 2,
	}

	output := formatter.Format(summary)

	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol")
	}
	if !strings.Contains(output, SymbolFail) {
		t.Error("Expected fail symbol")
	}
}
