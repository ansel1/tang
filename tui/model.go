package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ResultsEventMsg wraps results events for bubbletea
type ResultsEventMsg results.Event

// EOFMsg signals that stdin has been closed (kept for backward compatibility)
type EOFMsg struct{}

// TickMsg is used for timer updates to refresh elapsed times
type TickMsg struct{}

// TestState tracks the state of a single test.
//
// This structure maintains the current state of a test including its status,
// timing information, and output. The output is stored as a circular buffer
// of up to 6 lines per the specification.
//
// Fields:
//   - Name: The test name (e.g., "TestParseBasic")
//   - Package: The package containing the test
//   - Status: One of "running", "paused", "passed", "failed", "skipped"
//   - StartTime: When the test event was received
//   - ElapsedTime: Final elapsed time (only set when test finishes)
//   - SummaryLine: The most recent "===" or "---" line from the test output
//   - OutputLines: Circular buffer storing up to 6 log output lines
//   - MaxOutputLines: Maximum size of output buffer (set to 6 per spec)
type TestState struct {
	Name           string    // Test name (e.g., "TestParseBasic")
	Package        string    // Package name
	Status         string    // "running", "paused", "passed", "failed", "skipped"
	StartTime      time.Time // When the test started
	ElapsedTime    float64   // Final elapsed time (for finished tests)
	SummaryLine    string    // The "===" line or final "---" line
	OutputLines    []string  // Circular buffer of up to 6 output lines
	MaxOutputLines int       // Maximum output lines to keep (6)
}

// NewTestState creates a new test state
func NewTestState(name, pkg string) *TestState {
	return &TestState{
		Name:           name,
		Package:        pkg,
		Status:         "running",
		StartTime:      time.Now(),
		OutputLines:    make([]string, 0, 6),
		MaxOutputLines: 6,
	}
}

// AddOutputLine adds a line to the output buffer, removing oldest if at capacity
func (ts *TestState) AddOutputLine(line string) {
	if len(ts.OutputLines) >= ts.MaxOutputLines {
		// Remove oldest line
		ts.OutputLines = ts.OutputLines[1:]
	}
	ts.OutputLines = append(ts.OutputLines, line)
}

// GetElapsedTime returns the elapsed time for display
func (ts *TestState) GetElapsedTime() float64 {
	if ts.Status == "running" || ts.Status == "paused" {
		return time.Since(ts.StartTime).Seconds()
	}
	return ts.ElapsedTime
}

// PackageState tracks the state of a package and its tests.
//
// This structure maintains the current state of a package including its tests,
// aggregated test counts, and timing information. It serves as a container for
// all tests that belong to a particular package, organized in chronological order.
//
// Fields:
//   - Name: The package import path (e.g., "github.com/user/project/pkg")
//   - Status: One of "running", "passed", "failed", "skipped"
//   - StartTime: When the package testing started
//   - ElapsedTime: Final elapsed time (only set when package finishes)
//   - LastOutputLine: The most recent output line for the package (e.g., coverage info)
//   - Tests: Map from test name to TestState for quick lookup
//   - TestOrder: Chronological list of test names for consistent ordering
//   - Passed, Failed, Skipped, Running: Aggregated counts from contained tests
type PackageState struct {
	Name           string                // Package name
	Status         string                // "running", "passed", "failed", "skipped"
	StartTime      time.Time             // When the package started
	ElapsedTime    float64               // Final elapsed time (for finished packages)
	LastOutputLine string                // Last output line from the package
	Tests          map[string]*TestState // Test name -> TestState
	TestOrder      []string              // Chronological order of test starts
	Passed         int                   // Number of tests passed
	Failed         int                   // Number of tests failed
	Skipped        int                   // Number of tests skipped
	Running        int                   // Number of tests running
}

// NewPackageState creates a new package state
func NewPackageState(name string) *PackageState {
	return &PackageState{
		Name:      name,
		Status:    "running",
		StartTime: time.Now(),
		Tests:     make(map[string]*TestState),
		TestOrder: make([]string, 0),
	}
}

// GetElapsedTime returns the elapsed time for display
func (ps *PackageState) GetElapsedTime() float64 {
	if ps.Status == "running" {
		return time.Since(ps.StartTime).Seconds()
	}
	return ps.ElapsedTime
}

