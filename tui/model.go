package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EngineEventMsg wraps engine events for bubbletea
type EngineEventMsg engine.Event

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
// The Model implements the Bubbletea Model interface and manages the full state
// of a test run, including package states, test states, and timing information.
// It renders output in a hierarchical format grouped by package and test, with
// up to 6 output lines displayed per test.
//
// Architecture:
// - Packages contains all package states indexed by package name
// - PackageOrder maintains chronological order of package starts for consistent display
// - NonTestOutput stores build errors and compilation output to display at the top
// - Summary counters track overall test results (lightweight for realtime display)
// - SummaryCollector maintains detailed state for final summary display
// - TerminalWidth is used for line truncation and separator rendering
// - Timer ticks update elapsed times for running tests every second
//
// Rendering:
// The View() method delegates to renderHierarchical() which:
// 1. Renders non-test output (build errors) at the top
// 2. Groups tests by package with package headers
// 3. Indents tests under packages (2 spaces)
// 4. Shows up to 6 output lines per test (4 spaces indent)
// 5. Adds a separator line between content and summary
// 6. Shows overall summary with pass/fail counts
//
// Display features:
// - Elapsed times update every second for running tests
// - Package and test names truncated if they exceed terminal width
// - Right-aligned elapsed times and counts
// - Hierarchical indentation shows test organization
// - Summary shows total test counts with PASSED/FAILED prefix
type Model struct {
	// Package state
	Packages     map[string]*PackageState // Package name -> PackageState
	PackageOrder []string                 // Chronological order of package starts

	// Non-test output (build errors, compilation errors, etc.)
	NonTestOutput []string // Lines that are not part of a test

	// Summary counters (lightweight for realtime display)
	Passed  int
	Failed  int
	Skipped int
	Running int

	// Summary collector (detailed state for final summary)
	summaryCollector *SummaryCollector

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
}

