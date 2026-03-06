package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/ansel1/tang/results"
)

// formatDuration formats a duration according to the summary display rules.
//
// This function implements the HH:MM:SS.mmm format consistently.
//
// Parameters:
//   - d: Duration to format
//
// Returns:
//   - Formatted duration string
func formatDuration(d time.Duration) string {
	// Format as HH:MM:SS.mmm
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

// Symbol constants for test results
const (
	SymbolPass = "✓"
	SymbolFail = "✗"
	SymbolSkip = "∅"
)

// Indentation constants
const (
	IndentLevel1 = "  "   // 2 spaces
	IndentLevel2 = "    " // 4 spaces
)

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

// HasTestDetails reports whether the summary contains test-level detail
// messages (failures, skipped tests, slow tests, or build failures) that
// will be rendered above the package summary table.
func (s *Summary) HasTestDetails() bool {
	return len(s.Failures) > 0 || len(s.Skipped) > 0 || len(s.SlowTests) > 0 || len(s.BuildFailures) > 0
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

	// Calculate overall test statistics
	for _, testResult := range run.TestResults {
		summary.TotalTests++
		switch testResult.Status {
		case results.StatusPassed:
			summary.PassedTests++
		case results.StatusFailed:
			summary.FailedTests++
			summary.Failures = append(summary.Failures, testResult)
		case results.StatusSkipped:
			summary.SkippedTests++
			summary.Skipped = append(summary.Skipped, testResult)
		}

		// Detect slow tests
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
