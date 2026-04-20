package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ansel1/tang/results"
)

// DefaultSlowThreshold is the fallback threshold used when none is specified.
const DefaultSlowThreshold = 10 * time.Second

// RepaintMsg forces a redraw
type RepaintMsg struct{}

// QuitMsg signals the TUI to quit cleanly, rendering an empty final frame
// so the terminal is left clean for summary output.
type QuitMsg struct{}

// TickMsg is used for timer updates to refresh elapsed times
type TickMsg struct{}

const MaxOutputLines = 6

// Model represents the TUI state for the enhanced hierarchical test output display.
//
// The Model implements the Bubbletea Model interface.
// It is a passive view that renders the state from the collector.
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
	slowStyle    lipgloss.Style
	neutralStyle lipgloss.Style

	brightStyle   lipgloss.Style
	brightFail    lipgloss.Style
	brightPass    lipgloss.Style
	brightSkip    lipgloss.Style
	brightSlow    lipgloss.Style
	brightNeutral lipgloss.Style

	SlowThreshold time.Duration

	// Replay state
	ReplayRate float64

	spinner       spinner.Model // Bubbles spinner component ⏺
	frozenSpinner spinner.Model // Bubbles frozen spinner component

	interrupted bool
	quitting    bool

	// OnInterrupt, if set, is invoked when the user presses ctrl+c (or
	// otherwise interrupts the TUI). It runs before tea.Quit is returned so
	// callers can forward the interrupt (e.g. to a child go test process)
	// while bubbletea still has control of the terminal and will restore it
	// cleanly. Must be safe to call from a bubbletea goroutine and may be
	// invoked more than once across a program's lifetime.
	OnInterrupt func()

	NonTestOutput []string
}