// NewModel creates a new TUI model
func NewModel(replayMode bool, replayRate float64, summaryCollector *SummaryCollector) *Model {
	s := spinner.New()
	s.Spinner = spinner.Jump

	s.Spinner = spinner.Jump
	// s.Style is left empty so we can apply styles dynamically

	return &Model{
		Packages:         make(map[string]*PackageState),
		PackageOrder:     make([]string, 0),
		NonTestOutput:    make([]string, 0),
		summaryCollector: summaryCollector,
		TerminalWidth:    80,                                                  // Default width, will be updated by Bubbletea
		TerminalHeight:   24,                                                  // Default height, will be updated by Bubbletea
		passStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		failStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		skipStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		neutralStyle:     lipgloss.NewStyle(),
		events:           make([]parser.TestEvent, 0),
		spinner:          s,
		ReplayMode:       replayMode,
		ReplayRate:       replayRate,
		StartTime:        time.Now(),
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
	case EngineEventMsg:
		// Handle engine events
		// Note: EventRawLine is handled in main.go using p.Println()
		evt := engine.Event(msg)
		switch evt.Type {
		case engine.EventTest:
			// Process test event
			m.handleTestEvent(evt.TestEvent)

		case engine.EventComplete:
			// Stream finished
			m.Finished = true
			m.TotalElapsedTime = time.Since(m.StartTime).Seconds()
			// Don't display summary here - it will be displayed after TUI exits
			return m, tea.Quit

		case engine.EventError:
			// Error events are logged in main.go
			_ = evt.Error
		}

	case parser.TestEvent:
		// Keep backward compatibility for direct TestEvent messages
		m.handleTestEvent(msg)

	case tea.WindowSizeMsg:
		// Update terminal width
		// Update terminal width and height
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height

	case EOFMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			// Don't display summary here - it will be displayed after TUI exits
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleTestEvent processes a test event and updates the model state.
//
// This function is called for each event from the test stream and updates
// the model's Package and Test states accordingly. It handles three types of events:
//
// 1. Build-level events (empty Package field): Non-test output like build errors
// 2. Package-level events (empty Test field): Package start/pass/fail/skip
// 3. Test-level events: Individual test runs with output
//
// For test-level events, it extracts summary lines (starting with "===" or "---")
// separately from output lines, storing output in a circular buffer (max 6 lines).
//
// Event handling:
// - "run": Creates new test/package state if needed, increments running counters
// - "output": Stores as summary line if it's a "===" or "---" line, otherwise in output buffer
// - "pass"/"fail"/"skip": Updates status, elapsed time, and test/package counters
// - "build-output", "build-fail", "build-pass": Stored as non-test output
func (m *Model) handleTestEvent(event parser.TestEvent) {
	m.events = append(m.events, event)

	// Handle build-output and other non-package events (events with empty Package field)
	if event.Package == "" {
		switch event.Action {
		case "build-output", "build-fail", "build-pass":
			// These are non-test output lines (build errors, etc.)
			if event.Output != "" {
				m.NonTestOutput = append(m.NonTestOutput, strings.TrimRight(event.Output, "\n"))
			}
		}
		return
	}

	// Get or create package state
	pkgState, exists := m.Packages[event.Package]
	if !exists {
		pkgState = NewPackageState(event.Package)
		m.Packages[event.Package] = pkgState
		m.PackageOrder = append(m.PackageOrder, event.Package)
	}

	// Handle package-level events (no Test field)
	if event.Test == "" {
		switch event.Action {
		case "output":
			// Store last output line for package
			if event.Output != "" {
				pkgState.LastOutputLine = strings.TrimRight(event.Output, "\n")
			}

		case "pass":
			pkgState.Status = "passed"
			pkgState.ElapsedTime = event.Elapsed

		case "fail":
			pkgState.Status = "failed"
			pkgState.ElapsedTime = event.Elapsed

		case "skip":
			pkgState.Status = "skipped"
			pkgState.ElapsedTime = event.Elapsed
		}
		return
	}

	// Handle test-level events
	testState, testExists := pkgState.Tests[event.Test]
	if !testExists {
		testState = NewTestState(event.Test, event.Package)
		pkgState.Tests[event.Test] = testState
		pkgState.TestOrder = append(pkgState.TestOrder, event.Test)
		pkgState.Running++
		m.Running++
	}

	switch event.Action {
	case "run":
		// Test started (already handled above)
		testState.Status = "running"

	case "pause":
		testState.Status = "paused"

	case "resume":
		testState.Status = "running"

	case "output":
		if event.Output != "" {
			output := strings.TrimRight(event.Output, "\n")

			// Extract summary line (lines starting with "===" or "---")
			if strings.HasPrefix(output, "===") || strings.HasPrefix(output, "---") {
				testState.SummaryLine = output
			} else {
				// Regular output line
				testState.AddOutputLine(output)
			}
		}

	case "pass":
		testState.Status = "passed"
		testState.ElapsedTime = event.Elapsed
		pkgState.Running--
		pkgState.Passed++
		m.Running--
		m.Passed++

	case "fail":
		testState.Status = "failed"
		testState.ElapsedTime = event.Elapsed
		pkgState.Running--
		pkgState.Failed++
		m.Running--
		m.Failed++
		// m.spinner.Style = m.failStyle // Don't mutate global style

	case "skip":
		testState.Status = "skipped"
		testState.ElapsedTime = event.Elapsed
		pkgState.Running--
		pkgState.Skipped++
		m.Running--
		m.Skipped++
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
	if m.summaryCollector == nil {
		return
	}

	// Compute summary with 10 second slow test threshold
	summary := ComputeSummary(m.summaryCollector, 10*time.Second)

	// Format summary using terminal width
	formatter := NewSummaryFormatter(m.TerminalWidth)
	summaryText := formatter.Format(summary)

	// Print summary to stdout
	// Note: In TUI mode, this will appear after the TUI exits
	fmt.Println()
	fmt.Println(summaryText)
}
