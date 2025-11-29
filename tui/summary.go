package tui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// SummaryCollector accumulates test events and package results during test execution.
//
// This structure collects all test events and package completion information to enable
// comprehensive summary generation when testing completes or is interrupted. It maintains
// maps of packages and tests for efficient lookup and updates.
//
// The collector is designed to run as an independent goroutine consuming events from a
// channel. It uses a mutex for thread-safe access to its internal state, allowing
// GetSummary to be called safely while ProcessEvents is running.
//
// Fields:
//   - packages: Map from package name to PackageResult for quick lookup
//   - testResults: Map from "package/testname" to TestResult for quick lookup
//   - startTime: When collection started (first event received)
//   - endTime: When collection ended (set when GetSummary is called)
//   - packageOrder: Chronological order of package starts
//   - mu: Mutex for thread-safe access to collector state
type SummaryCollector struct {
	packages              map[string]*PackageResult
	testResults           map[string]*TestResult
	packageStartTimes     map[string]time.Time // Track event timestamp when each package started
	packageWallStartTimes map[string]time.Time // Track wall clock time when each package started
	startTime             time.Time
	endTime               time.Time
	packageOrder          []string // Chronological order of package starts
	mu                    sync.RWMutex
}

// PackageResult represents the final result of a package's test run.
//
// This structure captures the complete state of a package after testing,
// including aggregated test counts, timing information, and final status.
//
// Fields:
//   - Name: Package import path (e.g., "github.com/user/project/pkg")
//   - Status: Final status - "ok", "FAIL", or "?" for incomplete
//   - Elapsed: Total elapsed time for the package
//   - PassedTests: Number of tests that passed
//   - FailedTests: Number of tests that failed
//   - SkippedTests: Number of tests that were skipped
//   - Output: Final output line (e.g., coverage information)
type PackageResult struct {
	Name         string
	Status       string // "ok", "FAIL", "?"
	Elapsed      time.Duration
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Output       string
}

// TestResult represents the result of a single test.
//
// This structure captures the complete state of a test after execution,
// including its status, timing, and any output (failure messages or skip reasons).
//
// Fields:
//   - Package: Package containing the test
//   - Name: Test name (e.g., "TestParseBasic")
//   - Status: Final status - "pass", "fail", or "skip"
//   - Elapsed: Elapsed time for the test
//   - Output: Lines of output (failure messages or skip reasons)
type TestResult struct {
	Package string
	Name    string
	Status  string // "pass", "fail", "skip"
	Elapsed time.Duration
	Output  []string // failure/skip messages
}

// NewSummaryCollector creates a new summary collector.
func NewSummaryCollector() *SummaryCollector {
	return &SummaryCollector{
		packages:              make(map[string]*PackageResult),
		testResults:           make(map[string]*TestResult),
		packageStartTimes:     make(map[string]time.Time),
		packageWallStartTimes: make(map[string]time.Time),
		packageOrder:          make([]string, 0),
	}
}

// ProcessEvents consumes events from a channel and updates the collector state.
//
// This method is designed to run as a goroutine, processing events concurrently
// with other consumers (TUI, Simple Output). It handles all event types:
// - EventTest: Processes test events via AddTestEvent
// - EventComplete: Signals end of stream and returns
// - Other events: Ignored (EventRawLine, EventError handled by other consumers)
//
// The method returns when the event channel is closed or EventComplete is received.
// All state updates are protected by the collector's mutex for thread safety.
//
// Parameters:
//   - events: Channel of engine.Event to consume
//
// Example usage:
//
//	collector := NewSummaryCollector()
//	go collector.ProcessEvents(eventChannel)
//	// ... later ...
//	summary := ComputeSummary(collector, 10*time.Second)
func (sc *SummaryCollector) ProcessEvents(events <-chan engine.Event) {
	for event := range events {
		switch event.Type {
		case engine.EventTest:
			// Process test event with write lock
			sc.mu.Lock()
			sc.addTestEventLocked(event.TestEvent)
			sc.mu.Unlock()

		case engine.EventComplete:
			// End of stream - return
			return

			// Ignore other event types (EventRawLine, EventError)
			// These are handled by TUI/Simple Output consumers
		}
	}
}