// Model represents the TUI state for the enhanced hierarchical test output display.
//
// The Model implements the Bubbletea Model interface. It consumes results.Event
// from the results.Collector and reads state from the collector for rendering.
//
// Architecture:
// - Collector reference provides access to test run state
// - consumes results.Event to know when to re-render
// - View() reads from collector.GetState() for current data
type Model struct {
	// Collector reference (read-only from TUI perspective)
	collector *results.Collector

	// Package state (View Model)
	Packages     map[string]*PackageState // Package name -> PackageState
	PackageOrder []string                 // Chronological order of package starts

	// Non-test output (build errors, compilation errors, etc.)
	NonTestOutput []string // Lines that are not part of a test

	// Summary counters (lightweight for realtime display)
	Passed  int
	Failed  int
	Skipped int
	Running int

	// Terminal state
	TerminalWidth  int
	TerminalHeight int

	// Styles
	passStyle    lipgloss.Style
	failStyle    lipgloss.Style
	skipStyle    lipgloss.Style
	neutralStyle lipgloss.Style

	// For backward compatibility - keep raw events
	events []parser.TestEvent

	// Replay state
	ReplayMode bool
	ReplayRate float64

	// State tracking
	Finished         bool          // True if the event stream has completed
	StartTime        time.Time     // When the TUI started
	TotalElapsedTime float64       // Final elapsed time (set when finished)
	spinner          spinner.Model // Bubbles spinner component
	currentRunID     int           // Which run we're currently displaying
}

// NewModel creates a new TUI model
func NewModel(replayMode bool, replayRate float64, collector *results.Collector) *Model {
	s := spinner.New()
	s.Spinner = spinner.Jump

	return &Model{
		collector:      collector,
		Packages:       make(map[string]*PackageState),
		PackageOrder:   make([]string, 0),
		NonTestOutput:  make([]string, 0),
		TerminalWidth:  80,                                                  // Default width, will be updated by Bubbletea
		TerminalHeight: 24,                                                  // Default height, will be updated by Bubbletea
		passStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		failStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		skipStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		neutralStyle:   lipgloss.NewStyle(),
		events:         make([]parser.TestEvent, 0),
		spinner:        s,
		ReplayMode:     replayMode,
		ReplayRate:     replayRate,
		StartTime:      time.Now(),
		currentRunID:   1, // Start with run 1
	}
}

