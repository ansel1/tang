package output

import (
	"fmt"
	"io"
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
// Both modes buffer per-package and flush atomically on package completion.
// Verbose mode preserves the arrival order of output within each package,
// so parallel test output is interleaved exactly as go test -v would show it.
// Non-verbose mode prints only failed test output and package status lines
// (like go test ./...).
func (s *SimpleOutput) ProcessEvents(events <-chan engine.Event) error {
	testOutput := make(map[string][]string)
	pkgOutput := make(map[string][]string)
	pkgSummaryLine := make(map[string]string)

	// State for verbose mode: reconstruct go test -v formatting from JSON events
	lastActiveTest := make(map[string]string)                    // pkg -> last test that produced visible output
	pendingResults := make(map[string]map[string][]pendingResult) // pkg -> parent test -> buffered child results

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
				if te.Action == "output" && te.Output != "" {
					if s.verbose {
						s.handleVerboseTestOutput(te, pkgOutput, lastActiveTest, pendingResults)
					} else {
						testKey := te.Package + "/" + te.Test
						testOutput[testKey] = append(testOutput[testKey], te.Output)
					}
				}
			} else {
				s.handlePackageLevelEvent(te, testOutput, pkgOutput, pkgSummaryLine)
			}
		}
	}

	return s.writeSummary()
}

func (s *SimpleOutput) handlePackageLevelEvent(
	te parser.TestEvent,
	testOutput map[string][]string,
	pkgOutput map[string][]string,
	pkgSummaryLine map[string]string,
) {
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
				pkgOutput[te.Package] = append(pkgOutput[te.Package], te.Output)
			}
		}

	case "pass", "fail", "skip":
		s.flushPackage(te.Package, testOutput, pkgOutput, pkgSummaryLine)
	}
}

func (s *SimpleOutput) flushPackage(
	pkgName string,
	testOutput map[string][]string,
	pkgOutput map[string][]string,
	pkgSummaryLine map[string]string,
) {
	state := s.collector.State()
	var run *results.Run
	if len(state.Runs) > 0 {
		run = state.Runs[len(state.Runs)-1]
	}

	if s.verbose {
		for _, line := range pkgOutput[pkgName] {
			_, _ = fmt.Fprint(s.writer, line)
		}
	} else if run != nil {
		if pkg := run.Packages[pkgName]; pkg != nil {
			s.writeFailedTestOutput(run, pkg, testOutput)
		}
	}

	if line, ok := pkgSummaryLine[pkgName]; ok {
		_, _ = fmt.Fprint(s.writer, line)
	}

	delete(pkgSummaryLine, pkgName)
	delete(pkgOutput, pkgName)
}

func (s *SimpleOutput) writeFailedTestOutput(
	run *results.Run,
	pkg *results.PackageResult,
	testOutput map[string][]string,
) {
	for _, testName := range pkg.TestOrder {
		testKey := pkg.Name + "/" + testName
		tr, ok := run.TestResults[testKey]
		if !ok || tr.Status != results.StatusFailed {
			delete(testOutput, testKey)
			continue
		}

		depth := strings.Count(testName, "/")
		indent := strings.Repeat("    ", depth)

		if tr.SummaryLine != "" {
			_, _ = fmt.Fprintf(s.writer, "%s%s\n", indent, tr.SummaryLine)
		}
		for _, line := range tr.Output {
			_, _ = fmt.Fprintf(s.writer, "%s%s\n", indent, line)
		}

		delete(testOutput, testKey)
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
	pkgOutput map[string][]string,
	lastActiveTest map[string]string,
	pendingResults map[string]map[string][]pendingResult,
) {
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
			pkgOutput[te.Package] = append(pkgOutput[te.Package], te.Output)
			if pr := pendingResults[te.Package]; pr != nil {
				emitGroupedChildren(pkgOutput, te.Package, te.Test, pr, 1)
			}
			lastActiveTest[te.Package] = te.Test
		}
		return
	}

	// Regular output: inject === NAME when switching between parallel tests
	last := lastActiveTest[te.Package]
	if last != "" && last != te.Test && !isContextSwitchLine(trimmed) {
		pkgOutput[te.Package] = append(pkgOutput[te.Package],
			fmt.Sprintf("=== NAME  %s\n", te.Test))
	}
	lastActiveTest[te.Package] = te.Test
	pkgOutput[te.Package] = append(pkgOutput[te.Package], te.Output)
}

func emitGroupedChildren(
	pkgOutput map[string][]string,
	pkg string,
	parentTestName string,
	pr map[string][]pendingResult,
	depth int,
) {
	children := pr[parentTestName]
	delete(pr, parentTestName)
	for _, child := range children {
		indent := strings.Repeat("    ", depth)
		pkgOutput[pkg] = append(pkgOutput[pkg], indent+child.line)
		emitGroupedChildren(pkgOutput, pkg, child.test, pr, depth+1)
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