// AddTestEvent processes a test event and updates the collector state.
//
// This method handles test-level events (run, pass, fail, skip, output) and
// accumulates the necessary information for summary generation. It creates
// TestResult entries as needed and updates them based on event actions.
//
// This method is thread-safe and can be called from multiple goroutines.
//
// Event handling:
// - "run": Creates new TestResult if needed
// - "output": Accumulates output lines for the test
// - "pass"/"fail"/"skip": Sets final status and elapsed time
func (sc *SummaryCollector) AddTestEvent(event parser.TestEvent) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.addTestEventLocked(event)
}

// addTestEventLocked is the internal implementation of AddTestEvent.
// It must be called with the write lock held.
func (sc *SummaryCollector) addTestEventLocked(event parser.TestEvent) {
	// Set start time on first event
	if sc.startTime.IsZero() {
		sc.startTime = event.Time
	}

	// Update end time with each event
	if !event.Time.IsZero() && event.Time.After(sc.endTime) {
		sc.endTime = event.Time
	}

	// Get or create package result for any event with a package name
	// This ensures packages are registered in chronological order
	pkgResult, exists := sc.packages[event.Package]
	if !exists {
		pkgResult = &PackageResult{
			Name: event.Package,
		}
		sc.packages[event.Package] = pkgResult
		sc.packageOrder = append(sc.packageOrder, event.Package)
	}

	// Handle package-level events (no test name)
	if event.Test == "" {
		if event.Action == "output" {
			// Update output to the last line received
			if event.Output != "" {
				output := event.Output
				// Remove trailing newline if present
				if len(output) > 0 && output[len(output)-1] == '\n' {
					output = output[:len(output)-1]
				}
				// Only update if not empty (ignore empty lines)
				if output != "" {
					pkgResult.Output = output
				}
			}
			return
		}

		// Only process package completion events
		if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
			status := "ok"
			if event.Action == "fail" {
				status = "FAIL"
			} else if event.Action == "skip" {
				status = "?"
			}

			elapsed := time.Duration(event.Elapsed * float64(time.Second))

			// Update package status and timing
			pkgResult.Status = status
			pkgResult.Elapsed = elapsed

			// Aggregate test counts from test results
			for _, testResult := range sc.testResults {
				if testResult.Package == event.Package {
					switch testResult.Status {
					case "pass":
						pkgResult.PassedTests++
					case "fail":
						pkgResult.FailedTests++
					case "skip":
						pkgResult.SkippedTests++
					}
				}
			}
		}
		return
	}

	// Track package start time on first test from this package
	if _, exists := sc.packageStartTimes[event.Package]; !exists {
		sc.packageStartTimes[event.Package] = event.Time
		sc.packageWallStartTimes[event.Package] = time.Now()
	}

	// Create unique key for test
	testKey := event.Package + "/" + event.Test

	// Get or create test result
	testResult, exists := sc.testResults[testKey]
	if !exists {
		testResult = &TestResult{
			Package: event.Package,
			Name:    event.Test,
			Status:  "running",
			Output:  make([]string, 0),
		}
		sc.testResults[testKey] = testResult
	}

	// Process event based on action
	switch event.Action {
	case "run":
		// Test started - already handled by creation above
		testResult.Status = "running"

	case "output":
		// Accumulate output lines (trim newlines)
		if event.Output != "" {
			output := event.Output
			// Remove trailing newline if present
			if len(output) > 0 && output[len(output)-1] == '\n' {
				output = output[:len(output)-1]
			}
			testResult.Output = append(testResult.Output, output)
		}

	case "pass":
		testResult.Status = "pass"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))

	case "fail":
		testResult.Status = "fail"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))

	case "skip":
		testResult.Status = "skip"
		testResult.Elapsed = time.Duration(event.Elapsed * float64(time.Second))
	}
}