// Init initializes the model and returns the initial command
func (m *Model) Init() tea.Cmd {
	// Return a tick command to update elapsed times for running tests
	// and the spinner tick
	return m.spinner.Tick
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ResultsEventMsg:
		m.handleResultsEvent(results.Event(msg))

	case tea.WindowSizeMsg:
		// Update terminal width and height
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height

	case EOFMsg:
		m.Finished = true
		m.TotalElapsedTime = time.Since(m.StartTime).Seconds()
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.Finished = true
			m.TotalElapsedTime = time.Since(m.StartTime).Seconds()
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleResultsEvent processes a results event and updates the model state.
func (m *Model) handleResultsEvent(evt results.Event) {
	switch evt.Type {
	case results.EventNonTestOutput:
		m.NonTestOutput = append(m.NonTestOutput, evt.Output)

	case results.EventRunStarted:
		m.currentRunID = evt.RunID

	case results.EventRunFinished:
		// Could handle run completion here if needed

	case results.EventPackageUpdated:
		m.collector.WithRun(evt.RunID, func(run *results.Run) {
			pkgResult, exists := run.Packages[evt.PackageName]
			if !exists {
				return
			}

			// Update or create local PackageState
			pkgState, exists := m.Packages[evt.PackageName]
			if !exists {
				pkgState = NewPackageState(evt.PackageName)
				m.Packages[evt.PackageName] = pkgState
				m.PackageOrder = append(m.PackageOrder, evt.PackageName)
			}

			// Sync state
			pkgState.Status = mapPackageStatus(pkgResult.Status)
			pkgState.ElapsedTime = pkgResult.Elapsed.Seconds()
			pkgState.LastOutputLine = pkgResult.Output
			pkgState.Passed = pkgResult.PassedTests
			pkgState.Failed = pkgResult.FailedTests
			pkgState.Skipped = pkgResult.SkippedTests

			// Recalculate running count for package
			running := 0
			for _, t := range pkgState.Tests {
				if t.Status == "running" {
					running++
				}
			}
			pkgState.Running = running
		})

	case results.EventTestUpdated:
		m.collector.WithRun(evt.RunID, func(run *results.Run) {
			// Construct test key as used in results package
			testKey := evt.PackageName + "/" + evt.TestName
			testResult, exists := run.TestResults[testKey]
			if !exists {
				return
			}

			// Ensure package state exists
			pkgState, exists := m.Packages[evt.PackageName]
			if !exists {
				pkgState = NewPackageState(evt.PackageName)
				m.Packages[evt.PackageName] = pkgState
				m.PackageOrder = append(m.PackageOrder, evt.PackageName)
			}

			// Update or create local TestState
			testState, exists := pkgState.Tests[evt.TestName]
			if !exists {
				testState = NewTestState(evt.TestName, evt.PackageName)
				pkgState.Tests[evt.TestName] = testState
				pkgState.TestOrder = append(pkgState.TestOrder, evt.TestName)
				m.Running++ // New test starts as running
			}

			oldStatus := testState.Status
			newStatus := mapTestStatus(testResult.Status)
			testState.Status = newStatus
			testState.ElapsedTime = testResult.Elapsed.Seconds()
			testState.SummaryLine = testResult.SummaryLine

			// Update output lines (take last N lines)
			n := len(testResult.Output)
			if n > testState.MaxOutputLines {
				testState.OutputLines = testResult.Output[n-testState.MaxOutputLines:]
			} else {
				testState.OutputLines = testResult.Output
			}

			// Update global counters if status changed
			if oldStatus == "running" && newStatus != "running" {
				m.Running--
				switch newStatus {
				case "passed":
					m.Passed++
				case "failed":
					m.Failed++
				case "skipped":
					m.Skipped++
				}
			}
		})
	}
}

func mapPackageStatus(status string) string {
	switch status {
	case "ok":
		return "passed"
	case "FAIL":
		return "failed"
	case "?":
		return "skipped"
	default:
		return "running"
	}
}

func mapTestStatus(status string) string {
	switch status {
	case "pass":
		return "passed"
	case "fail":
		return "failed"
	case "skip":
		return "skipped"
	case "running":
		return "running"
	default:
		return "running"
	}
}

// View renders the TUI
func (m *Model) View() string {
	return strings.TrimRight(expandTabs(m.renderHierarchical(), 8), "\n")
}

// expandTabs replaces tab characters in a string with spaces.
// This is necessary because tab characters in some display environments
// do not overwrite characters but simply advance the cursor, leaving
// characters from the previous view bleeding through.
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

// String renders the TUI (for backward compatibility)
func (m *Model) String() string {
	return m.View()
}

// HasFailures returns true if any tests failed
func (m *Model) HasFailures() bool {
	return m.Failed > 0
}

// formatElapsedTime formats elapsed time according to spec
// Format: X.Xs for <60s, X.Xm for >=60s
func formatElapsedTime(seconds float64) string {
	if seconds < 0.05 {
		return "0.0s"
	}
	if seconds >= 60 {
		minutes := seconds / 60
		return fmt.Sprintf("%.1fm", minutes)
	}
	return fmt.Sprintf("%.1fs", seconds)
}

// truncateLine truncates a line to fit within width, preserving right-aligned content if needed
func truncateLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(line) <= width {
		return line
	}
	// Simple truncation at width
	// TODO: Could be enhanced to preserve right-aligned content
	return line[:width]
}

