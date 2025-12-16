package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/results"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EngineEventMsg wraps engine events for bubbletea
type EngineEventMsg engine.Event

// EngineEventBatchMsg wraps a batch of engine events
type EngineEventBatchMsg []engine.Event

// EOFMsg signals that stdin has been closed (kept for backward compatibility)
type EOFMsg struct{}

// TickMsg is used for timer updates to refresh elapsed times
type TickMsg struct{}

const MaxOutputLines = 6

// Model represents the TUI state for the enhanced hierarchical test output display.
//
// The Model implements the Bubbletea Model interface. It consumes engine.Event
// and pushes them to the collector, then reads state from the collector for rendering.
type Model struct {
	// Collector reference (read-only from TUI perspective)
	collector *results.Collector

	// Terminal state
	TerminalWidth  int
	TerminalHeight int

	// Styles
	passStyle    lipgloss.Style
	failStyle    lipgloss.Style
	skipStyle    lipgloss.Style
	neutralStyle lipgloss.Style
	boldStyle    lipgloss.Style

	// Replay state
	ReplayRate float64

	spinner       spinner.Model // Bubbles spinner component
	frozenSpinner spinner.Model // Bubbles frozen spinner component
}

// NewModel creates a new TUI model
func NewModel(replayMode bool, replayRate float64, collector *results.Collector) *Model {
	s := spinner.New()
	s.Spinner = spinner.Jump

	sf := spinner.New()
	sf.Spinner = spinner.Jump

	return &Model{
		collector:      collector,
		TerminalWidth:  80,                                                  // Default width, will be updated by Bubbletea
		TerminalHeight: 24,                                                  // Default height, will be updated by Bubbletea
		passStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		failStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		skipStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		neutralStyle:   lipgloss.NewStyle(),
		boldStyle:      lipgloss.NewStyle().Bold(true),
		spinner:        s,
		frozenSpinner:  sf,
		ReplayRate:     replayRate,
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
		cmd := m.handleEvent(engine.Event(msg))
		return m, cmd

	case EngineEventBatchMsg:
		var cmds []tea.Cmd
		for _, evt := range msg {
			if cmd := m.handleEvent(evt); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Sequence(cmds...)

	case tea.WindowSizeMsg:
		// Update terminal width and height
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height

	case EOFMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.collector.State().CurrentRun != nil {
				// Mark as interrupted
				m.collector.Finish()
				// Print final report
				// Since Finish() was just called, the run is now in the list of runs
				// and is no longer CurrentRun (CurrentRun is nil)
				state := m.collector.State()
				if len(state.Runs) > 0 {
					finishedRun := state.Runs[len(state.Runs)-1]
					return m, tea.Sequence(m.printFinalReport(finishedRun), tea.Quit)
				}
			}
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleEvent processes a single engine event and returns a command if output is needed
func (m *Model) handleEvent(evt engine.Event) tea.Cmd {
	if evt.Type == engine.EventRawLine {
		// Print raw line directly
		return tea.Println(string(evt.RawLine))
	}

	// 1. Snapshot state before update
	wasRunning := m.collector.State().CurrentRun != nil

	// 2. Update collector
	m.collector.Push(evt)

	// 3. Check state after update
	state := m.collector.State()
	currentRun := state.CurrentRun
	isRunning := currentRun != nil

	// Case 1: Run just finished
	if wasRunning && !isRunning {
		// Run finished, print final view and summary
		// We need the run that just finished. It should be the last one in the list.
		if len(state.Runs) > 0 {
			finishedRun := state.Runs[len(state.Runs)-1]
			return m.printFinalReport(finishedRun)
		}
	}

	// Case 3: Run just started or is ongoing
	// No immediate output command needed, View() handles rendering
	return nil
}

// printFinalReport handles the printing of the final run state and summary
func (m *Model) printFinalReport(run *results.Run) tea.Cmd {
	// Render the final state of the run
	finalView := expandTabs(m.renderRun(run), 8)

	// Generate summary
	summary := format.ComputeSummary(run, 10*time.Second)
	summaryText := ""
	if summary != nil {
		formatter := format.NewSummaryFormatter(m.TerminalWidth)
		summaryText = "\n" + formatter.Format(summary)
	}

	return tea.Println(finalView + summaryText)
}

// View renders the TUI
func (m *Model) View() string {
	state := m.collector.State()
	if state.CurrentRun == nil {
		return ""
	}
	// Pass the specific run to render
	return strings.TrimRight(expandTabs(m.renderRun(state.CurrentRun), 8), "\n")
}

// expandTabs replaces tab characters in a string with spaces.
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

func (m *Model) packageElapsed(pkg *results.PackageResult) time.Duration {
	if pkg.Status == results.StatusRunning {
		return m.scaledElapsedDuration(time.Since(pkg.WallStartTime))
	}
	return pkg.Elapsed
}

func (m *Model) testElapsed(test *results.TestResult) time.Duration {
	if test.Running() {
		return m.scaledElapsedDuration(time.Since(test.WallStartTime))
	}
	return test.Elapsed
}

func (m *Model) runElapsed(run *results.Run) time.Duration {
	if run.Status == results.StatusRunning {
		return m.scaledElapsedDuration(time.Since(run.WallStartTime))
	}
	return run.LastEventTime.Sub(run.FirstEventTime)
}

func (m *Model) scaledElapsedDuration(duration time.Duration) time.Duration {
	replayRate := m.ReplayRate
	if replayRate <= 0 {
		replayRate = 1.0
	}

	return time.Duration(float64(duration) / replayRate)
}

// formatElapsedTime formats elapsed time according to spec
func formatElapsedTime(d time.Duration) string {
	if d < 50*time.Millisecond {
		return "0.0s"
	}
	if d >= time.Minute {
		minutes := d.Seconds() / 60
		s := fmt.Sprintf("%.1fm", minutes)
		return s
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// truncateLine truncates a line to fit within width
func truncateLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(line) <= width {
		return line
	}
	return line[:width]
}

// renderRun renders the TUI for a specific run
func (m *Model) renderRun(run *results.Run) string {
	var b strings.Builder

	// Render non-test output first (build errors, etc.)

	for _, line := range run.NonTestOutput {
		// b.WriteString("  ") // Add padding
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(run.NonTestOutput) > 0 {
		b.WriteString("\n")
	}

	// Calculate max widths for each column
	var maxPassed, maxFailed, maxSkipped, maxElapsed int
	for _, pkg := range run.Packages {
		if passedLen := len(fmt.Sprintf("%d", pkg.Counts.Passed)); passedLen > maxPassed {
			maxPassed = passedLen
		}
		if failedLen := len(fmt.Sprintf("%d", pkg.Counts.Failed)); failedLen > maxFailed {
			maxFailed = failedLen
		}
		if skippedLen := len(fmt.Sprintf("%d", pkg.Counts.Skipped)); skippedLen > maxSkipped {
			maxSkipped = skippedLen
		}

		if elapsedLen := len(formatElapsedTime(m.packageElapsed(pkg))); elapsedLen > maxElapsed {
			maxElapsed = elapsedLen
		}
	}

	fixedLines := len(run.NonTestOutput)
	if len(run.NonTestOutput) > 0 {
		fixedLines++ // Newline
	}
	fixedLines += 1 // Summary line
	if len(run.PackageOrder) > 0 {
		fixedLines += 1 // Separator line
	}
	fixedLines += len(run.PackageOrder) // One header per package

	availableLines := m.TerminalHeight - fixedLines
	if availableLines < 0 {
		availableLines = 0
	}

	type renderItem struct {
		pkgName   string
		testName  string
		lineCount int
		priority  int
		startTime time.Time
	}

	var items []renderItem

	// Collect all potential test lines from running packages
	for _, pkgName := range run.PackageOrder {
		pkg := run.Packages[pkgName]
		if pkg.Status == results.StatusRunning || pkg.Status == results.StatusInterrupted {
			for _, testName := range pkg.TestOrder {
				testKey := pkgName + "/" + testName
				test := run.TestResults[testKey]

				// line for summary
				lineCount := 1

				// Only show output for running tests
				if test.Running() {
					// Update output lines (take last N lines)
					n := len(test.Output)
					if n < MaxOutputLines {
						lineCount += n
					} else {
						lineCount += MaxOutputLines
					}
				}

				// Priority:
				// 1. Running (Highest)
				// 2. Failed
				// 3. Passed/Skipped (Lowest)
				priority := 3
				if test.Running() {
					priority = 1
				} else if test.Status == results.StatusFailed {
					priority = 2
				}

				items = append(items, renderItem{
					pkgName:   pkgName,
					testName:  testName,
					lineCount: lineCount,
					priority:  priority,
					startTime: test.StartTime,
				})
			}
		}
	}

	// Allocate lines based on priority
	linesToShow := make(map[string]map[string]int)
	for _, pkgName := range run.PackageOrder {
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
	sortFunc := func(a, b renderItem) int {
		if a.startTime.After(b.startTime) {
			return -1
		}
		if a.startTime.Before(b.startTime) {
			return 1
		}
		return 0
	}
	slices.SortFunc(p1, sortFunc)
	slices.SortFunc(p2, sortFunc)
	slices.SortFunc(p3, sortFunc)

	allocate := func(group []renderItem) {
		for _, item := range group {
			if availableLines >= item.lineCount {
				linesToShow[item.pkgName][item.testName] = item.lineCount
				availableLines -= item.lineCount
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
	for _, pkgName := range run.PackageOrder {
		pkgState := run.Packages[pkgName]
		m.renderPackage(&b, run, pkgState, maxPassed, maxFailed, maxSkipped, maxElapsed, linesToShow[pkgName])
	}

	// Add separator line
	if len(run.PackageOrder) > 0 {
		b.WriteString(strings.Repeat("-", m.TerminalWidth))
		b.WriteString("\n")
	}

	// Summary line
	m.renderSummaryLine(&b, run)

	return b.String()
}

// renderPackage renders a single package and its tests
func (m *Model) renderPackage(b *strings.Builder, run *results.Run, pkg *results.PackageResult, wPassed, wFailed, wSkipped, wElapsed int, testLines map[string]int) {
	// Render package header
	m.renderPackageHeader(b, pkg, wPassed, wFailed, wSkipped, wElapsed)

	// Render tests if allocated
	if pkg.Status == results.StatusRunning || pkg.Status == results.StatusInterrupted {
		for _, testName := range pkg.TestOrder {
			count, ok := testLines[testName]
			if ok && count > 0 {
				testKey := pkg.Name + "/" + testName
				testState := run.TestResults[testKey]
				m.renderTest(b, testState, count)
			}
		}
	}
}

// renderPackageHeader renders the package summary line
func (m *Model) renderPackageHeader(b *strings.Builder, pkg *results.PackageResult, wPassed, wFailed, wSkipped, wElapsed int) {
	var leftPart string
	var rightPart string

	// Passed column
	passedStr := fmt.Sprintf("✓ %*d", wPassed, pkg.Counts.Passed)
	if pkg.Counts.Passed > 0 {
		passedStr = m.passStyle.Render(passedStr)
	} else {
		passedStr = m.neutralStyle.Render(passedStr)
	}

	// Failed column
	failedStr := fmt.Sprintf("✗ %*d", wFailed, pkg.Counts.Failed)
	if pkg.Counts.Failed > 0 {
		failedStr = m.failStyle.Render(failedStr)
	} else {
		failedStr = m.neutralStyle.Render(failedStr)
	}

	// Skipped column
	skippedStr := fmt.Sprintf("∅ %*d", wSkipped, pkg.Counts.Skipped)
	if pkg.Counts.Skipped > 0 {
		skippedStr = m.skipStyle.Render(skippedStr)
	} else {
		skippedStr = m.neutralStyle.Render(skippedStr)
	}

	// Elapsed column
	var elapsedVal string
	currentElapsed := m.packageElapsed(pkg)
	elapsedVal = formatElapsedTime(currentElapsed)
	elapsedStr := fmt.Sprintf("%*s", wElapsed, elapsedVal)

	rightPart = fmt.Sprintf("%s  %s  %s  %s", passedStr, failedStr, skippedStr, elapsedStr)
	leftPart = pkg.Name
	if pkg.Status != results.StatusRunning && pkg.Output != "" {
		// Expand tabs to ensure correct width calculation
		leftPart = expandTabs(pkg.Output, 8)
	}

	if pkg.Status == results.StatusRunning {
		// Bold the entire line for running packages
		leftPart = m.boldStyle.Render(leftPart)
		rightPart = m.boldStyle.Render(rightPart)
	}
	prefix := m.getStatusPrefix(pkg.Status, pkg.Counts.Failed > 0)

	m.renderAlignedLine(b, leftPart, rightPart, prefix)
}

// renderTest renders a test and its output lines
func (m *Model) renderTest(b *strings.Builder, test *results.TestResult, maxLines int) {
	// Render test summary line
	summary := m.formatTestSummary(test)

	var elapsedVal string
	currentElapsed := m.testElapsed(test)
	elapsedVal = formatElapsedTime(currentElapsed)

	prefix := "  "

	if test.Running() {
		// Bold the name and elapsed time for running tests
		summary = m.boldStyle.Render(summary)
		elapsedVal = m.boldStyle.Render(elapsedVal)
		prefix = m.getStatusPrefix(test.Status, false)
	}

	m.renderAlignedLine(b, summary, elapsedVal, prefix)
	maxLines--

	// Render output lines
	l := len(test.Output)
	if l > MaxOutputLines {
		l = MaxOutputLines
	}
	for _, outputLine := range test.Output[len(test.Output)-l:] {
		if maxLines <= 0 {
			break
		}
		line := "      " + outputLine // Increased indent (2 padding + 4 indent)
		b.WriteString(ensureReset(truncateLine(line, m.TerminalWidth)))
		b.WriteString("\n")

		maxLines--
	}
}

// formatTestSummary formats the test summary line (left part)
func (m *Model) formatTestSummary(test *results.TestResult) string {
	if test.SummaryLine != "" {
		return fmt.Sprintf("  %s", test.SummaryLine)
	}
	return fmt.Sprintf("  === RUN   %s", test.Name)
}

// getStatusPrefix returns the icon string with appropriate color/style for the status
func (m *Model) getStatusPrefix(status results.Status, hasFailures bool) string {

	switch status {
	case results.StatusRunning, results.StatusInterrupted:
		spinnerView := m.spinner.View()
		// For interrupted, we just show the last spinner frame (frozen)
		// logic is same as running for now from visual perspective in loop
		if hasFailures {
			return m.failStyle.Render(spinnerView) + " "
		}
		return m.passStyle.Render(spinnerView) + " "
	case results.StatusPassed:
		return m.passStyle.Render("✓") + " "
	case results.StatusFailed:
		return m.failStyle.Render("✗") + " "
	case results.StatusSkipped:
		return m.skipStyle.Render("∅") + " "
	case results.StatusPaused:
		// For interrupted, we just show the last spinner frame (frozen)
		// logic is same as running for now from visual perspective in loop
		if hasFailures {
			return m.failStyle.Render(m.frozenSpinner.View()) + " "
		}
		return m.passStyle.Render(m.frozenSpinner.View()) + " "
	default:
		return "  "
	}
}

// renderAlignedLine renders a line with left-aligned and right-aligned content
func (m *Model) renderAlignedLine(b *strings.Builder, left, right, prefix string) {
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
func (m *Model) renderSummaryLine(b *strings.Builder, run *results.Run) {
	total := run.Counts.Passed + run.Counts.Failed + run.Counts.Skipped + run.Counts.Running

	elapsedVal := formatElapsedTime(m.runElapsed(run))

	var statusPrefix string
	switch run.Status {
	case results.StatusRunning:
		statusPrefix = "RUNNING"
	case results.StatusFailed:
		statusPrefix = "FAILED"
	case results.StatusPassed:
		statusPrefix = "PASSED"
	case results.StatusInterrupted:
		statusPrefix = "INTERRUPTED"
	default:
		statusPrefix = "UNKNOWN"
	}

	leftPart := fmt.Sprintf("%s: %d passed, %d failed, %d skipped, %d running, %d total",
		statusPrefix, run.Counts.Passed, run.Counts.Failed, run.Counts.Skipped, run.Counts.Running, total)

	prefix := m.getStatusPrefix(run.Status, run.Counts.Failed > 0)
	if run.Status == results.StatusRunning {
		// Bold the summary line for running status
		leftPart = m.boldStyle.Render(leftPart)
		elapsedVal = m.boldStyle.Render(elapsedVal)
	}

	m.renderAlignedLine(b, leftPart, elapsedVal, prefix)
}

// DisplaySummary retrieves the summary from the collector and displays it.
func (m *Model) DisplaySummary() {
	if m.collector == nil {
		return
	}

	// Compute summary directly from state (single-threaded access assumed after runtime)
	var summary *format.Summary

	state := m.collector.State()
	if len(state.Runs) > 0 {
		run := state.Runs[len(state.Runs)-1]
		summary = format.ComputeSummary(run, 10*time.Second)
	}

	if summary == nil {
		return
	}

	formatter := format.NewSummaryFormatter(m.TerminalWidth)
	summaryText := formatter.Format(summary)

	fmt.Println()
	fmt.Println(summaryText)
}