// AddPackageResult records the completion of a package's test run.
//
// This method is called when a package finishes testing (pass, fail, or skip)
// and records the final package status and timing. It also aggregates test
// counts from the collected test results for this package.
//
// This method is thread-safe and can be called from multiple goroutines.
//
// Parameters:
//   - pkg: Package name
//   - status: Final status ("ok", "FAIL", or "?")
//   - elapsed: Total elapsed time for the package
func (sc *SummaryCollector) AddPackageResult(pkg string, status string, elapsed time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Get or create package result
	pkgResult, exists := sc.packages[pkg]
	if !exists {
		pkgResult = &PackageResult{
			Name: pkg,
		}
		sc.packages[pkg] = pkgResult
		sc.packageOrder = append(sc.packageOrder, pkg)
	}

	// Update package status and timing
	pkgResult.Status = status
	pkgResult.Elapsed = elapsed

	// Aggregate test counts from test results
	for _, testResult := range sc.testResults {
		if testResult.Package == pkg {
			switch testResult.Status {
			case "pass":
				pkgResult.PassedTests++
			case "fail":
				pkgResult.FailedTests++
			case "skip":
				pkgResult.SkippedTests++
			}
		}
	}
}

// GetSummary extracts the collected data and returns it as a Summary.
//
// This method should be called when summary display is needed (all tests complete
// or user interrupts). It sets the end time and returns all collected package
// and test results.
//
// This method is thread-safe and can be called while ProcessEvents is running.
// It uses a read lock to safely access the collector's state without blocking
// event processing.
//
// Returns:
//   - packages: Slice of PackageResult in chronological order
//   - testResults: Slice of all TestResult entries
//   - startTime: When collection started
//   - endTime: When GetSummary was called
func (sc *SummaryCollector) GetSummary() (packages []*PackageResult, testResults []*TestResult, startTime, endTime time.Time) {
	return sc.GetSummaryWithReplay(false, 1.0)
}

// GetSummaryWithReplay extracts the collected data with replay rate adjustment.
//
// For interrupted packages, this applies replay rate adjustment to calculate
// the simulated elapsed time based on wall clock time.
//
// Parameters:
//   - replayMode: Whether replay mode is active
//   - replayRate: Replay rate multiplier (e.g., 0.01 = 100x slower, 2.0 = 0.5x speed)
//
// Returns:
//   - packages: Slice of PackageResult in chronological order
//   - testResults: Slice of all TestResult entries
//   - startTime: When collection started
//   - endTime: When GetSummary was called
func (sc *SummaryCollector) GetSummaryWithReplay(replayMode bool, replayRate float64) (packages []*PackageResult, testResults []*TestResult, startTime, endTime time.Time) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Use end time from events, or current time if no events received
	endTime = sc.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	// Build packages slice in chronological order
	packages = make([]*PackageResult, 0, len(sc.packageOrder))
	for _, pkgName := range sc.packageOrder {
		if pkg, exists := sc.packages[pkgName]; exists {
			packages = append(packages, pkg)
		}
	}

	// Find packages with tests but no completion event (interrupted packages)
	// These are packages that have test results but no PackageResult entry
	packagesWithTests := make(map[string]bool)
	for _, testResult := range sc.testResults {
		packagesWithTests[testResult.Package] = true
	}

	// Add incomplete packages with status "?" and aggregate their test counts
	for pkgName := range packagesWithTests {
		if _, exists := sc.packages[pkgName]; !exists {
			// Calculate elapsed time for interrupted package
			var elapsed time.Duration
			if wallStart, hasStart := sc.packageWallStartTimes[pkgName]; hasStart {
				// Use wall clock time (time.Since) for interrupted packages
				wallElapsed := time.Since(wallStart)

				// Apply replay rate adjustment if in replay mode
				if replayMode && replayRate != 0 && replayRate != 1.0 {
					// Scale wall time to simulated time
					// e.g., if rate=0.01 (100x slower) and wall time is 30s, simulated time is 30s/0.01 = 3000s
					elapsed = time.Duration(float64(wallElapsed) / replayRate)
				} else {
					elapsed = wallElapsed
				}
			}

			// Create a PackageResult for this incomplete package
			pkgResult := &PackageResult{
				Name:    pkgName,
				Status:  "?",
				Elapsed: elapsed,
			}

			// Aggregate test counts from test results
			// Only count completed tests (pass/fail/skip), not running tests
			for _, testResult := range sc.testResults {
				if testResult.Package == pkgName {
					switch testResult.Status {
					case "pass":
						pkgResult.PassedTests++
					case "fail":
						pkgResult.FailedTests++
					case "skip":
						pkgResult.SkippedTests++
					}
				}
			}

			packages = append(packages, pkgResult)
		}
	}

	// Build test results slice
	testResults = make([]*TestResult, 0, len(sc.testResults))
	for _, testResult := range sc.testResults {
		testResults = append(testResults, testResult)
	}

	return packages, testResults, sc.startTime, endTime
}

