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
}

type pendingResult struct {
	test string
	line string
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

func NewSimpleOutput(w io.Writer, collector *results.Collector, slowThreshold time.Duration, summaryOptions format.SummaryOptions, verbose bool) *SimpleOutput {
	return &SimpleOutput{
		writer:         w,
		collector:      collector,
		slowThreshold:  slowThreshold,
		summaryOptions: summaryOptions,
		verbose:        verbose,
	}
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
	writers := make(map[string]*packageWriter)
	pkgSummaryLine := make(map[string]string)

	// State for verbose mode: reconstruct go test -v formatting from JSON events
	lastActiveTest := make(map[string]string)                    // pkg -> last test that produced visible output
	pendingResults := make(map[string]map[string][]pendingResult) // pkg -> parent test -> buffered child results

	// Streaming state: one focused package streams incrementally, others buffer
	var focusedPkg string
	var completedQueue []string

	for evt := range events {
		s.collector.Push(evt)

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
						s.handleVerboseTestOutput(te, writers, lastActiveTest, pendingResults)
					}
				} else if te.Action == "fail" {
					s.handleNonVerboseTestFailure(te, writers)
				}
			} else {
				completedPkg := s.handlePackageLevelEvent(te, writers, pkgSummaryLine)
				if completedPkg != "" {
					completedQueue = append(completedQueue, completedPkg)
				}
			}
		}

		// Pick initial focus: choose the package with most buffered output
		if focusedPkg == "" {
			focusedPkg = pickFocus(writers)
			if focusedPkg != "" {
				writers[focusedPkg].flush(s.writer)
				writers[focusedPkg].direct = s.writer
			}
		}

		// Check if focused package completed
		if focusedPkg != "" {
			if idx := slices.Index(completedQueue, focusedPkg); idx >= 0 {
				// Write focused package's summary line
				if line, ok := pkgSummaryLine[focusedPkg]; ok {
					_, _ = fmt.Fprint(s.writer, line)
					delete(pkgSummaryLine, focusedPkg)
				}
				delete(writers, focusedPkg)

				// Remove focused pkg from queue
				completedQueue = slices.Delete(completedQueue, idx, idx+1)

				// Flush all other completed packages (dump their buffers)
				for _, pkg := range completedQueue {
					s.flushPackage(pkg, writers, pkgSummaryLine)
				}
				completedQueue = completedQueue[:0]

				// Pick new focus
				focusedPkg = pickFocus(writers)
				if focusedPkg != "" {
					writers[focusedPkg].flush(s.writer)
					writers[focusedPkg].direct = s.writer
				}
			}
		}
	}

	// Flush any remaining completed packages (e.g., stream interrupted)
	for _, pkg := range completedQueue {
		s.flushPackage(pkg, writers, pkgSummaryLine)
	}

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

	w := getWriter(writers, te.Package)
	depth := strings.Count(te.Test, "/")
	indent := strings.Repeat("    ", depth)

	if tr.SummaryLine != "" {
		w.appendLine(fmt.Sprintf("%s%s\n", indent, tr.SummaryLine))
	}
	for _, line := range tr.Output {
		w.appendLine(fmt.Sprintf("%s%s\n", indent, line))
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

	summaryText := format.NewSummaryFormatter(80, s.summaryOptions).Format(summary)
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
