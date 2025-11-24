package tui

import (
	"strings"
	"testing"
	"time"
)

// TestSummaryFormatterBasic tests basic formatting functionality
func TestSummaryFormatterBasic(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create a simple summary
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "ok",
				Elapsed:      5 * time.Second,
				PassedTests:  10,
				FailedTests:  0,
				SkippedTests: 0,
			},
		},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	// Verify sections are present
	if !strings.Contains(output, "PACKAGES") {
		t.Error("Expected PACKAGES section")
	}
	if !strings.Contains(output, "OVERALL RESULTS") {
		t.Error("Expected OVERALL RESULTS section")
	}
	if !strings.Contains(output, "PACKAGE STATISTICS") {
		t.Error("Expected PACKAGE STATISTICS section")
	}

	// Verify symbols are used
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol")
	}
}

// TestSummaryFormatterWithFailures tests failure formatting
func TestSummaryFormatterWithFailures(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create summary with failures
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "FAIL",
				Elapsed:      5 * time.Second,
				PassedTests:  8,
				FailedTests:  2,
				SkippedTests: 0,
			},
		},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  2,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Failures: []*TestResult{
			{
				Package: "github.com/user/project/pkg1",
				Name:    "TestFailing",
				Status:  "fail",
				Elapsed: 1 * time.Second,
				Output:  []string{"Error: expected true, got false", "at line 42"},
			},
		},
	}

	output := formatter.Format(summary)

	// Verify FAILURES section is present
	if !strings.Contains(output, "FAILURES") {
		t.Error("Expected FAILURES section")
	}
	if !strings.Contains(output, "TestFailing") {
		t.Error("Expected test name in failures")
	}
	if !strings.Contains(output, "Error: expected true, got false") {
		t.Error("Expected failure output")
	}
}

// TestSummaryFormatterWithSkipped tests skipped test formatting
func TestSummaryFormatterWithSkipped(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create summary with skipped tests
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "ok",
				Elapsed:      5 * time.Second,
				PassedTests:  8,
				FailedTests:  0,
				SkippedTests: 2,
			},
		},
		TotalTests:   10,
		PassedTests:  8,
		FailedTests:  0,
		SkippedTests: 2,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped: []*TestResult{
			{
				Package: "github.com/user/project/pkg1",
				Name:    "TestSkipped",
				Status:  "skip",
				Elapsed: 0,
				Output:  []string{"Skipping: not implemented yet"},
			},
		},
	}

	output := formatter.Format(summary)

	// Verify SKIPPED section is present
	if !strings.Contains(output, "SKIPPED") {
		t.Error("Expected SKIPPED section")
	}
	if !strings.Contains(output, "TestSkipped") {
		t.Error("Expected test name in skipped")
	}
	if !strings.Contains(output, "Skipping: not implemented yet") {
		t.Error("Expected skip reason")
	}
}

// TestSummaryFormatterWithSlowTests tests slow test formatting
func TestSummaryFormatterWithSlowTests(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create summary with slow tests
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "ok",
				Elapsed:      65 * time.Second,
				PassedTests:  2,
				FailedTests:  0,
				SkippedTests: 0,
			},
		},
		TotalTests:   2,
		PassedTests:  2,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    65 * time.Second,
		PackageCount: 1,
		SlowTests: []*TestResult{
			{
				Package: "github.com/user/project/pkg1",
				Name:    "TestSlow",
				Status:  "pass",
				Elapsed: 65 * time.Second,
			},
		},
	}

	output := formatter.Format(summary)

	// Verify SLOW TESTS section is present
	if !strings.Contains(output, "SLOW TESTS") {
		t.Error("Expected SLOW TESTS section")
	}
	if !strings.Contains(output, "TestSlow") {
		t.Error("Expected test name in slow tests")
	}
	// Should use HH:MM:SS.mmm format for >= 60s
	if !strings.Contains(output, "00:01:05") {
		t.Error("Expected time in HH:MM:SS format")
	}
}

// TestSummaryFormatterNoFailuresOrSkips tests section omission
func TestSummaryFormatterNoFailuresOrSkips(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create summary with no failures or skips
	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "ok",
				Elapsed:      5 * time.Second,
				PassedTests:  10,
				FailedTests:  0,
				SkippedTests: 0,
			},
		},
		TotalTests:   10,
		PassedTests:  10,
		FailedTests:  0,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	// Verify FAILURES and SKIPPED sections are omitted
	if strings.Contains(output, "FAILURES") {
		t.Error("FAILURES section should be omitted when no failures")
	}
	if strings.Contains(output, "SKIPPED") {
		t.Error("SKIPPED section should be omitted when no skips")
	}
}