// Summary represents the computed summary statistics and results.
//
// This structure contains all the aggregated information needed to display
// a comprehensive test summary, including overall statistics, failures,
// skipped tests, slow tests, and package performance metrics.
//
// Fields:
//   - Packages: All package results in chronological order
//   - TotalTests: Total number of tests across all packages
//   - PassedTests: Number of tests that passed
//   - FailedTests: Number of tests that failed
//   - SkippedTests: Number of tests that were skipped
//   - TotalTime: Total elapsed time from start to end
//   - PackageCount: Number of packages tested
//   - Failures: All failed tests, grouped by package
//   - Skipped: All skipped tests, grouped by package
//   - SlowTests: Tests exceeding threshold, sorted by duration (descending)
//   - FastestPackage: Package with shortest elapsed time
//   - SlowestPackage: Package with longest elapsed time
//   - MostTestsPackage: Package with most tests
type Summary struct {
	Packages         []*PackageResult
	TotalTests       int
	PassedTests      int
	FailedTests      int
	SkippedTests     int
	TotalTime        time.Duration
	PackageCount     int
	Failures         []*TestResult
	Skipped          []*TestResult
	SlowTests        []*TestResult
	FastestPackage   *PackageResult
	SlowestPackage   *PackageResult
	MostTestsPackage *PackageResult
}

// ComputeSummary calculates summary statistics from collected test data.
//
// This function processes the raw collected data and computes all necessary
// statistics for display, including:
// - Overall test counts and percentages
// - Failure and skip collections grouped by package
// - Slow test detection and sorting
// - Package performance rankings
//
// Parameters:
//   - collector: The SummaryCollector with accumulated test data
//   - slowThreshold: Duration threshold for slow test detection (e.g., 10s)
//
// Returns:
//   - Summary with all computed statistics
func ComputeSummary(collector *SummaryCollector, slowThreshold time.Duration) *Summary {
	return ComputeSummaryWithReplay(collector, slowThreshold, false, 1.0)
}

// ComputeSummaryWithReplay calculates summary statistics with replay rate adjustment.
//
// This function is like ComputeSummary but applies replay rate adjustments to
// interrupted package elapsed times when in replay mode.
//
// Parameters:
//   - collector: The SummaryCollector with accumulated test data
//   - slowThreshold: Duration threshold for slow test detection (e.g., 10s)
//   - replayMode: Whether replay mode is active
//   - replayRate: Replay rate multiplier (e.g., 0.01 = 100x slower, 2.0 = 0.5x speed)
//
// Returns:
//   - Summary with all computed statistics
func ComputeSummaryWithReplay(collector *SummaryCollector, slowThreshold time.Duration, replayMode bool, replayRate float64) *Summary {
	packages, testResults, startTime, endTime := collector.GetSummaryWithReplay(replayMode, replayRate)

	summary := &Summary{
		Packages:     packages,
		PackageCount: len(packages),
		TotalTime:    endTime.Sub(startTime),
	}

	// Calculate overall test statistics
	for _, testResult := range testResults {
		summary.TotalTests++
		switch testResult.Status {
		case "pass":
			summary.PassedTests++
		case "fail":
			summary.FailedTests++
			// Collect failures
			summary.Failures = append(summary.Failures, testResult)
		case "skip":
			summary.SkippedTests++
			// Collect skipped tests
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
			pkgTestCount := pkg.PassedTests + pkg.FailedTests + pkg.SkippedTests
			mostTestCount := summary.MostTestsPackage.PassedTests + summary.MostTestsPackage.FailedTests + summary.MostTestsPackage.SkippedTests
			if pkgTestCount > mostTestCount {
				summary.MostTestsPackage = pkg
			}
		}
	}

	return summary
}

