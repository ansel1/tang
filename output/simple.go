package output

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/results"
)

type SimpleOutput struct {
	writer         io.Writer
	collector      *results.Collector
	slowThreshold  time.Duration
	summaryOptions format.SummaryOptions
	verbose        bool
	width          int

	// Per-event state (initialized by Init, used by ProcessEvent)
	writers                   map[string]*packageWriter
	pkgSummaryLine            map[string]string
	lastActiveTest            map[string]string
	pendingResults            map[string]map[string][]pendingResult
	pendingNonVerboseFailures map[string]map[string][]pendingNonVerboseFailure
	focusedPkg                string
	completedQueue            []string
}

type pendingResult struct {
	test string
	line string
}

type pendingNonVerboseFailure struct {
	test  string
	lines []string
}

// packageWriter buffers output for a single package. When direct is non-nil
// (the package is "focused"), output goes straight to the writer instead of
// being buffered.
type packageWriter struct {
	buf    []string
	direct io.Writer
}

func (pw *packageWriter) appendLine(line string) {
	if pw.direct != nil {
		_, _ = fmt.Fprint(pw.direct, line)
	} else {
		pw.buf = append(pw.buf, line)
	}
}

func (pw *packageWriter) flush(w io.Writer) {
	for _, line := range pw.buf {
		_, _ = fmt.Fprint(w, line)
	}
	pw.buf = pw.buf[:0]
}

func (pw *packageWriter) buffered() int {
	return len(pw.buf)
}

func getWriter(writers map[string]*packageWriter, pkg string) *packageWriter {
	if w, ok := writers[pkg]; ok {
		return w
	}
	w := &packageWriter{}
	writers[pkg] = w
	return w
}

// pickFocus selects the package with the most buffered output.
func pickFocus(writers map[string]*packageWriter) string {
	var best string
	var bestLen int
	for pkg, w := range writers {
		if n := w.buffered(); n > bestLen {
			best = pkg
			bestLen = n
		}
	}
	return best
}

func NewSimpleOutput(w io.Writer, collector *results.Collector, slowThreshold time.Duration, summaryOptions format.SummaryOptions, verbose bool, width int) *SimpleOutput {
	if width <= 0 {
		width = 80
	}
	return &SimpleOutput{
		writer:         w,
		collector:      collector,
		slowThreshold:  slowThreshold,
		summaryOptions: summaryOptions,
		verbose:        verbose,
		width:          width,
	}
}

// Init initializes the per-event processing state. Must be called before
// ProcessEvent. It is called automatically by ProcessEvents.
func (s *SimpleOutput) Init() {
	s.writers = make(map[string]*packageWriter)
	s.pkgSummaryLine = make(map[string]string)
	s.lastActiveTest = make(map[string]string)
	s.pendingResults = make(map[string]map[string][]pendingResult)
	s.pendingNonVerboseFailures = make(map[string]map[string][]pendingNonVerboseFailure)
	s.focusedPkg = ""
	s.completedQueue = nil
}

// ProcessEvent handles a single engine event. It does NOT call
// collector.Push — the caller is responsible for that. This allows
// ProcessEvent to be used in live mode where the main loop already
// pushes events to the collector.
func (s *SimpleOutput) ProcessEvent(evt engine.Event) {
	switch evt.Type {
	case engine.EventRawLine:
		_, _ = fmt.Fprint(s.writer, string(evt.RawLine))

	case engine.EventBuild:
		if evt.BuildEvent.Action == "build-output" && evt.BuildEvent.Output != "" {
			_, _ = fmt.Fprint(s.writer, evt.BuildEvent.Output)
		}

	case engine.EventTest:
		te := evt.TestEvent
		if te.Test != "" {
			if s.verbose {
				if te.Action == "output" && te.Output != "" {
					s.handleVerboseTestOutput(te, s.writers, s.lastActiveTest, s.pendingResults)
				}
			} else if te.Action == "fail" {
				s.handleNonVerboseTestFailure(te, s.writers)
			}
		} else {
			completedPkg := s.handlePackageLevelEvent(te, s.writers, s.pkgSummaryLine)
			if completedPkg != "" {
				s.completedQueue = append(s.completedQueue, completedPkg)
			}
		}
	}

	// Pick initial focus: choose the package with most buffered output
	if s.focusedPkg == "" {
		s.focusedPkg = pickFocus(s.writers)
		if s.focusedPkg != "" {
			s.writers[s.focusedPkg].flush(s.writer)
			s.writers[s.focusedPkg].direct = s.writer
		}
	}

	// Check if focused package completed
	if s.focusedPkg != "" {
		if idx := slices.Index(s.completedQueue, s.focusedPkg); idx >= 0 {
			// Write focused package's summary line
			if line, ok := s.pkgSummaryLine[s.focusedPkg]; ok {
				_, _ = fmt.Fprint(s.writer, line)
				delete(s.pkgSummaryLine, s.focusedPkg)
			}
			delete(s.writers, s.focusedPkg)

			// Remove focused pkg from queue
			s.completedQueue = slices.Delete(s.completedQueue, idx, idx+1)

			// Flush all other completed packages (dump their buffers)
			for _, pkg := range s.completedQueue {
				s.flushPackage(pkg, s.writers, s.pkgSummaryLine)
			}
			s.completedQueue = s.completedQueue[:0]

			// Pick new focus
			s.focusedPkg = pickFocus(s.writers)
			if s.focusedPkg != "" {
				s.writers[s.focusedPkg].flush(s.writer)
				s.writers[s.focusedPkg].direct = s.writer
			}
		}
	}
}

