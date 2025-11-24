package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/parser"
	"github.com/ansel1/tang/tui"
)

// SimpleOutput writes simple text output for -notty mode
// It accumulates output and displays summary at completion using shared summary collector
type SimpleOutput struct {
	writer           io.Writer
	output           []string
	summaryCollector *tui.SummaryCollector

	// Lightweight counters for exit code determination
	failed int
}

// NewSimpleOutput creates a simple output writer
func NewSimpleOutput(w io.Writer, summaryCollector *tui.SummaryCollector) *SimpleOutput {
	return &SimpleOutput{
		writer:           w,
		output:           make([]string, 0),
		summaryCollector: summaryCollector,
	}
}

// ProcessEvents consumes events from channel and writes to output
func (s *SimpleOutput) ProcessEvents(events <-chan engine.Event) error {
	for evt := range events {
		switch evt.Type {
		case engine.EventRawLine:
			// Accumulate raw (non-JSON) lines for output
			s.output = append(s.output, string(evt.RawLine))

		case engine.EventTest:
			// Track failures for exit code
			s.handleTestEvent(evt.TestEvent)

		case engine.EventComplete:
			// Write all accumulated output and summary
			return s.writeOutput()

		case engine.EventError:
			// Write error to stderr would be better, but we'll add to output
			s.output = append(s.output, fmt.Sprintf("Error: %v", evt.Error))
		}
	}
	return nil
}

// handleTestEvent processes a test event and updates state
func (s *SimpleOutput) handleTestEvent(evt parser.TestEvent) {
	// Track failures for exit code determination
	if evt.Action == "fail" && evt.Test != "" {
		s.failed++
	}

	// Accumulate output lines
	if evt.Output != "" {
		s.output = append(s.output, strings.TrimRight(evt.Output, "\n"))
	}
}

// writeOutput writes all accumulated output and summary
func (s *SimpleOutput) writeOutput() error {
	// Write all output lines first
	for _, line := range s.output {
		if _, err := fmt.Fprintln(s.writer, line); err != nil {
			return err
		}
	}

	// Display summary using shared summary collector and formatter
	if s.summaryCollector != nil {
		// Compute summary with 10 second slow test threshold
		summary := tui.ComputeSummary(s.summaryCollector, 10*time.Second)

		// Format summary using default terminal width (80 columns)
		formatter := tui.NewSummaryFormatter(80)
		summaryText := formatter.Format(summary)

		// Print summary
		fmt.Fprintln(s.writer)
		fmt.Fprintln(s.writer, summaryText)
	}

	return nil
}

// HasFailures returns true if any tests failed
func (s *SimpleOutput) HasFailures() bool {
	return s.failed > 0
}