// TestSummaryFormatterOutputTruncation tests output line truncation
func TestSummaryFormatterOutputTruncation(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create failure with more than 10 lines of output
	longOutput := make([]string, 15)
	for i := 0; i < 15; i++ {
		longOutput[i] = "Line " + string(rune('A'+i))
	}

	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "FAIL",
				Elapsed:      5 * time.Second,
				PassedTests:  0,
				FailedTests:  1,
				SkippedTests: 0,
			},
		},
		TotalTests:   1,
		PassedTests:  0,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Failures: []*TestResult{
			{
				Package: "github.com/user/project/pkg1",
				Name:    "TestWithLongOutput",
				Status:  "fail",
				Elapsed: 1 * time.Second,
				Output:  longOutput,
			},
		},
	}

	output := formatter.Format(summary)

	// Count how many lines appear in output
	lines := strings.Split(output, "\n")
	lineJCount := 0
	for _, line := range lines {
		if strings.Contains(line, "Line J") {
			lineJCount++
		}
	}

	// Line J is the 10th line (index 9), so it should appear
	if !strings.Contains(output, "Line J") {
		t.Error("Expected 10th line to appear")
	}

	// Line K is the 11th line (index 10), so it should NOT appear
	if strings.Contains(output, "Line K") {
		t.Error("Expected 11th line to be truncated")
	}
}

// TestSummaryFormatterSkipTruncation tests skip output truncation to 3 lines
func TestSummaryFormatterSkipTruncation(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	// Create skip with more than 3 lines of output
	longOutput := []string{
		"Line 1",
		"Line 2",
		"Line 3",
		"Line 4",
		"Line 5",
	}

	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "github.com/user/project/pkg1",
				Status:       "ok",
				Elapsed:      5 * time.Second,
				PassedTests:  0,
				FailedTests:  0,
				SkippedTests: 1,
			},
		},
		TotalTests:   1,
		PassedTests:  0,
		FailedTests:  0,
		SkippedTests: 1,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
		Skipped: []*TestResult{
			{
				Package: "github.com/user/project/pkg1",
				Name:    "TestWithLongSkipReason",
				Status:  "skip",
				Elapsed: 0,
				Output:  longOutput,
			},
		},
	}

	output := formatter.Format(summary)

	// Line 3 should appear
	if !strings.Contains(output, "Line 3") {
		t.Error("Expected 3rd line to appear")
	}

	// Line 4 should NOT appear (truncated)
	if strings.Contains(output, "Line 4") {
		t.Error("Expected 4th line to be truncated")
	}
}

// TestSummaryFormatterPercentages tests percentage calculations
func TestSummaryFormatterPercentages(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	summary := &Summary{
		Packages:     []*PackageResult{},
		TotalTests:   100,
		PassedTests:  97,
		FailedTests:  2,
		SkippedTests: 1,
		TotalTime:    5 * time.Second,
		PackageCount: 1,
	}

	output := formatter.Format(summary)

	// Check percentages are displayed with 1 decimal place
	if !strings.Contains(output, "97.0%") {
		t.Error("Expected 97.0% for passed tests")
	}
	if !strings.Contains(output, "2.0%") {
		t.Error("Expected 2.0% for failed tests")
	}
	if !strings.Contains(output, "1.0%") {
		t.Error("Expected 1.0% for skipped tests")
	}
}

// TestSummaryFormatterSymbols tests symbol usage
func TestSummaryFormatterSymbols(t *testing.T) {
	formatter := NewSummaryFormatter(80)

	summary := &Summary{
		Packages: []*PackageResult{
			{
				Name:         "pkg1",
				Status:       "ok",
				Elapsed:      1 * time.Second,
				PassedTests:  1,
				FailedTests:  0,
				SkippedTests: 0,
			},
			{
				Name:         "pkg2",
				Status:       "FAIL",
				Elapsed:      1 * time.Second,
				PassedTests:  0,
				FailedTests:  1,
				SkippedTests: 0,
			},
		},
		TotalTests:   2,
		PassedTests:  1,
		FailedTests:  1,
		SkippedTests: 0,
		TotalTime:    2 * time.Second,
		PackageCount: 2,
	}

	output := formatter.Format(summary)

	// Verify symbols are used correctly
	if !strings.Contains(output, SymbolPass) {
		t.Error("Expected pass symbol ✓")
	}
	if !strings.Contains(output, SymbolFail) {
		t.Error("Expected fail symbol ✗")
	}
	if !strings.Contains(output, SymbolSkipAlt) {
		t.Error("Expected skip symbol ⊘ in overall results")
	}
}