// Flush emits any remaining buffered package output. Call this after all
// events have been processed (e.g., when the event stream ends or the
// run finishes). It does NOT write the summary.
func (s *SimpleOutput) Flush() {
	for _, pkg := range s.completedQueue {
		s.flushPackage(pkg, s.writers, s.pkgSummaryLine)
	}
	s.completedQueue = s.completedQueue[:0]
}

// ProcessEvents consumes engine events and writes output progressively.
//
// One package at a time is "focused": its output streams incrementally to
// stdout while other packages buffer. When the focused package finishes,
// any other completed packages are flushed, then a new focus is picked
// from the remaining running packages (the one with the most buffered
// output).
//
// In verbose mode, all test output is streamed for the focused package.
// In non-verbose mode, test failure output is streamed as each test fails.
func (s *SimpleOutput) ProcessEvents(events <-chan engine.Event) error {
	s.Init()

	for evt := range events {
		s.collector.Push(evt)
		s.ProcessEvent(evt)
	}

	s.Flush()
	return s.writeSummary()
}

func (s *SimpleOutput) handlePackageLevelEvent(
	te parser.TestEvent,
	writers map[string]*packageWriter,
	pkgSummaryLine map[string]string,
) (completedPkg string) {
	switch te.Action {
	case "output":
		if te.Output != "" {
			trimmed := strings.TrimSpace(te.Output)
			// Package summary lines contain a tab (e.g. "ok\tpkg\ttime", "FAIL\tpkg\ttime").
			// Standalone "PASS" or "FAIL" lines (no tab) are regular output.
			isSummaryLine := strings.ContainsRune(trimmed, '\t') &&
				(strings.HasPrefix(trimmed, "ok") ||
					strings.HasPrefix(trimmed, "FAIL") ||
					strings.HasPrefix(trimmed, "?"))
			if isSummaryLine {
				pkgSummaryLine[te.Package] = te.Output
			} else if s.verbose {
				getWriter(writers, te.Package).appendLine(te.Output)
			}
		}

	case "pass", "fail", "skip":
		completedPkg = te.Package
	}
	return
}

func (s *SimpleOutput) flushPackage(
	pkgName string,
	writers map[string]*packageWriter,
	pkgSummaryLine map[string]string,
) {
	if w := writers[pkgName]; w != nil {
		w.flush(s.writer)
	}

	if line, ok := pkgSummaryLine[pkgName]; ok {
		_, _ = fmt.Fprint(s.writer, line)
	}

	delete(pkgSummaryLine, pkgName)
	delete(writers, pkgName)
}

// handleNonVerboseTestFailure formats and writes a single test's failure
// output through the packageWriter when the test's "fail" event arrives.
// Subtest failures are buffered and emitted under their parent test to
// match the ordering of go test output (parent first, then subtests).
func (s *SimpleOutput) handleNonVerboseTestFailure(
	te parser.TestEvent,
	writers map[string]*packageWriter,
) {
	state := s.collector.State()
	if len(state.Runs) == 0 {
		return
	}
	run := state.Runs[len(state.Runs)-1]
	testKey := te.Package + "/" + te.Test
	tr, ok := run.TestResults[testKey]
	if !ok {
		return
	}

	// Format the output lines for this test
	depth := strings.Count(te.Test, "/")
	indent := strings.Repeat("    ", depth)
	var lines []string
	if tr.SummaryLine != "" {
		lines = append(lines, fmt.Sprintf("%s%s\n", indent, tr.SummaryLine))
	}
	for _, line := range tr.Output {
		lines = append(lines, fmt.Sprintf("%s%s\n", indent, line))
	}

	if strings.Contains(te.Test, "/") {
		// Subtest: buffer under immediate parent for grouped emission
		parent := parentTest(te.Test)
		if s.pendingNonVerboseFailures[te.Package] == nil {
			s.pendingNonVerboseFailures[te.Package] = make(map[string][]pendingNonVerboseFailure)
		}
		s.pendingNonVerboseFailures[te.Package][parent] = append(
			s.pendingNonVerboseFailures[te.Package][parent],
			pendingNonVerboseFailure{test: te.Test, lines: lines},
		)
	} else {
		// Top-level test: emit now, then emit buffered children
		w := getWriter(writers, te.Package)
		for _, line := range lines {
			w.appendLine(line)
		}
		s.emitNonVerboseChildren(writers, te.Package, te.Test)
	}
}

