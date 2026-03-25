package format

import (
	"strings"
	"time"

	"github.com/ansel1/tang/results"
)

// formatDuration formats a duration using Go's native String() with up to 3
// fractional digits on the smallest unit (truncated, not rounded).
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	switch {
	case d >= time.Second:
		return d.Truncate(time.Millisecond).String()
	case d >= time.Millisecond:
		return d.Truncate(time.Microsecond).String()
	default:
		return d.Truncate(time.Nanosecond).String()
	}
}

// Symbol constants for test results
const (
	SymbolPass = "✓"
	SymbolFail = "✗"
	SymbolSkip = "∅"
)

// Indentation constants
const (
	IndentLevel1 = "  " // 2 spaces
	IndentLevel2 = "  " // 2 spaces
)

func testIndent(testName string) string {
	depth := strings.Count(testName, "/")
	return strings.Repeat(IndentLevel2, depth+1)
}

// expandTabs replaces tab characters with spaces.
func expandTabs(s string, tabWidth int) string {
	var b strings.Builder
	col := 0
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteRune(r)
			col = 0
		case '\t':
			spaces := tabWidth - (col % tabWidth)
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		default:
			b.WriteRune(r)
			col++
		}
	}
	return b.String()
}

// ensureReset appends a terminal reset sequence if the string doesn't end with one.
func ensureReset(s string) string {
	reset := "\x1b[0m"
	if strings.HasSuffix(s, reset) {
		return s
	}
	return s + reset
}

// Summary represents computed summary statistics from a test run.
type Summary struct {
	Packages         []*results.PackageResult
	TotalTests       int
	PassedTests      int
	FailedTests      int
	SkippedTests     int
	TotalTime        time.Duration
	PackageCount     int
	Failures         []*results.TestResult
	Skipped          []*results.TestResult
	SlowTests        []*results.TestResult
	BuildFailures    []*results.PackageResult // Packages that failed to build
	Run              *results.Run             // Reference to the run for accessing build errors
	FastestPackage   *results.PackageResult
	SlowestPackage   *results.PackageResult
	MostTestsPackage *results.PackageResult
}

// SummaryOptions controls which optional detail sections appear in the
// formatted summary output. Failures and build failures are always shown.
type SummaryOptions struct {
	IncludeSkipped bool // Show individual skipped test details
	IncludeSlow    bool // Show individual slow test details
}

// HasTestDetails reports whether the summary contains test-level detail
// messages (failures, skipped tests, slow tests, or build failures) that
// will be rendered above the package summary table.
func (s *Summary) HasTestDetails() bool {
	return s.HasTestDetailsWithOptions(SummaryOptions{IncludeSkipped: true, IncludeSlow: true})
}

// HasTestDetailsWithOptions is like HasTestDetails but respects the given options
// for which optional sections to consider.
func (s *Summary) HasTestDetailsWithOptions(opts SummaryOptions) bool {
	if len(s.Failures) > 0 || len(s.BuildFailures) > 0 {
		return true
	}
	if opts.IncludeSkipped && len(s.Skipped) > 0 {
		return true
	}
	if opts.IncludeSlow && len(s.SlowTests) > 0 {
		return true
	}
	return false
}

// ComputeSummary calculates summary statistics from a Run.
//
// This function processes the run data and computes all necessary
// statistics for display.
//
// Parameters:
//   - run: The Run to summarize
//   - slowThreshold: Duration threshold for slow test detection (e.g., 10s)
//
// Returns:
//   - Summary with all computed statistics
func ComputeSummary(run *results.Run, slowThreshold time.Duration) *Summary {
	summary := &Summary{
		PackageCount: len(run.PackageOrder),
		TotalTime:    run.LastEventTime.Sub(run.FirstEventTime),
		Run:          run,
	}

	// Build packages slice in chronological order
	packages := make([]*results.PackageResult, 0, len(run.PackageOrder))
	for _, pkgName := range run.PackageOrder {
		if pkg, exists := run.Packages[pkgName]; exists {
			packages = append(packages, pkg)
		}
	}
	summary.Packages = packages

	// Calculate overall test statistics from per-package counts.
	// This correctly counts all executions (e.g., when -count=N causes
	// tests to run multiple times), unlike run.TestResults which only
	// holds one entry per unique test name.
	for _, pkg := range packages {
		summary.PassedTests += pkg.Counts.Passed
		summary.FailedTests += pkg.Counts.Failed
		summary.SkippedTests += pkg.Counts.Skipped
	}
	summary.TotalTests = summary.PassedTests + summary.FailedTests + summary.SkippedTests

	// Collect failure details, skipped tests, and slow tests from the
	// unique test results map (detail display only needs one entry per test).
	for _, testResult := range run.TestResults {
		switch testResult.Status {
		case results.StatusFailed:
			summary.Failures = append(summary.Failures, testResult)
		case results.StatusSkipped:
			summary.Skipped = append(summary.Skipped, testResult)
		}
		if testResult.Elapsed >= slowThreshold {
			summary.SlowTests = append(summary.SlowTests, testResult)
		}
	}

	// Sort slow tests by elapsed time (descending)
	if len(summary.SlowTests) > 0 {
		sortSlowTests(summary.SlowTests)
	}

	// Collect packages with build failures
	for _, pkg := range packages {
		if pkg.FailedBuild != "" {
			summary.BuildFailures = append(summary.BuildFailures, pkg)
		}
	}

	// Calculate package statistics
	if len(packages) > 0 {
		summary.FastestPackage = packages[0]
		summary.SlowestPackage = packages[0]
		summary.MostTestsPackage = packages[0]

		for _, pkg := range packages {
			// Find fastest package
			if pkg.Elapsed < summary.FastestPackage.Elapsed {
				summary.FastestPackage = pkg
			}

			// Find slowest package
			if pkg.Elapsed > summary.SlowestPackage.Elapsed {
				summary.SlowestPackage = pkg
			}

			// Find package with most tests
			pkgTestCount := pkg.Counts.Passed + pkg.Counts.Failed + pkg.Counts.Skipped
			mostTestCount := summary.MostTestsPackage.Counts.Passed + summary.MostTestsPackage.Counts.Failed + summary.MostTestsPackage.Counts.Skipped
			if pkgTestCount > mostTestCount {
				summary.MostTestsPackage = pkg
			}
		}
	}

	return summary
}

// sortSlowTests sorts test results by elapsed time in descending order.
func sortSlowTests(tests []*results.TestResult) {
	n := len(tests)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if tests[j].Elapsed < tests[j+1].Elapsed {
				tests[j], tests[j+1] = tests[j+1], tests[j]
			}
		}
	}
}
