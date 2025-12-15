package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ansel1/tang/engine"
	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/results"
)

// SimpleOutput writes simple text output for -notty mode
// It accumulates output and displays summary at completion using shared collector
type SimpleOutput struct {
	writer    io.Writer
	output    []string
	collector *results.Collector
}

// NewSimpleOutput creates a simple output writer
func NewSimpleOutput(w io.Writer, collector *results.Collector) *SimpleOutput {
	return &SimpleOutput{
		writer:    w,
		output:    make([]string, 0),
		collector: collector,
	}
}

// ProcessEvents consumes events from channel and writes to output
func (s *SimpleOutput) ProcessEvents(events <-chan engine.Event) error {
	for evt := range events {
		// Update collector state
		s.collector.Push(evt)

		switch evt.Type {
		case engine.EventRawLine:
			// Accumulate raw (non-JSON) lines for output
			s.output = append(s.output, string(evt.RawLine))

		case engine.EventTest:
			// Accumulate test output
			// Only capture if there is actual output content
			if evt.TestEvent.Output != "" {
				s.output = append(s.output, strings.TrimRight(evt.TestEvent.Output, "\n"))
			}

		case engine.EventError:
			// Optionally handle error events
		}
	}

	// Write all accumulated output and summary
	return s.writeOutput()
}

// writeOutput writes all accumulated output and summary
func (s *SimpleOutput) writeOutput() error {
	// Write all output lines first
	for _, line := range s.output {
		if _, err := fmt.Fprintln(s.writer, line); err != nil {
			return err
		}
	}

	// Display summary using shared collector and formatter
	if s.collector != nil {
		var summary *format.Summary

		state := s.collector.State()
		if len(state.Runs) > 0 {
			run := state.Runs[len(state.Runs)-1]
			// Compute summary with 10 second slow test threshold
			summary = format.ComputeSummary(run, 10*time.Second)
		}

		if summary != nil {
			// Format summary using default terminal width (80 columns)
			formatter := format.NewSummaryFormatter(80)
			summaryText := formatter.Format(summary)

			// Print summary
			fmt.Fprintln(s.writer)
			fmt.Fprintln(s.writer, summaryText)
		}
	}

	return nil
}

// HasFailures returns true if any tests failed
func (s *SimpleOutput) HasFailures() bool {
	if s.collector == nil {
		return false
	}
	hasFailures := false

	state := s.collector.State()
	for _, run := range state.Runs {
		// Check run counts
		if run.Counts.Failed > 0 {
			hasFailures = true
			break
		}
		// Double check packages just in case
		for _, pkg := range run.Packages {
			if pkg.Status == results.StatusFailed {
				hasFailures = true
				break
			}
		}
		if hasFailures {
			break
		}
	}

	return hasFailures
}