// renderHierarchical renders output in hierarchical format with smart line elision
func (m *Model) renderHierarchical() string {
	var b strings.Builder

	// Render non-test output first (build errors, etc.)
	for _, line := range m.NonTestOutput {
		b.WriteString("  ") // Add padding
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(m.NonTestOutput) > 0 {
		b.WriteString("\n")
	}

	// Calculate max widths for each column
	var maxPassed, maxFailed, maxSkipped, maxElapsed int
	for _, pkg := range m.Packages {
		if passedLen := len(fmt.Sprintf("%d", pkg.Passed)); passedLen > maxPassed {
			maxPassed = passedLen
		}
		if failedLen := len(fmt.Sprintf("%d", pkg.Failed)); failedLen > maxFailed {
			maxFailed = failedLen
		}
		if skippedLen := len(fmt.Sprintf("%d", pkg.Skipped)); skippedLen > maxSkipped {
			maxSkipped = skippedLen
		}
		elapsed := formatElapsedTime(pkg.GetElapsedTime())
		if m.ReplayMode && m.ReplayRate != 1.0 && m.ReplayRate != 0 {
			// In replay mode, we want to show the simulated original time
			// For running packages, we scale the wall time by the rate (e.g. if rate is 0.5 (2x speed), 1s wall time = 2s simulated time)
			// For finished packages, we show the original elapsed time
			if pkg.Status == "running" {
				elapsed = formatElapsedTime(pkg.GetElapsedTime() / m.ReplayRate)
			} else {
				elapsed = formatElapsedTime(pkg.ElapsedTime)
			}
		}
		if elapsedLen := len(elapsed); elapsedLen > maxElapsed {
			maxElapsed = elapsedLen
		}
	}

	// Calculate available space for test lines
	// Fixed costs: NonTestOutput + Summary + Separator + Package Headers
	fixedLines := len(m.NonTestOutput)
	if len(m.NonTestOutput) > 0 {
		fixedLines++ // Newline
	}
	fixedLines += 1 // Summary line
	if len(m.PackageOrder) > 0 {
		fixedLines += 1 // Separator line
	}
	fixedLines += len(m.PackageOrder) // One header per package

	availableLines := m.TerminalHeight - fixedLines
	if availableLines < 0 {
		availableLines = 0
	}

	// Prioritize tests to show
	type renderItem struct {
		pkgName   string
		testName  string
		lines     []string
		priority  int
		startTime time.Time
	}

	var items []renderItem

	// Collect all potential test lines from running packages
	for _, pkgName := range m.PackageOrder {
		pkg := m.Packages[pkgName]
		if pkg.Status == "running" {
			for _, testName := range pkg.TestOrder {
				test := pkg.Tests[testName]

				var lines []string
				summary := m.formatTestSummary(test)
				lines = append(lines, summary)

				// Only show output for running tests
				if test.Status == "running" {
					for _, out := range test.OutputLines {
						lines = append(lines, fmt.Sprintf("    %s", out))
					}
				}

				// Priority:
				// 1. Running (Highest)
				// 2. Failed
				// 3. Passed/Skipped (Lowest)
				priority := 3
				if test.Status == "running" {
					priority = 1
				} else if test.Status == "failed" {
					priority = 2
				}

				items = append(items, renderItem{
					pkgName:   pkgName,
					testName:  testName,
					lines:     lines,
					priority:  priority,
					startTime: test.StartTime,
				})
			}
		}
	}

	// Allocate lines based on priority
	linesToShow := make(map[string]map[string]int)
	for _, pkgName := range m.PackageOrder {
		linesToShow[pkgName] = make(map[string]int)
	}

	// Sort items by priority (1 > 2 > 3)
	// We use a simple bucket approach since we have few priorities
	var p1, p2, p3 []renderItem
	for _, item := range items {
		if item.priority == 1 {
			p1 = append(p1, item)
		} else if item.priority == 2 {
			p2 = append(p2, item)
		} else {
			p3 = append(p3, item)
		}
	}

	// Sort buckets by StartTime descending (most recent first)
	sortItems := func(items []renderItem) {
		// Simple bubble sort for small lists, or use sort.Slice if we imported sort
		// Since we can't easily add imports in this block without context, let's use a simple swap
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].startTime.After(items[i].startTime) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
	sortItems(p1)
	sortItems(p2)
	sortItems(p3)

	allocate := func(group []renderItem) {
		for _, item := range group {
			if availableLines >= len(item.lines) {
				linesToShow[item.pkgName][item.testName] = len(item.lines)
				availableLines -= len(item.lines)
			} else if availableLines > 0 {
				linesToShow[item.pkgName][item.testName] = availableLines
				availableLines = 0
			}
		}
	}

	allocate(p1)
	allocate(p2)
	allocate(p3)

	// Render packages
	for _, pkgName := range m.PackageOrder {
		pkgState := m.Packages[pkgName]
		m.renderPackage(&b, pkgState, maxPassed, maxFailed, maxSkipped, maxElapsed, linesToShow[pkgName])
	}

	// Add separator line
	if len(m.PackageOrder) > 0 {
		b.WriteString(strings.Repeat("-", m.TerminalWidth))
		b.WriteString("\n")
	}

	// Add summary line
	m.renderSummaryLine(&b, maxElapsed)

	return b.String()
}

// renderPackage renders a single package and its tests
func (m *Model) renderPackage(b *strings.Builder, pkg *PackageState, wPassed, wFailed, wSkipped, wElapsed int, testLines map[string]int) {
	// Render package header
	m.renderPackageHeader(b, pkg, wPassed, wFailed, wSkipped, wElapsed)

	// Render tests if allocated
	if pkg.Status == "running" {
		for _, testName := range pkg.TestOrder {
			count, ok := testLines[testName]
			if ok && count > 0 {
				testState := pkg.Tests[testName]
				m.renderTest(b, testState, count)
			}
		}
	}
}

// renderPackageHeader renders the package summary line
func (m *Model) renderPackageHeader(b *strings.Builder, pkg *PackageState, wPassed, wFailed, wSkipped, wElapsed int) {
	var leftPart string
	var rightPart string

	// Passed column
	passedStr := fmt.Sprintf("✓ %*d", wPassed, pkg.Passed)
	if pkg.Passed > 0 {
		passedStr = m.passStyle.Render(passedStr)
	} else {
		passedStr = m.neutralStyle.Render(passedStr)
	}

	// Failed column
	failedStr := fmt.Sprintf("✗ %*d", wFailed, pkg.Failed)
	if pkg.Failed > 0 {
		failedStr = m.failStyle.Render(failedStr)
	} else {
		failedStr = m.neutralStyle.Render(failedStr)
	}

	// Skipped column
	skippedStr := fmt.Sprintf("∅ %*d", wSkipped, pkg.Skipped)
	if pkg.Skipped > 0 {
		skippedStr = m.skipStyle.Render(skippedStr)
	} else {
		skippedStr = m.neutralStyle.Render(skippedStr)
	}

	// Elapsed column
	var elapsedVal string
	currentElapsed := pkg.GetElapsedTime()
	if m.ReplayMode && m.ReplayRate != 1.0 && m.ReplayRate != 0 {
		if pkg.Status == "running" {
			// Scale wall time to simulated time
			currentElapsed = currentElapsed / m.ReplayRate
		} else {
			// Show original elapsed time
			currentElapsed = pkg.ElapsedTime
		}
	}
	elapsedVal = formatElapsedTime(currentElapsed)
	elapsedStr := fmt.Sprintf("%*s", wElapsed, elapsedVal)

	rightPart = fmt.Sprintf("%s  %s  %s  %s", passedStr, failedStr, skippedStr, elapsedStr)
	leftPart = pkg.Name
	if pkg.Status != "running" && pkg.LastOutputLine != "" {
		// Expand tabs to ensure correct width calculation
		leftPart = expandTabs(pkg.LastOutputLine, 8)
	}

	prefix := "  "
	if pkg.Status == "running" {
		prefix = m.getSpinnerPrefix(pkg.Failed > 0)
	}

	m.renderAlignedLine(b, leftPart, rightPart, prefix)
}

// renderTest renders a test and its output lines
func (m *Model) renderTest(b *strings.Builder, test *TestState, maxLines int) {
	// Render test summary line
	summary := m.formatTestSummary(test)

	var elapsedVal string
	currentElapsed := test.GetElapsedTime()
	if m.ReplayMode && m.ReplayRate != 1.0 && m.ReplayRate != 0 {
		if test.Status == "running" || test.Status == "paused" {
			// Scale wall time to simulated time
			currentElapsed = currentElapsed / m.ReplayRate
		} else {
			// Show original elapsed time
			currentElapsed = test.ElapsedTime
		}
	}
	elapsedVal = formatElapsedTime(currentElapsed)

	prefix := "  "
	if test.Status == "running" {
		prefix = m.getSpinnerPrefix(false)
	} else if test.Status == "paused" {
		prefix = "= "
	}

	m.renderAlignedLine(b, summary, elapsedVal, prefix)
	maxLines--

	// Render output lines
	for i, outputLine := range test.OutputLines {
		if maxLines <= 0 {
			break
		}
		line := fmt.Sprintf("      %s", outputLine) // Increased indent (2 padding + 4 indent)
		b.WriteString(ensureReset(truncateLine(line, m.TerminalWidth)))
		b.WriteString("\n")

		maxLines--
		_ = i // unused
	}
}

// formatTestSummary formats the test summary line (left part)
func (m *Model) formatTestSummary(test *TestState) string {
	if test.SummaryLine != "" {
		return fmt.Sprintf("  %s", test.SummaryLine)
	}
	return fmt.Sprintf("  === RUN   %s", test.Name)
}

// getSpinnerPrefix returns the spinner string with appropriate color
func (m *Model) getSpinnerPrefix(failed bool) string {
	spinnerView := m.spinner.View()
	if failed {
		return m.failStyle.Render(spinnerView) + " "
	}
	return m.passStyle.Render(spinnerView) + " " // Use passStyle (green) for neutral
}

// renderAlignedLine renders a line with left-aligned and right-aligned content
func (m *Model) renderAlignedLine(b *strings.Builder, left, right, prefix string) {
	// Construct the full line with prefix
	// Note: left usually doesn't have padding yet, except for test summary which had "  " in formatTestSummary.
	// I removed the "  " from formatTestSummary in the previous step (chunk 5).

	fullLeft := prefix + left

	if right == "" {
		b.WriteString(fullLeft)
		b.WriteString("\n")
		return
	}

	rightWidth := lipgloss.Width(right)
	leftWidth := lipgloss.Width(fullLeft)

	availableWidth := m.TerminalWidth - rightWidth - 2
	if availableWidth < 0 {
		availableWidth = 0
	}

	if leftWidth >= availableWidth {
		fullLeft = truncateLine(fullLeft, availableWidth)
		b.WriteString(fullLeft)
		b.WriteString("\033[0m")
		b.WriteString("  ")
		b.WriteString(right)
	} else {
		padding := availableWidth - leftWidth
		b.WriteString(fullLeft)
		b.WriteString("\033[0m")
		b.WriteString(strings.Repeat(" ", padding))
		b.WriteString("  ")
		b.WriteString(right)
	}
	b.WriteString("\n")
}

// renderSummaryLine renders the final summary line
func (m *Model) renderSummaryLine(b *strings.Builder, wElapsed int) {
	total := m.Passed + m.Failed + m.Skipped + m.Running

	// Calculate total elapsed time
	var elapsed float64
	if m.Finished {
		elapsed = m.TotalElapsedTime
	} else {
		elapsed = time.Since(m.StartTime).Seconds()
	}

	// Apply replay rate scaling if needed
	if m.ReplayMode && m.ReplayRate != 1.0 && m.ReplayRate != 0 {
		elapsed = elapsed / m.ReplayRate
	}

	elapsedVal := formatElapsedTime(elapsed)
	elapsedStr := fmt.Sprintf("%*s", wElapsed, elapsedVal)

	var leftPart string
	if !m.Finished {
		leftPart = fmt.Sprintf("RUNNING: %d passed, %d failed, %d skipped, %d running, %d total",
			m.Passed, m.Failed, m.Skipped, m.Running, total)
	} else {
		statusPrefix := "PASSED"
		if m.Failed > 0 {
			statusPrefix = "FAILED"
		}
		leftPart = fmt.Sprintf("%s: %d passed, %d failed, %d skipped, %d running, %d total",
			statusPrefix, m.Passed, m.Failed, m.Skipped, m.Running, total)
	}

	prefix := "  "
	if !m.Finished {
		prefix = m.getSpinnerPrefix(m.Failed > 0)
	}

	m.renderAlignedLine(b, leftPart, elapsedStr, prefix)
}

// DisplaySummary retrieves the summary from the collector and displays it.
//
// This method is called after the TUI exits, either when tests complete
// (EventComplete) or when the user interrupts the program (SIGINT/SIGTERM
// via ctrl+c, q, or esc).
//
// It computes the summary from the collector, formats it, and prints it to stdout.
// The summary includes:
// - Package results with pass/fail/skip counts
// - Overall statistics with percentages
// - Failures (if any) with output
// - Skipped tests (if any) with reasons
// - Slow tests (if any) sorted by duration
// - Package performance statistics
//
// Requirements: 7.1, 7.2, 7.3, 7.4
func (m *Model) DisplaySummary() {
	if m.collector == nil {
		return
	}

	// Compute summary within a callback to ensure thread-safe access
	var summary *format.Summary

	// Try to get the current run first
	m.collector.WithRun(m.currentRunID, func(run *results.Run) {
		summary = format.ComputeSummary(run, 10*time.Second)
	})

	// If no summary (run doesn't exist), try the last run
	if summary == nil {
		m.collector.WithState(func(state *results.State) {
			if len(state.Runs) > 0 {
				summary = format.ComputeSummary(state.Runs[len(state.Runs)-1], 10*time.Second)
			}
		})
	}

	// If still no summary, nothing to display
	if summary == nil {
		return
	}

	// Format summary using terminal width
	formatter := format.NewSummaryFormatter(m.TerminalWidth)
	summaryText := formatter.Format(summary)

	// Print summary to stdout
	// Note: In TUI mode, this will appear after the TUI exits
	fmt.Println()
	fmt.Println(summaryText)
}