// NewModel creates a new TUI model
func NewModel(replayMode bool, replayRate float64, collector *results.Collector) *Model {
	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	sf := spinner.New(spinner.WithSpinner(spinner.MiniDot))

	return &Model{
		collector:      collector,
		TerminalWidth:  80,                                                  // Default width, will be updated by Bubbletea
		TerminalHeight: 24,                                                  // Default height, will be updated by Bubbletea
		passStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		failStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		skipStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		slowStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("4")), // blue
		neutralStyle:   lipgloss.NewStyle(),
		brightStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		brightFail:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		brightPass:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		brightSkip:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")),
		brightSlow:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		brightNeutral:  lipgloss.NewStyle().Bold(true),
		SlowThreshold:  DefaultSlowThreshold,
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
	case RepaintMsg:
		return m, nil

	case tea.WindowSizeMsg:
		// Update terminal width and height
		m.TerminalWidth = msg.Width
		m.TerminalHeight = msg.Height

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.interrupted = true
			m.quitting = true
			if m.OnInterrupt != nil {
				m.OnInterrupt()
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

// View renders the TUI
func (m *Model) View() tea.View {
	return tea.NewView(m.renderView())
}

// renderView produces the rendered string for the TUI
func (m *Model) renderView() string {
	if m.quitting {
		return ""
	}

	m.collector.Lock()
	defer m.collector.Unlock()

	currentRun := m.collector.State().MostRecentRun()
	if currentRun == nil {
		return ""
	}
	// Pass the specific run to render
	return strings.TrimRight(expandTabs(m.renderRun(currentRun), 8), "\n")
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
	return m.renderView()
}

func (m *Model) packageElapsed(pkg *results.PackageResult) time.Duration {
	if pkg.Status == results.StatusRunning {
		return m.scaledElapsedDuration(time.Since(pkg.WallStartTime))
	}
	return pkg.Elapsed
}

func (m *Model) testElapsed(test *results.TestResult) time.Duration {
	switch test.Status {
	case results.StatusRunning:
		return m.scaledElapsedDuration(test.ActiveDuration + time.Since(test.LastResumeTime))
	case results.StatusPaused:
		return m.scaledElapsedDuration(test.ActiveDuration)
	default:
		return test.Elapsed
	}
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

	// Calculate max widths for each column (including run-level counts for the summary line)
	var maxRunning, maxPaused, maxPassed, maxFailed, maxSkipped, maxTotal, maxElapsed int

	// Include run-level counts in width calculation
	runTotal := run.Counts.Passed + run.Counts.Failed + run.Counts.Skipped
	maxRunning = len(fmt.Sprintf("%d", run.Counts.Running))
	maxPaused = len(fmt.Sprintf("%d", run.Counts.Paused))
	maxPassed = len(fmt.Sprintf("%d", run.Counts.Passed))
	maxFailed = len(fmt.Sprintf("%d", run.Counts.Failed))
	maxSkipped = len(fmt.Sprintf("%d", run.Counts.Skipped))
	maxTotal = len(fmt.Sprintf("%d", runTotal))
	maxElapsed = len(formatElapsedTime(m.runElapsed(run)))

	for _, pkg := range run.Packages {
		if runningLen := len(fmt.Sprintf("%d", pkg.Counts.Running)); runningLen > maxRunning {
			maxRunning = runningLen
		}
		if pausedLen := len(fmt.Sprintf("%d", pkg.Counts.Paused)); pausedLen > maxPaused {
			maxPaused = pausedLen
		}
		if passedLen := len(fmt.Sprintf("%d", pkg.Counts.Passed)); passedLen > maxPassed {
			maxPassed = passedLen
		}
		if failedLen := len(fmt.Sprintf("%d", pkg.Counts.Failed)); failedLen > maxFailed {
			maxFailed = failedLen
		}
		if skippedLen := len(fmt.Sprintf("%d", pkg.Counts.Skipped)); skippedLen > maxSkipped {
			maxSkipped = skippedLen
		}
		total := pkg.Counts.Passed + pkg.Counts.Failed + pkg.Counts.Skipped
		if totalLen := len(fmt.Sprintf("%d", total)); totalLen > maxTotal {
			maxTotal = totalLen
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

				// Only show output for actively running tests
				if test.Status == results.StatusRunning {
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
				// 3. Passed/Skipped/Paused (Lowest)
				priority := 3
				if test.Status == results.StatusRunning {
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
		switch item.priority {
		case 1:
			p1 = append(p1, item)
		case 2:
			p2 = append(p2, item)
		default:
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

	// Summary line at top
	m.renderSummaryLine(&b, run, maxRunning, maxPaused, maxPassed, maxFailed, maxSkipped, maxTotal, maxElapsed)

	// Add separator line
	if len(run.PackageOrder) > 0 {
		b.WriteString(strings.Repeat("-", m.TerminalWidth))
		b.WriteString("\n")
	}

	// Render packages
	for _, pkgName := range run.PackageOrder {
		pkgState := run.Packages[pkgName]
		m.renderPackage(&b, run, pkgState, maxRunning, maxPaused, maxPassed, maxFailed, maxSkipped, maxTotal, maxElapsed, linesToShow[pkgName])
	}

	return b.String()
}

// renderPackage renders a single package and its tests
func (m *Model) renderPackage(b *strings.Builder, run *results.Run, pkg *results.PackageResult, wRunning, wPaused, wPassed, wFailed, wSkipped, wTotal, wElapsed int, testLines map[string]int) {
	// Render package header
	m.renderPackageHeader(b, pkg, wRunning, wPaused, wPassed, wFailed, wSkipped, wTotal, wElapsed)

	// Render tests if allocated
	if pkg.Status == results.StatusRunning || pkg.Status == results.StatusInterrupted {
		for _, testName := range pkg.DisplayOrder {
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
func (m *Model) renderPackageHeader(b *strings.Builder, pkg *results.PackageResult, wRunning, wPaused, wPassed, wFailed, wSkipped, wTotal, wElapsed int) {
	var leftPart string
	var rightPart string

	running := pkg.Status == results.StatusRunning || pkg.Status == results.StatusInterrupted

	passColor, failColor, skipColor, neutralColor := m.passStyle, m.failStyle, m.skipStyle, m.neutralStyle
	if running {
		passColor, failColor, skipColor, neutralColor = m.brightPass, m.brightFail, m.brightSkip, m.brightNeutral
	}

	passedStr := fmt.Sprintf("%*s", wPassed+1, fmt.Sprintf("✓%d", pkg.Counts.Passed))
	if pkg.Counts.Passed > 0 {
		passedStr = passColor.Render(passedStr)
	} else {
		passedStr = neutralColor.Render(passedStr)
	}

	failedStr := fmt.Sprintf("%*s", wFailed+1, fmt.Sprintf("✗%d", pkg.Counts.Failed))
	if pkg.Counts.Failed > 0 {
		failedStr = failColor.Render(failedStr)
	} else {
		failedStr = neutralColor.Render(failedStr)
	}

	skippedStr := fmt.Sprintf("%*s", wSkipped+1, fmt.Sprintf("∅%d", pkg.Counts.Skipped))
	if pkg.Counts.Skipped > 0 {
		skippedStr = skipColor.Render(skippedStr)
	} else {
		skippedStr = neutralColor.Render(skippedStr)
	}

	total := pkg.Counts.Passed + pkg.Counts.Failed + pkg.Counts.Skipped
	totalStr := neutralColor.Render(fmt.Sprintf("%*d", wTotal, total))

	var elapsedVal string
	currentElapsed := m.packageElapsed(pkg)
	elapsedVal = formatElapsedTime(currentElapsed)
	elapsedStr := fmt.Sprintf("%*s", wElapsed, elapsedVal)
	if running {
		elapsedStr = m.brightStyle.Render(elapsedStr)
	}

	// Running/paused columns only shown for running packages; blank-padded otherwise
	// Display width: icon(1) + digits(wN) per column, plus 1 space between + 1 trailing space
	runPauseWidth := 1 + wRunning + 1 + 1 + wPaused + 1
	var runPausePart string
	if running {
		runningStr := neutralColor.Render(fmt.Sprintf("%*s", wRunning+1, fmt.Sprintf("▶%d", pkg.Counts.Running)))
		pausedStr := neutralColor.Render(fmt.Sprintf("%*s", wPaused+1, fmt.Sprintf("⏸%d", pkg.Counts.Paused)))
		runPausePart = fmt.Sprintf("%s %s ", runningStr, pausedStr)
	} else {
		runPausePart = strings.Repeat(" ", runPauseWidth)
	}

	rightPart = fmt.Sprintf("%s(%s %s %s) %s %s", runPausePart, passedStr, failedStr, skippedStr, totalStr, elapsedStr)
	leftPart = pkg.Name
	if !running && pkg.SummaryLine != "" {
		leftPart = expandTabs(stripSummaryStatusWord(pkg.SummaryLine), 8)
	}

	// Running/interrupted packages keep their bright highlight so the active
	// package stands out. Finished packages (passed/failed/skipped) leave the
	// name+info in the terminal's default foreground color; the colored gutter
	// icon alone conveys status. Failures in a running package are signaled
	// by the spinner flipping to red (see getStatusPrefix); the name stays
	// bright white so the active package reads consistently.
	switch pkg.Status {
	case results.StatusRunning, results.StatusInterrupted:
		leftPart = m.brightStyle.Render(leftPart)
		rightPart = m.brightStyle.Render(rightPart)
	}

	// Prefix uses a colored gutter icon for both running and finished packages so
	// the package name aligns at column 3 across all states.
	prefix := m.getStatusPrefix(pkg.Status, pkg.Counts.Failed > 0)

	m.renderAlignedLine(b, leftPart, rightPart, prefix)
}

// stripSummaryStatusWord removes the leading go test status token ("ok",
// "FAIL", or "?") and the whitespace that follows it from a package summary
// line. The gutter icon conveys the status visually, so repeating it as text
// would be redundant. The remaining text (package name, duration, "[no test
// files]", etc.) is returned untouched so that subsequent tab expansion keeps
// the original column layout.
func stripSummaryStatusWord(summary string) string {
	// Trim only leading spaces; preserve tabs so expandTabs can still align
	// the rest of the line.
	trimmed := strings.TrimLeft(summary, " ")
	for _, prefix := range []string{"FAIL", "ok", "?"} {
		if strings.HasPrefix(trimmed, prefix) {
			rest := trimmed[len(prefix):]
			// The go test runner always separates the status word from the
			// package name with at least one whitespace character (usually a
			// tab). Only strip if that separator is present, otherwise leave
			// the line alone (e.g. a package literally named "ok-foo").
			if len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
				return strings.TrimLeft(rest, " \t")
			}
		}
	}
	return summary
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
		summary = m.brightStyle.Render(summary)
		elapsedVal = m.brightStyle.Render(elapsedVal)
	} else {
		style := m.testStyle(test)
		if style != nil {
			summary = style.Render(summary)
			elapsedVal = style.Render(elapsedVal)
		}
	}

	m.renderAlignedLine(b, summary, elapsedVal, prefix)
	maxLines--

	// Render output lines
	l := len(test.Output)
	if l > MaxOutputLines {
		l = MaxOutputLines
	}
	logIndent := prefix + testIndent(test.Name)
	for _, outputLine := range test.Output[len(test.Output)-l:] {
		if maxLines <= 0 {
			break
		}
		line := logIndent + outputLine
		b.WriteString(ensureReset(truncateLine(line, m.TerminalWidth)))
		b.WriteString("\n")

		maxLines--
	}
}

func (m *Model) testStyle(test *results.TestResult) *lipgloss.Style {
	switch test.Status {
	case results.StatusFailed:
		return &m.failStyle
	case results.StatusSkipped:
		return &m.skipStyle
	case results.StatusPassed:
		if m.SlowThreshold > 0 && test.Elapsed >= m.SlowThreshold {
			return &m.slowStyle
		}
	}
	return nil
}

// formatTestSummary formats the test summary line (left part)
func (m *Model) formatTestSummary(test *results.TestResult) string {
	indent := testIndent(test.Name)
	if test.SummaryLine != "" {
		return indent + test.SummaryLine
	}
	return fmt.Sprintf("%s=== RUN   %s", indent, test.Name)
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

// renderSummaryLine renders the top summary line
func (m *Model) renderSummaryLine(b *strings.Builder, run *results.Run, wRunning, wPaused, wPassed, wFailed, wSkipped, wTotal, wElapsed int) {
	var leftPart string
	var rightPart string

	running := run.Status == results.StatusRunning

	totalPkgs := len(run.PackageOrder)
	donePkgs := totalPkgs - run.RunningPkgs
	if running {
		leftPart = fmt.Sprintf("(%d packages: %d running, %d done)", totalPkgs, run.RunningPkgs, donePkgs)
	} else {
		var statusLabel string
		switch run.Status {
		case results.StatusFailed:
			statusLabel = "FAILED"
		case results.StatusPassed:
			statusLabel = "PASSED"
		case results.StatusInterrupted:
			statusLabel = "INTERRUPTED"
		default:
			statusLabel = "UNKNOWN"
		}
		leftPart = statusLabel
	}

	passColor, failColor, skipColor, neutralColor := m.passStyle, m.failStyle, m.skipStyle, m.neutralStyle
	if running {
		passColor, failColor, skipColor, neutralColor = m.brightPass, m.brightFail, m.brightSkip, m.brightNeutral
	}

	passedStr := fmt.Sprintf("%*s", wPassed+1, fmt.Sprintf("✓%d", run.Counts.Passed))
	if run.Counts.Passed > 0 {
		passedStr = passColor.Render(passedStr)
	} else {
		passedStr = neutralColor.Render(passedStr)
	}

	failedStr := fmt.Sprintf("%*s", wFailed+1, fmt.Sprintf("✗%d", run.Counts.Failed))
	if run.Counts.Failed > 0 {
		failedStr = failColor.Render(failedStr)
	} else {
		failedStr = neutralColor.Render(failedStr)
	}

	skippedStr := fmt.Sprintf("%*s", wSkipped+1, fmt.Sprintf("∅%d", run.Counts.Skipped))
	if run.Counts.Skipped > 0 {
		skippedStr = skipColor.Render(skippedStr)
	} else {
		skippedStr = neutralColor.Render(skippedStr)
	}

	total := run.Counts.Passed + run.Counts.Failed + run.Counts.Skipped
	totalStr := neutralColor.Render(fmt.Sprintf("%*d", wTotal, total))

	runningStr := neutralColor.Render(fmt.Sprintf("%*s", wRunning+1, fmt.Sprintf("▶%d", run.Counts.Running)))
	pausedStr := neutralColor.Render(fmt.Sprintf("%*s", wPaused+1, fmt.Sprintf("⏸%d", run.Counts.Paused)))

	elapsedVal := formatElapsedTime(m.runElapsed(run))
	elapsedStr := fmt.Sprintf("%*s", wElapsed, elapsedVal)

	rightPart = fmt.Sprintf("%s %s (%s %s %s) %s %s", runningStr, pausedStr, passedStr, failedStr, skippedStr, totalStr, elapsedStr)

	prefix := m.getStatusPrefix(run.Status, run.Counts.Failed > 0)
	if running {
		leftPart = m.brightStyle.Render(leftPart)
		rightPart = m.brightStyle.Render(rightPart)
	}

	m.renderAlignedLine(b, leftPart, rightPart, prefix)
}