// sortSlowTests sorts test results by elapsed time in descending order.
//
// This function implements a simple bubble sort for sorting slow tests.
// Since the number of slow tests is typically small, this is efficient enough.
//
// Parameters:
//   - tests: Slice of TestResult pointers to sort in place
func sortSlowTests(tests []*TestResult) {
	n := len(tests)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if tests[j].Elapsed < tests[j+1].Elapsed {
				tests[j], tests[j+1] = tests[j+1], tests[j]
			}
		}
	}
}

// formatDuration formats a duration according to the summary display rules.
//
// This function implements dual format logic:
// - Durations < 60s are formatted as "X.Xs" (e.g., "5.2s", "59.9s")
// - Durations >= 60s are formatted as "HH:MM:SS.mmm" (e.g., "01:23:45.678")
//
// The format switch occurs exactly at the 60 second boundary to ensure
// consistent display across all summary sections (slow tests, package stats, etc).
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
	SymbolPass    = "✓"
	SymbolFail    = "✗"
	SymbolSkip    = "∅"
	SymbolSkipAlt = "⊘" // Used in overall results section
)

// Indentation constants
const (
	IndentLevel1 = "  "   // 2 spaces
	IndentLevel2 = "    " // 4 spaces
)

// SummaryFormatter formats a Summary for display.
//
// This structure handles the rendering of summary data into a human-readable
// format with proper alignment, symbols, and section organization.
//
// Fields:
//   - width: Terminal width for alignment calculations (default 80)
//   - passStyle: Style for passing tests/packages (green)
//   - failStyle: Style for failing tests/packages (red)
//   - skipStyle: Style for skipped tests/packages (yellow)
//   - neutralStyle: Style for neutral text
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
//
// This method orchestrates the rendering of all summary sections in the
// correct order with proper spacing and separators.
//
// Section order:
// 1. Package results
// 2. Overall results
// 3. Failures (if any)
// 4. Skipped tests (if any)
// 5. Slow tests (if any)
// 6. Package statistics
//
// Parameters:
//   - summary: The Summary to format
//
// Returns:
//   - Formatted summary string ready for display
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
//
// This method displays each package's final status line with pass/fail/skip
// counts and elapsed time, aligned in columns for readability.
//
// Format:
//
//	[symbol] package/name  [counts]  elapsed
//
// Parameters:
//   - packages: Slice of PackageResult to format
//
// Returns:
//   - Formatted package section string
func (sf *SummaryFormatter) formatPackageSection(packages []*PackageResult) string {
	if len(packages) == 0 {
		return ""
	}

	var result string
	result += "PACKAGES\n"
	result += sf.horizontalLine() + "\n"

	// Calculate column widths for alignment
	maxOutputLen := 0
	maxPassedLen := 0
	maxFailedLen := 0
	maxSkippedLen := 0
	maxElapsedLen := 0

	for _, pkg := range packages {
		// Use Output if available, otherwise Name
		// Replace tabs with spaces for consistent length calculation and alignment
		output := pkg.Output
		if output == "" {
			output = pkg.Name
		}
		output = expandTabs(output, 8)

		if len(output) > maxOutputLen {
			maxOutputLen = len(output)
		}

		passedStr := fmt.Sprintf("%d", pkg.PassedTests)
		if len(passedStr) > maxPassedLen {
			maxPassedLen = len(passedStr)
		}

		failedStr := fmt.Sprintf("%d", pkg.FailedTests)
		if len(failedStr) > maxFailedLen {
			maxFailedLen = len(failedStr)
		}

		skippedStr := fmt.Sprintf("%d", pkg.SkippedTests)
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
		if pkg.Status == "FAIL" {
			symbol = SymbolFail
			symbolStyle = sf.failStyle
		} else if pkg.Status == "?" {
			symbol = SymbolSkip
			symbolStyle = sf.skipStyle
		}

		// Use Output if available, otherwise Name
		output := pkg.Output
		if output == "" {
			output = pkg.Name
		}
		output = expandTabs(output, 8)

		// Format counts with right-aligned numbers
		// Omit counts if all are 0
		counts := ""
		if pkg.PassedTests > 0 || pkg.FailedTests > 0 || pkg.SkippedTests > 0 {
			// Format each count with color
			passedStr := fmt.Sprintf("%s %*d", SymbolPass, maxPassedLen, pkg.PassedTests)
			if pkg.PassedTests > 0 {
				passedStr = sf.passStyle.Render(passedStr)
			} else {
				passedStr = sf.neutralStyle.Render(passedStr)
			}

			failedStr := fmt.Sprintf("%s %*d", SymbolFail, maxFailedLen, pkg.FailedTests)
			if pkg.FailedTests > 0 {
				failedStr = sf.failStyle.Render(failedStr)
			} else {
				failedStr = sf.neutralStyle.Render(failedStr)
			}

			skippedStr := fmt.Sprintf("%s %*d", SymbolSkip, maxSkippedLen, pkg.SkippedTests)
			if pkg.SkippedTests > 0 {
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
			// If no counts, print symbol, output, and elapsed time
			// We align elapsed time to the right, leaving the counts column empty
			// Calculate visual width of counts column:
			// 3 symbols (length 1 each visually) + 7 spaces + max lengths
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
//
// This method displays aggregate test counts with percentages and total time.
//
// Format:
//
//	OVERALL RESULTS
//	---------------
//	Total tests:    123
//	Passed:         120 (97.6%)
//	Failed:         2 (1.6%)
//	Skipped:        1 (0.8%)
//	Total time:     01:23:45.678
//	Packages:       5
//
// Parameters:
//   - summary: The Summary containing statistics
//
// Returns:
//   - Formatted overall results string
func (sf *SummaryFormatter) formatOverallResults(summary *Summary) string {
	var result string
	result += "OVERALL RESULTS\n"
	result += sf.horizontalLine() + "\n"

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
	skipIcon := SymbolSkipAlt
	if sf.useColors {
		passIcon = sf.passStyle.Render(SymbolPass)
		failIcon = sf.failStyle.Render(SymbolFail)
		skipIcon = sf.skipStyle.Render(SymbolSkipAlt)
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
//
// This method displays all failed tests grouped by package, with up to
// 10 lines of failure output per test.
//
// Format:
//
//	FAILURES
//	--------
//	package/name
//	  TestName
//	    [failure output line 1]
//	    [failure output line 2]
//	    ...
//
// Parameters:
//   - failures: Slice of failed TestResult
//
// Returns:
//   - Formatted failures section string
func (sf *SummaryFormatter) formatFailures(failures []*TestResult) string {
	if len(failures) == 0 {
		return ""
	}

	var result string
	result += "FAILURES\n"
	result += sf.horizontalLine() + "\n"

	// Group failures by package
	packageMap := make(map[string][]*TestResult)
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
//
// This method displays all skipped tests grouped by package, with up to
// 3 lines of skip reason per test.
//
// Format:
//
//	SKIPPED
//	-------
//	package/name
//	  TestName
//	    [skip reason line 1]
//	    [skip reason line 2]
//	    ...
//
// Parameters:
//   - skipped: Slice of skipped TestResult
//
// Returns:
//   - Formatted skipped section string
func (sf *SummaryFormatter) formatSkipped(skipped []*TestResult) string {
	if len(skipped) == 0 {
		return ""
	}

	var result string
	result += "SKIPPED\n"
	result += sf.horizontalLine() + "\n"

	// Group skipped tests by package
	packageMap := make(map[string][]*TestResult)
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
//
// This method displays tests exceeding the slow threshold, sorted by
// elapsed time in descending order.
//
// Format:
//
//	SLOW TESTS (>10s)
//	-----------------
//	TestName  package/name  HH:MM:SS.mmm
//	...
//
// Parameters:
//   - slowTests: Slice of slow TestResult (already sorted)
//
// Returns:
//   - Formatted slow tests section string
func (sf *SummaryFormatter) formatSlowTests(slowTests []*TestResult) string {
	if len(slowTests) == 0 {
		return ""
	}

	var result string
	result += "SLOW TESTS (>10s)\n"
	result += sf.horizontalLine() + "\n"

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
//
// This method creates a line of dashes matching the formatter width.
//
// Returns:
//   - String of dashes for section separation
func (sf *SummaryFormatter) horizontalLine() string {
	line := ""
	for i := 0; i < sf.width; i++ {
		line += "-"
	}
	return line
}