// emitNonVerboseChildren recursively emits buffered subtest failures
// under their parent, ensuring parent-first ordering.
func (s *SimpleOutput) emitNonVerboseChildren(
	writers map[string]*packageWriter,
	pkg string,
	parentTestName string,
) {
	pending := s.pendingNonVerboseFailures[pkg]
	if pending == nil {
		return
	}
	children := pending[parentTestName]
	delete(pending, parentTestName)
	w := getWriter(writers, pkg)
	for _, child := range children {
		for _, line := range child.lines {
			w.appendLine(line)
		}
		s.emitNonVerboseChildren(writers, pkg, child.test)
	}
}

func (s *SimpleOutput) writeSummary() error {
	if s.collector == nil {
		return nil
	}

	state := s.collector.State()
	if len(state.Runs) == 0 {
		return nil
	}

	run := state.Runs[len(state.Runs)-1]
	summary := format.ComputeSummary(run, s.slowThreshold)
	if summary == nil {
		return nil
	}

	summaryText := format.NewSummaryFormatter(s.width, s.summaryOptions).Format(summary)
	if summary.HasTestDetailsWithOptions(s.summaryOptions) {
		_, _ = fmt.Fprintln(s.writer)
	}
	_, _ = fmt.Fprintln(s.writer, summaryText)

	return nil
}

// handleVerboseTestOutput processes a test-level output event in verbose mode.
// It reconstructs go test -v formatting by:
//   - Injecting "=== NAME" lines when output switches between parallel tests
//   - Grouping subtest results under their parent with proper indentation
func (s *SimpleOutput) handleVerboseTestOutput(
	te parser.TestEvent,
	writers map[string]*packageWriter,
	lastActiveTest map[string]string,
	pendingResults map[string]map[string][]pendingResult,
) {
	w := getWriter(writers, te.Package)
	trimmed := strings.TrimSpace(te.Output)

	// Test result lines (--- PASS/FAIL/SKIP:) need special handling for grouping
	if isResultLine(trimmed) {
		if strings.Contains(te.Test, "/") {
			// Subtest result: buffer under parent for later grouped emission
			parent := parentTest(te.Test)
			if pendingResults[te.Package] == nil {
				pendingResults[te.Package] = make(map[string][]pendingResult)
			}
			pendingResults[te.Package][parent] = append(
				pendingResults[te.Package][parent],
				pendingResult{test: te.Test, line: te.Output},
			)
		} else {
			// Top-level test result: emit with grouped children
			w.appendLine(te.Output)
			if pr := pendingResults[te.Package]; pr != nil {
				emitGroupedChildren(writers, te.Package, te.Test, pr, 1)
			}
			lastActiveTest[te.Package] = te.Test
		}
		return
	}

	// Regular output: inject === NAME when switching between parallel tests
	last := lastActiveTest[te.Package]
	if last != "" && last != te.Test && !isContextSwitchLine(trimmed) {
		w.appendLine(fmt.Sprintf("=== NAME  %s\n", te.Test))
	}
	lastActiveTest[te.Package] = te.Test
	w.appendLine(te.Output)
}

func emitGroupedChildren(
	writers map[string]*packageWriter,
	pkg string,
	parentTestName string,
	pr map[string][]pendingResult,
	depth int,
) {
	children := pr[parentTestName]
	delete(pr, parentTestName)
	for _, child := range children {
		indent := strings.Repeat("    ", depth)
		writers[pkg].appendLine(indent + child.line)
		emitGroupedChildren(writers, pkg, child.test, pr, depth+1)
	}
}

func isResultLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "--- PASS:") ||
		strings.HasPrefix(trimmed, "--- FAIL:") ||
		strings.HasPrefix(trimmed, "--- SKIP:")
}

func isContextSwitchLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "=== RUN") ||
		strings.HasPrefix(trimmed, "=== CONT") ||
		strings.HasPrefix(trimmed, "=== PAUSE")
}

func parentTest(test string) string {
	if i := strings.LastIndex(test, "/"); i >= 0 {
		return test[:i]
	}
	return ""
}

// HasFailures returns true if any tests failed.
func (s *SimpleOutput) HasFailures() bool {
	if s.collector == nil {
		return false
	}

	state := s.collector.State()
	for _, run := range state.Runs {
		if run.Counts.Failed > 0 {
			return true
		}
		for _, pkg := range run.Packages {
			if pkg.Status == results.StatusFailed {
				return true
			}
		}
	}

	return false
}
