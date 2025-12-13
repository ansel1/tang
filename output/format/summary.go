package format

import (
	"fmt"
	"strings"
	"time"

	"os"

	"github.com/ansel1/tang/results"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
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
	FastestPackage   *results.PackageResult
	SlowestPackage   *results.PackageResult
	MostTestsPackage *results.PackageResult
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
	// Handle case where EndTime wasn't set (shouldn't happen, but be defensive)
	endTime := run.EndTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	summary := &Summary{
		PackageCount: len(run.PackageOrder),
		TotalTime:    endTime.Sub(run.StartTime),
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

// SummaryFormatter formats a Summary for display.
type SummaryFormatter struct {
	width        int
	useColors    bool
	passStyle    lipgloss.Style
	failStyle    lipgloss.Style
	skipStyle    lipgloss.Style
	neutralStyle lipgloss.Style
}

// NewSummaryFormatter creates a new summary formatter.
//
// Colors are automatically enabled if stdout is a TTY.
//
// Parameters:
//   - width: Terminal width for alignment (use 80 if unknown)
//
// Returns:
//   - Pointer to new SummaryFormatter
func NewSummaryFormatter(width int) *SummaryFormatter {
	if width <= 0 {
		width = 80
	}
	// Detect if stdout is a TTY
	useColors := isatty.IsTerminal(os.Stdout.Fd())

	return &SummaryFormatter{
		width:        width,
		useColors:    useColors,
		passStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		failStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		skipStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		neutralStyle: lipgloss.NewStyle(),
	}
}

// Format renders a complete summary as a formatted string.
func (sf *SummaryFormatter) Format(summary *Summary) string {
	var result string

	// 1. Failures (if any)
	if len(summary.Failures) > 0 {
		result += sf.formatFailures(summary.Failures)
		result += "\n"
	}

	// 2. Skipped tests (if any)
	if len(summary.Skipped) > 0 {
		result += sf.formatSkipped(summary.Skipped)
		result += "\n"
	}

	// 3. Slow tests (if any)
	if len(summary.SlowTests) > 0 {
		result += sf.formatSlowTests(summary.SlowTests)
		result += "\n"
	}

	// 4. Package section
	result += sf.formatPackageSection(summary.Packages)
	result += "\n"

	// 5. Overall results
	result += sf.formatOverallResults(summary)
	result += "\n"

	return result
}

// formatPackageSection formats the package results section.
func (sf *SummaryFormatter) formatPackageSection(packages []*results.PackageResult) string {
	if len(packages) == 0 {
		return ""
	}

	var result string
	result += renderSectionHeader("PACKAGES")
	// result += sf.horizontalLine() + "\n"

	// Calculate column widths for alignment
	maxOutputLen := 0
	maxPassedLen := 0
	maxFailedLen := 0
	maxSkippedLen := 0
	maxElapsedLen := 0

	for _, pkg := range packages {
		var output string
		if pkg.Status == results.StatusInterrupted {
			// Add padding to align with "ok\t" prefix of completed packages
			output = "  \t" + pkg.Name + " [interrupted]"
		} else if pkg.Output != "" {
			output = pkg.Output
		} else {
			output = pkg.Name
		}
		output = expandTabs(output, 8)

		if len(output) > maxOutputLen {
			maxOutputLen = len(output)
		}

		passedStr := fmt.Sprintf("%d", pkg.Counts.Passed)
		if len(passedStr) > maxPassedLen {
			maxPassedLen = len(passedStr)
		}

		failedStr := fmt.Sprintf("%d", pkg.Counts.Failed)
		if len(failedStr) > maxFailedLen {
			maxFailedLen = len(failedStr)
		}

		skippedStr := fmt.Sprintf("%d", pkg.Counts.Skipped)
		if len(skippedStr) > maxSkippedLen {
			maxSkippedLen = len(skippedStr)
		}

		elapsedStr := formatDuration(pkg.Elapsed)
		if len(elapsedStr) > maxElapsedLen {
			maxElapsedLen = len(elapsedStr)
		}
	}

	// Format each package
	for _, pkg := range packages {
		symbol := SymbolPass
		symbolStyle := sf.passStyle
		if pkg.Status == results.StatusFailed {
			symbol = SymbolFail
			symbolStyle = sf.failStyle
		} else if pkg.Status == results.StatusSkipped {
			symbol = SymbolSkip
			symbolStyle = sf.skipStyle
		} else if pkg.Status == results.StatusInterrupted {
			// For interrupted packages:
			// - Use Fail icon if there were failures
			// - Use Skip icon if no tests were run (no pass/fail/skip)
			// - Use Pass icon otherwise (partial success)
			if pkg.Counts.Failed > 0 {
				symbol = SymbolFail
				symbolStyle = sf.failStyle
			} else if pkg.Counts.Passed == 0 && pkg.Counts.Failed == 0 {
				symbol = SymbolSkip
				symbolStyle = sf.skipStyle
			} else {
				symbol = SymbolPass
				symbolStyle = sf.passStyle
			}
		}

		var output string
		if pkg.Status == results.StatusInterrupted {
			// Add padding to align with "ok\t" prefix of completed packages
			output = "  \t" + pkg.Name + " [interrupted]"
		} else if pkg.Output != "" {
			output = pkg.Output
		} else {
			output = pkg.Name
		}
		output = expandTabs(output, 8)

		// Format counts with right-aligned numbers
		counts := ""
		if pkg.Counts.Passed > 0 || pkg.Counts.Failed > 0 || pkg.Counts.Skipped > 0 {
			passedStr := fmt.Sprintf("%s %*d", SymbolPass, maxPassedLen, pkg.Counts.Passed)
			if pkg.Counts.Passed > 0 {
				passedStr = sf.passStyle.Render(passedStr)
			} else {
				passedStr = sf.neutralStyle.Render(passedStr)
			}

			failedStr := fmt.Sprintf("%s %*d", SymbolFail, maxFailedLen, pkg.Counts.Failed)
			if pkg.Counts.Failed > 0 {
				failedStr = sf.failStyle.Render(failedStr)
			} else {
				failedStr = sf.neutralStyle.Render(failedStr)
			}

			skippedStr := fmt.Sprintf("%s %*d", SymbolSkip, maxSkippedLen, pkg.Counts.Skipped)
			if pkg.Counts.Skipped > 0 {
				skippedStr = sf.skipStyle.Render(skippedStr)
			} else {
				skippedStr = sf.neutralStyle.Render(skippedStr)
			}

			counts = fmt.Sprintf("%s  %s  %s", passedStr, failedStr, skippedStr)
		}

		// Format line with alignment
		if counts != "" {
			result += fmt.Sprintf("%s %-*s  %s  %*s\n",
				symbolStyle.Render(symbol),
				maxOutputLen, output,
				counts,
				maxElapsedLen, formatDuration(pkg.Elapsed))
		} else {
			countsWidth := 3 + 7 + maxPassedLen + maxFailedLen + maxSkippedLen
			emptyCounts := fmt.Sprintf("%*s", countsWidth, "")

			result += fmt.Sprintf("%s %-*s  %s  %*s\n",
				symbolStyle.Render(symbol),
				maxOutputLen, output,
				emptyCounts,
				maxElapsedLen, formatDuration(pkg.Elapsed))
		}
	}

	result += sf.horizontalLine()
	return result
}

// formatOverallResults formats the overall statistics section.
func (sf *SummaryFormatter) formatOverallResults(summary *Summary) string {
	var result string
	result += renderSectionHeader("OVERALL RESULTS")
	// result += sf.horizontalLine() + "\n"

	// Calculate percentages
	passPercent := 0.0
	failPercent := 0.0
	skipPercent := 0.0
	if summary.TotalTests > 0 {
		passPercent = float64(summary.PassedTests) / float64(summary.TotalTests) * 100
		failPercent = float64(summary.FailedTests) / float64(summary.TotalTests) * 100
		skipPercent = float64(summary.SkippedTests) / float64(summary.TotalTests) * 100
	}

	// Format icons with colors if TTY
	passIcon := SymbolPass
	failIcon := SymbolFail
	skipIcon := SymbolSkip
	if sf.useColors {
		passIcon = sf.passStyle.Render(SymbolPass)
		failIcon = sf.failStyle.Render(SymbolFail)
		skipIcon = sf.skipStyle.Render(SymbolSkip)
	}

	result += fmt.Sprintf("Total tests:    %d\n", summary.TotalTests)
	result += fmt.Sprintf("Passed:         %d %s (%.1f%%)\n", summary.PassedTests, passIcon, passPercent)
	result += fmt.Sprintf("Failed:         %d %s (%.1f%%)\n", summary.FailedTests, failIcon, failPercent)
	result += fmt.Sprintf("Skipped:        %d %s (%.1f%%)\n", summary.SkippedTests, skipIcon, skipPercent)
	result += fmt.Sprintf("Total time:     %s\n", formatDuration(summary.TotalTime))
	result += fmt.Sprintf("Packages:       %d\n", summary.PackageCount)

	result += sf.horizontalLine()
	return result
}

// formatFailures formats the failures section.
func (sf *SummaryFormatter) formatFailures(failures []*results.TestResult) string {
	if len(failures) == 0 {
		return ""
	}

	var result string
	result += renderSectionHeader("FAILURES")
	// result += sf.horizontalLine() + "\n"

	// Group failures by package
	packageMap := make(map[string][]*results.TestResult)
	packageOrder := make([]string, 0)
	for _, failure := range failures {
		if _, exists := packageMap[failure.Package]; !exists {
			packageOrder = append(packageOrder, failure.Package)
		}
		packageMap[failure.Package] = append(packageMap[failure.Package], failure)
	}

	// Format each package's failures
	for i, pkg := range packageOrder {
		if i > 0 {
			result += "\n"
		}
		result += pkg + "\n"

		for _, failure := range packageMap[pkg] {
			result += IndentLevel1 + failure.Name + "\n"

			// Truncate output to 10 lines
			outputLines := failure.Output
			if len(outputLines) > 10 {
				outputLines = outputLines[:10]
			}

			for _, line := range outputLines {
				result += IndentLevel2 + ensureReset(line) + "\n"
			}
		}
	}

	result += sf.horizontalLine()
	return result
}

// formatSkipped formats the skipped tests section.
func (sf *SummaryFormatter) formatSkipped(skipped []*results.TestResult) string {
	if len(skipped) == 0 {
		return ""
	}

	var result string
	result += renderSectionHeader("SKIPPED")
	// result += sf.horizontalLine() + "\n"

	// Group skipped tests by package
	packageMap := make(map[string][]*results.TestResult)
	packageOrder := make([]string, 0)
	for _, skip := range skipped {
		if _, exists := packageMap[skip.Package]; !exists {
			packageOrder = append(packageOrder, skip.Package)
		}
		packageMap[skip.Package] = append(packageMap[skip.Package], skip)
	}

	// Format each package's skipped tests
	for i, pkg := range packageOrder {
		if i > 0 {
			result += "\n"
		}
		result += pkg + "\n"

		for _, skip := range packageMap[pkg] {
			result += IndentLevel1 + skip.Name + "\n"

			// Truncate output to 3 lines
			outputLines := skip.Output
			if len(outputLines) > 3 {
				outputLines = outputLines[:3]
			}

			for _, line := range outputLines {
				result += IndentLevel2 + ensureReset(line) + "\n"
			}
		}
	}

	result += sf.horizontalLine()
	return result
}

// formatSlowTests formats the slow tests section.
func (sf *SummaryFormatter) formatSlowTests(slowTests []*results.TestResult) string {
	if len(slowTests) == 0 {
		return ""
	}

	var result string
	result += renderSectionHeader("SLOW TESTS (>10s)")
	// result += sf.horizontalLine() + "\n"

	// Calculate column widths for alignment
	maxNameLen := 0
	for _, test := range slowTests {
		if len(test.Name) > maxNameLen {
			maxNameLen = len(test.Name)
		}
	}

	// Format each slow test
	for _, test := range slowTests {
		result += fmt.Sprintf("%-*s  %s\n",
			maxNameLen, test.Name,
			formatDuration(test.Elapsed))
		result += fmt.Sprintf("  %s\n", test.Package)
	}

	result += sf.horizontalLine()
	return result
}

// horizontalLine returns a horizontal separator line.
func (sf *SummaryFormatter) horizontalLine() string {
	line := ""
	for i := 0; i < sf.width; i++ {
		line += "-"
	}
	return line
}

func renderSectionHeader(header string) string {
	return header + "\n" + strings.Repeat("-", len(header)) + "\n"
}
