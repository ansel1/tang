package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ansel1/tang/output/format"
	"github.com/ansel1/tang/results"
)

// SimpleOutput writes simple text output for -notty mode
// It accumulates output and displays summary at completion using shared collector
type SimpleOutput struct {
	writer    io.Writer
	output    []string
	collector *results.Collector

	// Lightweight counters for exit code determination
	failed int
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
func (s *SimpleOutput) ProcessEvents(events <-chan results.Event) error {
	for evt := range events {
		switch evt.Type {
		case results.EventRawOutput:
			// Accumulate raw (non-JSON) lines for output
			s.output = append(s.output, string(evt.RawLine))

		case results.EventTestOutput:
			// Accumulate test output
			s.output = append(s.output, strings.TrimRight(evt.Output, "\n"))

		case results.EventNonTestOutput:
			// Accumulate non-test output
			s.output = append(s.output, strings.TrimRight(evt.Output, "\n"))

		case results.EventTestUpdated:
			// Track failures for exit code
			// We need to check status from the event or collector?
			// The event doesn't carry status.
			// But we can check collector.
			// Or we can just rely on collector state at the end?
			// `HasFailures` is called at the end.
			// So we don't strictly need to track `failed` count here if we can query collector later.
			// But `HasFailures` is a method on `SimpleOutput`.
			// If we use collector, we can implement `HasFailures` by querying collector.
			// So we can remove `failed` field.

		case results.EventRunFinished:
			// We could write output here if we wanted to stream per run?
			// But requirement is to write at the end.
			// Wait, `ProcessEvents` returns when channel closes.
			// So we write at the end of function.
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
		s.collector.WithState(func(state *results.State) {
			if len(state.Runs) == 0 {
				return
			}
			run := state.Runs[len(state.Runs)-1]

			// Compute summary with 10 second slow test threshold
			summary = format.ComputeSummary(run, 10*time.Second)
		})

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
	s.collector.WithState(func(state *results.State) {
		for _, run := range state.Runs {
			for _, pkg := range run.Packages {
				if pkg.Counts.Failed > 0 || pkg.Status == results.StatusFailed {
					hasFailures = true
					return
				}
			}
		}
	})
	return hasFailures
}
